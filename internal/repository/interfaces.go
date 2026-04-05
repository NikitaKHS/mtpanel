package repository

import (
	"context"

	"github.com/mtpanel/mtpanel/internal/domain"
)

// NodeRepository manages node persistence.
type NodeRepository interface {
	Create(ctx context.Context, n *domain.Node) error
	Get(ctx context.Context, id string) (*domain.Node, error)
	List(ctx context.Context) ([]*domain.Node, error)
	UpdateStatus(ctx context.Context, id string, status domain.NodeStatus) error
	Delete(ctx context.Context, id string) error
}

// ProxyConfigRepository manages proxy configuration persistence.
type ProxyConfigRepository interface {
	Upsert(ctx context.Context, cfg *domain.ProxyConfig) error
	Get(ctx context.Context, id string) (*domain.ProxyConfig, error)
	GetActive(ctx context.Context) (*domain.ProxyConfig, error)
}

// ProxyLinkRepository manages shareable proxy link persistence.
type ProxyLinkRepository interface {
	Create(ctx context.Context, link *domain.ProxyLink) error
	Get(ctx context.Context, id string) (*domain.ProxyLink, error)
	List(ctx context.Context, activeOnly bool) ([]*domain.ProxyLink, error)
	Revoke(ctx context.Context, id string) error
}

// AuditRepository records and queries audit events.
type AuditRepository interface {
	Record(ctx context.Context, event *domain.AuditEvent) error
	List(ctx context.Context, limit, offset int) ([]*domain.AuditEvent, error)
}

// SettingsRepository is a simple key-value store for app settings.
type SettingsRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
	GetAll(ctx context.Context) (map[string]string, error)
}
