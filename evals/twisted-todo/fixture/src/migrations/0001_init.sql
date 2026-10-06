-- The first migration: a one-row table that proves the runner works. Feature tables come later,
-- one migration each.
CREATE TABLE IF NOT EXISTS app_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

INSERT OR IGNORE INTO app_meta (key, value) VALUES ('created_at', datetime('now'));
