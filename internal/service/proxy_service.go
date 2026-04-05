package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/systemd"
	"github.com/mtpanel/mtpanel/internal/repository"
)

const (
	mtproxyUnitName    = "mtproto-proxy.service"
	mtproxyUnitPath    = "/etc/systemd/system/mtproto-proxy.service"
	githubReleasesAPI  = "https://api.github.com/repos/TelegramMessenger/MTProxy/releases/latest"
	downloadURLPattern = "https://github.com/TelegramMessenger/MTProxy/releases/latest/download/mtproto-proxy-linux-%s"
	mtproxySourceRepo  = "https://github.com/TelegramMessenger/MTProxy.git"
)

var ErrProxyNotInstalled = errors.New("mtproxy is not installed")

var systemdUnitTmpl = template.Must(template.New("unit").Parse(`[Unit]
Description=MTProto Proxy
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=nobody
ExecStart={{.BinPath}} -u nobody -p {{.Port}} -H {{.Port}} -S {{.Secret}}{{if .Tag}} --aes-pwd proxy-secret proxy-multi.conf -M {{.Workers}} -t {{.Tag}}{{else}} --aes-pwd proxy-secret proxy-multi.conf -M {{.Workers}}{{end}}{{if .ExtraArgs}} {{.ExtraArgs}}{{end}}
Restart=on-failure
RestartSec=5
LimitNOFILE=65536
StandardOutput=journal
StandardError=journal
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`))

type unitData struct {
	BinPath   string
	Port      int
	Secret    string
	Tag       string
	Workers   int
	ExtraArgs string
}

// ProxyService manages the MTProxy daemon lifecycle.
type ProxyService struct {
	cfg       *config.Config
	systemd   *systemd.Manager
	proxyRepo repository.ProxyConfigRepository
	linkRepo  repository.ProxyLinkRepository
	audit     repository.AuditRepository
	settings  repository.SettingsRepository
}

// NewProxyService creates a ProxyService.
func NewProxyService(
	cfg *config.Config,
	sm *systemd.Manager,
	proxyRepo repository.ProxyConfigRepository,
	linkRepo repository.ProxyLinkRepository,
	audit repository.AuditRepository,
	settings repository.SettingsRepository,
) *ProxyService {
	return &ProxyService{
		cfg:       cfg,
		systemd:   sm,
		proxyRepo: proxyRepo,
		linkRepo:  linkRepo,
		audit:     audit,
		settings:  settings,
	}
}

// Install downloads the MTProxy binary, writes the systemd unit, and enables the service.
func (s *ProxyService) Install(ctx context.Context) error {
	s.recordAudit(ctx, domain.AuditEventProxyInstall, "", "started install")

	arch := detectArch()
	if arch == "" {
		return fmt.Errorf("proxy: unsupported architecture %s", runtime.GOARCH)
	}

	binURL := fmt.Sprintf(downloadURLPattern, arch)

	if err := os.MkdirAll(filepath.Dir(s.cfg.MTProxyBinPath), 0o755); err != nil {
		return fmt.Errorf("proxy: create bin dir: %w", err)
	}

	if err := downloadBinary(ctx, binURL, s.cfg.MTProxyBinPath); err != nil {
		if strings.Contains(err.Error(), "HTTP 404") {
			if buildErr := buildBinaryFromSource(ctx, s.cfg.MTProxyBinPath); buildErr != nil {
				return fmt.Errorf("proxy: download binary failed (%v), source build failed: %w", err, buildErr)
			}
		} else {
			return fmt.Errorf("proxy: download binary: %w", err)
		}
	}

	secret := s.cfg.MTProxySecret
	if secret == "" {
		var err error
		secret, err = generateSecret()
		if err != nil {
			return fmt.Errorf("proxy: generate secret: %w", err)
		}
	}

	// Persist proxy config.
	proxyCfg := &domain.ProxyConfig{
		ID:      "default",
		Port:    s.cfg.MTProxyPort,
		Secret:  secret,
		Workers: 4,
		MaxConn: 60000,
	}
	if err := s.proxyRepo.Upsert(ctx, proxyCfg); err != nil {
		return fmt.Errorf("proxy: persist config: %w", err)
	}

	if err := s.writeSystemdUnit(proxyCfg); err != nil {
		return fmt.Errorf("proxy: write unit file: %w", err)
	}

	if err := s.systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("proxy: daemon-reload: %w", err)
	}

	if err := s.systemd.EnableUnit(ctx, mtproxyUnitName); err != nil {
		return fmt.Errorf("proxy: enable unit: %w", err)
	}
	if err := s.systemd.StartUnit(ctx, mtproxyUnitName); err != nil {
		return fmt.Errorf("proxy: start unit: %w", err)
	}

	if err := s.settings.Set(ctx, "install_state", "installed"); err != nil {
		return fmt.Errorf("proxy: update install state: %w", err)
	}
	if err := s.settings.Set(ctx, "is_first_run", "false"); err != nil {
		return fmt.Errorf("proxy: update first_run: %w", err)
	}

	s.recordAudit(ctx, domain.AuditEventProxyInstall, "default", "install completed")
	return nil
}

