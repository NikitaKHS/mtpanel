package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mtpanel/mtpanel/internal/middleware"
	"golang.org/x/crypto/bcrypt"
)

// ErrBadCredentials is returned when the password does not match.
// It is intentionally vague to prevent user enumeration.
var ErrBadCredentials = errors.New("invalid credentials")

// ErrNoAdminConfigured is returned when the admin account has not been set up
// yet (first-run state).
var ErrNoAdminConfigured = errors.New("admin account not configured")

// AdminStore abstracts the persistence layer for admin credentials.
// Implemented by the SQLite settings repository.
type AdminStore interface {
	// GetPasswordHash returns the bcrypt hash stored for the admin account.
	// Returns ErrNoAdminConfigured if IsFirstRun is still true.
	GetPasswordHash(ctx context.Context) (string, error)

	// SetPasswordHash persists a new bcrypt hash and clears IsFirstRun.
	SetPasswordHash(ctx context.Context, hash string) error

	// IsFirstRun returns true if no admin password has been set yet.
	IsFirstRun(ctx context.Context) (bool, error)
}

// AuthService handles authentication logic: login, password setup, token
// issuance. It is deliberately thin — all JWT mechanics live in middleware.
type AuthService struct {
	store      AdminStore
	signingKey []byte
	expireHours int
}

// NewAuthService constructs an AuthService with the provided dependencies.
func NewAuthService(store AdminStore, signingKey []byte, expireHours int) *AuthService {
	return &AuthService{
		store:       store,
		signingKey:  signingKey,
		expireHours: expireHours,
	}
}

// LoginResult is returned on a successful login.
type LoginResult struct {
	Token       string    `json:"token"`
	ExpiresAt   time.Time `json:"expires_at"`
	ForceChange bool      `json:"force_password_change"`
}

// Login validates the provided password against the stored bcrypt hash and
// issues a JWT if correct.
//
// It uses bcrypt.CompareHashAndPassword which is inherently timing-safe.
// A deliberate constant-time sleep is NOT added here because bcrypt's own
// cost factor already dominates timing; adding a sleep would only mask errors.
func (s *AuthService) Login(ctx context.Context, password, actorIP string) (*LoginResult, error) {
	firstRun, err := s.store.IsFirstRun(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: check first run: %w", err)
	}
	if firstRun {
		return nil, ErrNoAdminConfigured
	}

	hash, err := s.store.GetPasswordHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: get hash: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		// Log at Info so failed attempts are visible in audit log without
		// leaking the password or specific error.
		slog.Info("login failed", "ip", actorIP)
		return nil, ErrBadCredentials
	}

	token, err := middleware.IssueToken(s.signingKey, false, actorIP, s.expireHours)
	if err != nil {
		return nil, fmt.Errorf("auth: issue token: %w", err)
	}

	return &LoginResult{
		Token:       token,
		ExpiresAt:   time.Now().Add(time.Duration(s.expireHours) * time.Hour),
		ForceChange: false,
	}, nil
}

// SetupAdmin is called on first-run to set the initial admin password.
// It hashes the password with bcrypt cost 12 and persists it.
// After this call IsFirstRun transitions to false.
//
// bcrypt cost 12 takes ~250 ms on a modern server — appropriate for a login
// endpoint; slow enough to resist offline dictionary attacks.
func (s *AuthService) SetupAdmin(ctx context.Context, password, actorIP string) (*LoginResult, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	if err := s.store.SetPasswordHash(ctx, string(hash)); err != nil {
		return nil, fmt.Errorf("auth: store hash: %w", err)
	}

	// Issue token with force_change=false: the user just chose their password.
	token, err := middleware.IssueToken(s.signingKey, false, actorIP, s.expireHours)
	if err != nil {
		return nil, fmt.Errorf("auth: issue token: %w", err)
	}

	return &LoginResult{
		Token:       token,
		ExpiresAt:   time.Now().Add(time.Duration(s.expireHours) * time.Hour),
		ForceChange: false,
	}, nil
}

// ChangePassword verifies the current password and replaces it with a new one.
func (s *AuthService) ChangePassword(ctx context.Context, currentPassword, newPassword, actorIP string) error {
	hash, err := s.store.GetPasswordHash(ctx)
	if err != nil {
		return fmt.Errorf("auth: get hash: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)); err != nil {
		return ErrBadCredentials
	}

	if err := validatePassword(newPassword); err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return fmt.Errorf("auth: hash new password: %w", err)
	}

	return s.store.SetPasswordHash(ctx, string(newHash))
}

// validatePassword enforces a minimal password policy.
// Deliberately not overly complex for a self-hosted tool.
func validatePassword(p string) error {
	if len(p) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	if len(p) > 128 {
		// Prevent bcrypt DoS via extremely long passwords.
		return errors.New("password must not exceed 128 characters")
	}
	return nil
}
