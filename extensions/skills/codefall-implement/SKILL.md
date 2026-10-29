---
name: codefall-implement
description: Execute the work the plan verb put into the graph — claim ready beads, build each task in its own worktree with tests as part of done, write the test case a bead's criteria name before the code, verify against the bead's acceptance criteria and the project's own checks, open pull requests, and walk the dependency graph in parallel waves until the frontier is empty. On an epic it then runs review and test, triages what they found, and runs the next round on its own until the delivery is done. Never merges to main, and never sets a test harness up.
argument-hint: "[a bead, an epic, a plan, or nothing to pick from ready work]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - Agent
  - Skill
---

# Implement

Walk the graph `codefall-plan` created. Claim what is ready, build it, verify it, open a pull
request, and let each close unblock the next task until the frontier is empty; then review it, test
it, and go again.

Implementing is not merging. A run ends at open pull requests and a link to the stack's top —
**a human performs every merge to `main`, and this skill never does**, in any mode, under any
instruction short of the user editing this file. Merges into an epic branch are the one exception;
the human gate sits at its aggregate PR.

Paths starting `reference/` or `../` are relative to this skill's directory, not the user's
project; one through `../../../.codefall/` names a file `codefall init` installed in the project's
own `.codefall/`.

## A delivery runs on its own

On an epic, one gate starts the run and nothing after it asks. The person may have left. Every
question the run would ask is answered as `../../../.codefall/shared/decisions.md` says: decided
with its reason when it is the run's; consulted once, then recorded as a human bead with the best
default taken, when it is not. When the frontier is empty the run runs `codefall-review` on the
epic's work, then `codefall-test`, then the round boundary in `reference/rounds.md`, and goes
again. Three things stop it: the delivery is done, nothing in the batch can be built, or the round
number is at the ceiling. Everything else is a signal, weighed with the default to continue. A run
that stops names `/codefall-implement <epic>` to resume, and that gate shows why it stopped. A
person who wants one round at a time says so at the gate.

## Files beside this one

Read each when its step says to.

- `reference/landing.md` — the two landing strategies, integration, later rounds, the diagrams.
  Read at steps 4 and 7.
- `reference/rounds.md` — the round boundary: revision beads, triage under the epic, the batch,
  the exits and signals, the ceiling, the ledger lines. Read at step 5 on a later round and at
  step 8.
- `reference/resume.md` — picking up an interrupted run. Read at step 2 for an epic with closed
  children.
- `reference/beads.md` — every `bd` command a run issues. Read at step 5.
- `../../../.codefall/shared/decisions.md` — what a run decides, consults on, or records as a
  human bead. Read at step 1.
- `../../../.codefall/shared/triage.md` — where a discovered item goes. Read at step 6.
- `../../../.codefall/shared/delivery.sh <epic>` — the chart line. Run at the gate, each wave
  boundary, and the report.
- `../../../.codefall/shared/stacks.md` — GitHub stacks, and how a report points at one. Read at
  steps 4, 7, and 9.
- `reference/workers.md` — launching a worker here or in another harness, effort, chains,
  results, failures, consulting. Read at steps 5 and 6. Names
  `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`,
  `../../../.codefall/shared/run-worker.sh`, `../../../.codefall/shared/consult-prompt.md`,
  `../../../.codefall/shared/consult.schema.json`.
- `reference/done.md` — when a bead is done, and the test case written before the code. Read at
  steps 3 and 6. Names `../codefall-test/reference/case-file.md` and
  `../../../.codefall/shared/check-cases.sh`.
- `reference/mirror.md` — the spec's tracker issue. Read at steps 6 and 9.
- `worker-prompt.md` — the prompt rendered for each worker.

## Scope — build, not decide

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Executing tasks from the graph | What the tasks are, or their edges | `codefall-plan` |
| Branches, worktrees, commits, PRs | Merging anything to `main` | the user |
| Every test the current work needs | Regression campaigns and fresh-context retesting | `codefall-test` |
| Writing the case file the criteria name | Deciding which tasks need a case; installing a runner | `codefall-plan`, `codefall-equip` |
| Harness checks on its own diffs | Independent review and verdicts | `review` |
| Bead lifecycle: claim, close, discovered work filed | Creating or re-cutting the task graph | `codefall-plan` |
| The vision's `Active` transition | Any other document transition | the owning verb |
| Mirroring work state to the spec's tracker issue | The mirror's lifecycle and labels | `codefall-specify` |

