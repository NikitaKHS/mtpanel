package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Panel self-update
// ---------------------------------------------------------------------------

// PanelUpdater manages safe in-place updates of the mtpanel binary itself.
type PanelUpdater struct {
	log       *slog.Logger
	client    *http.Client
	githubOrg string
	githubRepo string
	installDir string
	binaryName string
	serviceName string
}

// NewPanelUpdater creates a PanelUpdater with sensible defaults.
func NewPanelUpdater(log *slog.Logger, githubRepo string) *PanelUpdater {
	org, repo, _ := strings.Cut(githubRepo, "/")
	return &PanelUpdater{
		log:         log,
		client:      &http.Client{Timeout: 2 * time.Minute},
		githubOrg:   org,
		githubRepo:  repo,
		installDir:  "/opt/mtpanel",
		binaryName:  "mtpanel",
		serviceName: "mtpanel",
	}
}

// UpdateResult carries information about the completed update.
type UpdateResult struct {
	PreviousVersion string
	NewVersion      string
	BinaryPath      string
	RolledBack      bool
}

// Update performs a safe in-place update of the panel binary:
//  1. Download new binary to /tmp
//  2. Verify sha256 checksum
//  3. Stop service
//  4. Backup current binary
//  5. Move new binary into place
//  6. Start service
//  7. Health check — rollback on failure
func (u *PanelUpdater) Update(ctx context.Context, targetVersion string) (*UpdateResult, error) {
	u.log.Info("starting panel self-update", "target", targetVersion)

	// Resolve version to download
	var tag string
	var downloadURL string
	var checksumURL string

	if targetVersion == "" || targetVersion == "latest" {
		t, url, chk, err := u.resolveLatest(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve latest version: %w", err)
		}
		tag, downloadURL, checksumURL = t, url, chk
	} else {
		tag = targetVersion
		downloadURL, checksumURL = u.buildURLs(targetVersion)
	}

	u.log.Info("downloading new binary", "version", tag, "url", downloadURL)

	// 1. Download to /tmp
	tmpBin := filepath.Join("/tmp", "mtpanel-new")
	if err := u.downloadAndVerify(ctx, downloadURL, checksumURL, tmpBin); err != nil {
		_ = os.Remove(tmpBin)
		return nil, fmt.Errorf("download/verify: %w", err)
	}

	binPath := filepath.Join(u.installDir, u.binaryName)
	bakPath := filepath.Join(u.installDir, u.binaryName+".bak")

	// 3. Stop service
	u.log.Info("stopping panel service for update")
	if err := runCommand("systemctl", "stop", u.serviceName); err != nil {
		_ = os.Remove(tmpBin)
		return nil, fmt.Errorf("stop service: %w", err)
	}

	// 4. Backup current binary
	if err := copyFile(binPath, bakPath); err != nil {
		u.log.Warn("failed to backup current binary", "err", err)
		// Not fatal — continue anyway
	} else {
		u.log.Info("current binary backed up", "backup", bakPath)
	}

	// 5. Move new binary into place
	if err := os.Rename(tmpBin, binPath); err != nil {
		// Try copy as fallback (cross-device rename may fail on some setups)
		if err2 := copyFile(tmpBin, binPath); err2 != nil {
			_ = u.rollback(bakPath, binPath)
			_ = runCommand("systemctl", "start", u.serviceName)
			return &UpdateResult{RolledBack: true}, fmt.Errorf("install binary: %w", err2)
		}
		_ = os.Remove(tmpBin)
	}
	if err := os.Chmod(binPath, 0755); err != nil {
		_ = u.rollback(bakPath, binPath)
		_ = runCommand("systemctl", "start", u.serviceName)
		return &UpdateResult{RolledBack: true}, fmt.Errorf("chmod binary: %w", err)
	}

	// 6. Start service
	u.log.Info("starting panel service with new binary")
	if err := runCommand("systemctl", "start", u.serviceName); err != nil {
		_ = u.rollback(bakPath, binPath)
		_ = runCommand("systemctl", "start", u.serviceName)
		return &UpdateResult{RolledBack: true}, fmt.Errorf("start service after update: %w", err)
	}

	// 7. Health check
	if err := u.healthCheck(ctx); err != nil {
		u.log.Error("health check failed after update — rolling back", "err", err)
		if rbErr := u.rollback(bakPath, binPath); rbErr != nil {
			u.log.Error("rollback failed", "err", rbErr)
		}
		_ = runCommand("systemctl", "restart", u.serviceName)
		return &UpdateResult{RolledBack: true}, fmt.Errorf("post-update health check: %w", err)
	}

	u.log.Info("panel updated successfully", "version", tag)
	return &UpdateResult{
		NewVersion: tag,
		BinaryPath: binPath,
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type ghReleaseSimple struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

func (u *PanelUpdater) resolveLatest(ctx context.Context) (tag, binURL, chkURL string, err error) {
	apiURL := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/releases/latest",
		u.githubOrg, u.githubRepo,
	)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("GitHub API HTTP %d", resp.StatusCode)
	}

	var rel ghReleaseSimple
	if err := jsonDecode(resp.Body, &rel); err != nil {
		return "", "", "", err
	}

	binName := fmt.Sprintf("mtpanel-linux-%s", runtime.GOARCH)
	chkName := binName + ".sha256"

	for _, a := range rel.Assets {
		if a.Name == binName {
			binURL = a.BrowserDownloadURL
		}
		if a.Name == chkName {
			chkURL = a.BrowserDownloadURL
		}
	}

	if binURL == "" {
		binURL, chkURL = u.buildURLs(rel.TagName)
	}

	return rel.TagName, binURL, chkURL, nil
}

