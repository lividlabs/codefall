# The plan document

The shape of `docs/plans/PLAN-NNN-slug.md`. Read before planning (step 5) and again when
drafting (step 7).

## Contents

- The header
- Required sections
- Conditional sections
- Decisions needed
- Technical Context
- Hard Constraints
- Tasks
- Tasks, after creation
- Local IDs are append-only

## The header

A title heading and two rows, the same shape `VISION` and `SPEC` documents use. The body starts at
`## Overview`.

```markdown
# PLAN-007: Stage context handoff

**Status:** Ready — 2026-08-31
**Related:** spec: SPEC-004 · adr: ADR-003
```

`Status` is always present. `Related` appears only when there is something to relate, and is a
labeled list, not a set of rows.

| Label | Holds |
| --- | --- |
| `spec` | The specification this plan implements |
| `bug` | The bug report this plan fixes |
| `vision` | The vision that framed it — **only when there is no spec** |
| `adr` | The ADR or ADRs this plan produced |

**Never record a hop you can derive.** A spec names its vision in its own `**Vision:**` row, so a
plan records the spec alone. A plan written from a vision with no spec records the vision. A
plan for a bug records the report, and a spec only when the report's Spec row does not name it. A
plan with none of these has no `Related` row until it produces an ADR.

## Required sections

| Section | Holds |
| --- | --- |
| **Overview** | The approach in a paragraph, plus the key decisions and why each was made |
| **Architecture** | Components, their relationships, and data flow. Mermaid where a diagram earns its place. May be a paragraph for small work |
| **Tasks** | The task table, under a callout that says whether the beads exist yet |

**Research findings go inline**, in Overview or Architecture, next to the decision they bear on.
There is **no sibling `research.md`**, and no `data-model.md`, `quickstart.md`, or `contracts/`.

## Conditional sections

Include a section only when its trigger fires. Nothing is written to fill a heading.

| Section | Include when |
| --- | --- |
| **Components and Interfaces** | New or changed public types, APIs, or contracts |
| **Data Models** | Anything persists, or the shape of state changes |
| **Error Handling** | New failure modes, or the fix *is* a failure mode |
| **Testing Strategy** | The test approach is non-obvious. Skip it when "add a regression test" covers it |
| **Technical Context** | A new dependency, or infrastructure is touched |
| **Alternatives Considered** | A real choice was made. This is also the ADR trigger |
| **Hard Constraints** | There are invariants worth asserting directly |
| **Decisions needed** | A technical choice the run could not settle with the person, because the persona parks such choices or the person said they cannot decide it |

## Decisions needed

The section a plan carries when a technical choice was put to the person and not settled: under
the `product-manager` persona, every such choice; under any persona, one the person says they
cannot decide. The run never settles it in their place, and never reads silence as a choice. It
researches first, so what is parked is a judgment call and not a gap in reading.

One entry per decision:

```markdown
## Decisions needed

### 1. Signed payload or session lookup for the stage context
- **Affects:** SPEC-004-REQ-02, REQ-03
- **Options:** a signed payload — no store, every consumer verifies; a session lookup — one store,
  one place to expire, a network hop on every read
- **Needs an ADR:** yes — other components will build on it
- **Consulted:** codex:gpt-5-codex preferred the signed payload; the reasoning is under Overview
```

**A plan with a non-empty Decisions needed section is `Draft`, never `Ready`, and creates no
beads.** Before the status is set, the run asks the person once whether to settle the entries with
sensible defaults now or leave them for an engineer. Settled now, each entry's outcome moves into
the section it belongs to, the entry is removed, and the graph is created in the same run. Left,
the plan stays `Draft`, and an engineer's run of `codefall-plan` in Promote mode settles each
entry with that person, moves the outcome, removes the entry, and only then creates the graph. The
section is deleted when it is empty, like every conditional section.

## Technical Context

A fill-in-the-blank block. Drop a line whose answer is "unchanged" rather than writing the word.

