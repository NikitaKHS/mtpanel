package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/db"
)

// ---- ProxyConfigRepository ----

type sqliteProxyConfigRepository struct {
	db *db.DB
}

// NewProxyConfigRepository creates a ProxyConfigRepository backed by SQLite.
func NewProxyConfigRepository(d *db.DB) ProxyConfigRepository {
	return &sqliteProxyConfigRepository{db: d}
}

func (r *sqliteProxyConfigRepository) Upsert(ctx context.Context, cfg *domain.ProxyConfig) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO proxy_configs
			(id, port, secret, tag, workers, max_conn, nat_prefix, extra_args, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			port       = excluded.port,
			secret     = excluded.secret,
			tag        = excluded.tag,
			workers    = excluded.workers,
			max_conn   = excluded.max_conn,
			nat_prefix = excluded.nat_prefix,
			extra_args = excluded.extra_args,
			updated_at = excluded.updated_at`,
		cfg.ID, cfg.Port, cfg.Secret, cfg.Tag, cfg.Workers,
		cfg.MaxConn, cfg.NATPrefix, cfg.ExtraArgs, now, now,
	)
	return err
}

func (r *sqliteProxyConfigRepository) Get(ctx context.Context, id string) (*domain.ProxyConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, port, secret, tag, workers, max_conn, nat_prefix, extra_args, created_at, updated_at
		FROM proxy_configs WHERE id = ?`, id)
	return scanProxyConfig(row)
}

func (r *sqliteProxyConfigRepository) GetActive(ctx context.Context) (*domain.ProxyConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, port, secret, tag, workers, max_conn, nat_prefix, extra_args, created_at, updated_at
		FROM proxy_configs ORDER BY updated_at DESC LIMIT 1`)
	return scanProxyConfig(row)
}

func scanProxyConfig(s rowScanner) (*domain.ProxyConfig, error) {
	var c domain.ProxyConfig
	var createdStr, updatedStr string
	if err := s.Scan(
		&c.ID, &c.Port, &c.Secret, &c.Tag, &c.Workers,
		&c.MaxConn, &c.NATPrefix, &c.ExtraArgs,
		&createdStr, &updatedStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proxy config not found")
		}
		return nil, fmt.Errorf("proxy config scan: %w", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &c, nil
}

// ---- ProxyLinkRepository ----

type sqliteProxyLinkRepository struct {
	db *db.DB
}

// NewProxyLinkRepository creates a ProxyLinkRepository backed by SQLite.
func NewProxyLinkRepository(d *db.DB) ProxyLinkRepository {
	return &sqliteProxyLinkRepository{db: d}
}

func (r *sqliteProxyLinkRepository) Create(ctx context.Context, link *domain.ProxyLink) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO proxy_links (id, label, secret, host, port, link, active, created_at, revoked_at)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, NULL)`,
		link.ID, link.Label, link.Secret, link.Host, link.Port, link.Link, now,
	)
	return err
}

func (r *sqliteProxyLinkRepository) Get(ctx context.Context, id string) (*domain.ProxyLink, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, label, secret, host, port, link, active, created_at, revoked_at
		FROM proxy_links WHERE id = ?`, id)
	return scanProxyLink(row)
}

func (r *sqliteProxyLinkRepository) List(ctx context.Context, activeOnly bool) ([]*domain.ProxyLink, error) {
	query := `SELECT id, label, secret, host, port, link, active, created_at, revoked_at
	          FROM proxy_links`
	if activeOnly {
		query += ` WHERE active = 1`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*domain.ProxyLink
	for rows.Next() {
		l, err := scanProxyLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

func (r *sqliteProxyLinkRepository) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx,
		`UPDATE proxy_links SET active = 0, revoked_at = ? WHERE id = ? AND active = 1`,
		now, id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("link %q not found or already revoked", id)
	}
	return nil
}

func scanProxyLink(s rowScanner) (*domain.ProxyLink, error) {
	var l domain.ProxyLink
	var createdStr string
	var revokedStr sql.NullString
	var active int

	if err := s.Scan(
		&l.ID, &l.Label, &l.Secret, &l.Host, &l.Port,
		&l.Link, &active, &createdStr, &revokedStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("proxy link not found")
		}
		return nil, fmt.Errorf("proxy link scan: %w", err)
	}
	l.Active = active == 1
	l.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	if revokedStr.Valid {
		t, _ := time.Parse(time.RFC3339, revokedStr.String)
		l.RevokedAt = &t
	}
	return &l, nil
}
