package middleware

import "net/http"

// SecurityHeaders adds hardened HTTP response headers to every response.
// These are safe defaults for a self-hosted admin panel served over HTTPS via
// a reverse proxy. Adjust CSP if you load external fonts or analytics.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Prevent the browser from rendering the page inside a frame/iframe.
		h.Set("X-Frame-Options", "DENY")

		// Stop MIME-type sniffing; serve declared Content-Type only.
		h.Set("X-Content-Type-Options", "nosniff")

		// Full Referrer-Policy: no referrer sent to third-party origins.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy:
		//   - default-src 'self'          → only same-origin resources
		//   - script-src 'self'           → no inline scripts, no eval
		//   - style-src 'self' 'unsafe-inline' → SvelteKit inlines critical CSS
		//   - img-src 'self' data:         → data URIs for inline SVGs
		//   - connect-src 'self'           → fetch/XHR to same origin only
		//   - frame-ancestors 'none'       → belt-and-suspenders on framing
		//   - base-uri 'self'             → block base-tag injection
		//   - form-action 'self'          → block cross-origin form POSTs
		h.Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'")

		// Permissions-Policy: disable every powerful feature the panel doesn't use.
		h.Set("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=(), usb=()")

		// HSTS: 1 year, include subdomains.
		// Only effective when served over TLS; harmless over HTTP but noted.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Cross-Origin headers for Fetch isolation.
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")

		next.ServeHTTP(w, r)
	})
}