**A task that turns out to be wrong is reported, not re-planned.** When the plan's cut does not
survive contact with the code, say what you found and hand the graph back to `codefall-plan`. A
smaller disagreement the task can finish under — the plan's text against the code, or the
spec's — is amended by the worker in its own branch when the document is `Draft` or `Ready` and
the fix is text that moves no task row and no cited criterion; the PR body names it, and the root
re-mirrors a spec change. A fix that would move work is filed as a `plan-revision` bead per
`reference/beads.md`, and this run runs `codefall-plan`'s Revise mode on the document at the next
round boundary.

- **Implement writes every test the current work needs**, named in the plan or discovered
  mid-task; a missing test is written, not sent back. **`codefall-test` owns the run against the
  epic's work**: the cases the beads named, on the epic's branch before the merge.

## One bead or the graph

The argument fixes the scope; the skill never infers it.

| Invocation | Scope |
| --- | --- |
| `/implement booking-parser-trailing-comma` | That bead, alone |
| `/implement booking-PLAN-007` (an epic) | The epic's graph, then review, test, and the rounds after, until done or a stop |
| `/implement PLAN-007` | The plan's epic, `<prefix>-PLAN-007`, checked to exist with `bd show` |
| `/implement` | Show ready work grouped by epic and ask |

A plan whose Tasks callout still says `Staged. Not yet in Beads` has no graph to walk: refuse and
name `/plan`. An epic plain `bd ready` omits waits on another epic: say which and stop.

**An epic with closed children and `deferred` ones is the next round.** A run on it triages those
children at the round boundary per `reference/rounds.md`, picks the batch, and builds it on top of
the round before. The delivery is done when every child but the landed bead and any human bead is
closed and the last test run against the epic passed.

**Tier 0 is not a separate mode.**

## What gets read

Per bead, before any plan is formed: the bead, `bd show <id> --json`, for the body, the Plan ref,
the acceptance criteria, and `spec_id`, the plan's path; the plan's Overview and Architecture, the
Plan ref's section, Hard Constraints, and Technical Context and Testing Strategy when present; the
spec one hop up the plan's `Related` line, whose criteria are the externally observable contract,
or the vision when there is no spec; the ADRs on that line plus `docs/adrs/`, never rewritten, a
conflict with a task going to `codefall-plan`'s Revise mode as a superseding-ADR question; the
project's `AGENTS.md`, root and scoped; and any mockup referenced, a drawing to rebuild in the
app's stack, never markup to copy. A tier-0 bead has no document: bead, `AGENTS.md`, ADRs. A
followup epic, `<prefix>-FOLLOWUP-<area>-NNN`, holds such beads.

## The execution plan

Build the waves first, from the graph: `bd ready --mol <epic> --explain` and `bd dep tree`, every
bead in the earliest wave its blockers allow. Their shape picks the landing, per
`reference/landing.md`: every wave one bead wide is a chain and lands as a **serial stack**, one
branch atop the previous, workers one at a time; any wave wider than one lands on an **epic
branch**, `epic/<id>-<slug>`, waves off it, one aggregate PR. Work that must not land on `main` in
increments, a standing preference in `CUSTOMIZE.md`, and the strategy round one used each force
the epic branch. **There is no cap on a wave's width**; the graph sizes it.

## The go gate

One gate, before any work starts; on an epic, the only question of the delivery. Everything the run
will do, in one block:

- the delivery's chart line; on a later round the round number this run will set, the batch per
  `reference/rounds.md`, and what the boundary moved out or closed;
- the execution plan: the waves and their widths, the branch diagram, the strategy and its
  one-line reason;
- per bead, the agent and effort proposed from the `work` list, in the table
  `reference/workers.md` lays out, which the person may edit;
- what will be claimed, and that go flips a vision behind the work to `Active`;
- the consult order a question or a failure will be put to, resolved per `reference/workers.md`;
- the ceiling on rounds, and the root session's own effort;
- the permissions condition: nobody answers a prompt, so the session must allow edits and Bash
  without asking, or the run offers single-task mode instead;
- that the run continues through review, test, and the rounds after until the delivery is done,
  and the one thing to say if it should stop after this round instead.

Then wait for go; under the `product-manager` persona the gate is the sentence
`../../../.codefall/shared/personas.md` gives.

**Single-bead scope shrinks the gate to an approach approval**: the files to touch, the approach,
the test plan, one branch; the run ends at its pull request.

**After go, nothing asks.** A question is decided, consulted on, or recorded per `decisions.md`; a
failure is consulted on and retried once, then the bead is skipped and the run goes on. Agent and
effort are the root's, proposed at the gate; what ran goes in a `bd comment` beside the PR link and
in the bead's `agent` and `effort` metadata.

