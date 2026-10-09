# ADR-TS-03: Boundary Enforcement

## Status

Accepted — <date>

## Context

TypeScript has no `package-private` / `internal` visibility. So package-by-component's `index.ts`
facades (ADR-BASE-02) are conventions the compiler will not enforce — and without enforcement,
package-by-component silently degrades into "package by feature with everything public," which is the
weaker scheme. Enforcement is what makes the boundaries real. This matters *more* in an AI-assisted
workflow, where fast code generation drifts without hard guardrails.

## Decision

### Component unit

- Components are **folders within an app**, each fronted by an `index.ts` facade. Use **packages**
  only for genuinely shared / cross-app code (shared kernel, UI kit, technical infra).
  Package-per-feature is real enforcement but heavy bookkeeping — overkill.

### Tooling

- **`eslint-plugin-boundaries`** as primary — it gives **editor-time** feedback, catching a
  violation as it's written (the guardrail that matters most for AI-assisted work).
- **`dependency-cruiser`** optional, as a CI graph-level gate if ESLint rules aren't enough.
- On the **Next.js topology**, `server-only` is an optional extra: importing a module marked with it
  from a client component becomes a build error. Useful as a safety net on backend modules, but not
  what enforces the boundary here — the real server/client boundary is the HTTP call to a route
  handler, not the module graph.

### When

- Wire it in **at the project scaffold, day one.** Deferring defeats the purpose — boundaries only
  hold if enforced from the start.

### Rules to enforce

1. **Facade-only imports** — no reaching past a component's `index.ts` into its internals.
2. **Clean layer rule** — dependencies inward only: `domain` ← `application` ← `infrastructure` /
   `presentation`; no outward or sideways imports.
3. **Cross-component only via public facades.**
4. **Shared modules can't import components** (one-way).
5. **Frontend inner rings** (`domain` / `application`) can't import the UI framework or gateways
   directly.

## Consequences

- Setup + CI cost, and editor squiggles as you work — which is the point.
- Without this, the entire package-by-component choice is cosmetic.
- The lint config *is* the encoded architecture; keep it in review scope.

## Related

- ADR-BASE-02 — Package-by-component (the boundaries being enforced)
- ADR-BASE-01 — Clean Architecture (the layer rule at #2)
