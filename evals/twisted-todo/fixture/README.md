# Fixture: the Forfeit skeleton

The project the eval runs codefall's verbs against. It is a web app skeleton with a SQLite
database and nothing else: a page that says nothing is here, a health endpoint, a migration runner
with one migration, two unit tests, the local scripts, and a Playwright configuration with no specs
yet.

## The stack, and why

Node 24 with the standard library's `node:sqlite`, `node:http`, and server-rendered HTML; Playwright
as the end-to-end runner. Node 24 ships SQLite in the standard library, so the app has no runtime
dependencies and no native build step, and Playwright is the runner codefall-equip already declares
for a browser front end, so the fixture needs no tooling the skills do not know.

`node:sqlite` prints an `ExperimentalWarning` on Node 24; it is stable enough for this and the
warning is harmless.

## What is here

| Path | What it is |
| --- | --- |
| `src/main.js`, `src/server.js`, `src/page.js`, `src/db.js` | the server, the routes, the page shell and visual language, the database and migration runner |
| `src/migrations/0001_init.sql` | the one migration, so the runner is proven and later features add the next file |
| `test/*.test.js` | unit tests on `node:test`, run by `npm test` |
| `scripts/local.sh` | `start`, `update`, and `stop`; what `equip` would declare as the local scripts |
| `playwright.config.ts` | the runner, collecting `*.e2e.ts` from `testing/test-cases/` as `equip` configures it |
| `AGENTS.md` | the project's rules; `codefall init` adds its sections below them |
| `.github/workflows/land-document-stack.yml` | the recipe's Action, adapted for a throwaway |
| `setup.sh`, `teardown.sh` | create and delete the throwaway repository |

`codefall init` creates `testing/` with its `AGENTS.md`, `README.md`, and `CLAUDE.md`; the fixture
does not carry them.

## setup.sh

Creates everything a run needs and does not run the chain. Read the header of the script for the
exact steps. It needs `gh` logged in with the `repo`, `workflow`, `project`, and `delete_repo`
scopes, the `gh-stack` extension (it installs it when missing), `bd`, `jq`, Node 24, and Go.

```bash
evals/twisted-todo/fixture/setup.sh                 # a name is picked from the date
evals/twisted-todo/fixture/setup.sh my-run-name     # or your own
```

It writes `evals/twisted-todo/.throwaway.env` with the project directory and the repository name;
the driver and `teardown.sh` read it. That file is for one run and should not be committed.

Two things it does that `equip` would otherwise do in its own pull requests: it declares the
`local` scripts and the `playwright` runner in `.codefall/settings.json`, and it fills the Runners
line in `testing/AGENTS.md`. The chain under test does not include `equip`, and without those
declarations `implement` refuses to start a bead that names a test case. `equip` still works
against the fixture if someone wants to run it by hand: the scripts and the Playwright config are
where its search looks.

## teardown.sh

Deletes the repository on GitHub and stops the server. `--purge` also removes the project checkout.
Run transcripts under `evals/twisted-todo/runs/` are never touched.

```bash
evals/twisted-todo/fixture/teardown.sh --purge
```
