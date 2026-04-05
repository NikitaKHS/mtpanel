package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mtpanel/mtpanel/internal/config"
	"github.com/mtpanel/mtpanel/internal/repository"
	"github.com/mtpanel/mtpanel/internal/service"
)

type SettingsHandler struct {
	cfg      *config.Config
	settings repository.SettingsRepository
	proxySvc *service.ProxyService
}

func NewSettingsHandler(cfg *config.Config, settings repository.SettingsRepository, proxySvc *service.ProxyService) *SettingsHandler {
	return &SettingsHandler{cfg: cfg, settings: settings, proxySvc: proxySvc}
}

type settingsPayload struct {
	ListenAddr string `json:"listen_addr,omitempty"`
	ProxyPort  *int   `json:"proxy_port,omitempty"`
}

func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	resp := settingsPayload{
		ListenAddr: h.cfg.ListenAddr,
	}

	if v, err := h.settings.Get(r.Context(), "listen_addr"); err == nil && v != "" {
		resp.ListenAddr = v
	}

	port := h.cfg.MTProxyPort
	if v, err := h.settings.Get(r.Context(), "mtproxy_port"); err == nil && v != "" {
		if parsed, parseErr := strconv.Atoi(v); parseErr == nil {
			port = parsed
		}
	}
	resp.ProxyPort = &port

	respondOK(w, resp)
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req settingsPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ListenAddr != "" {
		if err := h.settings.Set(r.Context(), "listen_addr", req.ListenAddr); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.ProxyPort != nil {
		if err := h.proxySvc.SetPort(r.Context(), *req.ProxyPort); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := h.settings.Set(r.Context(), "mtproxy_port", strconv.Itoa(*req.ProxyPort)); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondNoContent(w)
}
