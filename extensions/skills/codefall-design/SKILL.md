---
name: codefall-design
description: Decide how a feature gets built and put the work into the graph — read the docs and the code, judge whether the change warrants a design document at all, write one scaled to the work at docs/designs/, record hard-to-reverse choices as ADRs, decide which tasks are verified through the wired product and draft their test case and its criteria into the bead, and create the task graph in Beads from the document's staged task plan.
argument-hint: "[the spec, the bug report, the vision, or what you want built]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - Skill
  - WebSearch
  - WebFetch
---

# Design

Decide how a feature gets built, and put the work into the dependency graph so `codefall-implement`
can pick it up.

Two outputs. A **design document** at `docs/designs/DESIGN-NNN-slug.md` — the approach, the
architecture, and a staged task plan — when the work earns one. **The tasks in Beads**, with their
dependency edges, always.

Designing is not implementing. Once the document is written and the graph exists, stop.

Paths that start with `../` are relative to this skill's directory, not the user's project. A path
through `../../../.codefall/` is the one that leaves the skills directory: it names a file `codefall
init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/lifecycle.md` — the three statuses, where each lives, and what each allows. Read at
  step 7 and for the other modes.
- `reference/document.md` — the design document's shape: header, sections and their triggers, the
  Technical Context and Hard Constraints blocks, the Task Plan and its callout. Read before step 5.
- `reference/adrs.md` — when a design writes an ADR, how it is numbered, and the template. Read at
  step 4 and step 7.
- `reference/beads.md` — what gets created in Beads, the test case a bead's criteria name, the plan
  file, the edge direction, and how to verify the graph. Read before step 9.
- `reference/bugs.md` — a fix for a bug: the report, reproducing it, the cause, the tier. Read at
  step 2 for a bug.
- `reference/revising.md` — reconciling the graph when a design changes after its beads exist, and
  settling the revision beads filed against it. Read for the Revise mode.
- `reference/consulting.md` — putting an unsettled technical point to the configured agents, and
  what their answer may do. Read at step 5 when one stays unsettled. Names
  `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`,
  `../../../.codefall/shared/consult-prompt.md`, `../../../.codefall/shared/consult.schema.json`.
- `templates/designs/AGENTS.md` — the operative rules this skill installs at `docs/designs/AGENTS.md`.

## Scope — how, not what and not whether

| In scope | Out of scope | Whose |
| --- | --- | --- |
| The approach, and the decisions inside it | Whether this is worth building | `codefall-envision` |
| Components, their relationships, and data flow | What a consumer observes when it works | `codefall-specify` |
| Contracts, types, schemas, storage, failure modes | What the screen looks like | `codefall-mock-up` |
| Which existing code changes and which is new | The project's architecture stance | `codefall-scaffold` |
| The task breakdown and its dependency edges | Writing the code | `codefall-implement` |
| Which tasks need a test case, and its criteria | Writing the case file, and running it | `codefall-implement`, `codefall-test` |
| Hard-to-reverse choices, recorded as ADRs | Estimates and assignment | the team |

**The project's stance is already decided** — layering, component boundaries, and how they are
enforced — in `docs/adrs/` and the scoped `AGENTS.md` files. Design within it. A design that needs
the stance changed says so once, then either follows the ADR or writes a superseding one.

## Upstream documents

A spec or vision this run finds wrong or incomplete is amended here, on this run's branch, when it
is `Draft` or `Ready`, the amendment is text that moves no work, and the user takes it at step 7. A
spec is amended by appending, and its requirement issue is regenerated per
`../codefall-specify/trackers/<name>/PROFILE.md`. An `Active` vision is frozen: say so and name
`codefall-envision`. The rule is in `../../../.codefall/shared/workflow.md`.

## Scale the artifact to the work

A small required core, everything else conditional, and **no empty or placeholder sections — omit
them.**

Three tiers. The work picks the tier; the user can overrule it.

| Tier | When | Output |
| --- | --- | --- |
| **0 — no document** | All three hold: the change is contained to one component, no public interface changes, and the task graph is one or two beads | Beads only |
| **1 — minimal document** | Anything that crosses a component boundary, or fans out past roughly three dependent tasks | The three required sections |
| **2 — full document** | The same, plus any conditional section whose trigger fires | Required three plus what triggered |

- **Tier 1 and 2 are not decided separately**: write the required three, then walk the triggers.
- **An ADR is not gated on the tier.**
- **Say the tier out loud before writing** — step 4.

