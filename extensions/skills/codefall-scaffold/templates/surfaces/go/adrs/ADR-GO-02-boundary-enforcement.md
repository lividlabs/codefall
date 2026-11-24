# ADR-GO-02: Boundary Enforcement

## Status

Accepted — <date>

## Context

Package-by-component (ADR-BASE-02) is only real if the boundaries are enforced; unenforced, it degrades
into package-by-feature with everything public. What enforcement costs depends on the language's
visibility model, which is why this ADR is per-profile.

Go is the favorable case. **`internal/` directories are enforced by the compiler**: a package under
`.../x/internal/...` is importable only from the tree rooted at `x`, and nesting works, so
`internal/orders/internal/domain` is reachable from `internal/orders/...` and nowhere else. A
reach-around is a build failure that no comment can suppress.

But the compiler rejects import *cycles*, not *outward* imports. A `domain/` package importing its
sibling `infrastructure/` is acyclic, compiles clean, and passes `go vet`. So the facade half of the
problem is free and the layer half is not.

## Decision

### Component unit

- A component is a **directory under `internal/`**, and its root package **is** the facade. Exported
  identifiers there are the public API; everything else lives in the component's nested `internal/`,
  split into `domain/` `application/` `infrastructure/` `presentation/`.
- Use separate **modules** only for genuinely cross-repo sharing. Within a repo, directories plus
  `internal/` already give real enforcement, and multi-module repos cost bookkeeping for nothing.
- A library module intended for import keeps the per-component nested `internal/` but places the
  component directories at the module root rather than under a root `internal/`.

### Tooling

- **The compiler is primary.** It covers the facade rules with no configuration and no opt-out.
- **`depguard`, via `golangci-lint`**, covers what the compiler will not: import direction inside a
  component, the shared-module rule, and what the inner rings may name. Checked in CI and in the
  editor through the language server.
- **`go-arch-lint`** optional, as a graph-level gate if the `depguard` rules stop expressing the
  architecture cleanly.

**The inner rings take strict allow-lists, not deny-lists.** `depguard`'s `deny.pkg` is prefix
matching with no glob support, so a deny-list has to name every forbidden package path literally and
cannot be written once for all components. `list-mode: strict` inverts it — anything not allowed is
forbidden — which is both shorter and safer, because a layer gains no new permissions when someone
adds a package. A strict list still takes a `deny`, which removes from what `allow` let in: that is
how the two standard-library packages rule 5 names are kept out while `$gostd` is allowed.

**The outer rings and the shared modules take deny-lists**, because what they may import is open and
what they may not is short: one entry per component.

```yaml
version: "2"
linters:
  default: standard
  enable: [depguard]
  settings:
    depguard:
      rules:
        domain-layer:
          list-mode: strict
          files: ["**/internal/domain/**"]
          allow:
            - $gostd
            - github.com/samber/mo
          deny:
            - pkg: net/http
              desc: "domain/ names no transport (ADR-GO-02 rule 5)"
            - pkg: database/sql
              desc: "domain/ names no storage (ADR-GO-02 rule 5)"
        application-layer:
          list-mode: strict
          files: ["**/internal/application/**"]
          allow:
            - $gostd
            - github.com/samber/mo
            - <module>/internal/<component>/internal/domain
          deny:
            - pkg: net/http
              desc: "application/ names no transport (ADR-GO-02 rule 5)"
            - pkg: database/sql
              desc: "application/ names no storage (ADR-GO-02 rule 5)"
        infrastructure-layer:
          files: ["**/internal/infrastructure/**"]
          deny:
            - pkg: <module>/internal/<component>/internal/presentation
              desc: "infrastructure/ does not import presentation/ (ADR-GO-02 rule 2)"
        presentation-layer:
          files: ["**/internal/presentation/**"]
          deny:
            - pkg: <module>/internal/<component>/internal/infrastructure
              desc: "presentation/ does not import infrastructure/ (ADR-GO-02 rule 2)"
        shared-modules:
          files: ["**/internal/shared/**"]
          deny:
            - pkg: <module>/internal/<component>
              desc: "shared modules do not import components (ADR-GO-02 rule 4)"
formatters:
  enable: [gofmt]
```

