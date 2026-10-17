# ADR-DART-03: Boundary Enforcement

## Status

Accepted — <date>

## Context

Package-by-component (ADR-BASE-02) is only real if the boundaries are enforced; unenforced, it
degrades into package-by-feature with everything public. What enforcement costs depends on the
language's visibility model, which is why this ADR is per-profile.

Dart has **library-private names and nothing else**: an identifier that starts with `_` is private
to its library, and every other declaration is public to the whole package. There is no
package-private level. The `implementation_imports` lint stops one *package* from importing another
package's `src/`, and says nothing about files inside the same package. So a component's facade file
is a convention the compiler will not enforce, exactly as in TypeScript, and the facade rule and
the layer rule both need a linter.

Two tools can do that. **Pub workspaces** (Dart 3.6 and later) make each component its own package,
so that a cross-component import needs a declared dependency and `implementation_imports` guards the
`src/` tree; that is compiler-grade enforcement at the cost of one `pubspec.yaml` per component and a
workspace root. **`import_lint`** is an analyzer plugin configured in `analysis_options.yaml`, with
rules of the form *files matching `target` may not import paths matching `from`, except `except`*,
modelled on ESLint's `no-restricted-paths`. It covers both rules in one file, inside one package.

This profile chooses `import_lint`, and names workspaces as the upgrade for a repository that holds
several apps or packages.

## Decision

### Component unit

- Components are **folders under `lib/components/`**, each fronted by a facade file
  `lib/components/<name>/<name>.dart` that `export`s the public API with `show`. Everything else
  lives under `lib/components/<name>/src/`, split into `domain/` `application/` `infrastructure/`
  `presentation/`.
- Shared technical modules live under `lib/shared/<module>/` with the same facade-and-`src/`
  shape.
- App-level wiring — the composition root, `lib/app/injection.dart`, the go_router configuration —
  lives under `lib/app/` and `lib/main*.dart`, and imports component facades only.
- Use **separate packages** only for genuinely cross-app code, or when the repository grows to
  several apps; then move to a pub workspace with one package per component, keep this file's layer
  rules inside each package, and let the dependency graph carry the facade rule.

### Tooling

- **`import_lint`** as primary, enabled under `plugins:` in `analysis_options.yaml` with
  `severity: error`. It reports in the editor through the analysis server and in CI through
  `dart run import_lint`, which exits non-zero on a violation. Dart 3.10 or later.
- **`flutter_lints`** and **`riverpod_lint`** beside it. They are not boundary tools, but
  `riverpod_lint` is what checks ADR-DART-02's provider rules.

### Rules to enforce

1. **Facade-only imports** — no reaching past a component's facade into its `src/`.
2. **Clean layer rule** — inward only: `domain` ← `application` ← `infrastructure` /
   `presentation`; no outward or sideways imports.
3. **Cross-component only via public facades.**
4. **Shared modules can't import components** — one way.
5. **Inner rings can't import the UI framework, the state tool, the locator, or a gateway
   library** — no `package:flutter`, `flutter_riverpod`, `go_router`, `get_it`, `dio`, `retrofit`,
   or a storage driver in `domain/` or `application/`. `package:injectable`'s annotations are the
   one allowed exception, per ADR-DART-01.

### The configuration

The layer rules (2, 4, 5) are written once with wildcards. The facade rules (1, 3) are **one rule
per component**, because the globs have no notion of "the same component": the rule for `orders`
forbids importing any component's `src/` except its own. This is why the file is **generated at
scaffold time** from the component names rather than copied from a template, and why adding a
component means adding its rule.

```yaml
plugins:
  import_lint: ^2.0.0

import_lint:
  severity: error
  rules:
    domain_imports_nothing_outward:
      target: "package:<app>/components/*/src/domain/**"
      from:
        - "package:<app>/components/*/src/application/**"
        - "package:<app>/components/*/src/infrastructure/**"
        - "package:<app>/components/*/src/presentation/**"
        - "package:flutter/**"
        - "package:flutter_riverpod/**"
        - "package:riverpod_annotation/**"
        - "package:go_router/**"
        - "package:get_it/**"
        - "package:dio/**"
        - "package:retrofit/**"
    application_imports_only_domain:
      target: "package:<app>/components/*/src/application/**"
      from:
        - "package:<app>/components/*/src/infrastructure/**"
        - "package:<app>/components/*/src/presentation/**"
        - "package:flutter/**"
        - "package:flutter_riverpod/**"
        - "package:riverpod_annotation/**"
        - "package:go_router/**"
        - "package:get_it/**"
        - "package:dio/**"
        - "package:retrofit/**"
    infrastructure_not_presentation:
      target: "package:<app>/components/*/src/infrastructure/**"
      from: "package:<app>/components/*/src/presentation/**"
    presentation_not_infrastructure:
      target: "package:<app>/components/*/src/presentation/**"
      from: "package:<app>/components/*/src/infrastructure/**"
    shared_not_components:
      target: "package:<app>/shared/**"
      from: "package:<app>/components/**"
    app_uses_facades:
      target: "package:<app>/app/**"
      from: "package:<app>/components/*/src/**"
    # one per component, generated:
    orders_uses_facades:
      target: "package:<app>/components/orders/**"
      from: "package:<app>/components/*/src/**"
      except: "package:<app>/components/orders/src/**"
```

`<app>` is the package name from `pubspec.yaml`. Relative imports resolve to the same `package:`
URIs, so the rules hold whichever form a file uses.

### When

- Wire it in **at the project scaffold, day one.** Deferring defeats the purpose — boundaries only
  hold if enforced from the start, and this matters more in an AI-assisted workflow, where fast
  generation drifts without a hard guardrail.

## Consequences

- Setup cost and editor squiggles as you work — which is the point.
- Without this, the entire package-by-component choice is cosmetic.
- The `import_lint` block *is* the encoded architecture; keep it in review scope. It is generated
  per project, so it is not a file to diff against a template.
- A rule whose `target` matches no file reports nothing and exits `0` — indistinguishable from a
  rule that works. Prove it by committing a deliberate violation, watching `dart run import_lint`
  fail, and reverting.
- The analysis server loads a plugin at start, so after `plugins:` changes the IDE needs a restart
  before the squiggles appear; the CLI command does not.
- Moving to a pub workspace later keeps every layer rule and replaces the per-component facade
  rules with `pubspec.yaml` dependencies, which is a mechanical change.

## Related

- ADR-BASE-02 — Package-by-component (the boundaries being enforced)
- ADR-BASE-01 — Clean Architecture (the layer rule at #2)
- ADR-DART-01 — Dependency injection (why `package:get_it` is allowed in two files and no others)
- ADR-DART-02 — App state (why `package:flutter_riverpod` stays in `presentation/`)
