# Surface profile: typescript-react

**Status:** supported.
**ADR prefix:** `TS` — this profile's ADRs are `ADR-TS-01` upward.

TypeScript on Node, and React web frontends. Covers either surface alone or both together.

## Fits when

- A backend or service written in TypeScript on Node, **or**
- A React frontend — web or **React Native** — regardless of what it talks to, **or**
- A **Tauri or Electron desktop app whose shell is only wiring** — the whole app, not half of it.

The frontend does not care what is on the other end of its gateways. Per ADR-BASE-01 a gateway is "an
interface to any external system" — an HTTP API, a Tauri `invoke` command, an Electron IPC channel,
and a WebSocket are all the same shape to a use case. A React frontend is this profile whether it is
served over HTTP or embedded in a native shell.

### Render targets

React is the component model; the renderer is an outer-ring detail. The same domain, use cases,
gateways, DI, and boundary rules serve React DOM and React Native alike — ADR-TS-02's three-tier split
is unchanged, since TanStack Query, Zustand, and `useState` all run on React Native. What varies is
toolchain:

| Target | Delta |
| --- | --- |
| React DOM (web) | The baseline. Topology varies — see below. |
| React Native | Metro transpiles with Babel, not `tsc`. Harmless under this profile's explicit-token rule; see the trap below for why. |

**The decorator-metadata trap, and why this profile sidesteps it.** Implicit Inversify resolution
reads `design:paramtypes`. That metadata records *runtime* types, and a TypeScript interface has
none — it erases to `Object`. Since ADR-TS-01 requires every dependency to be typed as an interface,
implicit resolution cannot work **under any toolchain, `tsc` included**. It is not a bundler problem.

Toolchain differences make it worse rather than causing it. Next.js detects `emitDecoratorMetadata`
in tsconfig and turns on SWC's transform, while Metro strips types with Babel and emits nothing
unless `babel-plugin-transform-typescript-metadata` is added. So a class-typed parameter with a
forgotten `@inject` resolves on Next and fails on React Native — the same code, two behaviours.

ADR-TS-01 makes `@inject(TYPES.Thing)` mandatory on every constructor parameter, which needs no
metadata at all. One rule, every renderer and every bundler, nothing to configure.

Verify it the way step 6 requires: resolve one decorated class from the container in a smoke test. A
container that fails only on the first real injection is the failure mode here, and it is the one
thing a type-check will never catch.

### App topologies

React DOM projects come in two shapes, and the choice changes the composition root and the lifetime
of the client-side libraries. It does **not** change the layering, the DI container, or the state
split.

| Topology | Shape | Pick it when |
| --- | --- | --- |
| **SPA + API** | A React bundle served statically, talking to a separate Node service over HTTP | No route needs *both* public reach and indexing |
| **Next.js SSR shell** | One app: `app/api/**/route.ts` holds the backend, client components hold the UI | Some routes must be **publicly reachable and worth indexing** |

The deciding question is *publicly reachable and worth indexing*, not static-versus-dynamic and not
first-party-versus-user-generated. A user-authored deck list at `/decks/must-have-cards-for-twin-suns`
is dynamic, user-generated, and exactly the thing that needs server rendering. A logged-in
collection page is none of those and can stay client-only.

**Keep the split.** Next.js here is an SSR shell over a conventional frontend/backend division, not
an invitation to server-component-first architecture. The backend lives behind route handlers; the
UI is client components calling them. Server Actions are not used — they blur the boundary this
stance exists to keep. In practice that means a lot of `'use client'` and no `'use server'`, which is
the correct smell for this topology.

Server rendering is then applied per route: public routes prefetch on the server and hydrate the
query cache (ADR-TS-02); authenticated routes fetch client-side as usual. That prefetch runs in a
server-component page, which is the **second and last** container-aware file kind alongside a route
handler — see ADR-TS-01. It resolves a use case and prefetches with it directly; it does not call the
app's own HTTP API, which would mean absolute URLs and forwarded cookies for no gain.

### The Next.js per-request trap

**Module-global is correct for stateless wiring and a data-leak bug for anything holding request
state.** One principle, two opposite conclusions, and getting either backwards is a real defect
rather than a style question.

The Inversify container **should** be a module-scoped singleton, cached on `globalThis` so dev
hot-reload does not rebuild it per request. It holds bindings, not data.

The `QueryClient` and every Zustand store **must not** be. A Next.js server handles concurrent
requests, so a module-level store or query cache is shared across users. TanStack Query's SSR guide
prescribes a fresh `QueryClient` per request on the server "to prevent data leakage between
requests," and Zustand's Next.js guide is blunter:

> **No global stores** — Because the store should not be shared across requests, it should not be
> defined as a global variable. Instead, the store should be created per request.

Zustand additionally forbids Server Components reading or writing stores at all, since RSCs cannot
use hooks or context. That falls out of the topology above anyway: state lives in client components.

This is the Next.js sibling of the React Native decorator trap — it looks correct, passes review, and
fails only under concurrency, where the symptom is one user seeing another's data.

### Thin-shell desktop and mobile apps

When `src-tauri/`, the Electron main process, or a React Native **native module** is stock
boilerplate plus a few commands wrapping OS APIs — file dialogs, notifications, the tray, the camera,
secure storage — those commands are **gateway implementations**, and the app is entirely this
profile. Kotlin or Swift in `android/` and `ios/` no more makes it a native surface than a Postgres
driver makes SQL one.

