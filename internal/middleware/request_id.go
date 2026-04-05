package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID injects a UUID v4 into every request context and response header.
// Downstream handlers and audit logging pull it via RequestIDFromCtx.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Honour an upstream proxy's request ID if present (e.g. nginx).
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromCtx retrieves the request ID from the context.
// Returns empty string if not present (should never happen in practice).
func RequestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}
