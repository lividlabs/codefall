---
name: codefall-review
description: Review something and fix what the user accepts — uncommitted work, a branch, an open pull request, a path, a document, or a description of what to look at. The reviewer is a subagent or another harness; this session triages the findings with the user and applies the ones they take. Every finding and what was decided about it is written to .codefall/reviews/.
argument-hint: "[what to review — nothing for uncommitted work] [via=<harness>[:model]]"
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

# Review

Read something, say what is wrong with it, decide with the user what to do about each finding, do
it, and write down what was found and decided.

One invocation is one complete review. Nothing carries over: a second review of the same thing is a
new review, with its own file.

**The reviewer and the fixer are different contexts.** The review runs in a subagent or another
harness; the triage and the fixes happen in this session.

Paths that start with `reference/` or `../` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills
directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/targets.md` — resolving an epic's work, several pull requests, document identifiers,
  and a prose argument. Read at step 1 when the argument is one of those.
- `reference/lenses.md` — what every code and document lens asks, and the notes on `docs`,
  `simplify`, `trace`, and edited ADRs. Read at the confirmation and at step 3.
- `reference/reviewers.md` — the resolved list, `current` and this session, the `via=` forms, the
  lens groups, subagent merging, the external call, and consulting on `notChecked`. Read at step 3.
  Names `../../../.codefall/shared/consult-prompt.md` and `../../../.codefall/shared/consult.schema.json`.
- `reference/findings-file.md` — naming, the Markdown shape, when the files are committed, and
  `revision`. Read at step 4.
- `reference/posting.md` — posting findings to a pull request. Read at step 5 when the target is
  an open pull request.
- `reviewer-prompt.md` and `findings.schema.json` — the prompt each reviewer is rendered from and
  the shape every reviewer returns.
- `../../../.codefall/shared/running-agents.md` — which harness this is, resolving and walking the
  agent list, `via=`. Read at step 1. `../../../.codefall/shared/run-agent.sh` runs one agent.
- `../../../.codefall/shared/delivery.sh <epic>` — the delivery's one-line chart. Run at the
  confirmation and in the report when the target is an epic's work.
- `../../../.codefall/shared/stacks.md` — *Pointing at a stack*, read at step 7.

## Targets

The argument's shape decides what is being reviewed.

| Argument | Target | Resolved by |
| --- | --- | --- |
| *(none)* | uncommitted work | `git diff`, `git diff --cached`, `git status --short` for untracked files |
| a branch name | that branch | `git diff <base>...<branch>` against the default branch |
| a PR number or github.com pull URL | that pull request | `gh pr view <ref> --json number,state,headRefName,headRefOid,baseRefName,baseRefOid`, `gh pr diff <ref>` |
| `<from>..<to>`, two commits | the work between them | `git diff <from> <to>`; both must be reachable from a live branch |
| a path to a file or directory | that path as it stands | the files under it |
| a codefall document identifier, or a path under `docs/` | that document | the file, plus the document upstream of it |
| an epic ID, or `DESIGN-NNN` | the epic's work | the epic branch, or the stack's top, per `reference/targets.md` |
| anything else — prose describing what to look at | that code | a search, confirmed with the user |

**The default branch** is `git symbolic-ref --short refs/remotes/origin/HEAD` with the `origin/`
prefix stripped, falling back to `git remote show origin` when that ref was never set locally, and
to the current checkout's initial branch when there is no remote. Resolve it once per run.

### What is not reviewable

Five things are refused rather than attempted, and the refusal says which:

- **A merged or closed pull request.** `state` is not `OPEN`.
- **A merged branch.** `git branch --merged <default>` lists it.
- **A superseded ADR.** Its Status line says `Superseded by`.
- **An archived vision or spec.** Its Status line says `Archived`, or it sits under `archive/` —
  either is enough, and a document where the two disagree is a `status` finding for whoever reviews
  the live one.
- **A specific commit.** Out of scope for this verb.

## The confirmation

Every target is confirmed before anything is reviewed. Name what was resolved and which lenses will
run, and offer to drop some:

```
Resolved SPEC-004 to docs/specs/SPEC-004-trip-sharing.md,
checked against VISION-002-trip-sharing.md.

Lenses: structure, status, trace, criteria, precision, stories.

