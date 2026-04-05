package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/repository"
	"github.com/mtpanel/mtpanel/internal/service"
)

type AuthHandler struct {
	svc      *service.AuthService
	settings repository.SettingsRepository
	audit    repository.AuditRepository
}

func NewAuthHandler(svc *service.AuthService, settings repository.SettingsRepository, audit repository.AuditRepository) *AuthHandler {
	return &AuthHandler{svc: svc, settings: settings, audit: audit}
}

type loginRequest struct {
	Password string `json:"password"`
}

type setupRequest struct {
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.svc.Login(r.Context(), req.Password, realClientIP(r))
	if err != nil {
		if errors.Is(err, service.ErrNoAdminConfigured) {
			respondError(w, http.StatusPreconditionRequired, "admin password not configured")
			return
		}
		if errors.Is(err, service.ErrBadCredentials) {
			h.recordAudit(r, domain.AuditEventLoginFailed, "admin", "bad password")
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.recordAudit(r, domain.AuditEventLogin, "admin", "login successful")
	respondOK(w, loginResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt.Format(time.RFC3339),
	})
}

func (h *AuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	var req setupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res, err := h.svc.SetupAdmin(r.Context(), req.Password, realClientIP(r))
	if err != nil {
		if isPasswordPolicyError(err) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.recordAudit(r, domain.AuditEventSettingChanged, "admin", "initial admin password configured")
	respondCreated(w, loginResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt.Format(time.RFC3339),
	})
}

func isPasswordPolicyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return msg == "password must be at least 12 characters" ||
		msg == "password must not exceed 128 characters"
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	respondNoContent(w)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), req.CurrentPassword, req.NewPassword, realClientIP(r)); err != nil {
		if errors.Is(err, service.ErrBadCredentials) {
			respondError(w, http.StatusUnauthorized, "invalid current password")
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	_ = h.settings.Set(r.Context(), "is_first_run", "false")
	h.recordAudit(r, domain.AuditEventSettingChanged, "admin", "password changed")
	respondNoContent(w)
}

func (h *AuthHandler) recordAudit(r *http.Request, et domain.AuditEventType, resID, detail string) {
	ev := &domain.AuditEvent{
		ID:         uuid.New().String(),
		EventType:  et,
		ActorID:    "admin",
		ActorIP:    realClientIP(r),
		ResourceID: resID,
		Detail:     detail,
		CreatedAt:  time.Now().UTC(),
	}
	_ = h.audit.Record(r.Context(), ev)
}

func realClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