## Verification and done

A bead is done when three things are true: **its acceptance criteria hold, the project's checks are
green, and its PR is open.** Done is not merged. Where the commands come from, what is checked, and
the test case a bead's criteria name are in `reference/done.md`. Done closes the task bead.

## Merges and the mirror

**The root never merges to `main`.** The extension ships a `PreToolUse` hook that denies it; a
denial from that hook is the system working as designed. What the root does merge: worker PRs into
the **epic branch**, one at a time, at wave boundaries. At close-out the run points at the stack
per *Pointing at a stack* in `../../../.codefall/shared/stacks.md` — the link to the top PR, never
a command or a bottom-up list. The spec's tracker issue walks the work's state per
`reference/mirror.md`. Every PR body carries `Relates to #<spec-issue>`.

## The vision transition

`codefall-envision` reserves one transition for this skill: `Status: Active — <date>`. At the
run's first claim, resolve the vision — the plan's `Related` line to the spec, the spec's
`**Vision:**` row to the vision, or the plan's `vision` label when there is no spec — and flip its
Status line, once, and report it: "VISION-012 → Active." Already `Active`, or no vision, is nothing
to do. Only the Status line is touched, ever.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Check preconditions

Run the shared check against the user's project.

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances. Otherwise read `beads_reason`, hand over the remedy, say to rerun this verb
after it, and stop: `not_installed` is `brew install beads`, `not_initialized` is `bd init`,
`unreadable` quotes `beads_detail`.

Then read the checkout lines. `behind` above `0` or `refresh=stale`: say so and run
`/codefall-refresh` before continuing. `refresh=undeclared` names `/codefall-equip` instead.
**Never run the other remedies.** They are the user's.

**Then read the `test=` line**, and hold it against the beads in scope once step 3 has read them. A
bead whose acceptance criteria name a test case needs an equipped harness: `equipped` proceeds;
`unequipped` names `/codefall-equip` and does not start the bead; `undeclared` names
`codefall upgrade` and does not start it; `unknown` reads the `test` block from
`.codefall/settings.json`, no runner there being `unequipped`. Beads verified by unit tests alone
proceed either way. **Never set the harness up** — that is `/codefall-equip`'s own pull request.

Then read the project's `AGENTS.md` (root and scoped), `../../../.codefall/shared/decisions.md`,
and `.codefall/skills/codefall-implement/CUSTOMIZE.md` — constraints, verify commands, a standing
strategy preference, a ceiling.

### 2. Fix the scope

