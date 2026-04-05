// Package installer handles the lifecycle of MTProxy on the host system:
// detecting the OS/arch, downloading the binary, creating the system user,
// writing the systemd unit, and managing the service via systemctl.
package installer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"
)

// ---------------------------------------------------------------------------
// Constants / defaults
// ---------------------------------------------------------------------------

const (
	MTProxyGitHubOrg  = "TelegramMessenger"
	MTProxyGitHubRepo = "MTProxy"

	MTProxyInstallDir = "/opt/mtproxy"
	MTProxyBinaryName = "mtproto-proxy"
	MTProxyUser       = "mtproxy"
	MTProxyGroup      = "mtproxy"

	MTProxyHTTPPort = 8888 // internal management port
	MTProxyWorkers  = 1    // default worker count

	ProxySecretURL = "https://core.telegram.org/getProxySecret"
	ProxyConfigURL = "https://core.telegram.org/getProxyConfig"

	ServiceName        = "mtproxy"
	ServiceUnitPath    = "/etc/systemd/system/mtproxy.service"
	UpdateServicePath  = "/etc/systemd/system/mtproxy-update.service"
	UpdateTimerPath    = "/etc/systemd/system/mtproxy-update.timer"

	httpTimeout = 30 * time.Second
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Config holds the parameters used to render the systemd unit template.
type Config struct {
	Port     int    // public-facing MTProxy port (e.g. 443)
	HTTPPort int    // internal management port (default 8888)
	Secret   string // 32-hex-char secret
	Workers  int    // worker count (default 1)
}

// InstallResult is returned after a successful MTProxy installation.
type InstallResult struct {
	Version string
	Secret  string
	Port    int
	Link    string // tg://proxy?server=...
}

// Installer orchestrates the MTProxy installation and lifecycle.
type Installer struct {
	log    *slog.Logger
	client *http.Client
}

// New creates a new Installer.
func New(log *slog.Logger) *Installer {
	return &Installer{
		log:    log,
		client: &http.Client{Timeout: httpTimeout},
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Install performs a full MTProxy installation.
// It is safe to call on an already-installed system (idempotent).
func (i *Installer) Install(ctx context.Context, cfg Config) (*InstallResult, error) {
	i.log.Info("starting MTProxy installation", "port", cfg.Port)

	if err := i.checkPlatform(); err != nil {
		return nil, fmt.Errorf("platform check: %w", err)
	}

	// Resolve defaults
	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = MTProxyHTTPPort
	}
	if cfg.Workers == 0 {
		cfg.Workers = MTProxyWorkers
	}
	if cfg.Secret == "" {
		s, err := generateSecret()
		if err != nil {
			return nil, fmt.Errorf("generate secret: %w", err)
		}
		cfg.Secret = s
	}

	// 1. Fetch latest release
	tag, downloadURL, err := i.latestReleaseURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	i.log.Info("latest MTProxy release", "tag", tag)

	// 2. Download binary to a temp file
	tmpBin, err := i.downloadBinary(ctx, downloadURL)
	if err != nil {
		return nil, fmt.Errorf("download binary: %w", err)
	}
	defer os.Remove(tmpBin) // cleaned up on success (we moved it) or error

	// 3. System user
	if err := i.ensureUser(MTProxyUser); err != nil {
		return nil, fmt.Errorf("ensure user: %w", err)
	}

	// 4. Directories
	if err := i.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("ensure directories: %w", err)
	}

	// 5. Install binary
	binPath := filepath.Join(MTProxyInstallDir, MTProxyBinaryName)
	if err := i.installBinary(tmpBin, binPath); err != nil {
		return nil, fmt.Errorf("install binary: %w", err)
	}

	// 6. Download Telegram config files
	if err := i.downloadTelegramFiles(ctx); err != nil {
		// Non-fatal: warn and continue; proxy will fail to start without these,
		// but the health check will catch that.
		i.log.Warn("failed to download Telegram config files", "err", err)
	}

	// 7. Write systemd units
	if err := i.writeServiceUnit(cfg); err != nil {
		return nil, fmt.Errorf("write service unit: %w", err)
	}
	if err := i.writeUpdateUnits(); err != nil {
		return nil, fmt.Errorf("write update units: %w", err)
	}

	// 8. Enable and start
	if err := i.enableAndStart(ctx); err != nil {
		return nil, fmt.Errorf("start service: %w", err)
	}

	// 9. Health check
	if err := i.waitHealthy(ctx, cfg.HTTPPort); err != nil {
		// Attempt rollback: stop and disable
		i.log.Error("health check failed — rolling back", "err", err)
		_ = runCommand("systemctl", "stop", ServiceName)
		_ = runCommand("systemctl", "disable", ServiceName)
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	serverIP, _ := getExternalIP(ctx, i.client)

	result := &InstallResult{
		Version: tag,
		Secret:  cfg.Secret,
		Port:    cfg.Port,
		Link:    buildProxyLink(serverIP, cfg.Port, cfg.Secret),
	}

	i.log.Info("MTProxy installed successfully",
		"version", tag, "port", cfg.Port, "link", result.Link)
	return result, nil
}

// Start starts the MTProxy systemd service.
func (i *Installer) Start(ctx context.Context) error {
	i.log.Info("starting MTProxy service")
	return runCommand("systemctl", "start", ServiceName)
}

// Stop stops the MTProxy systemd service.
func (i *Installer) Stop(ctx context.Context) error {
	i.log.Info("stopping MTProxy service")
	return runCommand("systemctl", "stop", ServiceName)
}

// Restart restarts the MTProxy systemd service.
func (i *Installer) Restart(ctx context.Context) error {
	i.log.Info("restarting MTProxy service")
	return runCommand("systemctl", "restart", ServiceName)
}

// Status returns true if the service is currently active.
func (i *Installer) Status() (active bool, err error) {
	out, err := exec.Command("systemctl", "is-active", ServiceName).Output()
	if err != nil {
		// is-active exits non-zero when inactive — that is not an error for us
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "active", nil
}

// Uninstall stops the service, removes the binary and systemd units.
// It does NOT remove the mtproxy system user or /opt/mtproxy directory.
func (i *Installer) Uninstall(_ context.Context) error {
	i.log.Info("uninstalling MTProxy")

	_ = runCommand("systemctl", "stop", ServiceName)
	_ = runCommand("systemctl", "disable", ServiceName)
	_ = runCommand("systemctl", "stop", "mtproxy-update.timer")
	_ = runCommand("systemctl", "disable", "mtproxy-update.timer")

	for _, f := range []string{ServiceUnitPath, UpdateServicePath, UpdateTimerPath} {
		_ = os.Remove(f)
	}
	_ = runCommand("systemctl", "daemon-reload")

	i.log.Info("MTProxy uninstalled")
	return nil
}

// UpdateConfig re-renders and applies a new configuration (e.g. changed port/secret).
func (i *Installer) UpdateConfig(ctx context.Context, cfg Config) error {
	i.log.Info("updating MTProxy configuration")

	if cfg.HTTPPort == 0 {
		cfg.HTTPPort = MTProxyHTTPPort
	}
	if cfg.Workers == 0 {
		cfg.Workers = MTProxyWorkers
	}

	if err := i.writeServiceUnit(cfg); err != nil {
		return fmt.Errorf("rewrite service unit: %w", err)
	}
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := runCommand("systemctl", "restart", ServiceName); err != nil {
		return fmt.Errorf("restart service: %w", err)
	}
	return i.waitHealthy(ctx, cfg.HTTPPort)
}

// ---------------------------------------------------------------------------
// Platform check
// ---------------------------------------------------------------------------

func (i *Installer) checkPlatform() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("MTProxy is only supported on Linux (got %s)", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return nil
	default:
		return fmt.Errorf("unsupported architecture: %s (need amd64 or arm64)", runtime.GOARCH)
	}
}

// ---------------------------------------------------------------------------
// GitHub release resolution
// ---------------------------------------------------------------------------

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func (i *Installer) latestReleaseURL(ctx context.Context) (tag, url string, err error) {
	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest",
		MTProxyGitHubOrg, MTProxyGitHubRepo,
	)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := i.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("GitHub API request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", fmt.Errorf("decode GitHub response: %w", err)
	}
	if rel.TagName == "" {
		return "", "", fmt.Errorf("no tag_name in GitHub release response")
	}

	// Find the correct asset for our arch
	wantSuffix := archSuffix()
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, wantSuffix) {
			return rel.TagName, a.BrowserDownloadURL, nil
		}
	}

	// Fallback: build a conventional URL
	conventionalURL := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s/mtproto-proxy-linux-%s",
		MTProxyGitHubOrg, MTProxyGitHubRepo, rel.TagName, runtime.GOARCH,
	)
	i.log.Warn("no matching asset found — trying conventional URL",
		"want_suffix", wantSuffix, "url", conventionalURL)
	return rel.TagName, conventionalURL, nil
}

