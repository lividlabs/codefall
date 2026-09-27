---
name: codefall-implement
description: Execute the work design put into the graph — claim ready beads, build each task in its own worktree with tests as part of done, write the test case a bead's criteria name before the code, verify against the bead's acceptance criteria and the project's own checks, open pull requests, and walk the dependency graph in parallel waves until the frontier is empty. Never merges to main, and never sets a test harness up.
argument-hint: "[a bead, an epic, a design, or nothing to pick from ready work]"
disable-model-invocation: true
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - Agent
---

# Implement

Walk the graph `codefall-design` created. Claim what is ready, build it, verify it, open a pull
request, and let each close unblock the next task until the frontier is empty.

Implementing is not merging. A run ends at open pull requests and a reported merge order — **a human
performs every merge to `main`, and this skill never does**, in any mode, under any instruction
short of the user editing this file. Merges into an epic branch are the one exception: the epic
branch exists to fan work back in, and the human gate sits at its aggregate PR.

Paths that start with `reference/` or `../` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills directory: it names
a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/landing.md` — the three landing strategies, file-scope prediction and the hotspot
  rule, and the branch diagrams. Read at step 4.
- `reference/beads.md` — every `bd` command a run issues: session start, claim and close, the
  landed bead and its gates, discovered work, session end. Read at step 5.
- `reference/workers.md` — launching a worker, the worktree seeding rule, chain sequencing, the
  result JSON, failure handling, and consulting. Read at step 6. Names
  `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`,
  `../../../.codefall/shared/consult-prompt.md`, `../../../.codefall/shared/consult.schema.json`.
- `reference/mirror.md` — how the spec's tracker issue walks the work's state. Read at step 6 and
  step 8.
- `worker-prompt.md` — the prompt rendered for each worker.

## Scope — build, not decide

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Executing tasks from the graph | What the tasks are, or their edges | `codefall-design` |
| Branches, worktrees, commits, PRs | Merging anything to `main` | the user |
| Every test the current work needs | Regression campaigns and fresh-context retesting | `codefall-test` |
| Writing the case file the criteria name | Deciding which tasks need a case; installing a runner | `codefall-design`, `codefall-equip` |
| Harness checks on its own diffs | Independent review and verdicts | `review` |
| Bead lifecycle: claim, close, discovered work | Creating or re-cutting the task graph | `codefall-design` |
| The vision's `Active` transition | Any other document transition | the owning verb |
| Mirroring work state to the spec's tracker issue | The mirror's lifecycle and labels | `codefall-specify` |

**A task that turns out to be wrong is reported, not redesigned.** When the design's cut does not
survive contact with the code, say what you found and hand the graph back to `codefall-design`. A
smaller disagreement the task can finish under — the design's text against the code, or the
spec's — is amended by the worker in its own branch when the document is `Draft` or `Ready` and
the fix is text that moves no task row and no cited criterion; the PR body names it, and the root
re-mirrors a spec change per `reference/mirror.md`. A fix that would move work is filed as a
`design-revision` bead per `reference/beads.md`, for `codefall-design`'s next run; the project's
`.codefall/shared/workflow.md` holds the rule.

- **Implement writes every test the current work needs** — planned by the design's Testing Strategy
  or discovered mid-task, unit through end-to-end. A missing test is written, not sent back to
  `codefall-design`. **`codefall-test` owns what comes after the work lands**: regression passes,
  coverage campaigns, agentic testing in a fresh context.

## One bead or the graph

The argument fixes the scope; the skill never infers it.

| Invocation | Scope |
| --- | --- |
| `/implement booking-parser-trailing-comma` | That bead, alone |
| `/implement booking-DESIGN-007` (an epic) | The epic's whole graph, until its ready set is empty or a gate stops the run |
| `/implement DESIGN-007` | The design's epic, `<prefix>-DESIGN-007`, checked to exist with `bd show` |
| `/implement` | Show ready work grouped by epic and ask |

A design whose Task Plan still says `Staged. Not yet in Beads` has no graph to walk. Refuse and
point at `/design`.

**Tier 0 is not a separate mode.**

## What gets read

Per bead, in order, before any plan is formed:

1. **The bead** — `bd show <id> --json`: the description, the Design ref, the acceptance criteria,
   and `spec_id`, the design document's path.
2. **The design document** — Overview and Architecture always; the specific section the Design ref
   names; Hard Constraints; Technical Context and Testing Strategy when present.
3. **The spec**, one hop up the design's `Related` line — its acceptance criteria are the
   externally observable contract. The vision only when there is no spec.
4. **ADRs** — the ones on the design's `Related` line plus the project's `docs/adrs/` baseline.
   Never rewritten; a conflict between a task and an ADR goes back to `codefall-design` as a
   superseding-ADR conversation, never a quiet exception.
5. **The project's `AGENTS.md`**, root and scoped — workflow rules, verify commands, conventions.
6. **Mockups** under `docs/mockups/` when referenced: a drawing to rebuild in the app's stack, never
   markup to copy.

A tier-0 bead has no document; the list collapses to bead + `AGENTS.md` + ADRs.

## Landing strategies

Three ways work reaches `main`; `reference/landing.md` has the rules and the diagrams.

| Strategy | When |
| --- | --- |
| **Parallel stacks** (default) | Independent chains with disjoint predicted file scopes |
| **Single stack** | Overlapping file scopes, uncertainty, or fan-in without a need for parallelism. Always correct |
| **Epic branch** | Fan-in across chains, or work that must not land on `main` in increments — *and* parallelism matters |

Hotspot files count as overlap until shown otherwise. When parallel stacks cannot be shown safe,
serialize.

## The go gate

One approval, before any work starts. Everything the run will do, in one block:

- the landing strategy and its one-line reason ("linear chain of 7, no fan-in → single stack");
- the branch diagram;
- the waves, and how many workers run concurrently in each — **there is no default cap**; the wave
  is sized by the graph and the file scopes, and the user trims it here if it is too wide;
- the model proposed per bead, and one session-level effort recommendation as the exact command —
  "recommend `/effort high` before go";
- what will be claimed in beads, and — when a vision sits behind the work — that go flips it to
  `Active`;
- that workers amend a design's or spec's text where the code disagrees, in their own PR, per
  `reference/beads.md`;
- the consult order a failure will be put to, resolved per `reference/workers.md`;
- the permissions condition: background workers cannot answer permission prompts, so the session
  must allow edits and Bash without prompting, or the run offers single-task mode instead.

**Single-bead scope shrinks the gate to a plan approval**: the files to touch, the approach, the
test plan, one branch.

**After go, waves proceed on their own.** Failures are the only mid-run stop.

Model choice is the root's, made at the gate. A bead's own execution metadata is a recommendation
shown in the table; **implement never writes bead metadata** — what ran goes in a `bd comment`
beside the PR link.

## Verification and done

A bead is done when three things are true: **its acceptance criteria hold, the project's checks are
green, and its PR is open.** Done is not merged.

**Where the commands come from.** Implement hardcodes no build, lint, or test invocation:

1. `.codefall/skills/codefall-implement/CUSTOMIZE.md` — verb-specific tuning, such as a fast subset per bead
   with the full suite reserved for pre-PR;
2. the project's `AGENTS.md` — scaffolded projects carry the command list in their verification
   section;
3. inference from the repo (`package.json` scripts, `Makefile`, `go.mod`) — stated at the go gate,
   with an offer to record the inferred commands in `AGENTS.md`.

The resolved list is passed into worker prompts. Workers re-derive nothing.

**The checks:**

- The project's own verification commands, run until clean.
- The harness's built-in passes on the bead's own diff, where the harness provides them:
  `simplify` always; `code-review` and `security-review` when available. These are checks inside
  implement, not review — implement renders no verdict on its own work.
- The bead's acceptance criteria, checked one by one. What passed goes into the close reason. A
  bead with no acceptance field falls back to the design's Hard Constraints plus the spec's
  criteria, and the close reason still records what was verified.

**Tests are part of done, not a follow-up.** So are the local scripts: a bead whose criteria name
the `start` and `update` change, or whose diff adds infrastructure, a dependency, a migration, or
generated code, changes the declared scripts in the same PR, following the local track in the
`codefall-equip` skill's section *When another verb follows this skill*. The bead is the
confirmation; the PR body names the change.

**A test case the criteria name is written before the code**, from those criteria and from nothing
else — never the sibling spec, the application's code, or a pull request's own text. Its format is
`../codefall-test/reference/case-file.md` and it lands at `<root>/test-cases/<area>/<slug>.md`
under the testing root; the spec follows it where the modality calls for one. The case counts
toward done. **Running it does not** — that is `codefall-test`'s. What is checked here is that
`../../../.codefall/shared/check-cases.sh` passes and that the runner's run-one command in
`<root>/AGENTS.md` collects the spec.

## Merges and the mirror

**The root never merges to `main`.** The extension ships a `PreToolUse` hook that denies it; a
denial from that hook is the system working as designed. What the root does merge: worker PRs into
the **epic branch**, one at a time, at wave boundaries.

At close-out the run reports the **merge order** — bottom-up per stack; GitHub retargets each PR as
its base merges — and stops. `bd gate check` turns the merges into bead state next session.

The spec's tracker issue walks the work's state per `reference/mirror.md`. Every PR body carries
`Relates to #<spec-issue>`.

