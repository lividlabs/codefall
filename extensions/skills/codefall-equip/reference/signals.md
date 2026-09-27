# Signals, entry points, and what they map to

What a repository says about the tools its local environment needs, where a project may already
keep the commands, and what each signal becomes in `start` and `update`. Read at step 2, and again
at step 3 while drafting.

## Contents

- [Entry points a project may already have](#entry-points-a-project-may-already-have)
- [Signals and what they map to](#signals-and-what-they-map-to)
- [What fails the contract](#what-fails-the-contract)
- [Consulting before the one question](#consulting-before-the-one-question)

## Entry points a project may already have

Search these first. A hit is a candidate; read its body before offering it.

| Where | What to look for |
| --- | --- |
| `Makefile`, `GNUmakefile` | targets named `up`, `dev`, `dev-up`, `start`, `setup`, `bootstrap`, `sync`, `migrate`, `install` |
| `justfile`, `Justfile` | recipes with the same names |
| `package.json` `scripts` | `dev:up`, `db:up`, `setup`, `bootstrap`, `db:migrate`, `migrate`, `postinstall` that runs codegen |
| `Taskfile.yml`, `.mise.toml` `[tasks]` | tasks with the same names |
| `scripts/`, `bin/`, `script/` | `setup`, `bootstrap`, `dev`, `up`, `start`, `update`, `sync`, `migrate`, with or without `.sh` |
| `README.md`, `CONTRIBUTING.md`, `docs/` | a "getting started" or "local development" section that lists commands nobody scripted |
| `.codefall/settings.json` | a `local` block already declared — the revision case |

A README section that lists commands is not an entry point; it is the evidence for a draft.

## Signals and what they map to

One row per signal. A draft's `update` runs the row's update step for every signal present, in the
order the rows appear: runtimes first, then the env file, then dependencies, then generated code,
then migrations. The env file comes before anything that reads it — a migration tool takes its
database URL from there, so on a fresh clone the copy has to happen before `migrate deploy` runs.
`start` runs every row's start step.

| Signal | Tool | `start` | `update` |
| --- | --- | --- | --- |
| `mise.toml`, `.tool-versions` | mise / asdf | — | `mise install` — first, before anything that needs a runtime |
| `.env.example`, `.env.sample`, or any `.env*` sample the README names | dotenv | — | copy the project's sample file to the file its tooling loads, only when that file is missing — `.env` for most dotenv loaders, `.env.local` for Next.js; the README or the loader's config says which. When two tools read different files — Next.js loads `.env.local`, the Prisma CLI loads only `.env` — the draft copies to each file a step reads, or runs the step through the loader that reads the app's file |
| `compose.yaml`, `docker-compose.yml` | Docker Compose | `docker compose up -d --wait` (`up -d` when the file has no healthchecks) | — |
| `package-lock.json` | npm | — | `npm ci`, guarded by a lockfile stamp (below) |
| `pnpm-lock.yaml` | pnpm | — | `pnpm install --frozen-lockfile` |
| `yarn.lock` | Yarn | — | `yarn install --immutable` (Berry) or `yarn install --frozen-lockfile` (1.x) |
| `bun.lock`, `bun.lockb` | Bun | — | `bun install --frozen-lockfile` |
| `go.mod` | Go | — | `go mod download` |
| `uv.lock` | uv | — | `uv sync` |
| `poetry.lock` | Poetry | — | `poetry install --sync` |
| `requirements.txt` with a venv | pip | — | `pip install -r requirements.txt` inside the venv |
| `Gemfile.lock` | Bundler | — | `bundle install` |
| `Cargo.lock` | Cargo | — | `cargo fetch` |
| `mix.lock` | Mix | — | `mix deps.get` |
| `sqlc.yaml` | sqlc | — | `sqlc generate` |
| `codegen.ts`, `codegen.yml` | GraphQL Codegen | — | `graphql-codegen` |
| `openapi*.yaml` with a generator config | OpenAPI generator | — | the project's generate command |
| `buf.yaml`, `*.proto` with a build rule | protobuf | — | `buf generate` / the project's `protoc` rule |
| `prisma/schema.prisma` | Prisma | — | `prisma generate`, then `prisma migrate deploy` |
| `drizzle.config.*` | Drizzle | — | `drizzle-kit migrate` |
| `knexfile.*` | Knex | — | `knex migrate:latest` |
| `alembic.ini` | Alembic | — | `alembic upgrade head` |
| `db/migrate/` with a `Rakefile` | Rails | — | `bin/rails db:migrate` |
| `manage.py` | Django | — | `python manage.py migrate` |
| `migrations/` with `goose` or `migrate` in `go.mod` or a Makefile | goose / golang-migrate | — | `goose up` / `migrate up` |

A signal with no row is still a signal: read the tool's documentation for its idempotent
install-or-sync form and its migrate-forward form, and use those.

**`npm ci` is the one install step that is not cheap on a no-op.** It removes `node_modules` and
reinstalls from the lockfile every run, so a draft guards it: hash `package-lock.json`, keep the
hash in a stamp inside `node_modules/` — already git-ignored, and gone whenever the directory
is — and run `npm ci` only when the hash differs. `templates/local.sh` shows the guard. The other
install rows compare the lockfile against what is installed and answer "nothing to do" on their
own: `pnpm install --frozen-lockfile`, `yarn install --immutable`, `bun install --frozen-lockfile`,
`go mod download`, `uv sync`, `poetry install --sync`, `bundle install`, `cargo fetch`, and
`mix deps.get` all do. A candidate that runs `npm ci` bare keeps the contract's safety and not its
cost; offer the guard as a revision rather than reporting the candidate.

## What fails the contract

A candidate or a draft with any of these is reported with the line and never declared:

- `db:reset`, `migrate reset`, `migrate fresh`, `drop`, `DROP DATABASE`, `flush`
- `rm -rf` on a data directory, a volume, or `node_modules` as a matter of course
- `docker compose down -v`, `--force-recreate`, `--renew-anon-volumes`
- `git clean`, `git reset --hard`, `git checkout .`, or any pull: the checkout is
  `codefall-refresh`'s, and the tree is the user's
- `npm install` where a lockfile exists: it may rewrite the lockfile, and `npm ci` is the form that
  installs exactly what is committed
- an interactive prompt: `refresh` runs these with nobody at the keyboard

A candidate that does one of these on a flag — `make reset` beside `make up` — is fine; only the
command being declared is judged.

The list holds two kinds of candidate, and the report says which. The first three entries, and
`npm install` beside a lockfile, are wrong for any caller: a script that resets, drops, or deletes
costs someone data whoever runs it. A pull or a prompt is usually a script written for a different
caller — a first-clone setup that asks for secrets once, a sync that someone runs by hand after
fetching — and it is correct for that caller. SKILL.md step 2 says how to name each kind, and what
the draft takes from the second.

## Consulting before the one question

The search usually settles which candidates keep the contract. When it does not — two entry points
that both start the services and nothing says which the team runs, a migration step whose tool the
signals do not name, a script whose effect cannot be read from its body — consult once before the
question, per *Consulting* in `../../../../.codefall/shared/running-agents.md`. Render `../../../../.codefall/shared/consult-prompt.md`:
`QUESTION` is what stayed ambiguous; `FILES` are the candidates and the signals they act on;
`CONTEXT` is [the contract](../SKILL.md#the-contract) in a sentence and what was found; `OPTIONS`
are the candidates, and "draft new"; `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`. Run the `consult`
order with `../../../../.codefall/shared/run-agent.sh`.

The answer is folded into the one question as the proposed option — "I found `make dev-up` and
`scripts/up.sh`; both keep the contract, and consulting `architect` reads `up.sh` as the one CI
runs, so I'd declare that. Declare it, or draft new?" — and the user still chooses. `low`,
`cannotSettle`, or no answer asks the question as it would have been asked. A consult never
declares a script and never judges the contract: a candidate that drops, resets, or deletes is
wrong whatever an agent says about it.

