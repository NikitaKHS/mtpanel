package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// envelope is the standard API response wrapper.
type envelope struct {
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		slog.Error("respondJSON encode", "err", err)
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: msg})
}

func respondOK(w http.ResponseWriter, data any) {
	respondJSON(w, http.StatusOK, data)
}

func respondCreated(w http.ResponseWriter, data any) {
	respondJSON(w, http.StatusCreated, data)
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
