# Forfeit

A small web app for one person on one laptop. The rules for working in it are here; `codefall init`
adds its own sections below.

## Stack

- Node 24 or newer. The server is `node:http`; the database is the standard library's `node:sqlite`
  (`DatabaseSync`). There are no runtime dependencies, and adding one is a decision to say out loud.
- Pages are rendered on the server as HTML strings through `layout()` in `src/page.js`, which also
  holds the whole visual language: the CSS variables, the type scale, and the `card`, `button`, and
  `empty` classes. A new screen uses those and adds nothing global.
- Forms post as `application/x-www-form-urlencoded`; `readForm()` in `src/server.js` parses them,
  and a successful post redirects with 303.

## Layout

- `src/main.js` starts the server; `src/server.js` holds `createApp()` and the routes; `src/db.js`
  opens the database and runs migrations; `src/page.js` renders the shell.
- `src/migrations/NNNN_what.sql` — one SQL file per schema change, applied in name order at startup
  and by `scripts/local.sh update`. Never edit a migration that has been committed; add the next one.
- `test/*.test.js` — unit tests on `node:test`, against an in-memory database.
- `testing/test-cases/<area>/<slug>.md` with a `<slug>.e2e.ts` beside it — end-to-end cases and
  their Playwright specs, run against the server `scripts/local.sh start` brought up.

## Verification

Run both before opening a pull request:

- `npm test` — the unit tests.
- `npm run e2e` — the Playwright specs under `testing/test-cases/`. Needs the server up:
  `scripts/local.sh start`.

## Local environment

- `scripts/local.sh start` brings the server up on port 3000 (`PORT` overrides) and waits until
  `/health` answers. `scripts/local.sh update` installs dependencies, installs the Playwright
  browser, and applies pending migrations. `scripts/local.sh stop` stops the server.
- The database file is `data/app.db` (`DB_PATH` overrides). It is git-ignored.
