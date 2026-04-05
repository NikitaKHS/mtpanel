package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/infrastructure"
	idb "github.com/mtpanel/mtpanel/internal/infrastructure/db"
	"github.com/mtpanel/mtpanel/internal/infrastructure/systemd"
	"github.com/mtpanel/mtpanel/internal/repository"
	"github.com/mtpanel/mtpanel/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	if strings.EqualFold(cfg.LogLevel, "debug") {
		level.Set(slog.LevelDebug)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o750); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := idb.Open(ctx, cfg.DBPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	settingsRepo := repository.NewSettingsRepository(db)
	proxyCfgRepo := repository.NewProxyConfigRepository(db)
	linkRepo := repository.NewProxyLinkRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	systemdM, err := systemd.New()
	if err != nil {
		slog.Warn("systemd not available", "err", err)
		systemdM = &systemd.Manager{}
	}

	signing := []byte(cfg.JWTSecret)
	if len(signing) == 0 {
		signing = []byte(mustRandomHex(32))
		slog.Warn("JWT secret was empty in config; generated ephemeral secret")
	}

	authStore := &adminStore{cfg: cfg, settings: settingsRepo}
	authSvc := service.NewAuthService(authStore, signing, cfg.JWTExpireHours)
	proxySvc := service.NewProxyService(cfg, systemdM, proxyCfgRepo, linkRepo, auditRepo, settingsRepo)
	systemSvc := service.NewSystemService(cfg, systemdM, settingsRepo)
	updateSvc := service.NewUpdateService(cfg, systemdM, auditRepo, settingsRepo)

	apiRouter := infrastructure.NewRouter(infrastructure.RouterConfig{
		DB:        db.DB,
		AppCfg:    cfg,
		Signing:   signing,
		AuthSvc:   authSvc,
		ProxySvc:  proxySvc,
		SystemSvc: systemSvc,
		UpdateSvc: updateSvc,
		Settings:  settingsRepo,
		Audit:     auditRepo,
		Version:   "dev",
	})

	handler := withSPA(apiRouter, filepath.Join("web", "dist"))

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	slog.Info("starting mtpanel", "listen", cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

type adminStore struct {
	cfg      *config.Config
	settings repository.SettingsRepository
}

func (s *adminStore) GetPasswordHash(ctx context.Context) (string, error) {
	hash, err := s.settings.Get(ctx, "admin_password_hash")
	if err == nil && hash != "" {
		return hash, nil
	}

	// Bootstrap from config if legacy hash exists.
	if s.cfg.AdminPasswordHash != "" {
		_ = s.settings.Set(ctx, "admin_password_hash", s.cfg.AdminPasswordHash)
		_ = s.settings.Set(ctx, "is_first_run", "false")
		return s.cfg.AdminPasswordHash, nil
	}
	return "", service.ErrNoAdminConfigured
}

func (s *adminStore) SetPasswordHash(ctx context.Context, hash string) error {
	if err := s.settings.Set(ctx, "admin_password_hash", hash); err != nil {
		return err
	}
	return s.settings.Set(ctx, "is_first_run", "false")
}

func (s *adminStore) IsFirstRun(ctx context.Context) (bool, error) {
	v, err := s.settings.Get(ctx, "is_first_run")
	if err != nil {
		return s.cfg.IsFirstRun, nil
	}
	return strings.EqualFold(v, "true"), nil
}

func mustRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func withSPA(api http.Handler, staticRoot string) http.Handler {
	spaFS := http.Dir(staticRoot)
	fileServer := http.FileServer(spaFS)
	indexPath := filepath.Join(staticRoot, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			api.ServeHTTP(w, r)
			return
		}

		if _, err := os.Stat(indexPath); err != nil {
			http.NotFound(w, r)
			return
		}

		// Serve existing static file, otherwise fallback to SPA entry.
		target := filepath.Join(staticRoot, filepath.FromSlash(strings.TrimPrefix(r.URL.Path, "/")))
		if info, err := os.Stat(target); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}