func archSuffix() string {
	switch runtime.GOARCH {
	case "arm64":
		return "arm64"
	default:
		return "amd64"
	}
}

// ---------------------------------------------------------------------------
// Binary download
// ---------------------------------------------------------------------------

func (i *Installer) downloadBinary(ctx context.Context, url string) (tmpPath string, err error) {
	i.log.Info("downloading MTProxy binary", "url", url)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := i.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "mtproto-proxy-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name())
		}
	}()

	hasher := sha256.New()
	if _, err = io.Copy(io.MultiWriter(tmp, hasher), resp.Body); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("write binary: %w", err)
	}
	_ = tmp.Close()

	digest := hex.EncodeToString(hasher.Sum(nil))
	i.log.Debug("binary downloaded", "sha256", digest, "path", tmp.Name())

	// Make it executable before moving
	if err = os.Chmod(tmp.Name(), 0755); err != nil {
		return "", fmt.Errorf("chmod binary: %w", err)
	}

	return tmp.Name(), nil
}

// ---------------------------------------------------------------------------
// System user
// ---------------------------------------------------------------------------

func (i *Installer) ensureUser(username string) error {
	// id returns exit code 0 if user exists
	if err := exec.Command("id", username).Run(); err == nil {
		i.log.Debug("system user already exists", "user", username)
		return nil
	}
	i.log.Info("creating system user", "user", username)
	return runCommand("useradd",
		"--system",
		"--no-create-home",
		"--shell", "/sbin/nologin",
		"--comment", "MTProxy service account",
		username,
	)
}

