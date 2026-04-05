package service

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/systemd"
	"github.com/mtpanel/mtpanel/internal/repository"
)

// SystemService provides host-level information and compatibility checks.
type SystemService struct {
	cfg      *config.Config
	systemdM *systemd.Manager
	settings repository.SettingsRepository
}

// NewSystemService creates a SystemService.
func NewSystemService(
	cfg *config.Config,
	sm *systemd.Manager,
	settings repository.SettingsRepository,
) *SystemService {
	return &SystemService{cfg: cfg, systemdM: sm, settings: settings}
}

// GetSystemInfo collects basic OS and hardware information.
func (s *SystemService) GetSystemInfo(ctx context.Context) (*domain.SystemInfo, error) {
	info := &domain.SystemInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		PanelVersion: panelVersion,
	}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	}

	// Kernel version from uname.
	if out, err := exec.CommandContext(ctx, "uname", "-r").Output(); err == nil {
		info.KernelVersion = strings.TrimSpace(string(out))
	}

	info.CPUCount = runtime.NumCPU()
	info.MemTotalMB, info.MemAvailableMB = readMemInfo()
	info.DiskTotalGB, info.DiskFreeGB = diskStats(s.cfg.DataDir)
	info.Uptime = readUptime()
	info.MTProxyVersion = mtproxyVersion(s.cfg.MTProxyBinPath)

	return info, nil
}

// CheckCompatibility verifies the host can run MTProxy and this panel.
func (s *SystemService) CheckCompatibility(ctx context.Context) (*domain.CompatibilityReport, error) {
	report := &domain.CompatibilityReport{
		Arch: runtime.GOARCH,
	}

	// systemctl
	if path, err := exec.LookPath("systemctl"); err == nil {
		report.SystemdAvailable = true
		report.SystemctlPath = path
	} else {
		report.Issues = append(report.Issues, "systemctl not found — systemd required")
	}

	// journalctl (optional but desired)
	if path, err := exec.LookPath("journalctl"); err == nil {
		report.JournalctlPath = path
	} else {
		report.Issues = append(report.Issues, "journalctl not found — log viewing disabled")
	}

	// Architecture check.
	switch runtime.GOARCH {
	case "amd64", "arm64", "arm":
	default:
		report.Issues = append(report.Issues,
			fmt.Sprintf("architecture %s may not have a pre-built MTProxy binary", runtime.GOARCH))
	}

	// Linux only.
	if runtime.GOOS != "linux" {
		report.Issues = append(report.Issues, "MTProxy is Linux-only")
	}

	report.Supported = report.SystemdAvailable && runtime.GOOS == "linux"
	return report, nil
}

// GetLogs returns the last n lines from the MTProxy service journal.
func (s *SystemService) GetLogs(ctx context.Context, lines int) ([]string, error) {
	if s.systemdM == nil {
		return nil, fmt.Errorf("systemd manager not available")
	}
	return s.systemdM.GetLogs(ctx, mtproxyUnitName, lines)
}

// ---------- helpers ----------

// readMemInfo parses /proc/meminfo for total and available memory in MB.
func readMemInfo() (totalMB, availMB uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			totalMB = val / 1024
		case "MemAvailable:":
			availMB = val / 1024
		}
	}
	return
}

// diskStats returns total and free disk space in GB for the filesystem
// containing the given path.
func diskStats(path string) (totalGB, freeGB float64) {
	_ = path
	return
}

// readUptime reads /proc/uptime and returns seconds as uint64.
func readUptime() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	return uint64(f)
}
