# ADR-016: Plan increments

## Status

Accepted — 2026-10-28

## Context

The chain's units line up one to one: a spec becomes a plan, a plan becomes an epic, and an epic is
a delivery that `implement` runs through rounds of build, review, test, and triage until a person
merges it (ADR-013.7). Nothing between the spec and the delivery asked how long the delivery would
run. `specify`'s only split is by cohesion: requirements that do not share a consumer and a purpose
become sibling specs, and its scope table names sizing and build order as not its business. `plan`
scales its document to the work with three tiers, and its one size test is per task. The round
ceiling in `implement` bounds a delivery after it has started, and ADR-013.7 records a large epic
running nine to eleven rounds.

A delivery goes best when it finishes in a few hours of agent time with a round or two: the person
reads one ledger and one stack of pull requests, and what review and test find is small enough to
build in the next round. A spec can hold several such deliveries, and the one verb that sees the
size in concrete terms before the delivery starts is `plan`, which reads the code and stages the
tasks.

Three places for the breakdown were weighed.

**Split the spec at `specify`, by size.** The person is present, so the increments come out
product-shaped. But the size is not visible until the code is read for a cut, which `specify` does
not do; its scope rule says sizing is not its concern; and its siblings are for different purposes,
which a size split of one cohesive feature contradicts.

**Tune the delivery.** A lower ceiling or a narrower batch. The rounds are where the cost shows,
not where it is decided, so this bounds the damage without changing the size.

**One spec, one plan per shippable increment, decided at `plan`.** Taken, below.

Two rules of the repository shape how it is written. A rule guides a run and does not gate it, so
the size is a judgment the run states with its reason, never a threshold it applies. A list lives in
the project's data, so the reference figures are the project's own closed deliveries, which Beads
records, rather than numbers in the skill.

## Decision

### A plan is one delivery, and a spec may have several plans

`plan` judges the size of the delivery at the step where it decides the tier, from the task count,
the components touched, the tasks verified through the wired product, and the depth of the
dependency chain, against the project's closed epics: their children, their `round` metadata, and
their dates. It says what it sees once, with a specific cut, and does what the person says; which
requirements ship together is a product decision under `shared/decisions.md`. It checks the
judgment again once the tasks are staged. A starting budget of four to six hours of agent time is a
reference point, and a project states its own in `CUSTOMIZE.md`. No count is a threshold.

### The cut is whole requirements in ship order, each leaving the product usable

A requirement is the smallest thing a consumer observes, so a plan never cuts one in two. Every
prefix of the order leaves the product whole: a consumer can use what shipped, nothing visible does
nothing, and no surface is half-changed. The build-boundary rule for tasks applies across
increments. When no order passes, there is no split, and the run says how long the one delivery
looks. A cut the person declines is recorded.

### One plan, one epic, one edge per increment

Each increment is a plan document like any other, numbered consecutively in ship order and written
in the same run and the same pull request. Its `Related` line names the requirements it covers on
the `spec` label and the plan before it on an `after` label. The first plan holds the architecture
that spans the feature; a later one holds only what is its own. Each has its own epic, and the
later epic depends on the one before it, `bd dep add <later epic> <earlier epic>`, so the later
epic and everything under it stays out of `bd ready` until the earlier one closes, which is when
its graph is empty and its code is on `main`.

`bd ready --mol <epic>` does not see a blocker outside the epic. So `plan` verifies a later
increment with it, which is the check of the rows against each other that `plan` wants, and
`implement` reads plain `bd ready` before it claims an epic, and stops, naming the epic that holds
it, when the epic is not listed.

### Split mode

A plan whose delivery turns out bigger than one is split on request: untouched rows move to a new
plan and epic numbered next, their IDs retired in the old table, their beads renamed and
reparented, and the edge added. A bead someone holds stays where it is, per the rule that a ticket
must not change under someone holding it.

## Consequences

- A long feature becomes several deliveries that each merge on their own, and a person sees the
  whole feature in the graph from the first plan.
- The judgment is the run's and improves with the project's history; the first deliveries in a
  project are the calibration, and the skill says when it has none.
- `plan`'s check that a plan already covers the work learns that several plans may share a spec,
  and the plan guide and `docs/plans/AGENTS.md` say so.
- `specify` is unchanged. An early signal at its recap, and an optional section naming the ship
  order while the person is present, are a possible later edition.
- `implement` does not record a delivery's elapsed agent time; the judgment reads the epic's dates
  and round metadata, which are close while a delivery runs continuously, and a resumed delivery
  overstates them.
- Nothing runs the Split mode on its own. A delivery that drags is a signal `implement`'s round
  boundary already weighs, and having it ask `plan` to split from there is a possible later edition.

## Related

- [ADR-013.7](ADR-013.7-deliveries.md) — the delivery, its rounds, and the ceiling this decision
  sizes against.
- [ADR-014](ADR-014-upstream-verbs-and-document-landing.md) — one pull request per document verb
  run, which the several plans share.
- [ADR-008](ADR-008-upstream-amendments.md) — amendments and revision beads, whose row rules Split
  mode follows.
- `extensions/skills/codefall-plan/reference/increments.md` — the procedure.
