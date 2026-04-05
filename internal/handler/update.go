package handler

import (
	"net/http"

	"github.com/mtpanel/mtpanel/internal/service"
)

type UpdateHandler struct {
	svc *service.UpdateService
}

func NewUpdateHandler(svc *service.UpdateService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

func (h *UpdateHandler) Check(w http.ResponseWriter, r *http.Request) {
	info, err := h.svc.CheckUpdate(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, info)
}

func (h *UpdateHandler) Apply(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Update(r.Context()); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondOK(w, map[string]any{
		"success": true,
		"message": "update applied",
	})
}