func (u *PanelUpdater) buildURLs(tag string) (binURL, chkURL string) {
	base := fmt.Sprintf(
		"https://github.com/%s/%s/releases/download/%s",
		u.githubOrg, u.githubRepo, tag,
	)
	name := fmt.Sprintf("mtpanel-linux-%s", runtime.GOARCH)
	return base + "/" + name, base + "/" + name + ".sha256"
}

func (u *PanelUpdater) downloadAndVerify(ctx context.Context, binURL, chkURL, dest string) error {
	// Download binary
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, binURL, nil)
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download binary: HTTP %d", resp.StatusCode)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, hasher), resp.Body); err != nil {
		_ = f.Close()
		return fmt.Errorf("write binary: %w", err)
	}
	_ = f.Close()

	gotHash := hex.EncodeToString(hasher.Sum(nil))

	// Fetch and compare checksum (best effort)
	if chkURL == "" {
		u.log.Warn("no checksum URL available — skipping verification")
		return nil
	}

	req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, chkURL, nil)
	resp2, err := u.client.Do(req2)
	if err != nil {
		u.log.Warn("could not fetch checksum file", "err", err)
		return nil
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		u.log.Warn("checksum file not available", "status", resp2.StatusCode)
		return nil
	}

	chkBody, err := io.ReadAll(io.LimitReader(resp2.Body, 256))
	if err != nil {
		return fmt.Errorf("read checksum: %w", err)
	}

	// Checksum file format: "<hash>  <filename>" or just "<hash>"
	wantHash := strings.Fields(string(chkBody))[0]
	if !strings.EqualFold(gotHash, wantHash) {
		return fmt.Errorf("checksum mismatch: want %s got %s", wantHash, gotHash)
	}

	u.log.Info("checksum verified", "sha256", gotHash)
	return nil
}

func (u *PanelUpdater) rollback(bakPath, binPath string) error {
	u.log.Info("rolling back binary", "from", bakPath, "to", binPath)
	if _, err := os.Stat(bakPath); os.IsNotExist(err) {
		return fmt.Errorf("backup binary not found at %s", bakPath)
	}
	if err := copyFile(bakPath, binPath); err != nil {
		return fmt.Errorf("restore backup: %w", err)
	}
	return os.Chmod(binPath, 0755)
}

func (u *PanelUpdater) healthCheck(ctx context.Context) error {
	// Read panel port from config (best effort)
	port := 8080
	cfg, err := os.ReadFile("/etc/mtpanel/config.json")
	if err == nil {
		// Simple scan for listen_addr port — avoid importing config package here
		s := string(cfg)
		if idx := strings.Index(s, `"listen_addr"`); idx != -1 {
			sub := s[idx:]
			if portIdx := strings.Index(sub, ":"); portIdx != -1 {
				var p int
				if _, err := fmt.Sscanf(sub[portIdx:], ":%d", &p); err == nil && p > 0 {
					port = p
				}
			}
		}
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/api/health", port)
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	u.log.Info("panel health check", "url", url)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("panel did not respond on %s within 30s", url)
}

// jsonDecode is a thin wrapper so selfupdate.go does not need its own json import.
// encoding/json is already imported in mtproxy.go (same package), so the compiler
// is happy.  We call it through the top-level package-level function to keep things
// explicit and testable.
func jsonDecode(r io.Reader, v any) error {
	return decodeJSON(r, v)
}
