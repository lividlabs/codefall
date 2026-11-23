# ADR-GO-01: Dependency Injection

## Status

Accepted — <date>

## Context

Clean Architecture (ADR-BASE-01) has inner layers depend on interfaces that outer layers implement;
something must bind implementations to those interfaces at a composition root.

Go changes what that costs. Interfaces are **implicitly satisfied and declared by the consumer**, so
the interface a use case depends on is defined next to the use case and no implementation ever names
it. The decoupling a container provides in TypeScript — separating a constructor from the concrete
type it receives — the language already provides. What remains is assembling the object graph and
managing the lifetimes of the things in it.

The options are manual constructor wiring, compile-time generation (`google/wire`), and runtime
containers (`uber-go/fx`, `samber/do`). This shop chooses **`samber/do`**: generics-based
registration without reflection-heavy tagging, lazy instantiation, scopes, and ordered shutdown,
at the cost of resolution happening at runtime rather than build time. That cost is real and the
Consequences section names how it is contained.

## Decision

### Container

- **`samber/do` v2**, one **injector per app**, built in the composition root at `cmd/<app>/main.go`.
  Concrete implementations are named only in the owning component's registration function — the
  root cannot name them, because `internal/<component>/internal/*` is unimportable from `cmd/`.
  The root assembles components by facade.
- Components export a registration function that the root calls. A component registers its own
  services; it never reaches into another component's registrations.

### Registration — the provider returns the interface

The service is keyed by the provider's **static return type**. Declare the interface there, never the
concrete type:

```go
do.Provide(injector, func(i do.Injector) (application.OrderRepository, error) {
    return infrastructure.NewPostgresOrderRepository(do.MustInvoke[*sql.DB](i)), nil
})
```

This provider lives in the component's registration function — the only place that can import
both `application/` and `infrastructure/`. The compiler checks satisfaction at that `return`. This
is the single most important rule in this ADR, and the failure mode when it is broken is described
in Consequences.

### Resolution

- **`do.Invoke[T]` / `do.MustInvoke[T]`** — exact-type lookup. The default, because its behaviour is
  predictable.
- **`do.As[Initial, Alias]`** — register an explicit alias when a concrete type must also be
  reachable through an interface it satisfies.
- **`do.InvokeAs[T]`** — resolves by scanning registered services for one assignable to `T`. Use only
  where the exact registered type genuinely is not known; it is a runtime scan and becomes ambiguous
  once two services satisfy the same interface.
- **Three places resolve from the injector, and no others**: the composition root, the providers
  in a component's registration function, and the functions a component's facade exports for the
  root to call, such as one that returns the component's routes or its commands. `MustInvoke` is
  correct in all three, where a failure should stop the process at startup.
- **Everything else takes what it needs through its constructor** and never holds a `do.Injector`:
  a handler, a command, a use case, a repository. A type that resolves its own dependencies hides
  them, because nothing in its constructor says what it uses, and a test cannot hand it a fake.

### Lifecycle

- Services needing ordered teardown implement `do.Shutdownable`; the root calls the injector's
  shutdown on exit. Do not hand-roll a teardown ordering the container already knows.
- `do.HealthCheck[T]` is available for services that can report readiness. Wire it where an operator
  would actually look at it, not reflexively.
- **Every service the injector holds lives as long as the process**, and every request running at
  the same moment shares it. A service holds nothing that belongs to one request.
- **What belongs to one request travels in `context.Context`**: its deadline, its cancellation, and
  whatever one layer attaches for the layers under it. Every use case and every gateway method
  takes a `context.Context` first and passes it on. Do not open a `samber/do` scope per request.

### Injection discipline

- **Inject only interfaces**, never concrete types. Value and config structs passed directly are the
  practical exception.
- **Interfaces are declared by the consumer**, in `application/`, alongside the use cases that need
  them. Never in `domain/`, and never in the package that implements them.

### Interface granularity — per component, not per use case

Consumer-declared does **not** mean one interface per use case. The interface lives in the
component's `application/` package and carries the methods that component's use cases collectively
need.

