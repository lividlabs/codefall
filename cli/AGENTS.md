# codefall

The CLI is a component of the repo in `cli/`; `extensions/` holds the skills, hooks, and shared
scripts that `codefall init` installs (the old "plugin" is renamed extension). Every path below is
relative to `cli/` unless it says otherwise.

A Go command-line tool. One surface, one app, one module.

**State: four components and nine shared modules.** `internal/doctor/` (`codefall doctor`) is the
first component and the reference for the rules below; `internal/initcmd/` (`codefall init` and
`codefall upgrade`, one use case behind two commands, exported through `Commands`) is the second, and
follows it; `internal/create/` (`codefall create`) is the third, and runs init's command in the
directory it makes; `internal/config/` (`codefall config`, an interactive editor in a terminal and
subcommands for scripts, over one use case) is the fourth, and reads and writes the agents list, one
entry per active agent with its review and consult lists, how each harness is called, whether review
posts, and the persona. `internal/shared/ui/` holds the palette,
the marks, the colour-profile writer, the spinner runner, and the reusable Bubble Tea models that are
generic over their data (`OrderList`); `internal/shared/process/` holds the command runner and the file
system; `internal/shared/buildinfo/` holds the version the binary reports, for `--version` and for
the manifest init writes. `internal/shared/harness/` holds the harnesses codefall can set up and where each one reads
skills, `internal/shared/manifest/` the `.codefall/manifest.json` format,
`internal/shared/settings/` the `.codefall/settings.json` format, `internal/shared/userfile/` the
per-user `.codefall/user.json` format, `internal/shared/version/` the release comparison, and
`internal/shared/text/` the string helpers both components need — all six **pure** (ADR-003), so the
inner layers may import them. `cmd/codefall/main.go` is the composition root and builds the
injector.

## Applicable ADRs

The why lives in the ADRs. This file is the operative rules only — never restate the reasoning here.

- [ADR-BASE-01 Clean Architecture](../docs/adrs/ADR-BASE-01-clean-architecture.md) ·
  [ADR-BASE-02 Package-by-component](../docs/adrs/ADR-BASE-02-package-by-component.md) ·
  [ADR-GO-01 Dependency Injection](../docs/adrs/ADR-GO-01-dependency-injection.md) ·
  [ADR-GO-02 Boundary enforcement](../docs/adrs/ADR-GO-02-boundary-enforcement.md) ·
  [ADR-GO-03 Optional values](../docs/adrs/ADR-GO-03-optional-values.md)
- This project's own decisions: [ADR-001 Facade contracts](../docs/adrs/ADR-001-facade-contracts.md) ·
  [ADR-002 CLI libraries](../docs/adrs/ADR-002-cli-libraries.md) ·
  [ADR-003 Pure shared modules](../docs/adrs/ADR-003-pure-shared-modules.md). New ones use
  [`docs/adrs/_TEMPLATE.md`](../docs/adrs/_TEMPLATE.md); decisions still moving live in
  [`docs/decision-log.md`](../docs/decision-log.md).

## Structure

- Package-by-component: capabilities as directories under `internal/<component>/`. The component's
  root package **is** the facade; exported identifiers there are its whole public API.
- What crosses a facade is a **contract** — a struct of primitives, ids, `mo.Option[T]`, and
  collections of those, declared in the component's root package. A facade never accepts or returns a
  `domain` type; reference another component's entity by id (ADR-001).
- Clean layers nest inside the component's own `internal/`: `domain/` `application/`
  `infrastructure/` `presentation/`.
- Shared technical modules under `internal/shared/<module>/`, each its own facade. A shared module
  never imports a component. A module is imported from a component's `presentation/` and
  `infrastructure/`, from its facade file, and from `main`; `domain/` and `application/` may import
  it only if it is a **pure shared module** — one whose own imports are the standard library and
  `samber/mo` and nothing else, named in `.golangci.yml` (ADR-003). `ui` and `process` are not pure.
- One composition root per app at `cmd/<app>/main.go`.

## Layer rules

- Dependencies point inward only. Interfaces live in `application/` with the use cases that need
  them; `domain/` = entities + value objects + errors.
- `domain/` imports the standard library, `samber/mo` (ADR-GO-03), and pure shared modules (ADR-003)
  and nothing else. `application/` adds only its own component's `domain/` and pure shared modules.
  No `net/http`, `database/sql`, drivers, or CLI framework in either.
- Handlers and commands are thin; use cases never see delivery types.

## DI

- `samber/do`, one injector per app, built in the composition root. Each component exports a
  registration function from its facade package; the root calls it. A component's concrete types are
  named only in that function — `main` cannot import `internal/<component>/internal/*`, so the
  compiler enforces this.
