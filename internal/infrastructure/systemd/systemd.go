package systemd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// UnitStatus represents the active state returned by systemctl.
type UnitStatus string

const (
	UnitActive   UnitStatus = "active"
	UnitInactive UnitStatus = "inactive"
	UnitFailed   UnitStatus = "failed"
	UnitUnknown  UnitStatus = "unknown"
)

// Manager controls systemd units via systemctl.
type Manager struct {
	systemctl  string
	journalctl string
	sudo       bool // whether to prepend sudo
}

// New creates a Manager, locating systemctl automatically.
func New() (*Manager, error) {
	ctl, err := exec.LookPath("systemctl")
	if err != nil {
		return nil, fmt.Errorf("systemd: systemctl not found: %w", err)
	}
	jctl, _ := exec.LookPath("journalctl") // optional; may be empty

	return &Manager{
		systemctl:  ctl,
		journalctl: jctl,
	}, nil
}

// StartUnit starts a systemd unit.
func (m *Manager) StartUnit(ctx context.Context, unit string) error {
	return m.run(ctx, "start", unit)
}

// StopUnit stops a systemd unit.
func (m *Manager) StopUnit(ctx context.Context, unit string) error {
	return m.run(ctx, "stop", unit)
}

// RestartUnit restarts a systemd unit.
func (m *Manager) RestartUnit(ctx context.Context, unit string) error {
	return m.run(ctx, "restart", unit)
}

// EnableUnit enables a systemd unit to start at boot.
func (m *Manager) EnableUnit(ctx context.Context, unit string) error {
	return m.run(ctx, "enable", unit)
}

// DisableUnit disables a systemd unit from starting at boot.
func (m *Manager) DisableUnit(ctx context.Context, unit string) error {
	return m.run(ctx, "disable", unit)
}

// DaemonReload tells systemd to re-read all unit files on disk.
func (m *Manager) DaemonReload(ctx context.Context) error {
	return m.run(ctx, "daemon-reload")
}

// GetUnitStatus returns the active-state of a unit.
func (m *Manager) GetUnitStatus(ctx context.Context, unit string) (UnitStatus, error) {
	args := m.args("is-active", "--quiet", unit)
	cmd := exec.CommandContext(ctx, m.systemctl, args...)

	// is-active exits 0 = active, 3 = inactive/unknown
	if err := cmd.Run(); err != nil {
		// Distinguish failed from simply inactive.
		state, stateErr := m.showProperty(ctx, unit, "ActiveState")
		if stateErr != nil {
			return UnitUnknown, nil
		}
		switch state {
		case "failed":
			return UnitFailed, nil
		case "inactive", "deactivating":
			return UnitInactive, nil
		default:
			return UnitUnknown, nil
		}
	}
	return UnitActive, nil
}

// IsActive returns true when the unit's active state is "active".
func (m *Manager) IsActive(ctx context.Context, unit string) (bool, error) {
	s, err := m.GetUnitStatus(ctx, unit)
	return s == UnitActive, err
}

// GetLogs returns the last n lines from the unit's journal.
func (m *Manager) GetLogs(ctx context.Context, unit string, lines int) ([]string, error) {
	if m.journalctl == "" {
		return nil, fmt.Errorf("journalctl not found on this system")
	}
	if lines <= 0 {
		lines = 100
	}
	args := []string{
		"-u", unit,
		"--no-pager",
		"--output", "short-iso",
		"-n", fmt.Sprintf("%d", lines),
	}
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, m.journalctl, args...)
	cmd.Stdout = &out
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("journalctl: %w: %s", err, stderr.String())
	}

	raw := strings.TrimRight(out.String(), "\n")
	if raw == "" {
		return []string{}, nil
	}
	return strings.Split(raw, "\n"), nil
}

// showProperty queries a single systemd unit property.
func (m *Manager) showProperty(ctx context.Context, unit, property string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	args := m.args("show", unit, "--property="+property, "--value")
	out, err := exec.CommandContext(ctx, m.systemctl, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// run executes a systemctl sub-command.
func (m *Manager) run(ctx context.Context, sub string, extra ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	args := m.args(sub)
	args = append(args, extra...)
	cmd := exec.CommandContext(ctx, m.systemctl, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl %s %s: %w: %s", sub,
			strings.Join(extra, " "), err, stderr.String())
	}
	return nil
}

// args prepends "--system" so we always target the system manager.
func (m *Manager) args(sub string, extra ...string) []string {
	base := []string{"--system", sub}
	return append(base, extra...)
}

// WriteUnitFile is a helper that writes content to a systemd unit file path.
// The caller is responsible for calling DaemonReload afterwards.
func WriteUnitFile(path, content string) error {
	return writeFileAtomic(path, []byte(content), 0o644)
}
