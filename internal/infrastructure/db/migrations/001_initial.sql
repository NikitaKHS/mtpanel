-- Migration 001: initial schema
-- Direction: UP

CREATE TABLE IF NOT EXISTS nodes (
    id          TEXT    PRIMARY KEY,
    name        TEXT    NOT NULL,
    host        TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'unknown',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS proxy_configs (
    id          TEXT    PRIMARY KEY,
    port        INTEGER NOT NULL DEFAULT 443,
    secret      TEXT    NOT NULL,
    tag         TEXT    NOT NULL DEFAULT '',
    workers     INTEGER NOT NULL DEFAULT 4,
    max_conn    INTEGER NOT NULL DEFAULT 60000,
    nat_prefix  TEXT    NOT NULL DEFAULT '',
    extra_args  TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS proxy_links (
    id          TEXT    PRIMARY KEY,
    label       TEXT    NOT NULL,
    secret      TEXT    NOT NULL,
    host        TEXT    NOT NULL,
    port        INTEGER NOT NULL,
    link        TEXT    NOT NULL,
    active      INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL,
    revoked_at  TEXT
);

CREATE INDEX IF NOT EXISTS idx_proxy_links_active ON proxy_links (active);

CREATE TABLE IF NOT EXISTS audit_events (
    id          TEXT    PRIMARY KEY,
    event_type  TEXT    NOT NULL,
    actor_id    TEXT    NOT NULL DEFAULT '',
    actor_ip    TEXT    NOT NULL DEFAULT '',
    resource_id TEXT    NOT NULL DEFAULT '',
    detail      TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_created ON audit_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_type    ON audit_events (event_type);

CREATE TABLE IF NOT EXISTS app_settings (
    key         TEXT    PRIMARY KEY,
    value       TEXT    NOT NULL,
    updated_at  TEXT    NOT NULL
);

-- Default settings
INSERT OR IGNORE INTO app_settings (key, value, updated_at)
VALUES
    ('is_first_run',      'true',  datetime('now')),
    ('install_state',     'none',  datetime('now')),
    ('mtproxy_version',   '',      datetime('now')),
    ('panel_version',     '1.0.0', datetime('now'));

CREATE TABLE IF NOT EXISTS schema_migrations (
    version     TEXT    PRIMARY KEY,
    applied_at  TEXT    NOT NULL
);
