# The architecture scaffold sets up

`/codefall-scaffold` starts a new project on one architecture and writes down why. This page
describes that architecture, the decision records that come with it, and how scaffold decides which
languages and frameworks a project can use.

## What scaffold does

Scaffold starts from a vision, the short document `/codefall-envision` writes about what you are
building and why. With a vision in hand, scaffold reads the parts of the product from it and asks you
to confirm them. Without one, it offers to write one first, and if you decline, it asks you to
describe the project in a sentence or two.

From that description, scaffold writes the project's architecture decisions into `docs/adrs/`, an
`AGENTS.md` for each part of the project, and, if you ask for them, the project files and the lint
rules that enforce the architecture. Each decision is an *ADR* (architecture decision record): a
short document that records one hard-to-reverse choice and the reasons for it.

## The architecture

Scaffold uses Clean Architecture, organized by component, with the boundaries checked by tools and
components that stay cheap to split out later.

### Clean Architecture

*Clean Architecture* keeps the business rules independent of frameworks, databases, and user
interfaces. Code sits in four rings, and dependencies point inward only:

| Ring | Holds |
| --- | --- |
| `domain/` | entities, value objects, and domain errors, with no framework and no I/O |
| `application/` | use cases, and the interfaces those use cases need |
| `infrastructure/` | implementations of those interfaces: databases, external services, and other *gateways* |
| `presentation/` | handlers, commands, and the UI |

A *gateway* is an interface to any external system. A database repository, an HTTP client, and a
desktop shell's native command are all gateways.

Scaffold uses the strict form of Clean Architecture: the interfaces a use case needs live with the
use case in `application/`, never in `domain/`. The domain holds only business objects.

### Package-by-component

The top level of the code is organized by *component*: one capability of the application, such as
orders or billing, with its own code and its own data. Each component holds its own Clean rings and
exposes one public API, its *facade*. Everything behind the facade is private to the component. Each
app has one *composition root*, the only place that names which implementation fills each interface.

In a Go project, one component looks like this:

```
cmd/<app>/main.go              composition root: the only place naming implementations
internal/<component>/          the facade: exported API only
  <component>.go
  internal/
    domain/                    entities, value objects, errors
    application/               use cases and the interfaces they need
    infrastructure/            gateway implementations
    presentation/              handlers, commands
internal/shared/<module>/      focused shared modules, each with its own facade
```

In TypeScript, the facade is the component's `index.ts`. On a front end, a component is a feature
module, not a React component.

The boundaries are checked by tools, not left to discipline. Without that check, any file could
import any other, and the components would stop being separate. Each profile's boundary-enforcement
ADR names the tools: the compiler's `internal/` rule and `depguard` for Go, and
`eslint-plugin-boundaries` for TypeScript.

When the product is one cohesive domain rather than several separable capabilities, scaffold asks
about it and can use ports-and-adapters instead: a framework-free core with adapters around it.

### A monolith that is cheap to split

The result is one deployable application, with no network calls between use cases. Scaffold also
keeps each component ready to become its own service later:

- Each component owns its own data.
- No transaction spans two components.
- What crosses a facade is a contract, not one of the component's entities.

With those rules in place, moving a component into its own service is a deployment change rather
than a redesign.

## The decision records scaffold writes

Scaffold writes two kinds of ADRs into the new project:

| ADRs | Apply to | Decide |
| --- | --- | --- |
| `ADR-BASE-01` to `ADR-BASE-03` | every project | Clean Architecture, package-by-component, and keeping components cheap to split |
| `ADR-TS-01` and up | `typescript-react` projects | Inversify for dependency injection; TanStack Query, Zustand, and `useState` for front-end state; `eslint-plugin-boundaries` for boundaries |
| `ADR-GO-01` and up | `go` projects | dependency injection, boundary enforcement, and optional values |

The base ADRs are in
[`extensions/skills/codefall-scaffold/templates/adrs/`](../../extensions/skills/codefall-scaffold/templates/adrs/).
`ADR-BASE-03`, on keeping components cheap to split, is left out for a part that could never be
split into services, such as a command-line tool, a desktop or mobile app, or a library.

