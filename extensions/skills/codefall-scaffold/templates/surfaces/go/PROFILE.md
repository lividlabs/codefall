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
| Facade enforcement | `internal/` packages — the compiler | ADR-GO-02 |
| Layer-direction enforcement | `depguard` via `golangci-lint`, strict allow-lists | ADR-GO-02 |
| Architecture gate (optional) | `go-arch-lint` | ADR-GO-02 |
| Formatting | `gofmt`, `gofumpt` optional | — |
| Tests | stdlib `testing`, table-driven, hand-written fakes | ADR-GO-01 |

## ADRs this profile supplies

- `adrs/ADR-GO-01-dependency-injection.md`
- `adrs/ADR-GO-02-boundary-enforcement.md`
- `adrs/ADR-GO-03-optional-values.md`

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

**`.golangci.yml` is generated, not copied.** The `application/` allow-lists must name each
component's own `domain` package by its full import path, so the file cannot exist until the module
path and component names are known. Emit one `domain-layer` rule (which needs only `$gostd`) plus one
`application-layer` rule per component. ADR-GO-02 has the shape.

Go needs no formatter config, no test-runner dependency, and no build tool — `gofmt` and `testing`
ship with the toolchain. Do not add `testify`, a mock generator, or a task runner reflexively; add
them when a project asks.

The local scripts at this tier: `update` runs `go mod download`, and `mise install` first when the
project pins a toolchain; `start` exits `0` with nothing to bring up. Where the tier added a
`Makefile`, they are two targets there rather than a script.

The **runnable skeleton** tier adds `cmd/<app>/main.go` wiring the composition root end to end with
no features in it, plus each component directory with its facade file and the four nested layer
packages.

Verify the boundary rules the way step 6 requires, and note that Go gives you a sharper test than
most profiles: a deliberate cross-component reach-around should **fail `go build`**, and a deliberate
outward layer import should fail `golangci-lint run` while still compiling. Check both. A `depguard`
config that matches no files passes silently and looks identical to one that works.
