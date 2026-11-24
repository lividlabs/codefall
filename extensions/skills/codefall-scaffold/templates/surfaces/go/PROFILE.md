# Surface profile: go

**Status:** supported.
**ADR prefix:** `GO` — this profile's ADRs are `ADR-GO-01` upward.

Go services, APIs, workers, daemons, and CLIs. Backend and tooling surfaces — anything whose delivery
mechanism is a socket, a queue, a schedule, or a terminal.

## Fits when

- An HTTP or gRPC service, **or**
- A queue consumer, worker, or long-running daemon, **or**
- A CLI or developer tool that holds real domain logic of its own.

Go is a single-surface language here: there is no renderer split, no second toolchain, and no
equivalent of the React DOM / React Native divergence. What varies between a service and a CLI is the
`presentation/` ring — handlers versus commands — and nothing inward of it.

## Does not fit

Anything with a client-side UI. Go has no analog to `typescript-react`'s frontend-state ADR, and a
Go backend paired with a browser
or mobile client is **two surfaces**, not one: the client needs its own profile and the project needs
a seam ADR naming which side owns the domain.

A Go binary that only implements gateways for another surface is **not a surface**. A sidecar that
proxies requests, a small exporter, a shim wrapping an OS API — those hold no entities and no use
cases, and per ADR-BASE-01 they are the `infrastructure/` ring of whatever surface they serve. Go being
present no more makes it a Go surface than a Postgres driver makes SQL one.

The test is where domain logic lives, and it is a question for the user. Do not infer it from a
`go.mod`.

## Visibility model

This is where Go diverges hardest from `typescript-react`, and the divergence runs in Go's favor.

**`internal/` directories are enforced by the compiler.** A package under `.../x/internal/...` is
importable only from the tree rooted at `x`. Nesting works: `internal/orders/internal/domain` is
reachable from `internal/orders/...` and from nowhere else. A violation is a build failure, not a
lint warning:

```
use of internal package example.com/app/internal/orders/internal/domain not allowed
```

So ADR-BASE-02's facade is not a convention in Go. The component's root package *is* the facade, its
internals live in a nested `internal/`, and the toolchain refuses to compile a reach-around. There is
no `eslint-plugin-boundaries` equivalent to install for this, and no way to disable it in a comment.

**What the compiler does not do is the layer rule.** Go rejects import *cycles*, not *outward*
imports. A `domain/` package importing its sibling `infrastructure/` is acyclic and compiles clean,
with `go vet` silent. That direction has to be enforced by a linter, which is why ADR-GO-02 is lighter
here but not absent.

**Interfaces are declared by the consumer**, implicitly satisfied. ADR-BASE-01's rule — that the
interfaces a use case needs live *with the use case* in `application/`, never in `domain/` — is a
discipline you enforce in TypeScript and simply how Go is written. "Accept interfaces, return
structs" is the same instruction from the other direction.

## Layout

```
cmd/<app>/main.go              composition root — the only place naming implementations
internal/<component>/          the facade: exported API only
  <component>.go
  internal/
    domain/                    entities, value objects, errors
    application/               use cases + the interfaces they need
    infrastructure/            gateway implementations
    presentation/              handlers, commands
internal/shared/<module>/      focused shared modules, each its own facade
```

A **library** module intended for import cannot put its public components under a root `internal/`.
Put those at the module root and keep the nested per-component `internal/` — the facade guarantee
still holds; only the outer wrapper is gone.

## Toolchain

| Concern | Choice | ADR |
| --- | --- | --- |
| Dependency injection | `samber/do` v2, one injector per app | ADR-GO-01 |
| Interface granularity | One per gateway role per **component**, not per use case | ADR-GO-01 |
| Optional values | `samber/mo` — `Option` only, not `Result` | ADR-GO-03 |
| Error handling | stdlib `error`, `%w` wrapping, `errors.Is` / `errors.As` | ADR-GO-03 |
| Web framework | `chi` v5, with an error-returning handler adapter the project owns | ADR-GO-04 |
| HTTP error responses | RFC 9457 problem details, written by one function; `log/slog` for what is not sent | ADR-GO-04 |
| Database | PostgreSQL unless the project chooses another | ADR-GO-05 |
| Data access | `sqlc` generating for `pgx` v5, `pgxpool` for connections; no ORM | ADR-GO-05 |
| Migrations | goose, compiled into the app, one set per component | ADR-GO-05 |
| Facade enforcement | `internal/` packages — the compiler | ADR-GO-02 |
| Layer-direction enforcement | `depguard` via `golangci-lint`: strict allow-lists for the inner rings, deny-lists for the outer rings and shared modules | ADR-GO-02 |
| Architecture gate (optional) | `go-arch-lint` | ADR-GO-02 |
| Formatting | `gofmt`, `gofumpt` optional | — |
| Tests | stdlib `testing`, table-driven, hand-written fakes | ADR-GO-01 |