Each profile numbers its ADRs from 01 under its own prefix, so two profiles never collide. The
project's own decisions form a third sequence, starting at `ADR-001`.

## Surfaces and profiles

Scaffold divides a project into *surfaces* and applies one *profile* to each. A surface is one part
of the product that holds its own domain logic, such as a web front end or a Go API. A profile is the
set of templates and ADRs scaffold applies to one kind of surface. A project uses as many profiles as
it has surfaces.

A surface is defined by where the domain logic lives, not by which languages appear in the
repository. A language that only implements gateways belongs to the outer ring of a surface that
already has a profile.

| Profile | Covers | Status |
| --- | --- | --- |
| `typescript-react` | TypeScript and Node backends; React front ends for the web and React Native, including Tauri, Electron, and React Native apps whose native side is only wiring | supported |
| `go` | Go services, APIs, workers, daemons, and command-line tools that hold domain logic | supported |
| `rust-native` | Rust services, and Tauri shells that hold domain logic | planned |
| `kotlin-native`, `swift-native` | Android and iOS | planned |
| `dart-flutter` | Flutter, on mobile and desktop | planned |
| `python`, `java` | backends and services | planned |

A profile counts as supported only once its `templates/surfaces/<name>/PROFILE.md` is complete.

### Desktop and mobile apps

You can scaffold a Tauri desktop app or a React Native mobile app today, as a whole, when its native
side is boilerplate plus a few commands that wrap operating-system APIs. Those commands are gateways
in the `infrastructure/` ring. As `ADR-BASE-01` puts it, a Tauri `invoke`, a native bridge call, an
HTTP request, and a platform channel are all gateways, and the use case never learns which one it
got. Rust or Kotlin in the repository does not make a native surface, just as a Postgres driver does
not make SQL a surface.

The app becomes two surfaces only when the native side holds real domain logic, such as heavy
computation or security-sensitive work that must not run in the web view. Then the native side needs
its own profile, plus a *seam ADR* in the project's own sequence that decides which side owns the
domain. Scaffold asks you directly whether the native side holds business logic, because the file
listing cannot tell it: React Native ships its `android/` and `ios/` directories empty.

### React on the web and on mobile

React DOM and React Native share the `typescript-react` profile. React is the component model, and
the renderer is a detail of the outer ring, so the domain, use cases, gateways, dependency
injection, and boundary rules are the same on both.

The toolchains differ, and the profile is written so the difference does not matter. Metro, React
Native's bundler, compiles TypeScript with Babel rather than `tsc` and emits no decorator metadata
unless a Babel plugin adds it. Inversify can only resolve a dependency without an explicit token by
reading that metadata, so code that relies on it can work on the web and fail on React Native.
`ADR-TS-01` avoids the problem by requiring an explicit `@inject(TYPES.Thing)` token on every
constructor parameter, and the profile tells you not to set `emitDecoratorMetadata`. No metadata
plugin is needed on any target.

## How scaffold matches a project to profiles

Scaffold reads your vision, or your description if you declined one, and splits the project into
surfaces. It then matches each surface to a profile:

- When the description names a stack, such as "a React app" or "a CLI in Go", scaffold names the
  matching profile and asks you to confirm it.
- When the description leaves a surface's stack open, such as "an API" with no language named,
  scaffold asks you to pick from the supported profiles or "none of these". It never offers a planned
  profile.

If any surface has no supported profile, scaffold stops. It does not substitute the nearest
supported profile, write ADRs for an unsupported language by hand, or scaffold only the part that
fits. It names which surfaces fit and which do not, says what is supported, and offers to record the
request.

When a surface's stack is open and the directory already holds files, scaffold first consults the
project's agents once, the same agents [Configuration](configuration.md) describes for questions of
fact. They read the files, such as `go.mod` or a lockfile, and a confident answer becomes the
proposed option in the question. You still pick, and a planned profile is never proposed.
