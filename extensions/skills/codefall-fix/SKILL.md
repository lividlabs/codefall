---
name: codefall-fix
description: Fix something small in one run — a bead, a bug report, an issue, or a description. Establish one bead at tier 0 with its cause and acceptance criteria, build it in a worktree with tests as part of done, and open a pull request, behind a single confirmation. Stops and names codefall-design when the work needs a design document, an ADR, or more than one bead. Never merges to main.
argument-hint: "[a bead, a bug report, an issue, or what to fix]"
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

# Fix

Take one small change from its description to an open pull request in one run: establish the bead,
confirm once, build it, verify it, and stop at the pull request.

This is the path `codefall-design` takes at tier 0 followed by the path `codefall-implement` takes for
a single bead, written as one procedure. Anything bigger is `codefall-design`'s.

Paths that start with `../` are relative to this skill's directory, not the user's project. A path
through `../../../.codefall/` is the one that leaves the skills directory: it names a file `codefall
init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `../codefall-design/reference/bugs.md` — the bug report as a target, reproducing it when the report
  could not, and finding the cause. Read at step 2 for a bug.
- `../codefall-design/reference/beads.md` — what a tier-0 bead carries, its ID, and the test case in
  its criteria. Read at step 2 and step 5. Names `../codefall-test/reference/case-file.md`, and
  `../codefall-design/reference/revising.md`, for a design whose graph exists, which a fix never
  reaches.
- `../codefall-implement/reference/done.md` — when a bead is done, where the verification commands
  come from, and the test case written before the code. Read at step 2 and step 6. Names
  `../../../.codefall/shared/check-cases.sh`.
- `../codefall-implement/reference/beads.md` — the `bd` commands: session start, claim and close,
  discovered work, session end. Read at step 5.
- `../codefall-implement/reference/workers.md` — launching the worker, its result, failure handling,
  and consulting. Read at step 6. Names `../codefall-implement/worker-prompt.md`,
  `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`,
  `../../../.codefall/shared/consult-prompt.md`, `../../../.codefall/shared/consult.schema.json`.

## Scope — one bead, tier 0

All of these hold, or it is not a fix:

- The change stays inside one component.
- No public interface changes.
- It is one task: one bead, one branch, one pull request.
- No ADR is called for — no new dependency, no schema or protocol decision other components will
  build on, no choice that is hard to reverse.

**When any of them fails, stop before anything is written**, say which one, and name
`/codefall-design <the input>`. This is an exit, not a disagreement about size: fix has no step that
writes a design document or an ADR, or builds a graph.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report; when it is not `engineer`, follow that persona's section in
`../../../.codefall/shared/personas.md` for this run, and say so.

## Process

### 1. Check preconditions

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances. Otherwise tell the user what is missing, give them the remedy, say to rerun
this verb after it, and stop: `not_installed` is `brew install beads`, `not_initialized` is
`bd init`, `unreadable` quotes `beads_detail`.

`behind` above `0` or `refresh=stale`: say so and run `/codefall-refresh` before continuing.
`refresh=undeclared` names `/codefall-equip`. Keep the `test=` line for step 3.

Read the project's `AGENTS.md`, root and scoped. **Never run a remedy.**

### 2. Establish the bead

`bd dolt pull`, then read the input:

| Input | What happens |
| --- | --- |
| A bead with acceptance criteria and a body that says what to do | Read it. It is the bead |
| A bead with a title and little else | Fill it in below: the cause, the criteria. Only if nobody holds it |
| A bug report, `BUG-NNN`, or the issue a report mirrors | `../codefall-design/reference/bugs.md`: the report is the target, reproduce if it could not be, find the cause |
| Another issue, or a description | Establish what to change from the code; for a bug, `bugs.md` as well, without its offer of `/codefall-report` |

A bead someone else holds is a stop: say who. A bead already closed, or with an open pull request,
is reported, and the run stops. A bead whose parent epic still has open gates on `<epic>-MERGED` —
a child of a delivery whose work has not merged — is a stop naming `/codefall-implement <epic>`:
this verb lands on the default branch, and that work lands on the epic's stack or branch. For
anything else, `bd search` first: work already in the graph is that bead, not a new one.

**Read what binds the change**: `docs/adrs/`, and the code the change passes through, against the
default branch as it stands. Cite file and line for the cause and for anything the bead assumed that
has moved.

**Write the acceptance criteria**, two to five checkable statements, per
`../codefall-design/reference/beads.md`: a bug report's criteria as they are, citing `BUG-NNN-AC-nn`;
a spec criterion cited in full. When only the wired product shows the change, the criteria name a
test case and its criteria as that file says.

### 3. Hold it against the scope

Walk [the scope](#scope--one-bead-tier-0) out loud — "one component, no interface change, one bead,
no ADR: a fix." Any failure is the exit.

A bead whose criteria name a test case needs an equipped harness: `test=unequipped` names
`/codefall-equip`, `test=undeclared` names `codefall upgrade`, and the run stops. Never set a harness
up.

### 4. The one confirmation

Show everything the run will do, in one block, and wait. Nothing is written before a yes.

- **The bead**: its ID, title, and body — the reproduction, the cause with its file and line, the
  acceptance criteria, and the test case when there is one. For an existing bead, what changes in it,
  or that nothing does.
- **The plan**: the files to touch, the approach, the test plan, the branch
  (`feat/<bead>-<slug>`), and the model the worker runs on.
- **The verification commands** and where they came from, per
  `../codefall-implement/reference/done.md`.
- **The permissions condition**: the worker runs in the background and cannot answer a permission
  prompt, so the session must allow edits and Bash without prompting.
- **Any amendment** to a bug report, which rides in the fix's pull request.

### 5. Create and claim the bead

Create it, or edit the thin bead, per `../codefall-design/reference/beads.md`: its ID,
`--external-ref` where a tracker issue exists, type `bug` where it is one, the body, and
`--acceptance`. Then claim it per `../codefall-implement/reference/beads.md`. `bd dolt push` after
each.

### 6. Build it

One worker, in its own worktree, per `../codefall-implement/reference/workers.md`: `BASE_REF` and
`PR_TARGET` are the default branch. The pull request body carries `Fixes #<issue>` when the bead
traces to a bug report with an issue, `Relates to #<spec-issue>` when it traces to a spec with a
mirror, and neither otherwise.