// InstallWithPort updates runtime port before running install.
func (s *ProxyService) InstallWithPort(ctx context.Context, port int) error {
	if err := ValidatePort(port); err != nil {
		return err
	}
	s.cfg.MTProxyPort = port
	return s.Install(ctx)
}

// Start starts the MTProxy systemd unit.
func (s *ProxyService) Start(ctx context.Context) error {
	s.recordAudit(ctx, domain.AuditEventProxyStart, mtproxyUnitName, "")
	return s.systemd.StartUnit(ctx, mtproxyUnitName)
}

// Stop stops the MTProxy systemd unit.
func (s *ProxyService) Stop(ctx context.Context) error {
	s.recordAudit(ctx, domain.AuditEventProxyStop, mtproxyUnitName, "")
	return s.systemd.StopUnit(ctx, mtproxyUnitName)
}

// Restart restarts the MTProxy systemd unit.
func (s *ProxyService) Restart(ctx context.Context) error {
	s.recordAudit(ctx, domain.AuditEventProxyRestart, mtproxyUnitName, "")
	return s.systemd.RestartUnit(ctx, mtproxyUnitName)
}

// Status returns the current status of the MTProxy process.
func (s *ProxyService) Status(ctx context.Context) (domain.ProxyStatus, error) {
	unitStatus, err := s.systemd.GetUnitStatus(ctx, mtproxyUnitName)
	if err != nil {
		return domain.ProxyStatusUnknown, err
	}
	switch unitStatus {
	case systemd.UnitActive:
		return domain.ProxyStatusRunning, nil
	case systemd.UnitInactive:
		return domain.ProxyStatusStopped, nil
	case systemd.UnitFailed:
		return domain.ProxyStatusFailed, nil
	default:
		return domain.ProxyStatusUnknown, nil
	}
}

// GetSystemdUnit returns the systemd unit name for MTProxy.
func (s *ProxyService) GetSystemdUnit() string {
	return mtproxyUnitName
}

// RotateSecret generates a new proxy secret, updates config, and rewrites the unit file.
func (s *ProxyService) RotateSecret(ctx context.Context) (string, error) {
	newSecret, err := generateSecret()
	if err != nil {
		return "", fmt.Errorf("proxy: generate secret: %w", err)
	}

	cfg, err := s.proxyRepo.GetActive(ctx)
	if err != nil {
		return "", fmt.Errorf("proxy: get active config: %w", err)
	}
	cfg.Secret = newSecret

	if err := s.proxyRepo.Upsert(ctx, cfg); err != nil {
		return "", fmt.Errorf("proxy: persist rotated secret: %w", err)
	}

	if err := s.writeSystemdUnit(cfg); err != nil {
		return "", fmt.Errorf("proxy: rewrite unit: %w", err)
	}

	if err := s.systemd.DaemonReload(ctx); err != nil {
		return "", err
	}

	s.recordAudit(ctx, domain.AuditEventSecretRotate, "default", "secret rotated")
	return newSecret, nil
}

// GenerateLink creates a new tg:// shareable proxy link.
func (s *ProxyService) GenerateLink(ctx context.Context, label string) (*domain.ProxyLink, error) {
	cfg, err := s.proxyRepo.GetActive(ctx)
	if err != nil {
		if isMissingProxyConfig(err) {
			return nil, ErrProxyNotInstalled
		}
		return nil, fmt.Errorf("proxy: no active config: %w", err)
	}

	host, err := publicIP(ctx)
	if err != nil {
		return nil, fmt.Errorf("proxy: detect public IP: %w", err)
	}

	id := uuid.New().String()
	tgLink := fmt.Sprintf("tg://proxy?server=%s&port=%d&secret=%s",
		host, cfg.Port, cfg.Secret)

	link := &domain.ProxyLink{
		ID:        id,
		Label:     label,
		Secret:    cfg.Secret,
		Host:      host,
		Port:      cfg.Port,
		Link:      tgLink,
		Active:    true,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("proxy: persist link: %w", err)
	}

	s.recordAudit(ctx, domain.AuditEventLinkCreate, id, label)
	return link, nil
}

// RevokeLink marks a proxy link as inactive.
func (s *ProxyService) RevokeLink(ctx context.Context, id string) error {
	if err := s.linkRepo.Revoke(ctx, id); err != nil {
		return fmt.Errorf("proxy: revoke link: %w", err)
	}
	s.recordAudit(ctx, domain.AuditEventLinkRevoke, id, "")
	return nil
}

// ListLinks returns all (or only active) proxy links.
func (s *ProxyService) ListLinks(ctx context.Context, activeOnly bool) ([]*domain.ProxyLink, error) {
	return s.linkRepo.List(ctx, activeOnly)
}

