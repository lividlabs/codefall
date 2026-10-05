import { test, after } from "node:test";
import assert from "node:assert/strict";
import { createApp } from "../src/server.js";
import { openDatabase } from "../src/db.js";

const server = createApp({ db: openDatabase(":memory:") });
await new Promise((resolve) => server.listen(0, resolve));
const base = `http://localhost:${server.address().port}`;
after(() => server.close());

test("GET /health answers ok", async () => {
  const res = await fetch(`${base}/health`);
  assert.equal(res.status, 200);
  assert.equal((await res.json()).ok, true);
});

test("GET / renders the skeleton page", async () => {
  const res = await fetch(`${base}/`);
  assert.equal(res.status, 200);
  const html = await res.text();
  assert.match(html, /<h1>Forfeit<\/h1>/);
  assert.match(html, /Nothing here yet/);
});

test("an unknown path is a 404 page", async () => {
  const res = await fetch(`${base}/nowhere`);
  assert.equal(res.status, 404);
});
