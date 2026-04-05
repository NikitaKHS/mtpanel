package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	// HTTP server
	ListenAddr string `json:"listen_addr"`

	// Storage
	DataDir string `json:"data_dir"`
	DBPath  string `json:"db_path"`

	// MTProxy binary
	MTProxyBinPath string `json:"mtproxy_bin_path"`
	MTProxyPort    int    `json:"mtproxy_port"`
	MTProxySecret  string `json:"mtproxy_secret"`

	// Security
	AdminPasswordHash string `json:"admin_password_hash"` // bcrypt hash
	JWTSecret         string `json:"jwt_secret"`
	JWTExpireHours    int    `json:"jwt_expire_hours"`

	// Logging
	LogLevel string `json:"log_level"`

	// Runtime
	IsFirstRun bool `json:"is_first_run"`
}

const defaultConfigPath = "/etc/mtpanel/config.json"

// Load reads config from a JSON file (path from MTPANEL_CONFIG env or default)
// and then overlays environment variable overrides.
func Load() (*Config, error) {
	cfg := defaults()

	configPath := envOr("MTPANEL_CONFIG", defaultConfigPath)
	if err := loadFile(cfg, configPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: reading %s: %w", configPath, err)
	}

	applyEnv(cfg)

	if err := validate(cfg); err != nil {
		return nil, fmt.Errorf("config: validation: %w", err)
	}

	// Resolve DBPath relative to DataDir if not absolute.
	if !filepath.IsAbs(cfg.DBPath) {
		cfg.DBPath = filepath.Join(cfg.DataDir, cfg.DBPath)
	}

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		ListenAddr:     ":8080",
		DataDir:        "/var/lib/mtpanel",
		DBPath:         "mtpanel.db",
		MTProxyBinPath: "/opt/telemt/telemt",
		MTProxyPort:    443,
		LogLevel:       "info",
		JWTExpireHours: 24,
		IsFirstRun:     true,
	}
}

func loadFile(cfg *Config, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(cfg)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("MTPANEL_LISTEN"); v != "" {
		cfg.ListenAddr = v
	}
	if v := os.Getenv("MTPANEL_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
	if v := os.Getenv("MTPANEL_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("MTPANEL_BIN_PATH"); v != "" {
		cfg.MTProxyBinPath = v
	}
	if v := os.Getenv("MTPANEL_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MTProxyPort = n
		}
	}
	if v := os.Getenv("MTPANEL_SECRET"); v != "" {
		cfg.MTProxySecret = v
	}
	if v := os.Getenv("MTPANEL_ADMIN_PASSWORD_HASH"); v != "" {
		cfg.AdminPasswordHash = v
	}
	if v := os.Getenv("MTPANEL_JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("MTPANEL_JWT_EXPIRE_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.JWTExpireHours = n
		}
	}
	if v := os.Getenv("MTPANEL_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
}

func validate(cfg *Config) error {
	if cfg.ListenAddr == "" {
		return fmt.Errorf("listen_addr must not be empty")
	}
	if cfg.DataDir == "" {
		return fmt.Errorf("data_dir must not be empty")
	}
	if cfg.JWTExpireHours <= 0 {
		return fmt.Errorf("jwt_expire_hours must be > 0")
	}
	if cfg.MTProxyPort < 1 || cfg.MTProxyPort > 65535 {
		return fmt.Errorf("mtproxy_port must be 1-65535")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		cfg.LogLevel = "info"
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
