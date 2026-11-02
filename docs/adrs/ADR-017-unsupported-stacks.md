# ADR-017: Unsupported stacks

## Status

Accepted — 2026-10-29

## Context

`scaffold` matches each surface of a project to a surface profile, and a profile supplies the
researched decisions for its stack: dependency injection, boundary enforcement, client state, and
the files each depth writes. Three profiles are supported, `typescript-react`, `go`, and
`dart-flutter`. A surface whose stack had none stopped the run. Scaffold said the stack was not
supported, named the three, offered to "record the request", and ended. `upgrade` did the same when
a project adopting the stance had such a surface.

The stop had a reason. A profile written from what a model knows about a language produces ADRs
marked Accepted that nobody decided, and the `dart-flutter` profile took a day of reading package
pages and framework documentation to decide item by item. But the stop also turned away a person
who wanted the stance and had a stack codefall does not cover. The stance, Clean Architecture
organized package-by-component with enforced boundaries, does not depend on the stack; only its
tooling does. The person usually knows that tooling, or can find it.

"Record the request" named no place. The tracker rule says a verb files on the project's own
tracker and nowhere else, and hands anything for codefall over as issue text. Issue text handed to
each person who meets the stop produces duplicate requests, or none, and codefall cannot see which
stacks people want.

Four options were weighed.

**Keep the stop, and offer the stack-agnostic core.** The core was already available to a person
who knew to ask for it. It leaves the surface with no decisions at all.

**Let the run pick the stack's tooling.** This is the improvised profile the stop was written to
prevent.

**The person describes the stack, and the decisions are the project's.** Taken, below.

**File the request on codefall's tracker.** A yes given inside a run is consent for this project,
and a new issue on another repository is a public act the person should write themselves.
Not taken for a new request. A reaction and a comment on a request that already exists are
smaller, and they are what makes the request count.

## Decision

### The person describes the stack, and scaffold records it as the project's own

When a surface has no supported profile, scaffold says so and offers to scaffold that surface on
the project's own stack. On yes, the person describes the architecture and practices for their
stack against what every profile settles: layout, dependency injection, boundary enforcement for
each of the five rules, errors and absence, tests, client state for a surface with a UI, and the
practices agents should follow. "Not decided" goes to the decision log's `Open`.

In the same question scaffold offers to search for how projects on that stack apply Clean
Architecture, reading the stack's own documentation and package pages first. It reports per concern
what the sources use, how it maps onto the rings and the facade, and where the stack works against
the stance, with sources linked. Options are presented flat, because codefall has no stance on a
stack it has no profile for. The person decides each concern. The stance itself does not change for
the stack.

Each decision becomes a project ADR, `ADR-001` upward, whose Context says codefall has no profile
for the stack and the decision is the project's own, and whose Related section links the sources.
The surface gets a scoped `AGENTS.md` written from those ADRs, using a supported skeleton's
sections and none of its content. The surface stays docs only, because the code tiers follow a
profile's depth notes and this surface has none, so its boundary enforcement is owed from the first
day. Provenance records the surface under `ownStack`, and `upgrade` compares only the
stack-agnostic templates for it. `upgrade`'s adoption path offers the same.

### The request is searched for, and an existing one is voted on

Before the own-stack offer, the run searches `lividlabs/codefall` for an issue requesting a
profile for the stack. Maintainers put the `profile request` label on each such issue, and the run
searches by that label and by title, since a new request may not be labelled yet. The label exists
because a title search misses a request worded differently, such as "Support Django" for a Python
profile. An open one is named, and the run offers a 👍 reaction and a comment, shown
before posting and posted on the person's account only on a yes. A closed, completed one means a
later codefall ships the profile, and the run points at `codefall upgrade`. With no issue, the run
hands over issue text as before. A comment or issue names the stack, the kind of surface, and the
codefall version, and nothing about the product, because the tracker is public.

The tracker rule in `docs/workflow.md` states this as its one exception.

## Consequences

- A person on Python, Rust, or Kotlin gets the stance and a recorded set of decisions instead of a
  stop, and a later run of any verb reads those ADRs like any others.
- Nothing in codefall checks the own-stack decisions. They are only as good as the person's
  description and the sources, and the ADRs say so.
- The surface starts with no boundary enforcement wired. Scaffold cannot verify a tool it has no
  profile for, so the person owes that task, as after any docs-only scaffold.
- Requests for a profile collect on one issue per stack, so the planned profiles in the catalog can
  be ordered by demand.
- A verb now writes to a repository other than the project's, in this one case. The comment is
  shown first, and the person's yes covers only what was shown.
- When a profile ships for a stack a project already described, nothing migrates the project's ADRs
  to it. `upgrade` treats the surface as `ownStack` until a later edition offers the move.

## Related

- [ADR-BASE-01](ADR-BASE-01-clean-architecture.md) and
  [ADR-BASE-02](ADR-BASE-02-package-by-component.md) — the stance every surface keeps.
- [ADR-010](ADR-010-upgrade.md) — `upgrade`, whose adoption path and applicable set this changes.
- `docs/workflow.md`, *Who is authoritative for what* — the tracker rule and its exception.
- `extensions/skills/codefall-scaffold/reference/unsupported.md` — the procedure.
