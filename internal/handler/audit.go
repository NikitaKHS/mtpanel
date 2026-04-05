package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mtpanel/mtpanel/internal/domain"
)

// AuditReader abstracts the persistence layer for reading audit events.
type AuditReader interface {
	List(ctx context.Context, limit, offset int) ([]*domain.AuditEvent, error)
}

// AuditHandler serves GET /api/audit.
type AuditHandler struct {
	repo AuditReader
}

// NewAuditHandler constructs an AuditHandler.
func NewAuditHandler(repo AuditReader) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// auditListResponse is the JSON envelope for the audit log.
type auditListResponse struct {
	Events []*domain.AuditEvent `json:"events"`
	Total  int                 `json:"total"`
}

// List handles GET /api/audit?limit=N.
// Requires authentication (enforced by JWTAuth on the route group).
// Defaults to the last 100 events; maximum 200.
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	limit := 100
	offset := 0
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if n, err := strconv.Atoi(ls); err == nil && n > 0 {
			limit = n
		}
	}
	if os := r.URL.Query().Get("offset"); os != "" {
		if n, err := strconv.Atoi(os); err == nil && n >= 0 {
			offset = n
		}
	}
	// Hard cap enforced in the repository layer too, but belt-and-suspenders.
	if limit > 200 {
		limit = 200
	}

	events, err := h.repo.List(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to retrieve audit log")
		return
	}

	if events == nil {
		events = []*domain.AuditEvent{}
	}

	respondOK(w, auditListResponse{
		Events: events,
		Total:  len(events),
	})
}
