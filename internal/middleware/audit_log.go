package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// auditActorKey is the context key under which the audit actor string is stored.
type auditActorKey struct{}

// WithAuditActor injects the actor string (e.g. "admin") into a context so
// audit middleware and handlers can retrieve it without reading the JWT again.
func WithAuditActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, auditActorKey{}, actor)
}

// AuditActorFromCtx retrieves the audit actor from the context.
func AuditActorFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(auditActorKey{}).(string)
	if v == "" {
		return "unknown"
	}
	return v
}

// AuditWriter is implemented by the audit repository to persist events.
type AuditWriter interface {
	Write(ctx context.Context, event AuditRecord) error
}

// AuditRecord is the in-process representation of an audit event before
// it is persisted. It mirrors domain.AuditEvent but lives in middleware to
// avoid circular imports.
type AuditRecord struct {
	ID         string
	EventType  string
	ActorID    string
	ActorIP    string
	ResourceID string
	Detail     string // JSON blob, may be empty
	CreatedAt  time.Time
	Result     string // "success" | "failure"
}

// auditResponseWriter wraps http.ResponseWriter to capture the status code.
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (arw *auditResponseWriter) WriteHeader(code int) {
	arw.statusCode = code
	arw.ResponseWriter.WriteHeader(code)
}

// AuditLog returns middleware that logs every state-changing request (non-GET)
// to the provided AuditWriter. It records the actor, IP, method, path, status,
// duration, and request ID.
//
// Pure-read requests (GET, HEAD, OPTIONS) are not audited to avoid noise, but
// this is trivially configurable.
func AuditLog(writer AuditWriter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only audit mutating methods.
			if r.Method == http.MethodGet ||
				r.Method == http.MethodHead ||
				r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			arw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			start := time.Now()

			next.ServeHTTP(arw, r)

			dur := time.Since(start)
			ctx := r.Context()
			actor := AuditActorFromCtx(ctx)
			ip := realIP(r)
			rid := RequestIDFromCtx(ctx)

			result := "success"
			if arw.statusCode >= 400 {
				result = "failure"
			}

			event := AuditRecord{
				ID:        rid,
				EventType: "http." + r.Method + "." + sanitizePath(r.URL.Path),
				ActorID:   actor,
				ActorIP:   ip,
				Result:    result,
				Detail:    buildDetail(r, arw.statusCode, dur),
				CreatedAt: start,
			}

			// Write asynchronously so audit I/O never slows down the response.
			go func() {
				if err := writer.Write(context.Background(), event); err != nil {
					slog.Error("audit write failed",
						"error", err,
						"event_type", event.EventType,
						"request_id", rid,
					)
				}
			}()

			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", arw.statusCode,
				"duration_ms", dur.Milliseconds(),
				"actor", actor,
				"ip", ip,
				"request_id", rid,
			)
		})
	}
}

// sanitizePath turns "/api/proxy/start" into "api.proxy.start" for use as an
// event_type field so it is safe as a structured-log field name.
func sanitizePath(path string) string {
	b := make([]byte, 0, len(path))
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case c == '/':
			if len(b) > 0 {
				b = append(b, '.')
			}
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

func buildDetail(r *http.Request, status int, dur time.Duration) string {
	// Deliberately minimal: method, status, duration. Do NOT log request bodies
	// (may contain credentials). Handlers that need richer detail should call
	// the AuditWriter directly with a domain-specific AuditRecord.
	return `{"method":"` + r.Method + `","status":` + itoa(status) +
		`,"duration_ms":` + itoa64(dur.Milliseconds()) + `}`
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 5)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

func itoa64(n int64) string {
	return itoa(int(n))
}