- **The provider returns the interface, never the concrete type** (this code lives in the
  component's registration function, not in `main`):

  ```go
  do.Provide(injector, func(i do.Injector) (application.OrderRepository, error) {
      return infrastructure.NewPostgresOrderRepository(do.MustInvoke[*sql.DB](i)), nil
  })
  ```

- Resolve with `do.Invoke` / `do.MustInvoke`. Use `do.As[Concrete, Interface]` to register an alias
  when a concrete type must also be reachable by interface. Avoid `do.InvokeAs` — it is a runtime
  scan and goes ambiguous once two services satisfy the same interface.
- Inject interfaces only, declared by the consumer in `application/`.
- **One interface per gateway role per component**, not per use case. Split narrower only when a
  different component needs a subset. If two interfaces in the same `application/` package differ by
  fewer than two methods, they are one interface.
- `Thing` / `<Qualifier>Thing` naming — `OrderRepository`, `PostgresOrderRepository`, `DefaultClock`.
  No `I` prefix, no `-er` suffix on domain roles. One implementation may satisfy several interfaces;
  do not force a canonical pairing.
- Tests pass hand-written fakes to the same constructors. Compile-time assertions, if written, go in
  `_test.go` in the pointer form: `var _ application.OrderRepository = (*PostgresOrderRepository)(nil)`.

## Optional values

- Absence is `mo.Option[T]`, not `*T` and not a sentinel zero value. `mo.Some("")` is present with an
  empty value and is not the same as `mo.None[string]()`.
- Errors stay `(T, error)` with `%w` wrapping and `errors.Is` / `errors.As`. Do not use `mo.Result`.
- If a caller needs to know *why* something is missing, it is an error, not a `None`.

## CLI

- Cobra (`github.com/spf13/cobra`) defines the command tree; `fang.Execute` runs it from
  `cmd/codefall/main.go`, the only importer of `github.com/charmbracelet/fang`. Fang owns help,
  usage, error, `--version`, `completion`, and `man` output — change it with `fang.With*` options,
  never with Cobra templates. The version is `buildinfo.Display()`, passed with `fang.WithVersion`.
- Commands are built in a component's `presentation/`, exported through its facade, and mounted on
  the root in the composition root. Flags are `pflag` — never the standard library `flag`.
- A component's facade exports `Register(do.Injector)` and `Command(do.Injector) *cobra.Command`;
  `main` calls the first and mounts the second. A component with several top-level commands adds
  `Commands(do.Injector) []*cobra.Command` when that happens. A command that runs another
  component's command takes it as a parameter — `create.Command(injector, initcmd.Command(injector))`
  — built by `main` for it alone, never the instance mounted on the root.
- Styling is Lip Gloss v2 (`charm.land/lipgloss/v2`), used where it helps. Plain text is the
  default; parsed output (anything piped or `--json`-style) is never styled; human output goes
  through the colour-profile writer `internal/shared/ui` builds (`ui.NewWriter`), never a rendered
  style written straight to `os.Stdout`. Colours are `ui.Tone` values, not hex literals in a
  component: a component maps its own statuses onto a tone and takes the style from `ui.Style`.
- Prompts are Huh v2 (`charm.land/huh/v2`), in `presentation/` only, collecting values into a
  contract for a use case. A command that prompts by default also runs without prompting: every
  prompted value is also a flag, and if `stdin` is not a terminal and a value is missing, fail
  naming the flag — never block on input. Run with `RunWithContext`; `WithAccessible` follows the
  `ACCESSIBLE` environment variable.
- Charm's v2 generation only. Nothing that imports the v1 paths
  (`github.com/charmbracelet/{lipgloss,bubbletea,bubbles,huh}`) is added. A screen is a Bubble Tea
  program written in the component's own `presentation/`, with Huh v2 fields embedded for entry and
  Bubbles for lists and keys (ADR-012). A model goes into `internal/shared/ui/` only when it knows
  nothing about what it shows; `presentation/` binds the data. Bubble Tea appears in `presentation/`
  and `internal/shared/ui/`, never inward. A slow job runs under `ui.RunWithSpinner`, which handles
  the no-terminal case too, rather than under a model of its own. A command that opens a screen
  prints its plain output instead when stdout is not a terminal, and never blocks on one.
- A facade exports no view or model type, and no alias to one (ADR-012). A component's screen is
  opened by its own command.

## Enforcement

- `internal/` facades are enforced by the compiler — a reach-around fails `go build`.
- Layer direction is enforced by `depguard` strict allow-lists in `.golangci.yml`, which is generated
  for this project rather than copied from a template.
- `gofmt` is enforced by golangci-lint's formatter stage: `golangci-lint run` fails on a file gofmt
  would rewrite.
- Run `go build ./...`, `go vet ./...`, `golangci-lint run`, `go test ./...` — or `make check`.
- **Verifying the rules takes two checks, because there are two mechanisms.** A cross-component
  reach-around must fail `go build`. An outward layer import must fail `golangci-lint run` *while
  still compiling*. Checking only the first tests the compiler, not the configuration.

## Workflow

- Conventional commits, with a body that says *why*.
- Work happens on a branch or worktree, never on `main`. Stacked branches are fine for landing
  large work as smaller reviewable pieces.
- Deliberately minimal: no branching model beyond this is decided here.

## Gotchas

- A `depguard` rule whose `files` pattern matches nothing reports `0 issues` and exits 0, looking
  exactly like a rule that works. Only a deliberate violation proves it.
- A `do.Provide` whose provider returns the concrete type compiles and then fails at runtime with
  `could not find service`. There is no compile-time guard for this — Go rejects
  `func As[Alias any, Initial Alias]()` with `cannot use a type parameter as constraint` — so the
  provider's return type is the discipline.
- **`.golangci.yml` needs a new `application-layer` allow-list entry for every component you add** —
  `depguard`'s `allow` is literal prefix matching with no globs, so a new component's `application/`
  package silently loses access to its own `domain/` until its import path is named there. The
  `shared-modules` rule needs a deny entry per component for the same reason.
- **A pure shared module needs four `.golangci.yml` entries** — a `domain-layer` allow entry, an
  `application-layer` allow entry, a `pure-shared-modules` `files` entry, and a `pure-shared-modules`
  `allow` entry naming the module itself (ADR-003). The first two let the inner layers import it; the
  third holds it to the standard library and `samber/mo`, and is the only reason the first two are
  safe; the fourth is what lets one pure module import another while the rule still denies everything
  impure. The purity rule matches nothing until the module exists, so it reports `0 issues` until
  then — prove it against a deliberate violation when the module lands.
- **Proving the `shared-modules` rule needs a throwaway package.** A real shared module can import a
  component only in a build that already has an import cycle, and a cycle fails `go build` — which
  is the compiler, not the configuration. Add `internal/shared/proof/` importing a component's
  facade, confirm `go build ./...` passes and `golangci-lint run` fails on the rule, and delete it.
  Proven that way on 2026-08-27, once per deny entry. The `domain-layer` and `application-layer`
  rules were re-proven the same day against a Cobra import from
  `internal/doctor/internal/application/` and against an `internal/shared/ui` import from
  `internal/initcmd/internal/application/` and `internal/doctor/internal/domain/`. The
  `pure-shared-modules` rule was proven the same way on 2026-08-27, against a
  `charm.land/lipgloss/v2` import in `internal/shared/settings/`. It was proven again for
  `internal/shared/harness/` on 2026-09-16, the same way, and that run also proved the module's
  `domain-layer` allow entry: a deliberate `harness` import from `internal/initcmd/internal/domain/`
  compiled and was not reported. Both proofs were made again for `internal/shared/manifest/` on
  2026-09-16; nothing in a `domain/` package imports that module, so its allow entry has no other
  proof. Both were made again for `internal/shared/userfile/` on 2026-09-27, against a
  `charm.land/lipgloss/v2` import in the module and a `userfile` import from
  `internal/doctor/internal/domain/`. The `config` component's `application-layer` and
  `shared-modules` entries were proven on 2026-09-27, against a Cobra import from its
  `application/` and a throwaway `internal/shared/proof/` importing its facade.
- **`depguard` matches `_test.go` too.** Inner-layer tests are internal test packages (`package
  domain`, `package application`), so a `domain` test cannot import `os`. The schema test that holds
  `schemas/settings.schema.json` equal to the settings constants lives in `internal/shared/settings`,
  beside the format it pins — a pure shared module is not an inner layer, so its tests may read the
  file.
- **`go.mod` requires `samber/do`, `samber/mo`, Cobra, Fang, Lip Gloss v2, Bubble Tea v2, Bubbles v2,
  `colorprofile`, `x/term`, and Huh v2** — `x/term` for the terminal detection Fang's own theme uses,
  Bubble Tea and Bubbles for the spinners, Huh for init's survey. Add each library with the code that
  needs it, not up front, and run `go mod tidy` after: a library nothing imports any more belongs in
  the indirect block.
- **A facade that returns a `domain` type compiles for its callers even though they cannot import
  that package.** Go's `internal/` rule restricts naming a package, not holding a value: `o :=
  orders.Find(id)` infers the type and `o.Total()` works, while `var o *domain.Order` does not
  compile. Neither the compiler nor `depguard` catches this — ADR-001 is a review rule.
- **The linker ignores a `-X` flag that names no variable, without a warning.** GoReleaser's
  default flags stamp `main.version`, which does not exist, and releases through 0.14.0 reported
  "unknown (built from source)" because of it. `.goreleaser.yml` names
  `internal/shared/buildinfo`'s variables instead, and `buildinfo_test.go` fails when the two
  disagree — moving the package or renaming a variable means changing both.
- **A v1-generation Charm import compiles and quietly doubles the dependency graph.** On any change
  to `go.mod`, check `go mod graph` for a `github.com/charmbracelet/lipgloss` or `bubbletea` line
  without `/v2`. Nothing else catches it (ADR-002).