```markdown
## Technical Context

- **Language / version** — Go 1.23
- **Primary dependencies** — `samber/do`, `pgx/v5`
- **Storage** — Postgres 16, existing `bookings` schema
- **Testing** — `go test`, `testcontainers` for the integration pass
- **Target platform** — Linux container, deployed as the existing API service
- **Performance goals** — p99 under 200ms for the read path
- **Constraints** — no new outbound network dependency
- **Scale** — roughly 40 writes/second at peak, unchanged by this work
```

## Hard Constraints

EARS-lite: `THE SYSTEM SHALL …`, for invariants only — concurrency, data integrity, security
boundaries. Only the ones worth testing directly. This is not `codefall-specify`'s full EARS: a
criterion is what a consumer observes; a hard constraint is an invariant of the implementation.

```markdown
## Hard Constraints

- THE SYSTEM SHALL hold no more than one open write transaction per booking at a time.
- THE SYSTEM SHALL reject a context payload whose signature does not verify, before parsing it.
```

## Tasks

One table under a callout. The table keeps its shape for the life of the plan; the callout is
**rewritten in place** when the beads are created.

```markdown
## Tasks
> Staged. Not yet in Beads.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | Add `StageContext` type + serde | — | Components |
| T2 | Wire context load into `/scaffold` | T1 | Architecture |
| T3 | Emit context on `/envision` exit | T1 | Data Models |
| T4 | Integration test: vision → scaffold | T2, T3 | Testing Strategy |
```

The local IDs make the dependency edges reviewable before the beads exist, and they name the beads
once they do. The **Plan ref** column names the section of this plan that motivated the task,
so `codefall-implement` opens the relevant fifteen lines rather than the whole document.

The table holds what the plan decided: the tasks, their edges, and the section each came from.
Work state — status, assignee, comments — is Beads' alone, and **the table never grows a column for
it.**

## Tasks, after creation

The callout is rewritten; the table stays.

```markdown
## Tasks
> Created in Beads 2026-08-29 as `booking-PLAN-007`, one bead per row as `booking-PLAN-007-Tn`.
> This is the plan as last approved. Beads is authoritative, and a difference between the two is
> reconciled by revising the plan.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | Add `StageContext` type + serde | — | Components |
| T2 | Wire context load into `/scaffold` | T1 | Architecture |
| T3 | Emit context on `/envision` exit | T1 | Data Models |
| T4 | Integration test: vision → scaffold | T2, T3 | Testing Strategy |
```

Bead identifiers are the document's own numbering behind the project's prefix —
`<prefix>-PLAN-NNN` for the epic, `<prefix>-PLAN-NNN-Tn` for each task — set by `bd rename`
after creation, per `beads.md`. The callout names the epic in full, so a reader sees the prefix
without running `bd`, and the per-row form says what every task is called; there is no column of
bead IDs because every one is derived. A replaced task's old bead is renamed
`<prefix>-PLAN-NNN-Tn-superseded`, and the current bead always holds the plain ID.

A row's title, edges, and plan ref change only through Revise, which edits the row and the bead
together (`revising.md`), and a revision adds its date to the callout: `Created in Beads
2026-08-29, revised 2026-09-10, as …`. A bead edited directly with `bd` is the one way the table
falls behind the graph, and the rules installed at `docs/plans/AGENTS.md` forbid it. Anything
filed under the epic since creation — discovered work, the landed bead — is Beads' alone and never
a row.

A plan created before the table stayed carries a mapping line in place of the table. Its epic
still resolves by number, and its first revision rebuilds the table from the beads before going row
by row.

## Local IDs are append-only

`T1`, `T2`, `T3` are permanent within a plan. **A retired local ID is never reused.** A removed
row comes out of the table and its ID goes on a `Retired:` line under it, so the next task added
takes the number after the highest ever used, never the first gap — that would silently make `T2`
name a different bead.

```markdown
| T5 | Load context in every verb's preamble | T1 | Architecture |

Retired: T2
```
