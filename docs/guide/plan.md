# /codefall-plan

`/codefall-plan` decides how a change gets built and turns the work into tasks that
[`/codefall-implement`](implement.md) can pick up. Run it once a feature has a spec, when you have a
bug report to fix, or when you can describe the work in a sentence.

```
/codefall-plan SPEC-003
```

The argument can be a spec or bug report identifier, a vision, or a description of what needs
building. With no argument, the skill asks what you want to plan.

## What it produces

A run produces up to two things:

- **A plan document** at `docs/plans/PLAN-007-itinerary-export.md`, when the change is big enough to
  need one. It is a software design document: the approach, the architecture, the decisions and
  why they were made, and a table of tasks at the end. `/codefall-implement` and `/codefall-review`
  check the code against it, and it stays as the description of that part of the system until the
  code moves away from it.
- **Tasks in Beads**, always. Beads is a task tracker that stores its data in your git repository,
  and each task in it is a *bead*. A bead records which other beads must finish before it can
  start; those records are *dependency edges*. The beads and their edges together form the *task
  graph* that `/codefall-implement` works through.

The examples on this page continue the trip planner from the [README](../../README.md#how-it-works):
a spec, SPEC-003, asks for travelers to be able to export their itinerary.

## What it reads first

Plan reads what already exists before it proposes anything:

- **The target.** When no spec was named, it looks in `docs/specs/` for one that matches and offers
  it. A spec is not required, but when there is none and the work is more than a fix, it says once
  that the plan has no written target and offers `/codefall-specify`.
- **What frames the target.** It reads the spec's vision, if the spec names one, or `docs/visions/`
  when no spec framed the work.
- **The project's rules.** It reads `docs/adrs/`, every `AGENTS.md`, and `docs/decision-log.md`,
  and plans within them.
- **Existing plans and beads.** When a plan or a bead already covers the work, it says so. A task
  this plan would create that is already a bead becomes a dependency edge, not a new task.
- **The code** the change touches.

A spec that is not ready is finished in the same run. When the spec still says `Status: Draft`,
plan runs `/codefall-specify` on it to settle what is open. When a requirement is still waiting for a
mockup, which its Design notes show as `Mockup: pending`, plan runs
`/codefall-mock-up` for it before planning. A bug report is a target in the same way as a spec. When
the report could not reproduce the bug, plan tries to reproduce it, and it finds the cause before it
chooses a tier.

## When does a change get a plan document?

Plan puts every change into one of three *tiers*, and it tells you which tier before it writes
anything. You can overrule it.

| Tier | When | What you get |
| --- | --- | --- |
| 0 | The change stays inside one component, changes no public interface, and comes to one or two tasks | Beads only, with no document |
| 1 | The change crosses a component boundary, or fans out past roughly three dependent tasks | A plan document with the three required sections |
| 2 | The same as tier 1, and one or more conditional sections applies | The required sections, plus each conditional section that applies |

A null check in one function is a tier 0 change. A plan document for it would add work without
telling anyone something new, so the run creates the bead and stops. Tier 0 has no document to
approve; you approve the beads themselves, with their titles, bodies, and edges.

Every plan document has three required sections: **Overview**, **Architecture**, and **Tasks**.
Eight more appear only when their condition holds:

| Section | Included when |
| --- | --- |
| Components and Interfaces | Public types, APIs, or contracts are new or changed |
| Data Models | Anything persists, or the shape of state changes |
| Error Handling | There are new failure modes, or the fix is a failure mode |
| Testing Strategy | The test approach is not obvious |
| Technical Context | A new dependency arrives, or infrastructure is touched |
| Alternatives Considered | A real choice was made between approaches |
| Hard Constraints | There are invariants worth stating directly |
| Decisions needed | A technical choice was left for an engineer to settle |

A section that does not apply is deleted, heading and all. Before writing, the run tells you which
sections it left out and why, for example "no Data Models section, because nothing here persists,"
so you can catch an omission that was a mistake.

Research goes next to the decision it supports. A plan has no separate `research.md`, no
`data-model.md`, and no `contracts/` directory, because a reader looking at a decision should find
its evidence in the same place.

## How the task table works

The task table is written before any bead exists. Each row has a local identifier, and the
**Depends on** column shows the edges:

```
## Tasks
> Staged. Not yet in Beads.

| ID | Task                               | Depends on | Plan ref     |
|----|------------------------------------|------------|--------------|
| T1 | Add the itinerary export format    | —          | Data Models  |
| T2 | Add the export button and download | T1         | Architecture |
```

Staging the table first lets you review the order while it is still cheap to change. A wrong order
or a missing prerequisite is the mistake a review of the plan is most likely to catch, and a flat
list of tasks would hide it. The **Plan ref** column names the section of the plan that motivated
each task, so the agent building it reads that section instead of the whole document.

Before the table is final, the skill checks each task that deletes or renames something other code
uses. If that code belongs to other tasks, the first task cannot be built and merged on its own,
because the moment it lands the other code stops compiling. The skill then either keeps the first
task add-only, with the new thing beside the old one and the deletion moved into the task that
updates the code using it, or merges the coupled tasks into one, and the plan says which.

When you approve the plan, each row becomes a bead, and the note above the table is rewritten:

```
> Created in Beads 2026-10-08 as `trips-PLAN-007`, one bead per row as `trips-PLAN-007-Tn`.
> This is the plan as last approved. Beads is authoritative, and a difference between the two is
> reconciled by revising the plan.
```

`trips-PLAN-007` is the *epic*: a bead that groups all of the plan's tasks. `trips` is the project's
Beads prefix, so T1 becomes `trips-PLAN-007-T1`. A person can read these identifiers aloud and find
the row each one came from. Tier 0 has no epic.

The table stays in the document after the beads exist, and from then on Beads is authoritative for
the state of the work. The table records only what the plan decided: the tasks, their edges, and
the section each came from. It never gets a status column. A row changes only through a revision,
which edits the row and its bead together so the two always match. When a row is removed, its
identifier goes on a `Retired:` line under the table and is never used again.

## When a spec is more than one delivery

A plan is one delivery: one epic that [`/codefall-implement`](implement.md) runs through rounds of
building, reviewing, and testing until it is merged. A delivery goes best when it finishes in a few
hours of agent time with a round or two, and a spec can easily hold more work than that.

So when plan decides the tier, it also judges the size of the delivery. It looks at how many tasks
the work comes to, how many parts of the system they touch, how many are verified through the
running product, and how deep the dependency chain is, and it compares that with the deliveries
this project has already run, which Beads records. None of those is a threshold. Plan says what it
sees once, with a specific cut, for example "fourteen tasks across three components, which is more
than any delivery here has run in one epic; I would ship requirements one to three first and four
and five after," and then does what you say. It checks the judgment again once the task table is
staged, because the count can change it either way.

A cut is made of whole requirements in the order they would ship, and each increment has to leave
the product whole and usable on its own. When no such order exists, because nothing in the spec
works until all of it does, there is no split, and plan says how long the one delivery looks
instead. You can also decline a cut that would work, and the plan records that you did.

When you take a cut, the run writes one plan per increment, numbered in ship order, in the same
pull request. Each plan's header names the requirements it covers and the plan it follows, for
example `spec: SPEC-004 (REQ-04, REQ-05) · after: PLAN-007`. The first plan holds the architecture
that spans the feature, and the later ones hold only what is theirs. Each plan gets its own epic,
and a later increment's work waits on the earlier epic closing, so `/codefall-implement` on the
second epic has nothing to build until the first is merged. The run ends by naming implement on
the first plan.

## How tasks get test cases

Plan decides which tasks need a *test case*: a Markdown file describing how to check a behavior
through the running product, which [`/codefall-test`](test.md) runs. A task gets one when it is
verified through the wired product, meaning the real interface running against the real services.
Plan writes the case into the bead's acceptance criteria:

```
Case: trips/export-itinerary · modalities: spec
1. Selecting export produces a file containing the itinerary.
   (SPEC-003-REQ-01-AC-01)
2. Exporting a trip with no segments produces a file that says the trip is empty.
   (derived from SPEC-003-REQ-01 — the empty case; the spec names only a trip with segments.)
```

The criteria name three things:

- **The case**, as `<area>/<slug>`. This is also the case file's path under the testing root's
  `test-cases/` directory.
- **The modalities.** A *modality* is a way of running the case. A `spec` case becomes a generated
  test that the project's own test runner executes. An `agentic` case is worked through step by step
  by the agent, which judges each criterion against what it observes. Plan chooses `agentic` only
  where checking the outcome needs judgement, `spec` otherwise, and neither when unit tests already
  verify the task.
- **Every criterion the case will hold.** A criterion that repeats a spec criterion cites it in
  full. A criterion that goes further is marked `derived`, names the requirement it elaborates, and
  says in one line what it adds.

You sign off the derived criteria when you approve the task table, which is why the run shows them
with it. Sometimes a derived criterion exposes a gap in the spec. Plan then offers it as a new
criterion appended to the spec. If you accept, the spec is edited on the plan's own branch, its
GitHub issue, when the project has one, is regenerated, and the bead cites the new identifier. If you decline, the criterion
stays `derived`.

## When does a plan write an ADR?

An *ADR* (architecture decision record) is a short document that records one hard-to-reverse
decision and the reasons for it. Plan writes one for a new dependency, a schema other components
will build on, or a rejected alternative that took real analysis. Most plans need none, and the
decision is separate from the tier.

The ADR takes the next number in the project's own `ADR-NNN` sequence and is shown to you before it
is written. An ADR is never rewritten once it is ratified. A later change of mind lands as a new ADR
that supersedes it.

## What happens when the run cannot settle a question?

Plan raises a concern once and lets you decide. When a technical point is still open after the
concern has been raised and the code and the ADRs have been read, plan asks the project's *consult*
agents about it, once. The consult agents are the second-opinion readers listed under `consult` in
`.codefall/settings.json`; with nothing configured, the consult is a fresh subagent of your current
coding agent. Plan sends the question, the relevant files, and the options it sees, and the answer
is handled this way:

| The answer | What happens |
| --- | --- |
| Confident, on a choice that is easy to reverse | Plan takes it as its own decision and names who was consulted |
| A hard-to-reverse choice the project's own rules give a default for | Plan takes the default and writes the ADR from it |
| Anything else | The analysis comes to you beside the concern, and you decide |
| No answer | The question goes into the document as a stated risk, with the options and their costs |

A consult never writes anything. Plan never consults on a preference you have already stated.
`.codefall/shared/decisions.md` holds the rule for what a run decides on its own.

## What do the statuses mean?

A plan's `Status` describes the document and never the work:

| Status | Meaning |
| --- | --- |
| `Draft` | Still being written, or it still has decisions waiting for an engineer. A `Draft` plan has no beads |
| `Ready` | Written and agreed, and its beads exist |
| `Archived` | Superseded or dropped. The file moves to `docs/plans/archive/`, where its identifier still resolves |

Whether the work is queued, underway, or done is for Beads to say. `/codefall-specify` divides a
spec's status from its issue tracker the same way.

## How a plan changes after its beads exist

Run `/codefall-plan` on an existing plan to revise it. The run reconciles the task graph with the
edited table instead of rebuilding it, and what happens to each bead depends on whether anyone has
acted on it:

- **A bead nobody has touched** is edited, whatever changed.
- **A bead someone has claimed, commented on, or closed** is replaced only when the work already
  done against the old wording would no longer count. A ticket should not change under the person
  holding it.
- **A task that leaves the plan** is reported to you and never closed automatically, because
  someone may still be working on it.

A plan that turns out to be too big can also be split. Ask for it on the plan, and the run moves
the rows you choose into a new plan with its own epic, numbered next, that follows this one. Only
rows nobody has started can move; a task someone has claimed, commented on, or closed stays where
it is, and the run cuts around it or tells you the split cannot be made.

A revision can also be requested from downstream as a `plan-revision` bead. When
[`/codefall-implement`](implement.md), `/codefall-review`, or `/codefall-test` finds that the code
disagrees with the plan, it amends the plan's text itself, in its own pull request. It files a
`plan-revision` bead instead when the fix would change a task row or a criterion a bead cites, or
when the document is frozen. The verb that filed it then runs `/codefall-plan`'s Revise mode
itself, in the same run or at the next round boundary, and settles each bead: the text amended, a
task added, or the request rejected, with a reason. A question only you can answer becomes a
*human bead*, a question for you in Beads, with the default the run took.
