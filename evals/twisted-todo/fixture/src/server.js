// The HTTP server, built on `node:http` and nothing else. `createApp` returns a server that is not
// yet listening, so tests can bind it to a free port; `src/main.js` starts it for real.
//
// Routes are matched in `route()`. A new screen adds a case there and renders through
// `layout()` from `page.js`. Form posts arrive as `application/x-www-form-urlencoded`; `readForm`
// parses them.

import { createServer } from "node:http";
import { openDatabase } from "./db.js";
import { layout, escapeHtml } from "./page.js";

export function createApp({ db = openDatabase() } = {}) {
  const server = createServer(async (req, res) => {
    try {
      await route(req, res, db);
    } catch (error) {
      console.error(error);
      send(res, 500, "text/plain; charset=utf-8", "Something went wrong on the server.");
    }
  });
  server.on("close", () => db.close());
  return server;
}

async function route(req, res, db) {
  const url = new URL(req.url, "http://localhost");

  if (req.method === "GET" && url.pathname === "/health") {
    const row = db.prepare("SELECT value FROM app_meta WHERE key = 'created_at'").get();
    return sendJson(res, 200, { ok: true, databaseCreatedAt: row?.value ?? null });
  }

  if (req.method === "GET" && url.pathname === "/") {
    const body = `
      <h2>Nothing here yet</h2>
      <p class="empty">This is the empty skeleton. The first feature replaces this page.</p>`;
    return sendHtml(res, 200, layout({ title: "Forfeit", body, current: "/" }));
  }

  const body = `<h2>Not found</h2><p class="empty">There is no page at <code>${escapeHtml(url.pathname)}</code>.</p>`;
  return sendHtml(res, 404, layout({ title: "Not found — Forfeit", body, current: "" }));
}

export function readForm(req) {
  return new Promise((resolve, reject) => {
    let raw = "";
    req.setEncoding("utf8");
    req.on("data", (chunk) => (raw += chunk));
    req.on("end", () => resolve(Object.fromEntries(new URLSearchParams(raw))));
    req.on("error", reject);
  });
}

export function redirect(res, location) {
  res.writeHead(303, { location });
  res.end();
}

function send(res, status, type, body) {
  res.writeHead(status, { "content-type": type, "content-length": Buffer.byteLength(body) });
  res.end(body);
}
export const sendHtml = (res, status, html) => send(res, status, "text/html; charset=utf-8", html);
export const sendJson = (res, status, value) =>
  send(res, status, "application/json; charset=utf-8", JSON.stringify(value));