**Verify the worker, never trust it**: the branch is on the remote, the pull request exists, and the
bead is done per `../codefall-implement/reference/done.md`. A failure is consulted on and retried
once, per `workers.md`; a second one goes to the user.

Comment the pull request link on the bead, close it with what was verified, check any `design`
discovery and record any amendment per `../codefall-implement/reference/workers.md`, file the
discovered work as `discovered-from` beads, and `bd dolt push`.

### 7. Report and stop

Do not merge, and do not wait for the merge.

Report the bead and how it was established, the cause for a bug, the pull request and its branch,
what the close verified, discovered work filed, every consult, and the worktree with the cleanup
offer. Anything decided here that no bead, PR, or document holds goes in a `bd comment` on the
bead, and the report says it is safe to `/clear`. **End with what the user does next**: review and
merge the pull request, then `/codefall-test <area>/<slug>` when the bead named a case.

## Rules

- **One bead at tier 0, or an exit** naming `/codefall-design`.
- **One confirmation, and nothing written before it.**
- **A human performs every merge to `main`; fix performs none.**
- **A ticket must not change under someone holding it.**
- **Scope is exactly the bead.** A tangent becomes a `discovered-from` bead, never a change in passing.
- **Tests are part of done.** A test case the criteria name is written first, from the criteria alone.
- **A test harness is never set up here.** An unequipped bead is not started.
- **Verify the worker, never trust it.**
- **Every `bd` write is pushed.**
- **Never overwrite a file that has drifted.** Show the difference and ask.
