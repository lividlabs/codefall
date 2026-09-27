---
name: codefall-review
description: Review something and fix what the user accepts — uncommitted work, a branch, an open pull request, a path, a document, or a description of what to look at. The reviewer is a subagent or another harness; this session triages the findings with the user and applies the ones they take. Every finding and what was decided about it is written to .codefall/reviews/.
argument-hint: "[what to review — nothing for uncommitted work] [via=<agent name>|<harness>[:model]]"
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

- `reference/lenses.md` — what every code and document lens asks, and the notes on `docs`,
  `simplify`, `trace`, and edited ADRs. Read at the confirmation and at step 3.
- `reference/reviewers.md` — the resolved order, `current` and this session, the `via=` forms, the
  lens groups, subagent merging, and the external call. Read at step 3.
- `reference/findings-file.md` — naming, the Markdown shape, when the files are committed, and
  `revision`. Read at step 4.
- `reference/posting.md` — posting findings to a pull request. Read at step 5 when the target is
  an open pull request.
- `reviewer-prompt.md` and `findings.schema.json` — the prompt each reviewer is rendered from and
  the shape every reviewer returns.
- `../../../.codefall/shared/running-agents.md` — which harness this is, resolving and walking the
  agent order, `via=`. Read at step 1. `../../../.codefall/shared/run-agent.sh` runs one agent.

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
| anything else — prose describing what to look at | that code | a search, confirmed with the user |

**The default branch** is `git symbolic-ref --short refs/remotes/origin/HEAD` with the `origin/`
prefix stripped, falling back to `git remote show origin` when that ref was never set locally, and
to the current checkout's initial branch when there is no remote. Resolve it once per run.

**Several pull requests** — `codefall-implement` leaves one per task. A stack, where each pull
request is based on the one below it, is one target: the top branch, or the range from the merge-base
of its tip with the default branch to its tip, holds every pull request's diff. Pull requests against the default branch share
nothing and are one invocation each, in the order the implement report listed them.

**Document identifiers** are the ones the other verbs define. Resolve one by globbing its directory
and stop if it matches nothing or more than one. Never guess at a near miss, and never invent a form
those verbs do not define.

### What is not reviewable

Five things are refused rather than attempted, and the refusal says which:

- **A merged or closed pull request.** `state` is not `OPEN`.
- **A merged branch.** `git branch --merged <default>` lists it.
- **A superseded ADR.** Its Status line says `Superseded by`.
- **An archived vision or spec.** Its Status line says `Archived`, or it sits under `archive/` —
  either is enough, and a document where the two disagree is a `status` finding for whoever reviews
  the live one.
- **A specific commit.** Out of scope for this verb.

### A prose argument

A prose argument is a scope and sometimes a narrowing. Resolve the scope by searching — the
description names components, behaviours, or domain terms, and those map to files.

**An ambiguous scope is interviewed, not guessed.** Search, show what you found, and ask what the
search could not settle — which of two components was meant, whether the boundary includes its
callers, whether a second subsystem matching the same terms is in or out. Search again with the
answer.

**Two rounds, then stop.** If the second round has not narrowed the set to something the user
recognises, say so and ask for a path or a document identifier instead.

Do not review until the user confirms the file list. Stop if the search finds nothing — say so and
ask for a different description rather than widening on your own.

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
names the resolved agent order, or the `via=` override, and says where fixes will land when that
would create a worktree or branch.

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

**A document target**: the document and the document upstream of it. The question is whether the
target faithfully refines what came before it, and what it added that nobody asked for.

| Target | Upstream | Found by |
| --- | --- | --- |
| vision | none | — |
| spec | its vision, when it has one | the `**Vision:**` header row |
| design | its spec, or its vision when there is no spec | the `**Related:**` row |
| ADR | the design that cites it, and every other accepted ADR | `grep -rl 'ADR-007' docs/designs/`, substituting the identifier, plus `docs/adrs/` |

