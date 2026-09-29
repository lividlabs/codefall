---
name: codefall-fix
description: Fix something small in one run — a bead, a bug report, an issue, or a description — by running codefall-design at tier 0 and codefall-implement on the one bead it produces, behind a single confirmation. Stops and names codefall-design when the work needs a design document. Never merges to main.
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

Take a small fix from its description to an open pull request in one run: `codefall-design` at tier
0 to establish the bead, then `codefall-implement` on that bead. **Fix is those two procedures, run
back to back with one confirmation between them.** It adds no rule of its own and relaxes none of
theirs.

Paths that start with `../` are relative to this skill's directory, not the user's project. A path
through `../../../.codefall/` is the one that leaves the skills directory: it names a file `codefall
init` installed in the project's own `.codefall/`.

## Files beside this one

**Read each of these in full, never a preview**, and resolve its own relative paths from its
directory. **After a compaction, read the `SKILL.md` for the step you are on again before you
continue**: only this file is restored, and the procedures it follows are not.

- `../codefall-design/SKILL.md` — the design procedure. Read at step 2, and follow it where this file
  says.
- `../codefall-design/reference/bugs.md` — reproducing a bug and finding its cause. Read at step 2
  for a bug.
- `../codefall-design/reference/beads.md` — creating the bead. Read at step 4. It names
  `../codefall-test/reference/case-file.md`, the test-case format, read at step 3 when the bead
  carries a case, and `../codefall-design/reference/revising.md`, for a design whose graph already
  exists, which a fix never reaches.
- `../codefall-implement/SKILL.md` — the implement procedure. Read at step 4, and follow it the same
  way, for single-bead scope.

## What fix is for

**One bead, at tier 0**: the change stays inside one component, changes no public interface, and is
one task. A bug fix, a small change a description states fully, or a bead already in the graph.

**What fix drops** from running the two verbs separately:

| Dropped | Because |
| --- | --- |
| Design's tier-0 confirmation and implement's plan approval as two stops | One confirmation, at step 3, carries both |
| A second preflight, and the report between the two verbs | One run, one preflight, one report |
| Design's offers of `/codefall-specify` and `/codefall-report` | A small fix proceeds from what it was given; the bead carries the reproduction |

**What fix does not do**: write a design document, write an ADR, or build more than one bead. When
design's step 4 finds any of those, stop before anything is written, say what it found, and name
`/codefall-design <the input>`. That is an exit, not a disagreement about size: fix has no step that
produces them.

## Project customizations and persona

Read `.codefall/skills/codefall-fix/CUSTOMIZE.md`, then `codefall-design`'s and
`codefall-implement`'s, per `../../../.codefall/shared/customizations.md`: the project's procedure
for either verb still holds when fix runs it. Read the `persona=` line of the preflight report; when
it is not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for
this run, and say so.

## Process

### 1. Check preconditions

Run `codefall-implement`'s step 1 once — the preflight, the Beads stop, the checkout lines, and the
`test=` line — and nothing is run a second time later.

### 2. Establish the bead

What the input is decides how much of design runs:

| Input | What runs |
| --- | --- |
| A bead with a reproduction or a clear task, and acceptance criteria | Nothing from design. Read it, and go to step 3 |
| A bead with a title and little else — a `discovered-from` bead, say | Design's steps 2–4 fill it in: the cause and the criteria. The bead is edited only if no one holds it |
| A bug report, `BUG-NNN`, or the issue a report mirrors | Design's steps 2–4 with `../codefall-design/reference/bugs.md`: reproduce if the report could not, find the cause |
| Another issue, or a description | Design's steps 2–4 at tier 0; for a bug, `../codefall-design/reference/bugs.md` as well |

A bead someone else has claimed is a stop: say who holds it. A bead that is already closed, or has an
open pull request, is reported and the run stops.

**When design's step 4 finds more than tier 0**, or an ADR trigger fires, stop here per
[What fix is for](#what-fix-is-for).

### 3. The one confirmation

Show everything the run will do, in one block, and wait:

- **The bead** — design's tier-0 confirmation: its ID, title, and body — the reproduction, the cause
  with its file and line, the acceptance criteria, and a test case with its criteria when the fix is
  only visible through the wired product. For an existing bead, what changes in it, or that nothing
  does.
- **The plan** — implement's single-bead plan approval: the files to touch, the approach, the test
  plan, the branch, and the model.
- **The permissions condition** from implement's go gate.
- **Any amendment** design would make to a bug report, which rides in the fix's pull request.

On yes, both approvals are given. Nothing is written before it.

### 4. Build it

Create or edit the bead per `../codefall-design/reference/beads.md` and push it. Then follow
`codefall-implement` from its step 5, for single-bead scope, with the go gate already passed: claim,
build, verify, open the pull request, close the bead. Every rule of implement holds, the merge rule
first.

### 5. Report

One report, not two: the bead and how it was established; the cause, for a bug; the pull request,
its branch, and what the close verified; discovered work filed; every consult; the worktree, with
the cleanup offer. **End with what the user does next**: review and merge the pull request, then
`/codefall-test <area>/<slug>` when the bead named a case.

## Rules

- **Fix is design at tier 0 and implement on one bead.** Their rules hold throughout.
- **Read the procedures in full, and again after a compaction.**
- **One confirmation**, and nothing written before it.
- **Tier 0 or an exit.** A design document, an ADR, or a second bead is `codefall-design`'s work.
- **A ticket must not change under someone holding it.**
- **A human performs every merge to `main`; fix performs none.**