## The document

`docs/designs/DESIGN-NNN-slug.md`, git-tracked. Three digits, zero-padded, the highest existing
number plus one. **A design is never renumbered and its identifier is never reused**, including
after it is archived — the file moves, the identifier does not.

Three required sections — **Overview**, **Architecture**, **Task Plan** — and seven conditional ones,
each with a trigger. The header, the section tables, the Technical Context and Hard Constraints
blocks, and the Task Plan's callout are in `reference/document.md`.

## ADRs

A separate artifact, `docs/adrs/ADR-NNN-title.md`, for a choice that is hard to reverse or that
other components will build on; most designs need none. The number continues the project's own
sequence, a ratified ADR is never rewritten, and the ADR is shown before it is written. The
trigger, the numbering, and the template are in `reference/adrs.md`.

## Status and lifecycle

`Draft`, `Ready`, or `Archived`, one word plus a date, describing the document and never the work.
The table and the rules are in `reference/lifecycle.md`.

## The designs directory

`docs/designs/AGENTS.md` is written from `templates/designs/AGENTS.md` when the directory is created,
and added on a later run if it is missing.

## Beads

One epic, `<prefix>-DESIGN-NNN`, and one task bead per row, `<prefix>-DESIGN-NNN-Tn`; tier 0 has no
epic. `reference/beads.md` has the rest.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report; when it is not `engineer`, follow that persona's section in
`../../../.codefall/shared/personas.md` for this run, and say so.

## Process

### 1. Check preconditions

Run the shared check against the user's project.

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances to step 2. Otherwise read `beads_reason`, tell the user what is missing, hand
over the command that fixes it, say to rerun this verb after it, and **stop**:

| `beads_reason` | What is wrong | Give them |
| --- | --- | --- |
| `not_installed` | `bd` is not on PATH | `brew install beads` |
| `not_initialized` | this repository has no beads database | `bd init` |
| `unreadable` | bd found a database and could not read it | quote `beads_detail` |

Then read the checkout lines. `behind` above `0` or `refresh=stale` means the environment may not
match `main`: say so and run `/codefall-refresh` before continuing. `refresh=undeclared` names
`/codefall-equip` instead.

**Never run the remedy.** That is the user's decision.

### 2. Take the input

One open question, unless the invocation already answered it:

> "What are we designing? A spec or bug report identifier, a vision, or just tell me what needs
> building."

**Then look for a spec.** Read `docs/specs/` — not `archive/` — and offer the relevant one:

> SPEC-004 covers booking history and looks like what you are describing. Design against it?

A spec is not required. If there is none and the work is more than a fix, say once that the design
has no written target and offer `/specify`. On no, continue.

**A bug report is a target the way a spec is**: read `reference/bugs.md` for one.

**A spec that is not ready is finished here, not refused.** Two cases:

- The spec, or the bug report, says `Status: Draft`: run `codefall-specify` on it in this run, on
  its branch, to settle what is open and promote it. Stop only on a question the person cannot
  settle.
- Its tracker issues carry `requires-mockup` (read per `../codefall-specify/trackers/<name>/PROFILE.md`;
  an unreachable tracker is asked about, never guessed): run `codefall-mock-up` for each such
  requirement in this run, on this run's branch, before designing.

**Then read what frames it.** The spec's vision, if it names one. `docs/visions/` if no spec
framed the work. A vision's **Environment & constraints** section is written for this moment.

### 3. Read the docs and the code

- **The project's stance** — `docs/adrs/`, every `AGENTS.md` in the tree, and
  `docs/decision-log.md`. Design within them.
- **The existing designs** — `docs/designs/`, not `archive/`. If one already covers this, say so
  and link it; the user may want to revise that one.
- **The code** — the components this touches, their facades, and what already exists that this can
  use.
- **The graph** — `bd dolt pull`, then `bd list` and `bd search` for existing work. A task
  this design would create that is already a bead is a dependency edge, not a new task.
- **Revision requests**, on an existing design — `bd list -l design-revision --spec <its path>`,
  filed by `codefall-implement` or `codefall-review`.
- **The cause**, for a bug, per `reference/bugs.md`. It decides the tier.

Report what you found before designing — open revision requests first. If the work already exists,
say so and stop.

### 4. Decide the tier, and whether there is an ADR

