package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// tokenBucket is a per-IP rate limiter using the token-bucket algorithm.
type tokenBucket struct {
	mu       sync.Mutex
	tokens   float64
	lastSeen time.Time
}

// RateLimiter holds per-IP state and configuration.
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*tokenBucket
	rate     float64 // tokens refilled per second
	capacity float64 // max burst size (== max tokens)
}

// NewRateLimiter creates a limiter that allows `capacity` bursts,
// then refills at `ratePerSec` tokens per second.
//
// For brute-force protection on /api/auth/login use:
//
//	NewRateLimiter(5, 5)   // 5 attempts burst, 1/s steady state is too slow;
//	                        // 5 burst then 5 req/s refill is fine for humans.
//
// Tune down for stricter protection: NewRateLimiter(3, 0.1) allows 3 attempts
// then one every 10 seconds.
func NewRateLimiter(capacity, ratePerSec float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:  make(map[string]*tokenBucket),
		rate:     ratePerSec,
		capacity: capacity,
	}
	// Sweep stale buckets every minute to avoid unbounded map growth.
	go rl.sweep(time.Minute)
	return rl
}

// Middleware returns a chi-compatible middleware that applies the rate limit.
// On limit exceeded it responds 429 and does NOT call the next handler.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		if !rl.allow(ip) {
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow consumes one token for the given key. Returns true if the request
// is permitted.
func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[key]
	if !ok {
		b = &tokenBucket{tokens: rl.capacity, lastSeen: time.Now()}
		rl.buckets[key] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rl.rate
	if b.tokens > rl.capacity {
		b.tokens = rl.capacity
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweep removes buckets that have been idle for longer than `ttl`.
func (rl *RateLimiter) sweep(ttl time.Duration) {
	ticker := time.NewTicker(ttl)
	for range ticker.C {
		cutoff := time.Now().Add(-ttl)
		rl.mu.Lock()
		for k, b := range rl.buckets {
			b.mu.Lock()
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, k)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// realIP extracts the client IP, honouring X-Forwarded-For set by a trusted
// reverse proxy. Falls back to RemoteAddr.
//
// WARNING: only trust X-Forwarded-For if the panel is deployed behind a
// controlled reverse proxy. If run directly on the internet, remove the XFF
// branch to prevent IP spoofing.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the left-most IP (the original client).
		for _, part := range splitCSV(xff) {
			if ip := net.ParseIP(trim(part)); ip != nil {
				return ip.String()
			}
		}
	}
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		if ip := net.ParseIP(trim(xri)); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func trim(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