// GetLogs returns recent logs from the MTProxy systemd unit.
func (s *ProxyService) GetLogs(ctx context.Context, lines int) ([]string, error) {
	if lines <= 0 {
		lines = 100
	}
	logs, err := s.systemd.GetLogs(ctx, mtproxyUnitName, lines)
	if err != nil {
		if isMissingUnitLogs(err) {
			return []string{"MTProxy пока не установлен или сервис mtproto-proxy.service не создан."}, nil
		}
		return nil, err
	}
	return logs, nil
}

// SetPort updates active proxy configuration and applies it immediately.
func (s *ProxyService) SetPort(ctx context.Context, port int) error {
	if err := ValidatePort(port); err != nil {
		return err
	}

	cfg, err := s.proxyRepo.GetActive(ctx)
	if err != nil {
		// If proxy isn't installed yet, update runtime default only.
		s.cfg.MTProxyPort = port
		return nil
	}

	cfg.Port = port
	if err := s.proxyRepo.Upsert(ctx, cfg); err != nil {
		return fmt.Errorf("proxy: persist port: %w", err)
	}
	if err := s.writeSystemdUnit(cfg); err != nil {
		return fmt.Errorf("proxy: rewrite unit: %w", err)
	}
	if err := s.systemd.DaemonReload(ctx); err != nil {
		return fmt.Errorf("proxy: daemon reload after port update: %w", err)
	}
	if err := s.systemd.RestartUnit(ctx, mtproxyUnitName); err != nil {
		return fmt.Errorf("proxy: restart after port update: %w", err)
	}

	s.cfg.MTProxyPort = port
	return nil
}

// ActiveConfig returns the latest persisted MTProxy configuration.
func (s *ProxyService) ActiveConfig(ctx context.Context) (*domain.ProxyConfig, error) {
	return s.proxyRepo.GetActive(ctx)
}

// ---------- helpers ----------

func (s *ProxyService) writeSystemdUnit(cfg *domain.ProxyConfig) error {
	var buf strings.Builder
	if err := systemdUnitTmpl.Execute(&buf, unitData{
		BinPath:   s.cfg.MTProxyBinPath,
		Port:      cfg.Port,
		Secret:    cfg.Secret,
		Tag:       cfg.Tag,
		Workers:   cfg.Workers,
		ExtraArgs: cfg.ExtraArgs,
	}); err != nil {
		return err
	}
	return systemd.WriteUnitFile(mtproxyUnitPath, buf.String())
}

func (s *ProxyService) recordAudit(ctx context.Context, et domain.AuditEventType, resID, detail string) {
	ev := &domain.AuditEvent{
		ID:         uuid.New().String(),
		EventType:  et,
		ActorID:    "system",
		ResourceID: resID,
		Detail:     detail,
		CreatedAt:  time.Now().UTC(),
	}
	_ = s.audit.Record(ctx, ev)
}

// generateSecret returns a 32-character hex string (16 random bytes).
func generateSecret() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// detectArch maps GOARCH to the asset suffix used in GitHub releases.
func detectArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	default:
		return ""
	}
}

// downloadBinary fetches url and saves it to dest as an executable.
func downloadBinary(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}

	tmp := dest + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// publicIP fetches the machine's public IPv4 using a lightweight external service.
func publicIP(ctx context.Context) (string, error) {
	services := []string{
		"https://api4.my-ip.io/ip",
		"https://ipv4.icanhazip.com",
		"https://api.ipify.org",
	}
	for _, svc := range services {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, svc, nil)
		if err != nil {
			continue
		}
		resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
		if err != nil {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err != nil {
			continue
		}
		ip := strings.TrimSpace(string(b))
		if ip != "" {
			return ip, nil
		}
	}
	return "", fmt.Errorf("could not detect public IP")
}

// mtproxyVersion tries to get the installed binary version.
func mtproxyVersion(binPath string) string {
	out, err := exec.Command(binPath, "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func buildBinaryFromSource(ctx context.Context, dest string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required for MTProxy source build: %w", err)
	}
	if _, err := exec.LookPath("make"); err != nil {
		return fmt.Errorf("make is required for MTProxy source build: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "mtproxy-src-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	repoDir := filepath.Join(tmpDir, "MTProxy")

	cloneCmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", mtproxySourceRepo, repoDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone MTProxy: %w: %s", err, strings.TrimSpace(string(out)))
	}

	makeCmd := exec.CommandContext(ctx, "make", "-j2")
	makeCmd.Dir = repoDir
	if out, err := makeCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("make MTProxy: %w: %s", err, strings.TrimSpace(string(out)))
	}

	srcBin := filepath.Join(repoDir, "objs", "bin", "mtproto-proxy")
	if _, err := os.Stat(srcBin); err != nil {
		return fmt.Errorf("compiled binary not found at %s: %w", srcBin, err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	in, err := os.Open(srcBin)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return nil
}

func isMissingProxyConfig(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "proxy config not found")
}

func isMissingUnitLogs(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "unit mtproto-proxy.service could not be found") ||
		strings.Contains(msg, "no journal files were found")
}
