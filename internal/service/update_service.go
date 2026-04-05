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

const (
	panelVersion      = "1.0.0"
	githubReleasesAPI = "https://api.github.com/repos/telemt/telemt/releases/latest"
)

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

	current, _ := u.settings.Get(ctx, "telemt_version")
	if current == "" {
		current, _ = u.settings.Get(ctx, "mtproxy_version")
	}
	if current == "" {
		current = mtproxyVersion(u.cfg.MTProxyBinPath)
		if current != "" && current != "unknown" {
			_ = u.settings.Set(ctx, "telemt_version", current)
			_ = u.settings.Set(ctx, "mtproxy_version", current)
		}
	}

	info := &domain.UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   release.TagName,
		UpdateAvailable: release.TagName != "" && release.TagName != current,
		ReleaseURL:      release.HTMLURL,
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt.Format(time.RFC3339),
	}
	return info, nil
}

// Update performs a safe binary replacement.
func (u *UpdateService) Update(ctx context.Context) error {
	release, err := fetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("update: fetch release: %w", err)
	}

	assetName, err := telemtAssetName()
	if err != nil {
		return fmt.Errorf("update: resolve asset name: %w", err)
	}

	downloadURL := ""
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}
	if downloadURL == "" {
		downloadURL, err = telemtLatestAssetURL()
		if err != nil {
			return fmt.Errorf("update: fallback url: %w", err)
		}
	}

	binPath := u.cfg.MTProxyBinPath
	backupPath := binPath + ".bak"

	stopErr := u.systemdM.StopUnit(ctx, telemtUnitName)
	_ = copyFile(binPath, backupPath)

	if err := downloadAndExtractTeleMT(ctx, downloadURL, binPath); err != nil {
		_ = copyFile(backupPath, binPath)
		if stopErr == nil {
			_ = u.systemdM.StartUnit(ctx, telemtUnitName)
		}
		return fmt.Errorf("update: download and install: %w", err)
	}

	if err := verifyBinary(binPath); err != nil {
		_ = copyFile(backupPath, binPath)
		if stopErr == nil {
			_ = u.systemdM.StartUnit(ctx, telemtUnitName)
		}
		return fmt.Errorf("update: binary verification failed: %w", err)
	}

	_ = u.settings.Set(ctx, "telemt_version", release.TagName)
	_ = u.settings.Set(ctx, "mtproxy_version", release.TagName)

	if err := u.systemdM.StartUnit(ctx, telemtUnitName); err != nil {
		return fmt.Errorf("update: restart after update: %w", err)
	}

	_ = u.audit.Record(ctx, &domain.AuditEvent{
		ID:        uuid.New().String(),
		EventType: domain.AuditEventUpdateApplied,
		ActorID:   "system",
		Detail:    fmt.Sprintf("telemt updated to %s", release.TagName),
	})

	_ = os.Remove(backupPath)
	return nil
}

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