Per [One bead or the graph](#one-bead-or-the-graph). With no argument, run the session-start
commands in `reference/beads.md` — `bd ready` unfiltered, since no epic is chosen yet — then show
the ready set grouped by epic and ask. Never infer a batch from an unprompted ready set. An epic
that already has closed children is read per `reference/resume.md`: a crashed round is reconciled,
`deferred` children wait for the round boundary.

### 3. Read

The full list in [What gets read](#what-gets-read), for every bead in scope. Reconcile the plan
against the code as it stands on the base branch — cite file and line for anything the plan
assumed that has moved — and default to preserving whatever the plan is silent about.

### 4. Build the execution plan

Read `reference/landing.md`. Build the waves and pick the strategy per
[The execution plan](#the-execution-plan), constrained by `AGENTS.md` and `CUSTOMIZE.md`; on a
later round, the strategy round one used.

### 5. The go gate

On a later round, first run the round boundary, steps 1 to 4 of `reference/rounds.md`, so the
gate shows the batch. Present the block per [The go gate](#the-go-gate), the chart line first, and
wait. On go, per `reference/beads.md`: flip the vision to `Active` if one is behind the work. Epic
scope, first round: create the epic branch if the strategy calls for one (`epic/<id>-<slug>` off
`main`, pushed), create the landed bead, claim the epic and the first wave, `bd dolt push`. Epic
scope, later round: the epic branch, the landed bead, and the epic's claim already exist and are
left alone; `bd update <epic> --set-metadata round=<N+1>`, claim the first wave of the batch,
`bd dolt push`. Single-bead scope: claim the bead, `bd dolt push`, nothing else.

A strategy overruled twice the same way is offered for `CUSTOMIZE.md`, never written unasked.

### 6. Execute the waves

Per `reference/workers.md` throughout. Per wave: render worker prompts, launch the batch at the
effort the table set, wait for results. Verify each success — branch on the remote, PR exists, or
it did not happen. A failed bead is consulted on and retried once; a second failure is consulted
on, and the bead is unclaimed with a note and skipped. Check each `plan` discovery, sending back
what the worker could have amended; give every other discovered item, a bug or not, the
disposition `../../../.codefall/shared/triage.md` gives it, and file it in the form its `kind`
names; record each `amended` entry as a closed `plan-amended` bead, per `reference/beads.md`.
Comment the PR link, close the bead with what was verified, gate the landed bead with the new PR
(stacked runs), `bd dolt push`. `--suggest-next` names the next wave; print the chart line, claim
the wave, and go again. Epic branch: merge each worker PR into the epic branch, serialized, at the
wave boundary. Update the mirror per `reference/mirror.md`.

Single-bead scope is one iteration of the same loop, in one worktree.

### 7. Integrate

When the frontier is empty: ask GitHub whether every open layer is mergeable, per *Integration* in
`reference/landing.md`. A layer that is not is fixed on its own branch and cascaded with
`gh stack`. Epic branch, first round: open the aggregate PR to `main`, titled as a release-worthy
conventional commit, and gate the landed bead with it — `Closes` nothing; the gate owns the epic's
close. A later round reuses the aggregate PR that exists.

### 8. Review, test, and the next round

Epic scope only. Run the `codefall-review` skill on the epic through the `Skill` tool, in this
session, then `codefall-test` the same way; each files what it finds and builds nothing. Then the
round boundary, per `reference/rounds.md`: revision beads to `codefall-plan`'s Revise mode, triage
of everything under the epic, the batch, the exits and signals, the boundary line. On continue:
the later-round go of step 5 without the gate, then step 6 again. On stop, or when the person
asked for one round at the gate: step 9.

### 9. Report and stop

Do not merge, and do not wait for merges; the next session's `bd gate check` finishes it.

- The chart line.
- Every bead built, with PR, branch, and what its close reason verified; every bead skipped and why.
- Upstream amendments, by document and PR, each with its `plan-amended` bead; every revision bead
  Revise mode settled; discovered work by the disposition each got.
- Every decision taken and every human bead filed, per `decisions.md`, by bead; every consult.
- The tracker mirror's state, the vision transition if one fired, the worktree list with the
  cleanup offer.
- The epic's notes lines per `reference/rounds.md`; that it is safe to `/clear`; a final
  `bd dolt push`.
- **Last, what the user does next**: when the delivery is done, the link to the pull request the
  person merges, per *Pointing at a stack*, and the human beads to answer; otherwise why the run
  stopped and `/codefall-implement <epic>` to resume. A single bead ends at its pull request.

## Other modes

- **Resume** an interrupted run — per `reference/resume.md`, which reconciles beads, git, and
  GitHub and tells a crashed round from the next one. Then continue the normal loop.
- **Abandon** a run: unclaim what is claimed and unbuilt, note why on each bead, report branches
  and PRs left standing; delete nothing.
- **Drain assistance is reporting only.** After merges, `bd gate check` and the mirror update;
  never a merge.

## Rules

- **A person merges every code PR to `main`; this skill performs none, in any mode.**
- **After the one gate, nothing asks.** A question is decided, consulted on, or recorded as a
  human bead, per `decisions.md`, and the run goes on.
- **A stop is one of the three exits, or a judgment written down with its reason.** When the
  signals are mixed, the round runs.
- **Every `bd` write is the root's, in the primary checkout.** Workers never run `bd`.
- **Closed means done — criteria verified, checks green, PR open.** Merged is the gates' to say.
- **Publish the claim before the work.** `bd dolt push` follows every claim and every close.
- **The graph is the sequencer.** `bd ready` decides what runs next; never start a blocked bead.
- **Scope is exactly the bead.** Tangents are triaged by the root, never fixed in passing; a plan
  or spec found wrong is amended or becomes a `plan-revision` bead, never a quiet workaround.
- **Nothing is rebased or force-pushed by hand.** `gh stack` cascades; GitHub rebases on merge.
- **Bead IDs ride every commit message.**
- **Verify workers, never trust them.**
- **One consult, one automatic retry, one more consult, then skip and go on.** The root consults;
  a worker never does.
- **Tests are part of done.** Planned or discovered, written now, never deferred to `test`.
- **A test case the criteria name is written first, from those criteria alone**, before the code
  and its spec.
- **A test harness is never set up inside a task's pull request.** An unequipped bead is not
  started; `/codefall-equip` is named.
- **The local scripts are part of done**, changed in the same PR.
- **Implement never re-cuts the graph**, and writes no bead metadata but the epic's `round`.
- **Never overwrite a file that has drifted.** Show the difference and ask.
