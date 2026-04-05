package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mtpanel/mtpanel/internal/domain"
	"github.com/mtpanel/mtpanel/internal/infrastructure/db"
)

type sqliteNodeRepository struct {
	db *db.DB
}

// NewNodeRepository creates a NodeRepository backed by SQLite.
func NewNodeRepository(d *db.DB) NodeRepository {
	return &sqliteNodeRepository{db: d}
}

func (r *sqliteNodeRepository) Create(ctx context.Context, n *domain.Node) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO nodes (id, name, host, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		n.ID, n.Name, n.Host, string(n.Status), now, now,
	)
	if err != nil {
		return fmt.Errorf("node.Create: %w", err)
	}
	return nil
}

func (r *sqliteNodeRepository) Get(ctx context.Context, id string) (*domain.Node, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, host, status, created_at, updated_at
		FROM nodes WHERE id = ?`, id)
	return scanNode(row)
}

func (r *sqliteNodeRepository) List(ctx context.Context) ([]*domain.Node, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, host, status, created_at, updated_at
		FROM nodes ORDER BY created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("node.List: %w", err)
	}
	defer rows.Close()

	var nodes []*domain.Node
	for rows.Next() {
		n, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (r *sqliteNodeRepository) UpdateStatus(ctx context.Context, id string, status domain.NodeStatus) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx,
		`UPDATE nodes SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), now, id,
	)
	if err != nil {
		return fmt.Errorf("node.UpdateStatus: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("node.UpdateStatus: id %q not found", id)
	}
	return nil
}

func (r *sqliteNodeRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM nodes WHERE id = ?`, id)
	return err
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanNode(s rowScanner) (*domain.Node, error) {
	var n domain.Node
	var createdStr, updatedStr string
	if err := s.Scan(&n.ID, &n.Name, &n.Host, &n.Status, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found")
		}
		return nil, fmt.Errorf("node scan: %w", err)
	}
	n.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	n.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &n, nil
}