## The vision transition

`codefall-envision` reserves one transition for this skill: `Status: Active — <date>`. At the
run's first claim, resolve the vision — the design's `Related` line to the spec, the spec's
`**Vision:**` row to the vision, or the design's `vision` label when there is no spec — and flip
its Status line. Automatically, and report it — "VISION-012 → Active."

Once, idempotently. Already `Active`, or no vision in the lineage: nothing to do. Only the Status
line is touched, ever.

## Picking up an interrupted run

State lives in three places — beads, git, GitHub — and a resumed session reconciles them rather
than re-running anything:

| Found | Meaning | Do |
| --- | --- | --- |
| Bead closed, PR merged | Finished and landed | `bd gate check` records it; nothing else |
| Bead closed, PR open | Done, awaiting the human | Leave it; it is in the merge order |
| Bead claimed, branch pushed, no PR | Worker stopped before `gh pr create` | Verify the branch, open the PR from the root — do not re-run the work |
| Bead claimed, no branch | Work never started or never landed anywhere | Relaunch the worker with the same rendered prompt |
| Bead unclaimed but `bd ready` says ready | Never started | Normal flow |

A stacked chain resumes from its highest link with an open PR; everything below is merged or
awaiting merge, and everything above follows the normal sequence.

## Project customizations

Follow `../../../.codefall/shared/customizations.md` for this verb.

