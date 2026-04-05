package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// FirstRunConfig is the output of the first-run bootstrapping process.
type FirstRunConfig struct {
	JWTSecret     string // 64-char hex (256-bit entropy)
	InitialPassword string // 24-char random alphanumeric, printed to stdout once
}

// ConfigPersister saves generated secrets into the config file.
type ConfigPersister interface {
	// SaveJWTSecret writes the generated JWT signing key to persistent config.
	SaveJWTSecret(ctx context.Context, secret string) error

	// SaveAdminHash persists the initial bcrypt-hashed password.
	SaveAdminHash(ctx context.Context, hash string) error
}

// Bootstrap performs first-run security setup:
//  1. Generates a 256-bit random JWT signing key.
//  2. Generates a random initial admin password.
//  3. Hashes it with bcrypt cost 12.
//  4. Persists both.
//  5. Prints the initial password to stdout ONCE and never again.
//
// This function is idempotent: if JWTSecret is already set it will not
// overwrite it (controlled by caller — check cfg.IsFirstRun before calling).
func Bootstrap(ctx context.Context, persister ConfigPersister) (*FirstRunConfig, error) {
	jwtSecret, err := generateHex(32) // 32 bytes = 64 hex chars = 256 bits
	if err != nil {
		return nil, fmt.Errorf("firstrun: generate JWT secret: %w", err)
	}

	initialPassword, err := generatePassword(24)
	if err != nil {
		return nil, fmt.Errorf("firstrun: generate initial password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(initialPassword), 12)
	if err != nil {
		return nil, fmt.Errorf("firstrun: hash initial password: %w", err)
	}

	if err := persister.SaveJWTSecret(ctx, jwtSecret); err != nil {
		return nil, fmt.Errorf("firstrun: save JWT secret: %w", err)
	}

	if err := persister.SaveAdminHash(ctx, string(hash)); err != nil {
		return nil, fmt.Errorf("firstrun: save admin hash: %w", err)
	}

	// Print the initial password to stdout exactly once.
	// This is the ONLY place the plaintext password ever appears.
	printInitialPassword(initialPassword)

	return &FirstRunConfig{
		JWTSecret:       jwtSecret,
		InitialPassword: initialPassword,
	}, nil
}

// printInitialPassword writes the initial password to stdout with a clear
// banner. This is intentionally written to stdout (not the log) because:
//   - Logs may be forwarded to a remote system (bad for secrets).
//   - The operator is expected to copy it from the terminal.
func printInitialPassword(password string) {
	// Write directly to fd 1 to avoid log redirection.
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "╔══════════════════════════════════════════════════════╗\n")
	fmt.Fprintf(os.Stdout, "║           MTPanel — First-Run Setup                  ║\n")
	fmt.Fprintf(os.Stdout, "╠══════════════════════════════════════════════════════╣\n")
	fmt.Fprintf(os.Stdout, "║  Initial admin password:                             ║\n")
	fmt.Fprintf(os.Stdout, "║  %-52s║\n", password)
	fmt.Fprintf(os.Stdout, "║                                                      ║\n")
	fmt.Fprintf(os.Stdout, "║  This password will NOT be shown again.              ║\n")
	fmt.Fprintf(os.Stdout, "║  Change it immediately at /setup after first login.  ║\n")
	fmt.Fprintf(os.Stdout, "╚══════════════════════════════════════════════════════╝\n")
	fmt.Fprintf(os.Stdout, "\n")

	slog.Info("first-run setup complete; initial password printed to stdout")
}

// generateHex returns n random bytes as a lowercase hex string.
func generateHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// generatePassword returns a URL-safe random password of length n using only
// characters that are unambiguous when read aloud: a-z, A-Z, 0-9 minus
// visually similar characters (0/O, 1/l/I).
func generatePassword(n int) (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b), nil
}
