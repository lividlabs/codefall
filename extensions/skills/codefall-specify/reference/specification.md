# What a specification is

The rules for the requirements, criteria, and optional sections inside a spec document. Read before
shaping the requirements (step 7) and again when writing (step 10).

## Contents

- The user story
- Acceptance criteria are written in EARS
- Criteria must be observable in a running system
- Implementation vocabulary is prohibited
- Numbering
- Criteria exist for coverage, never for symmetry
- Edge cases
- Key entities
- Assumptions

## The user story

Each requirement opens with one:

```
As a <consumer>, I want <capability>, so that <benefit>.
```

**The "so that" clause is mandatory.** If the user cannot supply the benefit, the requirement is not
understood yet — ask, do not fill it in.

**The consumer is not always a person.** A system can be the consumer:

> As the BFF, I want to cache search results, so that a repeated search does not pay the upstream
> round trip again.

Reach for a human consumer first, but do not manufacture one. `As an engineer` is legitimate when
building developer tooling, evasive otherwise.

**When no real persona exists, say so in the story rather than inventing one:**

> A clean persona was hard to identify here; the closest consumer is the on-call engineer,
> because they are the only party who observes the outcome.

**Some obligations have no story.** Audit logging, data retention, and authorization rules are
written as a requirement with a system consumer, or as criteria under the requirement they
constrain. Do not invent a traveler who wants their security events logged.

## Acceptance criteria are written in EARS

The generic form:

```
While <optional pre-condition>, when <optional trigger>,
the <system> shall <system response>
```

Zero or many preconditions, zero or one trigger, one system, one or many responses. Six patterns:

| Pattern | Shape | Use for |
| --- | --- | --- |
| Ubiquitous | `The system shall <response>` | facts that are always true |
| State driven | `While <precondition>, the system shall <response>` | obligations that hold throughout a state |
| Event driven | `When <trigger>, the system shall <response>` | responses to an expected event |
| Optional feature | `Where <feature is included>, the system shall <response>` | behavior present only in some configurations |
| Unwanted behaviour | `If <trigger>, then the system shall <response>` | responses to an undesired situation |
| Complex | `While <precondition>, when <trigger>, the system shall <response>` | combinations of the above |

Worked:

```
SPEC-003-REQ-01-AC-01
  WHEN a traveler selects export on a trip that has a destination and both dates,
  the system SHALL produce a file containing the destination, departure date,
  return date, and traveler count

SPEC-003-REQ-01-AC-02
  IF the trip is missing a departure or return date, THEN the system SHALL make
  export unavailable and SHALL name the missing field

SPEC-003-REQ-01-AC-03
  WHILE an export is in progress, the system SHALL show progress and SHALL allow
  the traveler to cancel it

SPEC-003-REQ-01-AC-04
  The system SHALL order exported segments by departure time
```

- **Pick the pattern that fits, not the one that is familiar.** `WHEN a traveler views a trip THEN
  the system SHALL display its tags` is a ubiquitous requirement wearing an event. Write `The system
  SHALL display a trip's tags`.
- **Failure behavior uses `IF … THEN`, not `WHEN`.** It keeps the failure paths visible as a group.
- **Say "the system" or name the actual system**, consistently within a spec. Never drop the
  subject: a criterion without one is an observation, not an obligation.
- **EARS governs criteria and nothing else.** The story, Context, Key entities, Edge cases,
  Assumptions, Out of scope, and Open questions are plain prose. Do not write `SHALL` in them.

## Criteria must be observable in a running system

Not observable in a test — observable in the product. For the BFF cache above, "a repeated search
does not call upstream" is only observable if something emits that fact — a metric, a log line, a
response header, an admin view. **That instrumentation is now part of the requirement.** Say so out
loud when it happens:

> Making this observable means the service has to expose cache hits somewhere. That is a real
> addition to the work. In scope, or do you want to leave it unverifiable for now?

Never quietly inflate the requirement, and never write a criterion nobody can check.

## Implementation vocabulary is prohibited

`WHEN I POST to /api/trips` passes a syntax check and defeats the purpose. So does naming a table, a
component, a queue, or a library. The test: could this criterion still be true after a complete
rewrite of the implementation? If not, it is describing the solution.

## Numbering

Three identifiers, nested, all hyphenated to match `VISION-002` and `ADR-BASE-01`:

```
SPEC-003                        the spec
SPEC-003-REQ-01                 a requirement within it
SPEC-003-REQ-01-AC-01           a criterion within that requirement
```

Spec numbers are three digits, requirement and criterion numbers two, all zero-padded.

- **Criterion numbers scope to their requirement**, so `AC-01` restarts under each one.
- **Write identifiers in full inside the document**, so a reference copies straight into a test name
  or a ticket.
- **Numbering is append-only, at every level.** A deleted criterion retires its number within its
  requirement; a deleted requirement retires its number within its spec; a spec identifier is never
  reused. `SPEC-003-REQ-01-AC-07` must never silently come to mean a different criterion.
- **Do not invent an alternate short form.** `grep SPEC-003` finds the document, its requirements,
  and every citation in one pass.

## Criteria exist for coverage, never for symmetry

**Never write a criterion to fill a section.** Requirements within one spec will have different
numbers of criteria, and a document that looks lopsided is not a defect. The only question is
whether the behavior is covered.

## Edge cases

Optional, per requirement. Boundary conditions that were considered and **deliberately left
unhandled**.

An edge case with a decided answer is an `IF … THEN` criterion and belongs above. An edge case nobody
has an answer for is an **Open question**. What stays here is the third kind: something the user
considered, decided not to specify behavior for, and wants on the record.

> - A trip edited while its export is running — the exported file may be stale; not addressed in
>   this spec

Do not use this section as a worksheet. Edge cases raised during the interview get resolved into
criteria, open questions, or an out-of-scope line, and only what survives that gets written.

## Key entities

Optional, per spec. The domain nouns the specification uses, and what each one means.

```
- **Trip** — a planned journey a traveler has saved
- **Itinerary** — the traveler-readable rendering of a trip
```

**Names and meanings only.** No fields, no types, no relations, no identifiers, no storage. `Trip
has a departureDate: Date and belongs to a User` is a data model, and that is `codefall-plan`'s
output.

Include the section when the feature involves data and the vocabulary needs settling. Omit it when
the nouns are ordinary English that nobody could misread.

## Assumptions

Optional, per spec. Calls made without asking, written down so the user can overrule them.

| Section | Means |
| --- | --- |
| **Assumptions** | I decided this without asking. Correct me if it is wrong. |
| **Out of scope** | We decided together that this is not included. |
| **Open questions** | Nobody knows yet, and someone has to answer before this is built. |

Assumptions are for the defensible defaults underneath a question nobody would think to ask.
Anything the interview should have surfaced belongs in the interview.

> - Recipients read the itinerary outside the app; no recipient account is required
> - The existing share sheet is reused rather than replaced