Links point one way: reaching a design from a spec or an ADR is a grep for the identifier. A
vision has no upstream and is reviewed for internal consistency alone.

## The lenses

Eleven code lenses — `correctness`, `failures`, `behaviour`, `tests`, `types`, `conventions`,
`comments`, `docs`, `simplify`, `local`, `security` — and, for documents, `structure` and `status` plus the
lenses for the document's kind. What each asks is in `reference/lenses.md`. Each verb owns the rules
its documents are held to; read them there when a lens needs the detail, never restate them.

## Calibration

- **Be certain before calling something a bug.** Investigate. If still unsure, it is not a finding —
  it goes in `notChecked`, which is where everything the review could not settle belongs.
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

Severity: a **blocker** is wrong and will be observed; **important** is wrong under conditions that
will occur; **minor** is worth fixing and costs nothing to leave. A `REVIEW.md` at the project root,
if present, states what the project cares about and wins.

## Who reviews

The project's agent order, resolved and walked per `../../../.codefall/shared/running-agents.md`;
the first agent that answers is the reviewer. `current` is a subagent of this harness, and with no
`agents` configured it is the whole order, as before. `via=` replaces the order for one run. This
session reviews only when the work came from somewhere else. The lens groups, how each reviewer runs
them, and the external call are in `reference/reviewers.md`. Every reviewer runs read-only, and every
agent tried is named in the report and the findings file.

## Where the fixes go

**The target decides, not where the user is standing.**

| Target | Fixes land on |
| --- | --- |
| Uncommitted work | The working tree, in place |
| A branch | That branch |
| An open pull request | That PR's branch |
| A commit range | The branch whose tip is `<to>`; if no branch has it, stop and ask |
| A document or path, when something is already checked out for it | There |
| A document or path on the default branch | A new worktree, branched from the default branch |

**Getting there.** A branch, PR, or range target that is not already checked out is fetched and
checked out before any fix is applied — `git fetch origin` then `git checkout <branch>`, taking the
branch name from `headRefName` for a pull request and the branch whose tip is `<to>` for a range. A new worktree is `git worktree add` off the default branch.

**A dirty working tree stops the move.** When the tree is dirty and the fixes belong somewhere else,
report the findings, say the fixes were not applied and why, and leave the tree exactly as it is.
Uncommitted work is never moved.

## Triage and fixes

Present the findings as one numbered list, most severe first, each with its location, its claim, and
the conditions under which it matters. Then ask which to fix.

| Status | Meaning |
| --- | --- |
| `fixed` | The user took it; the fix is applied in this invocation |
| `dismissed` | The user rejected it, with a reason |
| `deferred` | Real, but not now |

Apply the accepted fixes with this session's tools, code and documents alike. A finding's proposed
`patch` is a starting point, not a script — apply the intent, matching the surrounding code. Fixes
are not re-reviewed here.

**A finding that an upstream document is wrong is fixed like any other document finding.** When the
design behind the work says one thing and the code needed another, or the design and its spec
disagree, and the document is `Draft` or `Ready`, the fix is text on the target's branch: the
design's section amended, or a criterion appended to the spec with the requirement's tracker issue
regenerated per `../codefall-specify/trackers/<name>/PROFILE.md`. Every document between the change
and the code that restates the point is fixed together, or none is.

**What that fix cannot do is offered as a bead.** A fix that would move work — a Task Plan row, a
criterion a bead cites — or touch a frozen document, or one the user defers, leaves
`codefall-design` never hearing of it if it stays in the findings file alone. Offer, at triage, to
file it in the `design` form under *Discovered work* in `../codefall-implement/reference/beads.md`:
`--spec-id` the design's path, the label `design-revision`, and a `discovered-from` edge to the
bead the branch names when there is one. On yes, create it, `bd dolt push`, and record its ID as
the finding's `bead`. Never file one unasked, and never for a finding whose cause is the code.

## The findings file

