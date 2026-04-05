package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Custom claim keys used in every issued token.
const (
	ClaimSubject       = "sub"     // always "admin"
	ClaimIssuedAt      = "iat"
	ClaimExpiresAt     = "exp"
	ClaimJTI           = "jti"     // unique token ID for future revocation list
	ClaimForceChange   = "pwd_chg" // true → first-login, force password change
	ClaimActorIP       = "ip"      // IP at issue time (informational, not enforced)
)

type claimsKey struct{}

// JWTAuth returns middleware that validates a Bearer token in the Authorization
// header against `signingKey` using HS256.
//
// On success it stores the parsed token in the request context so handlers can
// read claims. On failure it returns 401 with a JSON body.
func JWTAuth(signingKey []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := bearerToken(r)
			if !ok {
				unauthorized(w, "missing or malformed Authorization header")
				return
			}

			tok, err := jwt.ParseString(
				tokenStr,
				jwt.WithKey(jwa.HS256, signingKey),
				jwt.WithValidate(true),
			)
			if err != nil {
				unauthorized(w, "invalid or expired token")
				return
			}

			// Store the parsed token for downstream use.
			ctx := context.WithValue(r.Context(), claimsKey{}, tok)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TokenFromCtx retrieves the parsed JWT from the request context.
// Returns nil if not present (i.e. the route is not behind JWTAuth).
func TokenFromCtx(ctx context.Context) jwt.Token {
	tok, _ := ctx.Value(claimsKey{}).(jwt.Token)
	return tok
}

// MustChangePassword returns true when the token carries the first-login flag,
// meaning the handler should redirect to the password-change flow.
func MustChangePassword(ctx context.Context) bool {
	tok := TokenFromCtx(ctx)
	if tok == nil {
		return false
	}
	v, ok := tok.Get(ClaimForceChange)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

// bearerToken extracts the raw JWT string from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", false
	}
	tok := strings.TrimPrefix(h, "Bearer ")
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return "", false
	}
	return tok, true
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `Bearer realm="mtpanel"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + msg + `"}`)) //nolint:errcheck
}

// --- Token issuance (used by auth handler, kept here to centralise JWT logic) ---

// IssueToken creates and signs a new HS256 JWT for the admin user.
//
//   - subject:     always "admin"
//   - jti:         random UUID to allow future token revocation
//   - forceChange: true on first login
//   - actorIP:     client IP at issuance time
//   - expiry:      now + expireHours
func IssueToken(signingKey []byte, forceChange bool, actorIP string, expireHours int) (string, error) {
	now := time.Now()

	tok, err := jwt.NewBuilder().
		Subject("admin").
		IssuedAt(now).
		Expiration(now.Add(time.Duration(expireHours) * time.Hour)).
		Claim(ClaimJTI, newJTI()).
		Claim(ClaimForceChange, forceChange).
		Claim(ClaimActorIP, actorIP).
		Build()
	if err != nil {
		return "", err
	}

	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, signingKey))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

// newJTI returns a 32-char random hex token ID (128 bits of entropy).
func newJTI() string {
	id, err := randomHex(16)
	if err != nil {
		// Extremely unlikely; crypto/rand failure means the OS is broken.
		return "fallback-" + strings.ReplaceAll(time.Now().UTC().String(), " ", "-")
	}
	return id
}
