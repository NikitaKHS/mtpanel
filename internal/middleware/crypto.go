package middleware

import (
	"crypto/rand"
	"encoding/hex"
)

// randomHex returns n crypto-random bytes encoded as a lowercase hex string.
// n=16 → 32 hex chars (suitable for JWT JTI or MTProxy secrets).
// n=32 → 64 hex chars (suitable for JWT signing keys).
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
