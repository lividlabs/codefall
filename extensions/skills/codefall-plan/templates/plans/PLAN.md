# PLAN-NNN: {Title}

**Status:** Ready — {date}
**Related:** spec: SPEC-004 · adr: ADR-003

[Every bracketed instruction in this template is stripped on emit. The Related row is a labeled
list, and is omitted when there is nothing to relate; its labels, and which hops to leave out, are
in `reference/document.md`. The status is `Draft` while Decisions needed has an entry. A section
below that opens "Only when" is omitted when its trigger does not fire: nothing is written to fill
a heading.]

## Overview

[The approach in a paragraph, then the key decisions and why each was made. Research findings go
here or under Architecture, next to the decision they bear on.]

<the approach>

- **<decision>** — <why it was made>

## Architecture

[Components, their relationships, and data flow. A Mermaid diagram where one earns its place. A
paragraph for small work.]

## Components and Interfaces

[Only when public types, APIs, or contracts are new or changed. Otherwise omit.]

- `<type, endpoint, or message>` — <what it carries, and who produces and consumes it>

## Data Models

[Only when anything persists, or the shape of state changes. Otherwise omit. Name the migration it
implies, if any.]

## Error Handling

[Only when the work adds a failure mode, or the fix is one. Otherwise omit.]

- <failure mode> — <what the system does about it>

## Testing Strategy

[Only when the test approach is non-obvious. Omit it when "add a regression test" covers it.]

## Technical Context

[Only when the work adds a dependency or touches infrastructure. Otherwise omit. Drop a line whose
answer is "unchanged" rather than writing the word.]

- **Language / version** — <language and version>
- **Primary dependencies** — <libraries this work relies on>
- **Storage** — <the store and schema it reads or writes>
- **Testing** — <the test tools and passes>
- **Target platform** — <where it runs>
- **Performance goals** — <the number it must meet>
- **Constraints** — <what it must not add or break>
- **Scale** — <the load it must carry>

## Alternatives Considered

[Only when a real choice was made, which is also the ADR trigger. Otherwise omit.]

| Option | Why it was set aside |
| --- | --- |
| <option> | <the reason> |

## Hard Constraints

[Only when there are invariants worth testing directly: concurrency, data integrity, security
boundaries. Otherwise omit.]

- THE SYSTEM SHALL <invariant>

## Decisions needed

[Only when a technical choice was put to the person and not settled. Otherwise omit. One entry per
decision; the Consulted line is omitted when nobody was consulted.]

### 1. <the choice>
- **Affects:** <the requirement identifiers>
- **Options:** <each option, and what it costs>
- **Needs an ADR:** <yes or no — why>
- **Consulted:** <who answered, and what they preferred>

## Tasks
> Staged. Not yet in Beads.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | <task> | — | <section> |
| T2 | <task> | T1 | <section> |

[The callout is rewritten in place when the beads are created, per `reference/document.md`. The
table keeps this shape for the life of the plan.]