Concretely: the `invoke` call or bridge module lives in the `infrastructure/` ring of whichever
component needs it, behind an interface owned by that component's `application/` layer. Use cases
never import Tauri or `NativeModules`, and ADR-TS-03 rule 5 already forbids it. Swapping the desktop
shell for a web deployment, or Android for iOS, then touches one gateway implementation and nothing
else.

## Does not fit

Non-Node backends and non-React frontends.

A shell or native module that carries **real domain logic** — heavy compute, native integrations with
their own rules, security-sensitive work kept out of the JS thread — is a second surface, not wiring.
It needs its own profile, and the project needs a seam ADR deciding which side owns the domain.
ADR-TS-01 anticipates this: non-TS layers use idiomatic constructor injection and trait objects, not
Inversify.

The test is **where domain logic lives**, and it is a question for the user. Do not infer it from the
presence of a `src-tauri/` or `android/` directory — React Native ships those empty.

## Visibility model

TypeScript has **no `package-private`**, so component facades are unenforceable by the compiler. This
is the fact ADR-TS-03 is built around, and it is why this profile requires a boundary linter rather
than merely recommending one. Profiles for languages that *do* have real visibility (Kotlin
`internal`, Rust `pub(crate)`, Go's `internal/` packages, Java modules) need a much lighter
boundary-enforcement ADR of their own — do not copy this one across. `ADR-GO-02` is the worked
example: the compiler covers the facade rules outright, and only the layer rule needs a linter.

## Toolchain

| Concern | Choice | ADR |
| --- | --- | --- |
| DI container | Inversify, one composition root per app; explicit `@inject` tokens always | ADR-TS-01 |
| App topology | SPA + separate API, or Next.js SSR shell | — |
| Next.js lifetimes | Container module-global; `QueryClient` and stores per request | ADR-TS-01, ADR-TS-02 |
| Server state | TanStack Query | ADR-TS-02 |
| Shared client state | Zustand | ADR-TS-02 |
| Local UI state | `useState` / `useReducer` | ADR-TS-02 |
| Boundary enforcement | `eslint-plugin-boundaries`, `dependency-cruiser` optional | ADR-TS-03 |

## ADRs this profile supplies

- `adrs/ADR-TS-01-dependency-injection.md`
- `adrs/ADR-TS-02-frontend-state.md` — **backend-only projects skip this one.**
- `adrs/ADR-TS-03-boundary-enforcement.md`

Shared ADR-BASE-01 through ADR-BASE-03 come from `templates/adrs/` and apply to every profile.
ADR-BASE-03 is dropped only for a surface that could never be split into services at all — a CLI,
a desktop or mobile app, a library — not for one that merely has no current plan to split.

## Depth notes

Beyond docs, this profile's **project files** tier owes: `package.json`, a `tsconfig` with
`experimentalDecorators`, ESLint including the `eslint-plugin-boundaries` rules, formatter, test
runner, and CI running all of it.

**Add every dependency with the package manager** — `npm install -D <name>`, `pnpm add -D <name>`,
`yarn add -D <name>` — never by writing a version into `package.json` by hand. The package manager
resolves the current release and writes the lockfile to match it; a version written from memory is
whatever was current when the model was trained.

**Do not set `emitDecoratorMetadata`.** Nothing here needs it, and enabling it lets a forgotten
`@inject` on a class-typed parameter resolve on some toolchains and not others — a portability bug
that the explicit-token rule exists to make impossible. No metadata plugin is needed on any target
for the same reason.

On the **Next.js topology**, add `next.config.ts`, the `app/` tree with at least one
`app/api/**/route.ts`, and the container module described in ADR-TS-01. Wire the `QueryClient` and
any Zustand provider as per-request instances from the start — retrofitting that after the fact means
auditing every store for cross-request leakage.

The **runnable skeleton** tier adds the component folders with their `index.ts` facades and nested
Clean layers, plus one Inversify composition root per app that boots with no features in it.

The local scripts at either code tier: `update` runs the package manager's frozen install for the
lockfile the tier wrote — `npm ci`, `pnpm install --frozen-lockfile`, `yarn install --immutable`
— and `mise install` first when the project pins a runtime; `start` exits `0` with nothing to bring
up. They are two `package.json` scripts, `local:start` and `local:update`, declared as
`npm run local:start` and `npm run local:update` in the package manager's own form.

---

## Adding a new profile

Copy this file's shape. A profile is complete when it states: an **ADR prefix**, what it fits, what it
explicitly does not, its **visibility model** (which determines how much its boundary-enforcement ADR
has to do), its toolchain choices, the ADRs it supplies, an `AGENTS.md.skeleton`, and its depth notes.
Until all of that exists, the surface stays `planned` in the catalog and `codefall-scaffold` refuses projects
that need it.

**Pick a short uppercase ADR prefix** — `TS`, `GO` — and check that no existing profile has claimed
it. Numbering restarts at 01 inside each profile, so `ADR-<PREFIX>-01` is simply the first ADR this
profile supplies and nothing has to be coordinated with any other profile. Two profiles in one
project never collide, and a profile with no use for a given concern just doesn't ship that ADR —
there is no gap to explain and nothing to renumber.

Scope a profile to **one surface**, not to one kind of product. "Desktop app" is not a profile — it
is a web surface plus a native shell, and each of those is reusable on its own. Profiles that
describe products rather than surfaces duplicate each other and rot.
