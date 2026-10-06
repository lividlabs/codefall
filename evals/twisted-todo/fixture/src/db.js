// The database: one SQLite file, opened with Node's own `node:sqlite`, and a small migration runner.
//
// Migrations are plain SQL files under `src/migrations/`, named `NNNN_what.sql`, applied in name
// order. Each is recorded in `schema_migrations` once it has run, so applying is safe to repeat.
// `openDatabase()` applies pending migrations; `scripts/local.sh update` applies them ahead of a
// start through `src/migrate.js`.

import { DatabaseSync } from "node:sqlite";
import { mkdirSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
export const MIGRATIONS_DIR = join(here, "migrations");
export const DEFAULT_DB_PATH = process.env.DB_PATH ?? "data/app.db";

export function openDatabase(path = DEFAULT_DB_PATH) {
  if (path !== ":memory:") mkdirSync(dirname(path), { recursive: true });
  const db = new DatabaseSync(path);
  db.exec("PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;");
  migrate(db);
  return db;
}

// Applies every migration not yet recorded. Returns the names it applied, in order.
export function migrate(db) {
  db.exec(
    `CREATE TABLE IF NOT EXISTS schema_migrations (
       name TEXT PRIMARY KEY,
       applied_at TEXT NOT NULL DEFAULT (datetime('now'))
     )`,
  );
  const applied = new Set(db.prepare("SELECT name FROM schema_migrations").all().map((r) => r.name));
  const files = readdirSync(MIGRATIONS_DIR).filter((f) => f.endsWith(".sql")).sort();
  const record = db.prepare("INSERT INTO schema_migrations (name) VALUES (?)");
  const fresh = [];
  for (const file of files) {
    if (applied.has(file)) continue;
    db.exec("BEGIN");
    try {
      db.exec(readFileSync(join(MIGRATIONS_DIR, file), "utf8"));
      record.run(file);
      db.exec("COMMIT");
    } catch (error) {
      db.exec("ROLLBACK");
      throw new Error(`migration ${file} failed: ${error.message}`);
    }
    fresh.push(file);
  }
  return fresh;
}