**Every library in this table is a default.** Scaffold raises one of them itself, the database, as
the override below describes. A user who names a different library for any other gets it: say once
what the default is and why, then amend that ADR for the project, with its decisions kept and its
code rewritten for the library chosen, and report the amendment. A project that wants Echo in place
of chi keeps what ADR-GO-04 decides — a handler returns its error, one function writes an error
response, the handler gives a domain error its status, each component registers its own routes —
and writes them with Echo's types. The code in each ADR was run with the default alone, so run the
amended code before the ADR records it. What a user cannot replace is the stance: the layers,
package-by-component, and boundary enforcement.

## ADRs this profile supplies

- `adrs/ADR-GO-01-dependency-injection.md`
- `adrs/ADR-GO-02-boundary-enforcement.md`
- `adrs/ADR-GO-03-optional-values.md`
- `adrs/ADR-GO-04-http-delivery.md` — **surfaces that serve no HTTP skip this one.**
- `adrs/ADR-GO-05-data-access.md` — **surfaces with no database skip this one.**

**Each of the two follows what the surface does, not what kind of program it is.** Take ADR-GO-04
when the surface answers HTTP requests, and ADR-GO-05 when it reads or writes a relational database.
A surface takes both, either, or neither: an API with no database, a CLI with one, a worker that
only consumes a queue. Read each answer from step 1's description, and ask once when the description
leaves one open. For an ADR the surface does not take, drop it, and delete its section and its link
from the `AGENTS.md` written from the skeleton.

**Whatever else a surface talks to has no ADR in this profile**: a message broker, a queue, a cache,
another service's API. Each is a gateway like any other (ADR-BASE-01), with its interface in
`application/` and its implementation in `infrastructure/`. Codefall has no opinion on the client
library, and the project records its choice in an ADR of its own.

**The same holds for any other way a surface is reached**: gRPC, a queue it consumes, a CLI's
commands. Each is delivery in `presentation/` (ADR-BASE-01), with no ADR in this profile, and the
project records its choice of library in an ADR of its own.

**The database is the one override this profile offers.** The interview says *this will use
PostgreSQL unless you want a different database now*, and proposes nothing else. On MySQL or SQLite,
amend ADR-GO-05 for the project: `sqlc` and goose stay, `sqlc` generates for `database/sql`, and the
driver is that database's own. On SQLite more than the driver changes: there is no `compose.yaml`,
a transaction is a `*sql.Tx`, and a component's tables take a prefix where PostgreSQL gave them a
schema. The ADR's code was run on PostgreSQL only, so on another database run each piece of it
before the amended ADR records it. On a database `sqlc` does not read, drop ADR-GO-05; the project
records the decision in an ADR of its own.

Numbering is per profile and starts at 01, so this profile has no gaps and no relationship to any
other profile's numbers. `ADR-GO-01` and `ADR-TS-01` are different decisions that happen to sit first
in their own sets.

Shared ADR-BASE-01 through ADR-BASE-03 come from `templates/adrs/` and apply to every profile.
ADR-BASE-03 is dropped only for a surface that could never be split into services at all — a CLI,
a desktop or mobile app, a library — not for one that merely has no current plan to split.

## Depth notes

The **project files** tier owes: `go.mod`, `.golangci.yml` carrying the `depguard` layer rules
(ADR-GO-02), a `Makefile` or `Taskfile` fronting the standard targets, and CI running `go build ./...`,
`go vet ./...`, `golangci-lint run`, and `go test ./...`.

**`.golangci.yml` is generated, not copied.** Its rules name each component by its full import
path, so the file cannot exist until the module path and component names are known. Emit the five
rules ADR-GO-02 shows, with its `default: standard` and its `gofmt` formatter:

- `domain-layer` — strict; allows `$gostd` and `github.com/samber/mo`; denies `net/http` and
  `database/sql`.
- `application-layer` — the same, plus one allow entry per component for its `domain` package. One
  rule serves every component.
- `infrastructure-layer` and `presentation-layer` — one deny entry per component, each for that
  component's other outer ring.
- `shared-modules` — one deny entry per component.

A component added later needs its four entries: one allow and three deny.

Go needs no formatter config, no test-runner dependency, and no build tool — `gofmt` and `testing`
ship with the toolchain. Do not add `testify`, a mock generator, or a task runner reflexively; add
them when a project asks.

**Add every dependency with `go get`**, never by writing a version into `go.mod` by hand: a version
written from memory is whatever was current when the model was trained. Run `go mod tidy` once the
code that imports a module exists; until then `go.mod` marks it `// indirect`. A dependency arrives
with the code that uses it: `samber/do` with the composition root, by
`go get github.com/samber/do/v2`, and `samber/mo` with the first optional value.

The local scripts at this tier: `update` runs `go mod download`, and `mise install` first when the
project pins a toolchain; `start` exits `0` with nothing to bring up. Where the tier added a
`Makefile`, they are two targets there rather than a script.

The **runnable skeleton** tier adds `cmd/<app>/main.go` wiring the composition root end to end with
no features in it, plus each component directory with its facade file and the four nested layer
packages.

**On a surface that serves HTTP**, the skeleton also adds the shared module
`internal/shared/httpx/`, written from ADR-GO-04:

- `handler.go` — the `Handler` type and its `ServeHTTP`.
- `error.go` — `Error`, `NewError`, the `problem` body, and `writeError`.
- `json.go` — `Decode` and `Encode`.
- `router.go` — `NewRouter`, the recover middleware, the not-found handler, and `GET /healthz`.

`cmd/<app>/main.go` sets the default `slog` logger, provides the `*http.Server` through the
injector, listens on the port in `PORT` or on 8080, serves until `SIGINT` or `SIGTERM`, and shuts
the injector down with a deadline. `.golangci.yml` gains the `http-router` rule. Add chi with
`go get github.com/go-chi/chi/v5`.

A component gets no routes in the skeleton. Its `presentation/routes.go`, its facade's `Routes`,
and its mount in `main.go` arrive with the component's first handler.

The module's tests use `testing` and `net/http/httptest`, and cover each of these:

- A returned `*httpx.Error` gives its status and detail.
- The same error wrapped with `%w` gives the same response.
- Any other error is a 500 whose text is logged and is absent from the body.
- A `nil` return leaves the handler's own response untouched.
- `GET /healthz` answers through `NewRouter()`.
- An unknown path and a panic each get the error body, and the panic's stack is in the log.
- A wrong method keeps chi's 405 and its `Allow` header.
- `Decode` gives a 400 for a body that does not parse.
- `Encode` sends nothing when the value cannot be marshalled.

**On a surface with a database**, the project files tier also owes, written from ADR-GO-05:

- `compose.yaml` with one PostgreSQL service: its user, password, and database named for the app, a
  named volume, and its port bound to `127.0.0.1`. The healthcheck runs
  `pg_isready -h 127.0.0.1 -U <user> -d <database>`, over TCP, because the image answers on its
  socket while it is still setting itself up. Pin the image to the current major version, read from
  `docker run --rm postgres postgres --version` and not written from memory. The local script
  `start` runs `docker compose up -d --wait`.
- A default for `DATABASE_URL` in the `Makefile`, naming the compose database and exported, so
  that `migrate` and a local run need nothing set. A value already in the environment wins.
- The `database-driver` rule in `.golangci.yml`.
- A `migrate` target in the `Makefile` that runs `go run ./cmd/<app> migrate`. The local script
  `update` runs it after `go mod download`.
- A PostgreSQL service in CI, with `TEST_DATABASE_URL` set to it, so the repository tests run there.
  Locally the variable may name the compose server: each test makes and drops a database of its own.

The skeleton then adds, with no table in it:

- In `cmd/<app>/main.go`: the pool opened from `DATABASE_URL` and provided to the injector, with
  an unset variable stopping the app, and the `migrate` command over every component. A service
  checks for the argument before it serves; a CLI adds `migrate` beside its other commands.
- In each component: `infrastructure/migrations/00001_create_schema.sql`, which creates the
  component's PostgreSQL schema and nothing else, `CREATE SCHEMA <component>;` up and
  `DROP SCHEMA <component>;` down; `infrastructure/migrate.go`; and on the facade a `Migrate`
  function beside `Register`, which is empty until the component has something to register.

Add `pgx` and goose with `go get github.com/jackc/pgx/v5 github.com/pressly/goose/v3`.

**`sqlc` arrives with a component's first query**, not with the skeleton: `sqlc generate` fails on a
component that has no query, and on a `sqlc.yaml` that has no entry. The first query adds the
component's entry to `sqlc.yaml`, its `queries/` and `db/` directories, a `generate` target, and
`sqlc diff` in CI. Pin the tool the first time with `mise use --pin sqlc@latest`, which writes the
current version into `mise.toml`; without `--pin` it writes `latest`. The `generate` target runs
`mise exec -- sqlc generate`, which works in a shell where mise is not activated, and CI installs
the pinned tool through mise before it runs `sqlc diff`. The local script `update` then runs
`mise install` first. It does not generate: the generated code is committed. Add
`github.com/google/uuid` with `go get` when the first `uuid` column is generated.

Verify the boundary rules the way step 6 requires, and note that Go gives you a sharper test than
most profiles: a deliberate cross-component reach-around should **fail `go build`**, and a deliberate
outward layer import should fail `golangci-lint run` while still compiling. Check both. A `depguard`
config that matches no files passes silently and looks identical to one that works.

Prove each lint rule with its own violation, the five ADR-GO-02 lists under Consequences. A rule
that holds one entry per component is proven once per component, because a missing entry passes
unnoticed. Once the project imports `samber/mo`, confirm that it passes in a `domain/` package.

On a surface that serves HTTP, prove two more things. Run the binary and request `GET /healthz`.
Then import chi from an `infrastructure/` package: `golangci-lint run` should fail on the
`http-router` rule while `go build` still passes.

On a surface with a database, prove four more. Bring the database up with the local `start`
script. Run `<app> migrate` twice: the second run logs `applied=0` for every component and exits
`0`. Run the app with `DATABASE_URL` unset: it stops with a message naming the variable. Then
import `pgx` from a `presentation/` package: `golangci-lint run` should fail on the
`database-driver` rule while `go build` still passes.
