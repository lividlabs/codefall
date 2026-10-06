import { test } from "node:test";
import assert from "node:assert/strict";
import { openDatabase, migrate } from "../src/db.js";

test("migrations apply once and record themselves", () => {
  const db = openDatabase(":memory:");
  const names = db.prepare("SELECT name FROM schema_migrations ORDER BY name").all().map((r) => r.name);
  assert.ok(names.includes("0001_init.sql"));
  assert.deepEqual(migrate(db), [], "a second run applies nothing");
  const meta = db.prepare("SELECT value FROM app_meta WHERE key = 'created_at'").get();
  assert.ok(meta?.value, "the first migration seeded app_meta");
  db.close();
});
