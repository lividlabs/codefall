# ADR-DART-01: Dependency Injection

## Status

Accepted — <date>

## Context

Clean Architecture (ADR-BASE-01) has inner layers depend on interfaces that outer layers implement;
something must bind implementations to those interfaces at a composition root.

Dart gives the inner rings everything they need without a framework: an `abstract interface class`
declared in `application/`, implemented in `infrastructure/`, and received through a constructor.
What remains is assembling the object graph once, per app, and swapping implementations per
environment and in tests.

Three options were weighed. **Riverpod providers as the container** — a `Provider<OrderRepository>`
per binding, overridden in `ProviderScope` — is a common community pattern, and it is not what
Riverpod is for. Its maintainers describe it as "a reactive caching and data-binding framework";
its Do/Don't page reserves providers for shared business state; and the idiom its API reference
suggests for an object with many dependencies, passing `ref` into the object, puts a Riverpod import
inside the application layer, which ADR-BASE-01 forbids. **A hand-written composition root** with
no library is what Flutter's own architecture sample does, and it grows one line per binding with
nothing checking that the graph is complete. **`get_it` with `injectable`** is a service locator
plus a generator: bindings are declared by annotation where the implementation lives, the generator
writes the registration in dependency order and reports a dependency it cannot find, and
environments switch implementations without touching the root.

This profile chooses `get_it` with `injectable`, and confines the locator so that it never becomes
a global reached from inside the code it wires.

## Decision

### Container

- **`get_it`, registered by `injectable`**, one container per app, built in the composition root.
  The root is `lib/main.dart` (and one `lib/main_<env>.dart` per extra environment), which calls
  the generated `configureDependencies(environment:)` from `lib/app/injection.dart` before
  `runApp`.
- **Bindings are declared where the implementation lives.** An implementation in a component's
  `infrastructure/` carries `@LazySingleton(as: OrderRepository)`; a use case in `application/`
  carries `@injectable`. The generator scans the package, so a component needs no registration
  function and the root names no concrete class.
- **Third-party objects** — a `Dio` instance, `SharedPreferences`, a database — are registered in a
  `@module` class in the shared module that owns them, never inline in the root.
- **Environments.** `@Environment('dev')` and `@prod` on an implementation select it; a fake
  repository for local development is a `dev` binding beside the `prod` one. The entry file picks
  the environment, which is the whole difference between `main.dart` and `main_development.dart`.

### Injection discipline

- **Constructor injection, interfaces only.** Every dependency a use case or gateway implementation
  receives is typed as an `abstract interface class` declared in `application/`. Value and
  configuration objects are the practical exception.
- **Interfaces are declared by the consumer**, in `application/`, alongside the use cases that need
  them. Never in `domain/`, never in the file that implements them.
- **One interface per gateway role per component**, not per use case. Go narrower only when a
  different component needs a subset.

### Naming

- **Interface** = the bare noun — `OrderRepository`, not `IOrderRepository`, not
  `OrderRepositoryInterface`.
- **Implementation** = `<Qualifier>Noun` — `RemoteOrderRepository`, `InMemoryOrderRepository`.
- **`Default`** is the qualifier for the single canonical implementation (`DefaultClock`).

### The locator boundary

`GetIt.I` is called in **two places and no others**:

1. The composition root, to build the graph and resolve what `runApp` needs.
2. One **bridge provider per use case** in the component's `presentation/` ring, which is how the
   state tool named in ADR-DART-02 reaches the graph:

   ```dart
   @riverpod
   CreateOrder createOrder(Ref ref) => GetIt.I<CreateOrder>();
   ```

   A notifier or widget watches the bridge; it never calls the locator and never constructs a use
   case. Tests override the bridge with a fake, which is the whole test seam for the presentation
   ring.

The state tool itself — Riverpod's `ProviderScope`, its providers and notifiers — is **not
registered in the container**. It stays idiomatic in the presentation ring; the container holds
framework-agnostic logic only.

### Non-Dart layers

Platform code in Kotlin or Swift that only wraps an OS API is a gateway implementation behind a
platform channel and needs no container on its side. It becomes a surface of its own, with its own
dependency-injection ADR, only when it holds domain logic.

### Test seam

- **Use cases and gateways** are tested by calling the same constructors with **hand-written
  fakes** declared in the component's own test files. No container is built.
- **Notifiers and widgets** are tested through `ProviderContainer.test()` or a `ProviderScope`
  with the bridge providers overridden. No container is built.
- **The container is tested once**, by a smoke test that calls `configureDependencies` for each
  environment and resolves every use case. This is the test that catches a binding left out of the
  graph, which otherwise fails at first resolution on a device.

## Consequences

- `injectable` runs under `build_runner` and writes `lib/app/injection.config.dart`. The generated
  file is committed, and CI regenerates it and fails on a diff, so the registration in the
  repository is always the one the annotations describe.
- The generator reports a dependency it cannot find at build time. What it cannot catch is a class
  nobody annotated: that resolves at runtime, with `Object/factory with type X is not registered`,
  which is what the container smoke test exists for.
- Annotations put `package:injectable` into `application/`. That is the same trade `ADR-TS-01`
  makes with Inversify's decorators: metadata only, no behaviour, and the inner rings still name no
  transport, storage, or widget.
- A service locator is only as safe as the places allowed to call it. Two call sites, both in the
  outer ring, is the rule that keeps this one from becoming a global; ADR-DART-03's lint keeps
  `package:get_it` out of every other file.
- A different state tool (ADR-DART-02 amended) changes the shape of the bridge and nothing above
  this section.

## Related

- ADR-BASE-01 — Clean Architecture (the interfaces being bound, and where they live)
- ADR-BASE-02 — Package-by-component (the components whose implementations are annotated)
- ADR-DART-02 — App state (why the state tool stays outside the container)
- ADR-DART-03 — Boundary enforcement (what keeps the locator out of the inner rings)
