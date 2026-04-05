package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mtpanel/mtpanel/internal/service"
)

// ProxyHandler handles all /api/proxy/* endpoints.
type ProxyHandler struct {
	svc *service.ProxyService
}

// NewProxyHandler creates a ProxyHandler.
func NewProxyHandler(svc *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{svc: svc}
}

// Install handles POST /api/proxy/install.
func (h *ProxyHandler) Install(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port *int `json:"port"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	var err error
	if req.Port != nil {
		err = h.svc.InstallWithPort(r.Context(), *req.Port)
	} else {
		err = h.svc.Install(r.Context())
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]any{
		"success": true,
		"message": "MTProxy installed",
	})
}

// Start handles POST /api/proxy/start.
func (h *ProxyHandler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Start(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]string{"status": "started"})
}

// Stop handles POST /api/proxy/stop.
func (h *ProxyHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Stop(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]string{"status": "stopped"})
}

// Restart handles POST /api/proxy/restart.
func (h *ProxyHandler) Restart(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Restart(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]string{"status": "restarted"})
}

// Status handles GET /api/proxy/status.
func (h *ProxyHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.svc.Status(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := map[string]any{"status": string(status)}
	if cfg, cfgErr := h.svc.ActiveConfig(r.Context()); cfgErr == nil && cfg != nil {
		resp["port"] = cfg.Port
		resp["secret"] = cfg.Secret
	}
	respondOK(w, resp)
}

// GetLogs handles GET /api/proxy/logs.
// Optional query param: ?lines=N (default 100, max 1000).
func (h *ProxyHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	lines := 100
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 1000 {
			lines = n
		}
	}
	logLines, err := h.svc.GetLogs(r.Context(), lines)
	if err != nil {
		if errors.Is(err, service.ErrProxyNotInstalled) {
			respondError(w, http.StatusConflict, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]any{"lines": logLines})
}

// SetPort handles POST /api/proxy/port.
func (h *ProxyHandler) SetPort(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port int `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetPort(r.Context(), req.Port); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondNoContent(w)
}

// RotateSecret handles POST /api/proxy/rotate-secret.
func (h *ProxyHandler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	secret, err := h.svc.RotateSecret(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]string{"secret": secret})
}

// ListLinks handles GET /api/proxy/links.
func (h *ProxyHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active") != "false"
	links, err := h.svc.ListLinks(r.Context(), activeOnly)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]any{"links": links})
}

type createLinkRequest struct {
	Label string `json:"label"`
}

// CreateLink handles POST /api/proxy/links.
func (h *ProxyHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	var req createLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Label == "" {
		req.Label = "link"
	}

	link, err := h.svc.GenerateLink(r.Context(), req.Label)
	if err != nil {
		if errors.Is(err, service.ErrProxyNotInstalled) {
			respondError(w, http.StatusConflict, "MTProxy пока не установлен. Сначала установите его в разделе Прокси.")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondCreated(w, link)
}

// RevokeLink handles DELETE /api/proxy/links/:id.
func (h *ProxyHandler) RevokeLink(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, http.StatusBadRequest, "link id required")
		return
	}
	if err := h.svc.RevokeLink(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondNoContent(w)
}
