# ADR-DART-02: App State

## Status

Accepted — <date>

## Context

A Flutter app juggles four genuinely different kinds of state. Conflating them — one store holding
API responses, derived flags, the current route, and form text — forces hand-rolled cache
invalidation for server data and globalizes state that one widget owns.

Flutter is less opinionated here than React: the framework ships `setState`, `ValueNotifier`, and
`InheritedWidget`, and leaves the rest to packages. Its architecture guide calls the choice
"personal preference". This profile makes one, and it is the one decision in this profile a project
may override at the interview: scaffold says *we will use Riverpod for state management unless you
want to override that decision now*, and an override amends this ADR for the project. No
alternative is proposed.

**Riverpod 3** is chosen because it is reactive, framework-agnostic in its core package, testable
through overrides without widget scaffolding, and carries a lint package that checks the rules
below at analysis time.

## Decision

Four categories, one tool each:

| State kind | Tool | Examples |
|---|---|---|
| Server state | **Riverpod async providers** — `FutureProvider`, `AsyncNotifier`, `StreamProvider` | anything owned by an API or a local store |
| Navigation state | **go_router** | current screen, back stack, route parameters, deep links |
| Shared client state | **Riverpod `Notifier` providers** | selection, filters, theme — client-owned, cross-screen |
| Local widget state | **`setState`**, or `flutter_hooks` | form fields, toggles, animation controllers |

### Decision rule

Ask in order: server-owned? → an async provider. Router-owned? → go_router. Shared across screens?
→ a `Notifier`. One widget's own? → the widget.

### Rules

- **Providers call use cases, never gateways.** An async provider watches the bridge provider from
  ADR-DART-01 and calls the use case; it never imports a repository, a client, or anything in
  `infrastructure/`.
- **Do not mirror server state into a `Notifier`.** One async provider owns a piece of server data;
  anything derived watches it. A `Notifier` may hold a key — a selected id — that an async provider
  resolves.
- **Do not mirror navigation state into a provider.** go_router owns the route, the back stack, and
  the parameters. A provider may hold state *keyed by* a route, never the route itself.
- **Providers are for shared business state, not ephemeral state.** A `TextEditingController`, an
  animation, a hover flag, belong to the widget. This is Riverpod's own rule.
- **Real-time transports** — a WebSocket, a server-sent event stream — arrive through a
  `StreamProvider` fed by a use case, or invalidate the async provider that owns the data. Never a
  parallel store.
- **Riverpod 3 forms only.** `Notifier`, `AsyncNotifier`, `StreamNotifier`, and the generated
  `@riverpod` forms. The legacy `StateProvider`, `StateNotifierProvider`, and
  `ChangeNotifierProvider` under `package:riverpod/legacy.dart` are not used.
- **`riverpod_generator` and `riverpod_lint` are on.** The generator removes the choice of provider
  class and supports stateful hot reload; the lint catches a scoped provider with no override and a
  provider that is not statically known.
- **Providers live in `presentation/`**, in the component that owns the screen. Per ADR-DART-01 the
  state tool stays out of the container.

### Lifetimes

- `ProviderScope` wraps the app once, in the composition root. Scoped `ProviderScope`s are for
  per-screen overrides and nothing else.
- Providers auto-dispose by default. Mark one `keepAlive` only when a screen that leaves and returns
  must not refetch, and say why in a comment.

## Consequences

- Loading, error, retry, and cache invalidation for server data are the library's, not bespoke
  code. Riverpod 3 retries a failed provider with exponential backoff out of the box.
- Four tools to learn, but each does one job; the decision rule removes "where does this go?".
- The no-mirroring rules are enforced in review; `riverpod_lint` covers the rest.
- **BLoC and the `provider` package are deliberately not chosen.** BLoC adds ceremony per state
  change and ships no injection of its own; `provider` is Riverpod's predecessor by the same author
  and is bound to the widget tree.
- An override at the interview rewrites this ADR's Context and Decision for the project and changes
  the shape of ADR-DART-01's bridge provider. It changes nothing in the inner rings, which never
  import the state tool.

## Related

- ADR-BASE-01 — Clean Architecture (the inner rings the state tool never touches)
- ADR-DART-01 — Dependency injection (the bridge providers and the locator boundary)
- ADR-DART-03 — Boundary enforcement (what keeps `package:flutter_riverpod` in `presentation/`)
