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
| `scripts/local.sh` | `start`, `update`, and `stop`; what the chain's `equip local` session finds and declares |
| `playwright.config.ts` | the runner, collecting `*.e2e.ts` from `testing/test-cases/`; what the chain's `equip test` session finds and declares |
| `AGENTS.md` | the project's rules; `codefall init` adds its sections below them |
| `setup.sh`, `teardown.sh` | create and delete the throwaway repository |

The document-landing Action is not in the fixture. `setup.sh` copies it from the template codefall
ships, `extensions/skills/codefall-equip/templates/codefall-land-documents.yml`, so the throwaway
runs the same file a project gets from `/codefall-equip landing`, and a change to the template
reaches the eval without a second copy to keep in step. `setup.sh` also creates the `auto-merge`
label the Action listens for, which the landing track would create.

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

It declares nothing under `local` or `test.runners` and writes no Runners line: the chain's first
two sessions are `/codefall-equip local` and `/codefall-equip test`, which find the scripts and the
Playwright configuration and declare them in their own pull requests, so `equip` is under test too.
The one thing it installs that `equip` would otherwise install is the document landing: the Action,
copied from codefall's template, and the `auto-merge` label, so that a document pull request merges
once the driver, playing the person, adds the label.

## teardown.sh

Deletes the repository on GitHub and stops the server. `--purge` also removes the project checkout.
Run transcripts under `evals/twisted-todo/runs/` are never touched.

```bash
evals/twisted-todo/fixture/teardown.sh --purge
```