Both judgements, stated together, before any writing:

> Four tasks across the context store and the scaffold command: a design document. The signed
> payload versus session lookup choice is hard to reverse: an ADR as well.

Walk the [tier table](#scale-the-artifact-to-the-work) and the [ADR trigger](#adrs) explicitly. They
are independent.

**At tier 0, skip to step 7.**

### 5. Design it

Read `reference/document.md`. Settle, in this order, and only what applies:

1. **The approach** — how this gets built, in a paragraph, and the decisions inside it.
2. **The components** — what changes, what is new, what talks to what.
3. **The contracts** — the types, APIs, and messages that cross a boundary.
4. **What persists** — the data model, and any migration it implies.
5. **What fails** — the new failure modes, and what the system does about each.
6. **What is left out** — the approaches considered and set aside.

**Research inline as you go**, attached to the decision it bears on.

**"Like $LIBRARY does it."** Offer once to look it up; on yes, summarize only what changes a decision
here, and confirm the summary before it reaches the document.

**Raise a concern once, then defer or park.** Name it, say why, and let them decide; cap at two
rounds. A technical point still unsettled after research is consulted on once, per
`reference/consulting.md`. What stays unsettled goes into the document as a stated risk, or, when
the persona is `product-manager` or the person says they cannot decide it, into **Decisions needed**
per `reference/document.md`: never decided by the run, never taken from silence.

**Do not bikeshed** over naming or two equivalent shapes.

### 6. Stage the tasks

**Tasks decompose by what can be built and verified on its own**, against the design — a different
cut from `codefall-specify`'s, where requirements decompose by what a consumer observes, and one
requirement routinely becomes several tasks.

The sizing test: could one person pick the task up, finish it, and have something that either works
or does not? A task nobody can tell is done is too big or too vague.

**Wire the edges, and check the ordering by reading it backwards**: for each task, what must exist
before it can start? A task with no answer is a root. A cycle is an error in the cut — fix the cut,
not the edges.

**A task that introduces a tool carries the script change as a criterion.** A compose file, a
migrations directory, a lockfile, a codegen config, or an `.env.example` among its predicted files
means one criterion says the declared `start` and `update` scripts were changed for it, per
`codefall-equip`; a project with no `local` block gets "equip the project" on the first such task.

**A task verified through the wired product carries a test case.** Its criteria name the case
(`<area>/<slug>`), its modalities, and every criterion the case will hold, each cited in full or
marked `derived`, per `reference/beads.md`. `agentic` only where verifying an outcome needs
judgement; `spec` otherwise; neither where unit tests verify the task. A gap the criteria expose in
the spec is an appended criterion, offered at step 7 per [Upstream documents](#upstream-documents);
`derived` is what it stays when the user declines. The case-file format is
`../codefall-test/reference/case-file.md`.

Write the staging table.

### 7. Draft and confirm

Compose the full document and **show it before anything is written**. Nothing lands unapproved.

Say which conditional sections you left out and why — "no Data Models section, because nothing here
persists" — so the user can catch an omission that was a gap.

**Show the acceptance criteria of every task that carries a test case**, derived criteria included,
and every spec or vision amendment. Approving the plan is the sign-off those need; nothing later
asks for it.

Show the ADR too, if there is one; it ships `Accepted`.

Then set the status: `Ready`, unless they are stopping and coming back, which is `Draft`.

**When Decisions needed is not empty**, ask one question before setting the status, under any
persona:

> I set aside N technical decisions. Settle them with sensible defaults now so building can start,
> or leave them for an engineer?

On **settle now**: pick the default for each — the simplest choice that fits the project's stance
and what it already uses — say it in one plain sentence, move it into its section, write the ADR
where one needs it per `reference/adrs.md`, and the status is `Ready`. On **leave them**: `Draft`,
no beads, steps 9 and 10 skipped, and the pull request stays a draft.

**At tier 0, this is the confirmation instead**: the beads you would create, their titles, their
bodies, and their edges. The user approves the graph, not a document.

### 8. Branch, then write the document

Branch first, per `../../../.codefall/shared/landing.md` — `design/DESIGN-NNN-slug`, from the
default branch — unless this is a tier 0 run with no ADR, which writes no file.

Write `docs/designs/DESIGN-NNN-slug.md` with the Task Plan **staged**, the ADR if there is one,
`docs/designs/AGENTS.md` if it was missing, and the amendments the user took: a spec's mirrored, a
vision's committed beside the design.

Write the document before the beads, so a failed creation is resumable.

### 9. Create the graph

Read `reference/beads.md`. Build the plan file from the table, dry-run it, create it, rename to the
IDs, set `--spec-id` and `--acceptance`, verify with `bd ready` and `bd dep cycles`, and push.

If the ready set does not match the table's roots, fix the edges now, before the callout is
rewritten.

### 10. Mark the Task Plan created

Rewrite the callout above the table per `reference/document.md`: the date, the epic's full ID, the
per-row form, and that Beads is authoritative. **The table stays.**

At tier 0 there is no document to mark.

### 11. Link back, commit, and report

Fill in the `plan:` field on the framing vision's `Related` line with this design's identifier.
Where a spec framed the work, the vision is the one named in the spec's `**Vision:**` row. Add the
identifier and change nothing else in the file.

There is no back-link to write into the spec: the design's `spec` label carries the connection.

Then land it per `../../../.codefall/shared/landing.md`, with `.beads/interactions.jsonl` when it
changed: push, open the pull request as a draft, and mark it ready for review when the status is
`Ready`. The Action merges it; no person is asked to.

Report:

- the design's path, identifier, and status, or that this was tier 0 and why;
- any ADR written, and what it decided;
- every bead created, with its local ID, its title, and any test case its criteria name;
- every upstream amendment written, and any the user declined;
- every consult: the point, who answered, what it changed;
- the ready set — which tasks `codefall-implement` can start on today;
- anything left unresolved, the decisions set aside and how many, whether the person had them
  settled now or left for an engineer, and any concern the user overruled;
- the branch and the pull request, with its state worded as the landing procedure says: merged,
  ready and being merged by the Action, or ready and waiting for a person because the project has no
  Action; a `Draft` design keeps a draft pull request;
- **last, one command**: `/codefall-implement DESIGN-NNN`, or `/codefall-implement <bead>` at
  tier 0; with decisions left for an engineer, `/codefall-design DESIGN-NNN`.

## Other modes

Invoking this skill on an existing design does one of four things. Ask which if it is not obvious.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft` when no beads exist yet.
  Settling every item under **Decisions needed** empties the section first; steps 9 and 10 then
  create the graph.
- **Revise** a design and reconcile its graph, per `reference/revising.md`, which also settles
  every open `design-revision` bead. A design the code has moved past is revised, not labelled.
- **Archive** a design: set `Status: Archived`, add `**Replaced by:**` if something took its place,
  move the file to `docs/designs/archive/`, and report its open beads to the user rather than
  closing them.
- **Add tasks** to an existing design — new rows appended to the table, each with its bead, and
  the callout dated. Retired local IDs stay retired.

Every one of these is the user's decision. Report the state and offer; never transition a design on
your own initiative.

## Rules

- **Nothing is written without the user confirming it first** — the document, the ADR, and at tier 0
  the beads.
- **Scale the artifact to the work.** Tier 0 is a real outcome, not a failure to write a document.
- **Never fill a heading.** An empty conditional section is deleted.
- **A design still carrying Decisions needed is `Draft`** and has no beads. The one question at
  step 7 is how they are settled in this run; silence never settles them.
- **Design within the project's ADRs.** Changing the stance is a superseding ADR, said out loud; a
  ratified ADR is never rewritten.
- **Identifiers are append-only** — design numbers, and local task IDs within a design.
- **Beads is authoritative for work state once the tasks exist.** The Task Plan never grows a
  status column.
- **Every task bead carries acceptance criteria**, checkable, citing spec IDs and the test case
  where one applies.
- **A task that introduces infrastructure, a dependency, a migration, or generated code names the
  local-script change in its criteria.**
- **Verify the graph before marking the table created.** `bd ready` and `bd dep cycles`, against
  its roots.
- **Every bead write is pushed**: step 9, and every revision.
- **A removed task's bead is reported, never closed silently.**
- **A revision bead ends closed** — amended, made a task row, or rejected with why.
- **An upstream document found wrong is amended here**, within [Upstream documents](#upstream-documents).
- **A ticket must not change under someone holding it.** An untouched bead is edited; a held one is
  replaced when the work done would no longer count.
- **Push back once, then defer** — on the tier, on the approach, on the cut. A consult informs,
  never decides.
- **Never overwrite a file that has drifted.** Show the difference and ask.
