package infrastructure

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/handler"
	"github.com/mtpanel/mtpanel/internal/middleware"
	"github.com/mtpanel/mtpanel/internal/repository"
	"github.com/mtpanel/mtpanel/internal/service"
)

type RouterConfig struct {
	DB       *sql.DB
	AppCfg   *config.Config
	Signing  []byte
	AuthSvc  *service.AuthService
	ProxySvc *service.ProxyService
	SystemSvc *service.SystemService
	UpdateSvc *service.UpdateService
	Settings repository.SettingsRepository
	Audit    repository.AuditRepository
	Version  string
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.RequestID)
	r.Use(chimiddleware.Recoverer)
	r.Use(structuredLogger())

	authH := handler.NewAuthHandler(cfg.AuthSvc, cfg.Settings, cfg.Audit)
	proxyH := handler.NewProxyHandler(cfg.ProxySvc)
	systemH := handler.NewSystemHandler(cfg.SystemSvc)
	updateH := handler.NewUpdateHandler(cfg.UpdateSvc)
	settingsH := handler.NewSettingsHandler(cfg.AppCfg, cfg.Settings, cfg.ProxySvc)
	auditH := handler.NewAuditHandler(cfg.Audit)
	healthH := handler.NewHealthHandler(cfg.DB, func() string {
		s, err := cfg.ProxySvc.Status(context.Background())
		if err != nil {
			return "unknown"
		}
		return string(s)
	}, cfg.Version)

	loginLimiter := middleware.NewRateLimiter(5, 0.1)
	r.Get("/api/health", healthH.ServeHTTP)
	r.With(loginLimiter.Middleware).Post("/api/auth/login", authH.Login)
	r.With(loginLimiter.Middleware).Post("/api/auth/setup", authH.Setup)

	r.Group(func(auth chi.Router) {
		auth.Use(middleware.JWTAuth(cfg.Signing))
		auth.Post("/api/auth/logout", authH.Logout)
		auth.Post("/api/auth/change-password", authH.ChangePassword)

		auth.Get("/api/audit", auditH.List)

		auth.Get("/api/proxy/status", proxyH.Status)
		auth.Post("/api/proxy/install", proxyH.Install)
		auth.Post("/api/proxy/start", proxyH.Start)
		auth.Post("/api/proxy/stop", proxyH.Stop)
		auth.Post("/api/proxy/restart", proxyH.Restart)
		auth.Post("/api/proxy/rotate-secret", proxyH.RotateSecret)
		auth.Get("/api/proxy/logs", proxyH.GetLogs)
		auth.Post("/api/proxy/port", proxyH.SetPort)

		auth.Get("/api/links", proxyH.ListLinks)
		auth.Post("/api/links", proxyH.CreateLink)
		auth.Delete("/api/links/{id}", proxyH.RevokeLink)

		// Compatibility aliases
		auth.Get("/api/proxy/links", proxyH.ListLinks)
		auth.Post("/api/proxy/links", proxyH.CreateLink)
		auth.Delete("/api/proxy/links/{id}", proxyH.RevokeLink)

		auth.Get("/api/system/info", systemH.Info)
		auth.Get("/api/system/compatibility", systemH.Compatibility)

		auth.Get("/api/updates/check", updateH.Check)
		auth.Post("/api/updates/apply", updateH.Apply)

		auth.Get("/api/settings", settingsH.Get)
		auth.Put("/api/settings", settingsH.Update)
		auth.Post("/api/settings/password", authH.ChangePassword)
	})

	return r
}

func structuredLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			slog.Debug("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"request_id", middleware.RequestIDFromCtx(r.Context()),
			)
		})
	}
}
