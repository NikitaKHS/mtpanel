package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mtpanel/mtpanel/internal/infrastructure/db"
)

type sqliteSettingsRepository struct {
	db *db.DB
}

// NewSettingsRepository creates a SettingsRepository backed by SQLite.
func NewSettingsRepository(d *db.DB) SettingsRepository {
	return &sqliteSettingsRepository{db: d}
}

func (r *sqliteSettingsRepository) Get(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx,
		`SELECT value FROM app_settings WHERE key = ?`, key,
	).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("settings key %q not found", key)
	}
	return value, err
}

func (r *sqliteSettingsRepository) Set(ctx context.Context, key, value string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, now,
	)
	return err
}

func (r *sqliteSettingsRepository) GetAll(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}