Review now, or exclude any of these?
```

Every lens that applies to the target runs unless the user drops it here. A prose argument that
narrows — "review the auth code for security issues" — shows the reduced list. The confirmation also
names the resolved agent list, or the `via=` override, and says where fixes will land when that
would create a worktree or branch. For an epic's work it opens with the chart line.

## What gets read

**A target with a diff** — uncommitted work, a branch, a pull request, a range:

1. The diff, by the command in the table above.
2. Every modified file in full.
3. Every untracked file in full. A new file has no diff; its whole content is the change.
4. The project's `AGENTS.md` and `docs/adrs/`. The root `AGENTS.md` always; a scoped one whenever a
   touched file sits under its directory, nearest first. A project with neither is reviewed against
   its own surrounding code, and the report says so.
5. The design behind the work when there is one. A branch named `feat/booking-DESIGN-007-T1-…` carries a bead ID;
   `bd show <id> --json` gives `spec_id`, the design document's path. From the design,
   `codefall-design` defines the row that reaches the spec, and `codefall-specify` the row that
   reaches the vision.

**Every hop in item 5 is optional, and a missing one is never an error.** Review against
conventions alone, and record which hop was missing in `notChecked`.

**A path or prose target**: every file in the resolved set, in full, plus item 4 above.

**A document target**: the document and the document upstream of it, found as the *Document
identifiers* section of `reference/targets.md` says. The question is whether the target faithfully
refines what came before it, and what it added that nobody asked for.

## The lenses

Eleven code lenses — `correctness`, `failures`, `behaviour`, `tests`, `types`, `conventions`,
`comments`, `docs`, `simplify`, `local`, `security` — and, for documents, `structure` and `status` plus the
lenses for the document's kind, all in `reference/lenses.md`. Each verb owns the rules its documents
are held to; read them there, never restate them.

## Calibration

- **Be certain before calling something a bug.** Investigate. If still unsure, it goes in
  `notChecked`, with everything else the review could not settle.
- **Review the target, nothing else.** Where the target has a diff, the scope is the changed lines,
  and pre-existing code the diff did not touch is out of bounds. Where it does not — a path, a
  document — the scope is the whole of what was named.
- **No hypothetical edge cases.** Name the realistic scenario that reaches it, or drop it.
- **State the conditions up front.** Empty input, concurrency, cold cache — the first sentence says
  when it matters. The conditions are most of the severity.
- **Do not be a zealot about style.** Verify the project actually holds the convention. Some
  violations are the simplest option and are fine. Excessive nesting is a finding regardless.
- **Do not overstate severity or inflate the count.**
- **No flattery.** No summary of what the work does well.

Severity, with the Beads priority a finding's bead takes: a **blocker** (P1) is wrong and will be
observed; **important** (P2) is wrong under conditions that will occur; **minor** (P3) is worth
fixing and costs nothing to leave. `reference/findings-file.md` has the table; P0 is a person's to
set. A `REVIEW.md` at the project root, if present, states what the project cares about and wins.

## Who reviews

The `review` list of the `agents` entry for this harness, else of the `default` entry, resolved and
walked per `../../../.codefall/shared/running-agents.md`; the first agent that answers is the
reviewer. `current` is a subagent of this harness, and with no `agents` configured it is the whole
list. `via=` replaces the list for one run. This session reviews only when the work came
from somewhere else. The lens groups, how each reviewer runs them, and the external call are in
`reference/reviewers.md`. Every reviewer runs read-only, and every agent tried is named in the
report and the findings file.

## Where the fixes go

**The target decides, not where the user is standing.**

| Target | Fixes land on |
| --- | --- |
| Uncommitted work | The working tree, in place |
| A branch | That branch |
| An open pull request | That PR's branch |
| A commit range | The branch whose tip is `<to>`; if no branch has it, stop and ask |
| An epic's work | A document fix: the stack's top branch, or the epic branch. A code fix: a child of the epic, built by `codefall-implement` in this session |
| A document or path, when something is already checked out for it | There |
| A document or path on the default branch | A new worktree, branched from the default branch |

**Getting there.** A branch, PR, or range target that is not already checked out is fetched and
checked out before any fix is applied — `git fetch origin` then `git checkout <branch>`. A new
worktree is `git worktree add` off the default branch.

**A dirty working tree stops the move.** When the fixes belong somewhere else, report the findings,
say the fixes were not applied and why, and leave the tree as it is.

## Triage and fixes

Present the findings as one numbered list, most severe first, each with its priority and severity,
its location, its claim, and the conditions under which it matters. Then ask which to fix. **On an
epic's work the question is one line** — "I found N problems. Fix them all? (I recommend yes.)" — and the person may take all,
some, or none.

| Status | Meaning |
| --- | --- |
| `fixed` | The user took it; the fix is applied in this invocation |
| `dismissed` | The user rejected it, with a reason |
| `deferred` | Real, but not now |

A dismissed finding is written with its `reason`; a dismissal with no reason is asked for one
before the write.

Apply the accepted fixes, code and documents alike. A finding's proposed `patch` is a starting
point, not a script — apply the intent, matching the surrounding code. Fixes are not re-reviewed
here. **On an epic's work, a taken code finding is not fixed here**: it is
filed as a `deferred` child of the epic in the `code` form below, or the `design` form where the
design caused it, its status is `deferred` with the bead's ID, and step 8 runs `codefall-implement`
on the epic to build them in this session. Document fixes on an epic's work still land
here as text, under the rule below. **A code fix that changes behaviour the design or spec describes amends
that document in the same commit**, under the rule below: the document is part of the fix, not a
second finding.

**A finding that an upstream document is wrong is fixed like any other document finding.** When the
design behind the work says one thing and the code needed another, or the design and its spec
disagree, and the document is `Draft` or `Ready`, the fix is text on the target's branch: the
design's section amended, or a criterion appended to the spec with the requirement's tracker issue
regenerated per `../codefall-specify/trackers/<name>/PROFILE.md`. Fix every document between the
change and the code that restates the point; when one of them fails these tests, fix none, and name
each document in the bead offered below, so the whole chain reaches `codefall-design` together.
Each document fix is recorded as a closed `design-amended` bead, per *Amendments* in
`../codefall-implement/reference/beads.md`, with a `discovered-from` edge to the bead the branch
names when there is one.

**What that fix cannot do, and what the user defers, is offered as a bead.** A finding left in the
findings file alone is one no verb reads again. Offer, at triage, to file each in the form under
*Discovered work* in `../codefall-implement/reference/beads.md`: the `design` form — `--spec-id`
the design's path, the label `design-revision` — for a finding the design caused that a fix here
cannot settle because it moves work, the document is frozen, or the user deferred it; the `code`
form for a `deferred` finding whose cause is the code. Either carries a `discovered-from` edge to
the bead the branch names when there is one, and is a `deferred` child of that bead's epic, read
from `bd show <bead> --json`, when there is one. On yes, create it with `-p` set to the finding's
priority, `bd dolt push`, and record its ID as the finding's `bead`. Never file one unasked.

## The findings file

Two files per invocation under `.codefall/reviews/`, a `.json` record and a `.md` written to be
read, sharing one timestamped stem. Written three times — after the review, after triage, after the
fixes — and committed with the fixes, except for uncommitted work, where they are left unstaged.
Naming, the Markdown shape, and `revision` are in `reference/findings-file.md`.

### Kept out of codebase search

A `.ignore` file beside `.codefall/` holds one line, `.codefall/reviews/`, so ripgrep-backed
harnesses skip old findings. When the line is missing, offer to append it before the review, in
the words `reference/findings-file.md` gives; never replace the file, and say nothing when the line
is there.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

1. **Resolve and confirm.** Resolve the target per [Targets](#targets); refuse what is not
   reviewable. Interview a prose scope until the file list is recognised. Read
   `.codefall/settings.json`'s `review` block and `.codefall/skills/codefall-review/CUSTOMIZE.md`
   if present. Answer which harness this is and take its `agents` entry's `review` list per
   `../../../.codefall/shared/running-agents.md`, or take the `via=` override. Check the `.ignore`
   entry and offer to add it if it is missing. Read `reference/lenses.md`, present
   [the confirmation](#the-confirmation), and wait.
2. **Read.** Everything in [What gets read](#what-gets-read).
3. **Review.** Read `reference/reviewers.md`. Walk the list: `current` runs the lens groups as
   parallel subagents, any other agent runs one `../../../.codefall/shared/run-agent.sh` call
   carrying every lens, this session runs pass by pass. Every candidate finding checked against
   [Calibration](#calibration) before it becomes one. Then each `notChecked` entry that is an
   unsettled question about the target is consulted on once, per `reference/reviewers.md`; a
   consult never becomes a finding on its own.
4. **Write.** Read `reference/findings-file.md`. Merge the JSON and write the findings files, every
   finding `open`.
5. **Triage.** Present the list, take a decision on each, update the files. Post to the pull request
   per `reference/posting.md` if that is enabled and the target is one.
6. **Fix.** Apply what was taken, in the place [Where the fixes go](#where-the-fixes-go) names.
   Update the files.
7. **Report.** The chart line for an epic's work; the target, the reviewer by harness and model,
   and every agent tried before it with why each was skipped or failed; every consult and what it
   changed; what was found, most severe first; what was fixed, dismissed, deferred, and the beads
   filed; what could not be checked and why; where the files are; and the branch or worktree the
   fixes landed on if one was created. On an epic's work, one line goes on the epic's notes —
   `round N (review) [date]: what was fixed and filed` — per *Session end* in
   `../codefall-implement/reference/beads.md`. Anything decided here that no bead, file, or PR
   holds — a dismissal's reason is already in the file — goes in a `bd comment` on the epic or the
   bead, and the report says it is safe to `/clear`. **End with what the user does next**: on an
   epic's work, that this run is now running `codefall-implement <epic>` to fix the problems the
   person took, or `/codefall-test <epic>` when none were taken; on another pull request or branch,
   push the fixes and give the link to the pull request to merge, per *Pointing at a stack* in
   `../../../.codefall/shared/stacks.md`; on uncommitted work, the findings files are left unstaged
   to commit with the work or not at all; otherwise nothing is pending. Where revision beads were
   filed, `/codefall-design DESIGN-NNN` comes first, so the document is corrected while the work
   that found it wrong is still in view.
8. **Fix the epic's problems.** On an epic's work, when the person took at least one code finding,
   run the `codefall-implement` skill with the epic as its argument, through the `Skill` tool, in
   this session. Its go gate reopens the children this run filed, its workers build them, and its
   report ends the session with its own one command.

**Three runs end early, and each ends cleanly.**

- **No findings.** Write the files — the only record that this revision was looked at. Skip triage
  and fixing, and report what ran and what it covered; `notChecked` is the part worth reading.
- **Nothing accepted.** Every finding `dismissed` or `deferred`. The second write records it; there
  is no third.
- **This session reviewed.** Say so in the report: the context that found these is the context that
  fixed them.

## Rules

- **The reviewer never fixes.** Review in a subagent or another harness; triage and fix in this
  session.
- **Nothing is changed before triage.** The user decides what gets fixed.
- **Confirm the scope before reviewing it.** A review of the wrong files costs the whole run.
- **Be certain, or put it in `notChecked`.** Uncertainty is recorded, never dressed as a finding.
- **A consult proposes.** It may move a `notChecked` entry to a finding this session has verified;
  it never decides a status.
- **Review the target, nothing else.**
- **Never restate a rule another verb owns.** Point at it; a copy drifts.
- **Every finding carries the conditions under which it manifests.**
- **No flattery, no filler findings.**
- **Read the whole file, never only the hunk.**
- **An external reviewer runs read-only.** A failure advances the list and is reported, never
  worked around silently; the end of the list is a stop.
- **The target decides where fixes land**, never where the session started.
- **Uncommitted work is never moved to a worktree**, and a dirty tree stops a move rather than
  carrying changes onto another branch.
- **Every finding ends with a status**, and the files are written even when nothing was found or
  fixed.
- **Refuse what is not reviewable**, and say which of the five it is.
- **An identifier that resolves to nothing is a stop**, not a guess.
- **Only `fixed` and `deferred` findings reach a pull request.** A dismissed one was judged wrong.
- **A bead is offered, never filed unasked**: a `design-revision` bead for a finding the design
  caused that a fix here cannot settle, a `code` bead for a deferred finding the code caused, each a
  `deferred` child of the epic when there is one.
- **On an epic's work, the problems the person takes are built by `codefall-implement`**, run by
  this session after the report, never fixed by hand on the stack. The person's one typed command
  was the review.
- **The `.ignore` entry is offered, never added unasked**, and appended rather than written over.