// ---------------------------------------------------------------------------
// Directories
// ---------------------------------------------------------------------------

func (i *Installer) ensureDirectories() error {
	dirs := []struct {
		path  string
		owner string
		mode  os.FileMode
	}{
		{MTProxyInstallDir, MTProxyUser, 0750},
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d.path, d.mode); err != nil {
			return fmt.Errorf("mkdir %s: %w", d.path, err)
		}
		if err := runCommand("chown", fmt.Sprintf("%s:%s", d.owner, d.owner), d.path); err != nil {
			return fmt.Errorf("chown %s: %w", d.path, err)
		}
		i.log.Debug("directory ready", "path", d.path)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Binary installation
// ---------------------------------------------------------------------------

func (i *Installer) installBinary(src, dst string) error {
	// Backup existing binary
	if _, err := os.Stat(dst); err == nil {
		bak := dst + ".bak"
		if err := copyFile(dst, bak); err != nil {
			return fmt.Errorf("backup existing binary: %w", err)
		}
		i.log.Info("existing binary backed up", "backup", bak)
	}

	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("install binary to %s: %w", dst, err)
	}

	if err := os.Chmod(dst, 0755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}

	if err := runCommand("chown", fmt.Sprintf("root:%s", MTProxyGroup), dst); err != nil {
		return fmt.Errorf("chown binary: %w", err)
	}

	i.log.Info("binary installed", "path", dst)
	return nil
}