`domain/` allows the standard library and `samber/mo`, the one module ADR-GO-03 puts there, and
nothing else. `application/` adds each component's own `domain` package, named by its full import
path, and the three deny rules name each component the same way. That is why **this file is
generated at scaffold time rather than copied from a template** — the module path and component
names are not known until then — and why **adding a component is four entries**: one allow entry in
`application-layer`, and one deny entry in each of `infrastructure-layer`, `presentation-layer`, and
`shared-modules`. One `application-layer` rule serves every component, because the compiler already
keeps one component out of another's `domain/`.

`default: standard` runs golangci-lint's standard linters beside `depguard`. With `default: none`,
`golangci-lint run` checks the boundary rules and nothing else.

### When

- Wire it in **at the project scaffold, day one**, same as every profile. The `internal/` layout is
  the harder half to retrofit: moving packages after the fact rewrites every import path in the repo.

### Rules to enforce, and by what

1. **Facade-only imports** — no reaching past a component's root package into its internals.
   *Compiler.*
2. **Clean layer rule** — inward only: `domain` ← `application` ← `infrastructure` / `presentation`;
   no outward imports, and no sideways ones: `infrastructure/` and `presentation/` do not import
   each other. *Linter.*
3. **Cross-component only via public facades.** *Compiler* — the same nested `internal/` that gives
   rule 1.
4. **Shared modules can't import components** — one way. *Linter*; both are ordinary importable
   packages and the compiler has no opinion.
5. **Inner rings can't import delivery or gateway packages** — no `net/http`, `database/sql`, driver
   packages, or CLI framework in `domain/` or `application/`. *Linter*: the strict lists reject every
   third-party package, and their `deny` entries reject the two from the standard library. This is
   the Go reading of the frontend rule: the inner rings name no transport and no storage.

## Consequences

- Rules 1 and 3 are free, absolute, and cannot be disabled — stronger than any lint-based scheme, and
  the reason this ADR is a third the size of the `typescript-react` one.
- The cost is verbosity in paths. `internal/orders/internal/application` reads awkwardly, and this is
  the price of compiler-backed facades. Do not flatten it to make imports prettier.
- The `depguard` block in `.golangci.yml` *is* the encoded layer rule; keep it in review scope. It is
  generated per project, so it is not a file to diff against a template.
- A component whose `application-layer` entry is missing cannot import its own `domain/`, and lint
  says so at once. A component whose deny entries are missing passes lint unguarded, and nothing
  says so.
- A `depguard` rule whose `files` patterns match nothing reports `0 issues` and exits 0 —
  indistinguishable from a rule that works. Prove it by committing a deliberate violation, watching
  it fail, and reverting.
- Verification has two halves here, unlike other profiles: a cross-component reach-around must fail
  `go build`, and an outward layer import must fail `golangci-lint run` **while still compiling**.
  Checking only the first tests the compiler, not your configuration.
- Each lint rule is proven by its own violation: `net/http` in `domain/`, `infrastructure/` in
  `application/`, `presentation/` in `infrastructure/` and the reverse, and a component in a shared
  module. Prove the last from a throwaway package under `internal/shared/`: a shared module that a
  component already imports cannot import it back without an import cycle, which the compiler
  reports before the linter runs.

## Related

- ADR-BASE-02 — Package-by-component (the boundaries being enforced)
- ADR-BASE-01 — Clean Architecture (the layer rule at #2)
- ADR-GO-01 — Dependency injection (why `application/` names no implementations)
- ADR-GO-03 — Optional values (why `samber/mo` is allowed in the inner rings)
