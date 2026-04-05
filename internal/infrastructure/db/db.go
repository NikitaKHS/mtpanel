package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Migrations holds SQL migration files.
// By default migrations are embedded into the binary for reliable deployments.
var Migrations fs.FS = embeddedMigrations

// MigrationsDir is the path inside the embed.FS that contains *.sql files.
var MigrationsDir = "migrations"

// DB wraps *sql.DB with helpers.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at dsn, applies all
// pragmas needed for safe concurrent use, then runs pending migrations.
func Open(ctx context.Context, dsn string) (*DB, error) {
	// modernc/sqlite DSN: file path or file::memory:
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", dsn, err)
	}

	// Single writer, multiple readers is fine for a management panel.
	// WAL mode allows concurrent reads while a write is in progress.
	raw.SetMaxOpenConns(1)
	raw.SetMaxIdleConns(1)
	raw.SetConnMaxLifetime(0)

	if err := applyPragmas(ctx, raw); err != nil {
		raw.Close()
		return nil, err
	}

	d := &DB{DB: raw}
	if err := d.migrate(ctx); err != nil {
		raw.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}

	return d, nil
}

// applyPragmas sets per-connection SQLite settings.
func applyPragmas(ctx context.Context, db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA cache_size = -8000;", // 8 MB
		"PRAGMA temp_store = MEMORY;",
	}
	for _, p := range pragmas {
		if _, err := db.ExecContext(ctx, p); err != nil {
			return fmt.Errorf("db: pragma %q: %w", p, err)
		}
	}
	return nil
}

// migrate runs all SQL files from the embedded FS that have not yet been applied.
func (d *DB) migrate(ctx context.Context) error {
	// Ensure tracking table exists (bootstrap).
	if _, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	files, err := listMigrationFiles()
	if err != nil {
		return err
	}

	for _, name := range files {
		version := migrationVersion(name)

		var existing string
		err := d.QueryRowContext(ctx,
			`SELECT version FROM schema_migrations WHERE version = ?`, version,
		).Scan(&existing)
		if err == nil {
			continue // already applied
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("checking migration %s: %w", version, err)
		}

		content, err := fs.ReadFile(Migrations, filepath.Join(MigrationsDir, name))
		if err != nil {
			return fmt.Errorf("reading migration file %s: %w", name, err)
		}

		slog.Info("applying migration", "version", version)

		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("executing migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`,
			version, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("committing migration %s: %w", version, err)
		}
	}
	return nil
}

func listMigrationFiles() ([]string, error) {
	entries, err := fs.ReadDir(Migrations, MigrationsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("migrations dir %q was not found in embedded filesystem", MigrationsDir)
		}
		return nil, fmt.Errorf("reading migrations dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return nil, fmt.Errorf("no migration files found in %q", MigrationsDir)
	}
	return names, nil
}

// migrationVersion strips the .sql extension to get a sortable version key.
func migrationVersion(filename string) string {
	return strings.TrimSuffix(filename, ".sql")
}

// WithTx runs fn inside a transaction, rolling back on error.
func (d *DB) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