## Process

### 1. Check preconditions

Run the shared check against the user's project.

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances. Otherwise read `beads_reason`, tell the user what is missing, hand over the
command that fixes it, say to rerun this verb after it, and stop:

| `beads_reason` | What is wrong | Give them |
| --- | --- | --- |
| `not_installed` | `bd` is not on PATH | `brew install beads` |
| `not_initialized` | this repository has no beads database | `bd init` |
| `unreadable` | bd found a database and could not read it | quote `beads_detail` |

**Never run the remedy.** That is the user's decision.

Then read the checkout lines. `behind` above `0` or `refresh=stale` means the environment may not
match what the work will build on: say so and run `/codefall-refresh` before continuing.
`refresh=undeclared` names `/codefall-equip` instead.

**Then read the `test=` line**, and hold it against the beads in scope once step 3 has read them. A
bead whose acceptance criteria name a test case needs an equipped harness:

| `test=` | What happens |
| --- | --- |
| `equipped` | those beads proceed |
| `unequipped` | say so, name `/codefall-equip`, and do not start them |
| `undeclared` | say so, name `codefall init`, and do not start them |
| `unknown` | read the `test` block from `.codefall/settings.json` and judge it the same way; no runner there is `unequipped` |

Beads verified by unit tests alone proceed either way. **Never set the harness up** — that is
`/codefall-equip`'s own pull request.

Then read the project's `AGENTS.md` (root and scoped) and
`.codefall/skills/codefall-implement/CUSTOMIZE.md` — workflow constraints, verify commands, pinned
board IDs, a standing strategy preference.

### 2. Fix the scope