Two files per invocation under `.codefall/reviews/`, a `.json` record and a `.md` written to be
read, sharing one timestamped stem. Written three times — after the review, after triage, after the
fixes — and committed with the fixes, except for uncommitted work, where they are left unstaged.
Naming, the Markdown shape, and `revision` are in `reference/findings-file.md`.

### Kept out of codebase search

A `.ignore` file beside `.codefall/` holds one line, `.codefall/reviews/`, so ripgrep-backed
harnesses skip old findings. `codefall init` writes it and `codefall doctor` warns when it is
missing.

**Check it before the review, and offer:**

> `.ignore` doesn't list `.codefall/reviews/`, so findings from this review will show up in
> codebase searches. Add the line?

On yes, append it — never replace the file. On no, carry on and say nothing further. Say nothing at
all when the line is already there.

## Project customizations

Follow `../../../.codefall/shared/customizations.md` for this verb.

## Process

1. **Resolve and confirm.** Resolve the target per [Targets](#targets); refuse what is not
   reviewable. Interview a prose scope until the file list is recognised. Read
   `.codefall/settings.json`'s `review` block and `.codefall/skills/codefall-review/CUSTOMIZE.md`
   if present. Resolve the agent order per `../../../.codefall/shared/running-agents.md`, or take
   the `via=` override. Check the `.ignore` entry and offer to add it if it is missing. Read
   `reference/lenses.md`, present [the confirmation](#the-confirmation), and wait.
2. **Read.** Everything in [What gets read](#what-gets-read).
3. **Review.** Read `reference/reviewers.md`. Walk the order: `current` runs the lens groups as
   parallel subagents, any other agent runs one `../../../.codefall/shared/run-agent.sh` call
   carrying every lens, this session runs pass by pass. Every candidate finding checked against
   [Calibration](#calibration) before it becomes one.
4. **Write.** Read `reference/findings-file.md`. Merge the JSON and write the findings files, every
   finding `open`.
5. **Triage.** Present the list, take a decision on each, update the files. Post to the pull request
   per `reference/posting.md` if that is enabled and the target is one.
6. **Fix.** Apply what was taken, in the place [Where the fixes go](#where-the-fixes-go) names.
   Update the files.
7. **Report.** The target, the reviewer, and every agent tried before it with why each was skipped
   or failed; what was found, most severe first; what was fixed,
   dismissed, deferred; what could not be checked and why; where the files are; and the branch or
   worktree the fixes landed on if one was created. **End with what the user does next**: on a pull
   request or a branch, push the fixes and merge; on uncommitted work, the findings files are left
   unstaged to commit with the work or not at all; otherwise nothing is pending. Where revision
   beads were filed, `/codefall-design DESIGN-NNN` comes before the merge, so the document is
   corrected while the work that found it wrong is still in view.

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
- **Review the target, nothing else.**
- **Never restate a rule another verb owns.** Point at it; a copy drifts.
- **Every finding carries the conditions under which it manifests.**
- **No flattery, no filler findings.**
- **Read the whole file, never only the hunk.**
- **An external reviewer runs read-only.** A failure advances the order and is reported, never
  worked around silently; the end of the order is a stop.
- **The target decides where fixes land**, never where the session started.
- **Uncommitted work is never moved to a worktree**, and a dirty tree stops a move rather than
  carrying changes onto another branch.
- **Every finding ends with a status**, and the files are written even when nothing was found and
  even when nothing was fixed.
- **Refuse what is not reviewable** — a merged or closed pull request, a merged branch, a superseded
  ADR, an archived document, a specific commit — and say which.
- **An identifier that resolves to nothing is a stop**, not a guess.
- **Only `fixed` and `deferred` findings reach a pull request.** A dismissed one was judged wrong.
- **A revision bead is offered, never filed unasked**, and only for a finding the design caused
  that a fix here cannot settle: it moves work, the document is frozen, or the user deferred it.
- **The `.ignore` entry is offered, never added unasked**, and appended rather than written over.
