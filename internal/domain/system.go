package domain

// SystemInfo holds runtime information about the host.
type SystemInfo struct {
	Hostname        string  `json:"hostname"`
	OS              string  `json:"os"`
	Arch            string  `json:"arch"`
	KernelVersion   string  `json:"kernel_version"`
	CPUCount        int     `json:"cpu_count"`
	MemTotalMB      uint64  `json:"mem_total_mb"`
	MemAvailableMB  uint64  `json:"mem_available_mb"`
	DiskTotalGB     float64 `json:"disk_total_gb"`
	DiskFreeGB      float64 `json:"disk_free_gb"`
	Uptime          uint64  `json:"uptime_seconds"`
	MTProxyVersion  string  `json:"mtproxy_version"`
	PanelVersion    string  `json:"panel_version"`
}

// CompatibilityReport lists what the host supports.
type CompatibilityReport struct {
	SystemdAvailable  bool   `json:"systemd_available"`
	SystemctlPath     string `json:"systemctl_path"`
	JournalctlPath    string `json:"journalctl_path"`
	Arch              string `json:"arch"`
	Supported         bool   `json:"supported"`
	Issues            []string `json:"issues,omitempty"`
}

// UpdateInfo describes a new release found upstream.
type UpdateInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	UpdateAvailable bool  `json:"update_available"`
	ReleaseURL     string `json:"release_url"`
	ReleaseNotes   string `json:"release_notes"`
	PublishedAt    string `json:"published_at"`
}
