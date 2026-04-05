package domain

import "time"

// AuditEventType classifies the kind of action recorded.
type AuditEventType string

const (
	AuditEventLogin          AuditEventType = "auth.login"
	AuditEventLoginFailed    AuditEventType = "auth.login_failed"
	AuditEventProxyStart     AuditEventType = "proxy.start"
	AuditEventProxyStop      AuditEventType = "proxy.stop"
	AuditEventProxyRestart   AuditEventType = "proxy.restart"
	AuditEventProxyInstall   AuditEventType = "proxy.install"
	AuditEventSecretRotate   AuditEventType = "proxy.secret_rotate"
	AuditEventLinkCreate     AuditEventType = "link.create"
	AuditEventLinkRevoke     AuditEventType = "link.revoke"
	AuditEventUpdateApplied  AuditEventType = "update.applied"
	AuditEventSettingChanged AuditEventType = "settings.changed"
)

// AuditEvent is an immutable record of an administrative action.
type AuditEvent struct {
	ID         string         `json:"id"`
	EventType  AuditEventType `json:"event_type"`
	ActorID    string         `json:"actor_id"`   // user / "system"
	ActorIP    string         `json:"actor_ip"`
	ResourceID string         `json:"resource_id,omitempty"`
	Detail     string         `json:"detail,omitempty"` // JSON blob
	CreatedAt  time.Time      `json:"created_at"`
}