Per [One bead or the graph](#one-bead-or-the-graph). With no argument, run the session-start
commands in `reference/beads.md` — `bd ready` unfiltered, since no epic is chosen yet — then show
the ready set grouped by epic, with title, priority, and what each unblocks, and ask. Never infer a
batch from an unprompted ready set.

### 3. Read

The full list in [What gets read](#what-gets-read), for every bead in scope. Reconcile the design
against the code as it stands on the base branch — cite file and line for anything the design
assumed that has moved — and default to preserving whatever the design is silent about.

### 4. Classify the landing strategy

Read `reference/landing.md`. Build the graph picture (`bd ready --mol <epic> --explain`,
`bd dep tree`), find the chains and any fan-in, derive each chain's predicted file scope from its
Design refs, apply the hotspot rule. Pick, constrained by `AGENTS.md` and `CUSTOMIZE.md`.

### 5. The go gate

Present the block per [The go gate](#the-go-gate) and wait. On go, per `reference/beads.md`: flip
the vision to `Active` if one is behind the work. Epic scope: create the epic branch if the
strategy calls for one (`epic/<id>-<slug>` off `main`, pushed), create the landed bead, claim the
epic and the first wave, `bd dolt push`. Single-bead scope: claim the bead, `bd dolt push`, nothing
else.

When the user overrules the classifier the same way twice, offer to record the preference in
`CUSTOMIZE.md` — offer, never write unasked.

### 6. Execute the waves

Per `reference/workers.md`. Per wave: render worker prompts, launch the batch, wait for results.
Verify each success — branch on the remote, PR exists, or it did not happen. A failed bead is
consulted on and retried once; a second failure is consulted on and escalates, per
`reference/workers.md`. File discovered work, in the form
its `kind` names, and read each `amended` list per `reference/beads.md`. Comment the PR link, close
the bead with what was verified, gate the landed bead with the new PR (stacked runs), `bd dolt push`. `--suggest-next` names the next wave; claim it and
go again. Epic branch: merge each worker PR into the epic branch, serialized, at the wave boundary.
Update the mirror per `reference/mirror.md`.

Single-bead scope is one iteration of the same loop, in one worktree.

### 7. Integrate

When the frontier is empty: restack stack bottoms onto current `main`, re-run verification, resolve
nothing silently. Epic branch: open the aggregate PR to `main`, titled as a release-worthy
conventional commit, and gate the landed bead with it — `Closes` nothing; the gate owns the epic's
close.

### 8. Report and stop

Do not merge, and do not wait for merges; the next session's `bd gate check` finishes it.

- Every bead built, with PR, branch, and what its close reason verified.
- The merge order, bottom-up per stack, and what is blocked on the user.
- Upstream amendments, by document and PR; then discovered work filed, in two buckets: code
  follow-ups, and what was handed back to design — "DESIGN-NNN has N revision beads" — with the
  `bd comment` on the epic that names them.
- Every consult: the bead, who answered, what it changed.
- The tracker mirror's state, the vision transition if one fired.
- The worktree list, with the cleanup offer.
- Final `bd dolt push`.
- **Last, what the user does next**: review the pull requests — a stack as one target, its top
  branch; pull requests against the default branch one each — merge them in the order above, then
  `/codefall-test <area>/<slug>` for each case a bead named, and `/codefall-design DESIGN-NNN`
  where revision beads were filed.

## Other modes

- **Resume** an interrupted run — per
  [Picking up an interrupted run](#picking-up-an-interrupted-run). Reconcile, then continue the
  normal loop.
- **Abandon** a run: unclaim what is claimed and unbuilt, note why on each bead, report branches
  and PRs left standing. The user decides their fate; delete nothing.
- **Drain assistance is reporting only.** After merges, `bd gate check` and the mirror update are
  welcome; performing merges is not, and the hook enforces it.

## Rules

- **A human performs every merge to `main`; this skill performs none, in any mode.**
- **Every `bd` write is the root's, in the primary checkout.** Workers never run `bd`; their prompt
  carries what they need and their result JSON carries what they found.
- **Closed means done — criteria verified, checks green, PR open.** Merged is the gates' to say.
- **Publish the claim before the work.** `bd dolt push` follows every claim and every close.
- **The graph is the sequencer.** `bd ready` decides what runs next; never start a blocked bead.
- **Scope is exactly the bead.** Tangents become `discovered-from` beads, filed by the root, never
  fixed in passing; a design or spec found wrong is amended in the PR or becomes a
  `design-revision` bead, never a quiet workaround.
- **Bead IDs ride every commit message.**
- **Verify workers, never trust them.**
- **One consult, one automatic retry, one more consult, then a human.** The root consults; a worker
  never does.
- **No permission prompts mid-run.** The go gate holds the condition.
- **Tests are part of done.** Planned or discovered, written now, never deferred to `test`.
- **A test case the criteria name is written first, from those criteria**, before the code and
  before its spec. Never from the sibling spec, the code, or a pull request's text. The case counts
  toward done; running it is `codefall-test`'s.
- **A test harness is never set up inside a task's pull request.** A bead that needs one and finds
  the project unequipped is not started, and `/codefall-equip` is named.
- **The local scripts are part of done.** A change that would leave a teammate's refresh stale
  changes `start` and `update` in the same PR, per `codefall-equip`'s local track.
- **Implement never writes bead metadata and never redesigns the graph.**
- **`Active` is a fact, recorded once.**
- **Worktrees are cleaned up by offer, never by default.**
- **Never overwrite a file that has drifted.** Show the difference and ask.