// ---------------------------------------------------------------------------
// Telegram config files
// ---------------------------------------------------------------------------

func (i *Installer) downloadTelegramFiles(ctx context.Context) error {
	files := []struct {
		url  string
		dest string
	}{
		{ProxySecretURL, filepath.Join(MTProxyInstallDir, "proxy-secret")},
		{ProxyConfigURL, filepath.Join(MTProxyInstallDir, "proxy-multi.conf")},
	}

	for _, f := range files {
		if err := i.downloadFile(ctx, f.url, f.dest); err != nil {
			return fmt.Errorf("download %s: %w", f.url, err)
		}
		if err := runCommand("chown",
			fmt.Sprintf("%s:%s", MTProxyUser, MTProxyGroup), f.dest); err != nil {
			return err
		}
		if err := os.Chmod(f.dest, 0640); err != nil {
			return err
		}
		i.log.Info("downloaded Telegram config", "dest", f.dest)
	}
	return nil
}

func (i *Installer) downloadFile(ctx context.Context, url, dest string) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := i.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), ".dl-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	_ = tmp.Close()
	return os.Rename(tmp.Name(), dest)
}

// ---------------------------------------------------------------------------
// Systemd unit template
// ---------------------------------------------------------------------------

var serviceUnitTemplate = template.Must(template.New("mtproxy.service").Parse(`[Unit]
Description=MTProxy - Telegram MTProto Proxy
Documentation=https://github.com/TelegramMessenger/MTProxy
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=mtproxy
Group=mtproxy
WorkingDirectory=/opt/mtproxy

ExecStart=/opt/mtproxy/mtproto-proxy \
    -u mtproxy \
    -p {{.HTTPPort}} \
    -H {{.Port}} \
    -S {{.Secret}} \
    --aes-pwd /opt/mtproxy/proxy-secret \
    /opt/mtproxy/proxy-multi.conf \
    -M {{.Workers}}

ExecReload=/bin/kill -USR1 $MAINPID

Restart=always
RestartSec=5s
TimeoutStartSec=30s
TimeoutStopSec=15s

StandardOutput=journal
StandardError=journal
SyslogIdentifier=mtproxy

NoNewPrivileges=true
PrivateTmp=true
PrivateDevices=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/mtproxy
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictSUIDSGID=true
RestrictRealtime=true
LockPersonality=true
MemoryDenyWriteExecute=false
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
SystemCallFilter=@system-service @network-io
SystemCallArchitectures=native
SystemCallErrorNumber=EPERM

[Install]
WantedBy=multi-user.target
`))

var updateServiceTemplate = template.Must(template.New("mtproxy-update.service").Parse(`[Unit]
Description=MTProxy Config Update - Download latest proxy-secret and proxy-multi.conf
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=mtproxy
Group=mtproxy
ExecStart=/bin/bash -c '\
  set -euo pipefail; \
  TMP_SECRET=$(mktemp); TMP_CONF=$(mktemp); \
  trap "rm -f $TMP_SECRET $TMP_CONF" EXIT; \
  curl -fsSL --retry 3 --retry-delay 5 {{.ProxySecretURL}} -o "$TMP_SECRET" && \
  curl -fsSL --retry 3 --retry-delay 5 {{.ProxyConfigURL}} -o "$TMP_CONF" && \
  mv "$TMP_SECRET" /opt/mtproxy/proxy-secret && \
  mv "$TMP_CONF" /opt/mtproxy/proxy-multi.conf && \
  chmod 640 /opt/mtproxy/proxy-secret /opt/mtproxy/proxy-multi.conf && \
  systemctl reload-or-restart mtproxy.service \
'
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mtproxy-update
ReadWritePaths=/opt/mtproxy
ProtectSystem=strict
NoNewPrivileges=true
PrivateTmp=true
`))

