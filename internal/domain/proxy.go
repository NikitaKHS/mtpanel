package domain

import "time"

// ProxyStatus represents the runtime state of the MTProxy process.
type ProxyStatus string

const (
	ProxyStatusRunning  ProxyStatus = "running"
	ProxyStatusStopped  ProxyStatus = "stopped"
	ProxyStatusFailed   ProxyStatus = "failed"
	ProxyStatusUnknown  ProxyStatus = "unknown"
	ProxyStatusInstalling ProxyStatus = "installing"
)

// ProxyConfig holds the active configuration for the MTProxy daemon.
type ProxyConfig struct {
	ID         string    `json:"id"`
	Port       int       `json:"port"`
	Secret     string    `json:"secret"`      // hex-encoded 16-byte secret
	Tag        string    `json:"tag"`         // Telegram advertising tag (optional)
	Workers    int       `json:"workers"`
	MaxConn    int       `json:"max_connections"`
	NATPrefix  string    `json:"nat_prefix"`  // for NAT traversal
	ExtraArgs  string    `json:"extra_args"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ProxyLink represents a shareable tg:// connection link.
type ProxyLink struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Secret    string    `json:"secret"`   // may be per-link or global
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Link      string    `json:"link"`     // full tg://proxy?... URI
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}
