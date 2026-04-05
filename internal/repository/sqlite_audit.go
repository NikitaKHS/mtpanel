package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/db"
)

type sqliteAuditRepository struct {
	db *db.DB
}

// NewAuditRepository creates an AuditRepository backed by SQLite.
func NewAuditRepository(d *db.DB) AuditRepository {
	return &sqliteAuditRepository{db: d}
}

func (r *sqliteAuditRepository) Record(ctx context.Context, e *domain.AuditEvent) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_events (id, event_type, actor_id, actor_ip, resource_id, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID,
		string(e.EventType),
		e.ActorID,
		e.ActorIP,
		e.ResourceID,
		e.Detail,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("audit.Record: %w", err)
	}
	return nil
}

func (r *sqliteAuditRepository) List(ctx context.Context, limit, offset int) ([]*domain.AuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, event_type, actor_id, actor_ip, resource_id, detail, created_at
		FROM audit_events
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("audit.List: %w", err)
	}
	defer rows.Close()

	var events []*domain.AuditEvent
	for rows.Next() {
		var ev domain.AuditEvent
		var createdStr string
		if err := rows.Scan(
			&ev.ID, &ev.EventType, &ev.ActorID, &ev.ActorIP,
			&ev.ResourceID, &ev.Detail, &createdStr,
		); err != nil {
			return nil, fmt.Errorf("audit.List scan: %w", err)
		}
		ev.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		events = append(events, &ev)
	}
	return events, rows.Err()
}