var updateTimerContent = `[Unit]
Description=MTProxy Config Update Timer
Requires=mtproxy-update.service

[Timer]
OnCalendar=*-*-* 03:00:00
RandomizedDelaySec=3600
OnBootSec=2min
AccuracySec=10min
Persistent=true

[Install]
WantedBy=timers.target
`

func (i *Installer) writeServiceUnit(cfg Config) error {
	var buf bytes.Buffer
	if err := serviceUnitTemplate.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("render service unit: %w", err)
	}
	if err := atomicWrite(ServiceUnitPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write %s: %w", ServiceUnitPath, err)
	}
	i.log.Info("service unit written", "path", ServiceUnitPath)
	return nil
}

func (i *Installer) writeUpdateUnits() error {
	type updateVars struct {
		ProxySecretURL string
		ProxyConfigURL string
	}
	vars := updateVars{ProxySecretURL: ProxySecretURL, ProxyConfigURL: ProxyConfigURL}

	var buf bytes.Buffer
	if err := updateServiceTemplate.Execute(&buf, vars); err != nil {
		return fmt.Errorf("render update service: %w", err)
	}
	if err := atomicWrite(UpdateServicePath, buf.Bytes(), 0644); err != nil {
		return err
	}
	if err := atomicWrite(UpdateTimerPath, []byte(updateTimerContent), 0644); err != nil {
		return err
	}
	i.log.Info("update units written")
	return nil
}

// ---------------------------------------------------------------------------
// Service management
// ---------------------------------------------------------------------------

func (i *Installer) enableAndStart(ctx context.Context) error {
	if err := runCommand("systemctl", "daemon-reload"); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := runCommand("systemctl", "enable", ServiceName); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}
	if err := runCommand("systemctl", "enable", "mtproxy-update.timer"); err != nil {
		i.log.Warn("failed to enable update timer", "err", err)
	}
	if err := runCommand("systemctl", "start", ServiceName); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	if err := runCommand("systemctl", "start", "mtproxy-update.timer"); err != nil {
		i.log.Warn("failed to start update timer", "err", err)
	}
	return nil
}

// waitHealthy polls the MTProxy internal HTTP port until it responds or ctx expires.
func (i *Installer) waitHealthy(ctx context.Context, httpPort int) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/stats", httpPort)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	deadline := time.Now().Add(60 * time.Second)
	client := &http.Client{Timeout: 2 * time.Second}

	i.log.Info("waiting for MTProxy to become healthy", "url", url)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case t := <-ticker.C:
			if t.After(deadline) {
				return fmt.Errorf("MTProxy did not become healthy within 60s (checked %s)", url)
			}
			resp, err := client.Get(url)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					i.log.Info("MTProxy is healthy")
					return nil
				}
			}
			i.log.Debug("health check pending", "url", url, "err", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// generateSecret returns a random 32-hex-char MTProxy secret.
func generateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildProxyLink constructs the Telegram proxy deep link.
func buildProxyLink(server string, port int, secret string) string {
	if server == "" {
		server = "YOUR_SERVER_IP"
	}
	return fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s", server, port, secret)
}

// getExternalIP fetches the machine's public IP from a reliable endpoint.
func getExternalIP(ctx context.Context, client *http.Client) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	return strings.TrimSpace(string(b)), err
}

// runCommand runs an external command and returns a descriptive error on failure.
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err,
			strings.TrimSpace(string(out)))
	}
	return nil
}

// atomicWrite writes data to path via a temp file + rename.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// decodeJSON is a shared helper used by both mtproxy.go and selfupdate.go.
func decodeJSON(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// copyFile copies src to dst, creating dst if needed.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