The component is already the unit of cohesion (ADR-BASE-02), and its `application/` layer is one package.
Splitting per use case draws boundaries inside a boundary that has already been drawn, and it buys
near-duplicate interfaces, duplicated fakes, and a reader who has to work out which of five similar
interfaces is in play at a given call site.

Go narrower only when a **different component** needs a subset — a real second consumer with a
genuinely different contract, which is how `billing/application.OrderLookup` comes to exist alongside
`orders/application.OrderRepository`. Never split for a hypothetical one.

Rob Pike's "the bigger the interface, the weaker the abstraction" is about interfaces published at
package API boundaries, like `io.Reader`. It is not an instruction to shard every dependency.

**The test:** if two interfaces in the same `application/` package differ by fewer than two methods,
they are one interface.

### Naming

- **Interface** = the bare noun — `OrderRepository`, not `IOrderRepository` and not
  `OrderRepositorer`.
- **Implementation** = `<Qualifier>Noun` — `PostgresOrderRepository`, `InMemoryOrderRepository`.
- Use **`Default`** for the single canonical implementation (`DefaultClock`); adding an edge-case
  sibling later does not force a rename.
- This diverges from the `-er` convention, which suits small single-method interfaces (`Reader`,
  `Stringer`) and reads badly on domain roles.

**The pairing is not recorded anywhere, and that is the difference from TypeScript.** Both languages
are structurally typed and in both a single implementation may satisfy several interfaces. But
TypeScript writes the pairing down twice — in `class X implements Y` and again in the container
binding — so one interface reads as *the* interface for an implementation. Go has nowhere to write
it: there is no `implements` clause, and `do.Provide` derives its key from a return type rather than
a declared relationship. So do not force a canonical interface per implementation, and do not widen a
consumer's interface to make it "match" the thing behind it.

### Test seam

- There is no rebind. Tests call the same constructors with a **hand-written fake** defined in the
  component's own test files. Component-scoped interfaces stay small enough that this beats a mock
  generator; add `mockery` only when a specific interface makes that false.
- **Compile-time interface assertions live in `_test.go`**, in the pointer form:

  ```go
  var _ application.OrderRepository = (*PostgresOrderRepository)(nil)
  ```

  The value form `T{}` fails when any method has a pointer receiver, so the pointer form is the one to
  write. Keeping it in a test file preserves the check under `go vet` and `go test` while leaving the
  production import graph unchanged.
- Assertions are **optional and additive**. Satisfaction is already checked at the provider's return
  type; the assertion only moves the error to the implementation's own package.

## Consequences

- **A provider that returns the concrete type compiles and fails at runtime.** This is the sharp edge
  of the choice:

  ```
  DI: could not find service `app.OrderRepository`,
      available services: `*app.PostgresOrderRepository`
  ```

  It is not a library defect. Implicit satisfaction means there is no enumerable set of interfaces a
  type implements, so no container can auto-register a concrete type under them. Nor can `do.As` be
  made compile-time safe — `func As[Alias any, Initial Alias]()` is rejected with `cannot use a type
  parameter as constraint`, so the check must be a runtime error. Discipline plus `do.As` is the
  whole mitigation.
- Missing registrations surface at startup rather than at build. Resolve every component facade in
  the composition root so that failure happens on the first line of `main`, not on the first
  request. Resolution is lazy, so constructing a facade invokes its providers and builds the
  component's whole graph.
- Lazy instantiation, scopes, and ordered shutdown come free, which is the trade being made for the
  point above.
- Component-scoped interfaces keep the number of interfaces bounded by components times gateway
  roles, and keep hand-written fakes small enough to stay hand-written.

## Related

- ADR-BASE-01 — Clean Architecture (the interfaces being bound, and where they live)
- ADR-BASE-02 — Package-by-component (the components the root assembles)
- ADR-GO-02 — Boundary enforcement (what keeps `application/` from importing its implementations)
- ADR-GO-03 — Optional values (the other place this profile takes a stance on modelling)
