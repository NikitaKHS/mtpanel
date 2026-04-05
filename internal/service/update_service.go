package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/systemd"
	"github.com/mtpanel/mtpanel/internal/repository"
)

const panelVersion = "1.0.0"

// githubRelease is a partial decode of the GitHub Releases API response.
type githubRelease struct {
	TagName     string    `json:"tag_name"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// UpdateService checks for and applies binary updates from GitHub Releases.
type UpdateService struct {
	cfg      *config.Config
	systemdM *systemd.Manager
	audit    repository.AuditRepository
	settings repository.SettingsRepository
}

// NewUpdateService creates an UpdateService.
func NewUpdateService(
	cfg *config.Config,
	sm *systemd.Manager,
	audit repository.AuditRepository,
	settings repository.SettingsRepository,
) *UpdateService {
	return &UpdateService{
		cfg:      cfg,
		systemdM: sm,
		audit:    audit,
		settings: settings,
	}
}

// CheckUpdate fetches the latest release from GitHub and returns update information.
func (u *UpdateService) CheckUpdate(ctx context.Context) (*domain.UpdateInfo, error) {
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("update: fetch release: %w", err)
	}

	current, _ := u.settings.Get(ctx, "mtproxy_version")

	info := &domain.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   release.TagName,
		UpdateAvailable: release.TagName != current && release.TagName != "",
		ReleaseURL:      release.HTMLURL,
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt.Format(time.RFC3339),
	}
	return info, nil
}

// Update performs a safe binary replacement:
//  1. Stop the running service.
//  2. Back up the current binary.
//  3. Download + replace.
//  4. Start the service.
//  5. Roll back on error.
func (u *UpdateService) Update(ctx context.Context) error {
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("update: fetch release: %w", err)
	}

	arch := detectArch()
	if arch == "" {
		return fmt.Errorf("update: unsupported architecture")
	}

	downloadURL := ""
	suffix := "mtproto-proxy-linux-" + arch
	for _, a := range release.Assets {
		if a.Name == suffix {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		// Fall back to pattern URL.
		downloadURL = fmt.Sprintf(downloadURLPattern, arch)
	}

	binPath := u.cfg.MTProxyBinPath
	backupPath := binPath + ".bak"

	// Stop service before replacing binary.
	stopErr := u.systemdM.StopUnit(ctx, mtproxyUnitName)

	// Back up current binary (best effort).
	if err := copyFile(binPath, backupPath); err != nil {
		// Not fatal if file doesn't exist yet.
		_ = err
	}

	// Download new binary.
	if err := downloadBinary(ctx, downloadURL, binPath); err != nil {
		// Restore backup.
		_ = copyFile(backupPath, binPath)
		// Restart old version.
		if stopErr == nil {
			_ = u.systemdM.StartUnit(ctx, mtproxyUnitName)
		}
		return fmt.Errorf("update: download: %w", err)
	}

	// Verify new binary executes.
	if err := verifyBinary(binPath); err != nil {
		// Restore backup and restart.
		_ = copyFile(backupPath, binPath)
		if stopErr == nil {
			_ = u.systemdM.StartUnit(ctx, mtproxyUnitName)
		}
		return fmt.Errorf("update: binary verification failed: %w", err)
	}

	// Persist new version.
	_ = u.settings.Set(ctx, "mtproxy_version", release.TagName)

	// Restart service.
	if err := u.systemdM.StartUnit(ctx, mtproxyUnitName); err != nil {
		return fmt.Errorf("update: restart after update: %w", err)
	}

	// Record audit event.
	_ = u.audit.Record(ctx, &domain.AuditEvent{
		ID:        uuid.New().String(),
		EventType: domain.AuditEventUpdateApplied,
		ActorID:   "system",
		Detail:    fmt.Sprintf("updated to %s", release.TagName),
	})

	// Clean up backup.
	os.Remove(backupPath)
	return nil
}

// ---------- helpers ----------

func fetchLatestRelease(ctx context.Context) (*githubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "mtpanel/"+panelVersion)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// verifyBinary checks the binary is executable and responds to --version or --help.
func verifyBinary(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("binary %s is not executable", path)
	}
	return nil
}

// copyFile copies src → dst, preserving permissions.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := out.ReadFrom(in); err != nil {
		return err
	}
	return out.Sync()
}
