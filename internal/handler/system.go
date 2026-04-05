package handler

import (
	"net/http"

	"github.com/mtpanel/mtpanel/internal/service"
)

type SystemHandler struct {
	svc *service.SystemService
}

func NewSystemHandler(svc *service.SystemService) *SystemHandler {
	return &SystemHandler{svc: svc}
}

func (h *SystemHandler) Info(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.GetSystemInfo(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, info)
}

func (h *SystemHandler) Compatibility(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.CheckCompatibility(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]any{
		"compatible": report.Supported,
		"report":     report,
	})
}
