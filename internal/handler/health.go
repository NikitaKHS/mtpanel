package handler

import (
	"database/sql"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// startTime records when the process started.
var startTime = time.Now()

// HealthHandler serves GET /api/health.
type HealthHandler struct {
	db          *sql.DB
	proxyStatus func() string // callback: returns "running"|"stopped"|"unknown"
	version     string
}

// NewHealthHandler constructs a HealthHandler.
//
//   - db:          live database connection (used to probe db_ok).
//   - proxyStatus: a function that returns the current proxy status string.
//   - version:     application version string (e.g. "1.2.3").
func NewHealthHandler(db *sql.DB, proxyStatus func() string, version string) *HealthHandler {
	if version == "" {
		version = buildVersion()
	}
	return &HealthHandler{
		db:          db,
		proxyStatus: proxyStatus,
		version:     version,
	}
}

// healthResponse is the JSON shape of GET /api/health.
type healthResponse struct {
	Status        string `json:"status"`           // "ok" | "degraded"
	Version       string `json:"version"`
	ProxyStatus   string `json:"proxy_status"`     // "running"|"stopped"|"unknown"|"failed"
	UptimeSeconds int64  `json:"uptime_seconds"`
	DBOK          bool   `json:"db_ok"`
}

// ServeHTTP handles GET /api/health.
// It is intentionally cheap: one DB ping, one proxy status call.
// The endpoint is unauthenticated so external monitors can use it.
// It does NOT expose sensitive data.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:        "ok",
		Version:       h.version,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
	}

	// Probe the database with a 2-second timeout.
	ctx := r.Context()
	if err := h.db.PingContext(ctx); err != nil {
		resp.DBOK = false
		resp.Status = "degraded"
	} else {
		resp.DBOK = true
	}

	// Fetch proxy status from the runtime registry.
	if h.proxyStatus != nil {
		resp.ProxyStatus = h.proxyStatus()
	} else {
		resp.ProxyStatus = "unknown"
	}

	status := http.StatusOK
	if resp.Status != "ok" {
		status = http.StatusServiceUnavailable
	}

	respondJSON(w, status, resp)
}

// buildVersion tries to read the module version from the embedded build info.
// Falls back to "dev" if not available.
func buildVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return "dev"
}

// --- Optional Prometheus metrics (post-MVP) ---

// ProxyRunningGauge is an atomic flag (0 or 1) read by the /metrics handler.
// Set it from the proxy service when state changes.
var ProxyRunningGauge atomic.Int32

// ProxyRestartsTotal counts total proxy restarts since process start.
var ProxyRestartsTotal atomic.Int64

// ActiveLinksCount holds the current count of active proxy links.
var ActiveLinksCount atomic.Int32
