# Beads: the commands a run issues

Every `bd` command in a run is the root session's, executed against the primary checkout — with
`-C` when the session's working directory is elsewhere. The default beads setup is an embedded Dolt
database, one writer at a time, whose sync channel is the repo's own git remote under
`refs/dolt/data`. Read at step 5, before the first claim.

## Contents

- Session start
- Claim, work, close
- The landed bead and its gates
- Discovered work
- Amendments
- Session end

## Session start

```bash
bd dolt pull            # teammates' claims and closes, when a remote is wired
bd gate check           # resolve gates for PRs merged since last session
bd ready --mol <epic>   # the claimable frontier
```

A project with no Dolt remote works identically; state is this-machine-only. Say so once; `bd dolt
pull` failing for lack of a remote is that statement's trigger, not a stop.

`bd ready` leaves `deferred` beads out, so the children a review or a test run filed for the next
round stay off the frontier until that round's go reopens them (`bd update <id> -s open`); from
then on `bd ready --mol <epic>` pulls them in like any other child.

`bd gate check` fails safely. With `gh` missing, unauthenticated, or no GitHub remote, every gate
stays open, the command reports per-gate errors, and it still exits 0: a report, never a falsely
resolved gate. An `ESCALATE` line is different — the gate was checked and its PR is missing — and
is surfaced to the user.

## Claim, work, close

```bash
bd update <epic> <bead> --claim     # epic scope: the epic itself is claimed on the run's first claim
bd dolt push                        # the claim is visible to the team before the work, not after
```

Single-bead scope claims only its bead — the epic stays unclaimed so coworkers can take siblings —
and creates no landed bead; the PR-link comment is the merge trail.

Work happens in git; commits carry the bead ID — `feat: add StageContext type
(booking-DESIGN-007-T1)`.

```bash
bd comment <bead> "PR #101 · feat/booking-DESIGN-007-T1-stage-context · built with opus/high"
bd close <bead> -r "done: criteria R1,R2 verified, checks green, PR #101 open" --suggest-next
bd dolt push
```

**The close is the engine.** Closing a bead removes it as a blocker, so `--suggest-next` prints the
next wave straight from the graph. **Closed means done, not merged**; the merge is tracked by the
gates below.

## The landed bead and its gates

An epic cannot be gated directly and refuses to close while children are open. So an epic run's
first claim also creates one extra child task — `Land: all PRs merged to main`, named
`<epic>-MERGED` — with gates blocking it. `--id` and `--parent` do not combine on one `bd create`,
so the parent is set in a second call:

```bash
bd create "Land: all PRs merged to main" --id <epic>-MERGED
bd update <epic>-MERGED --parent <epic>
bd gate create --type=gh:pr --blocks <epic>-MERGED --await-id=<pr-number> -r "PR #<n>"
```

`bd close` refuses an issue with unsatisfied gates, so the landed bead cannot close — and therefore
the epic cannot close — until every PR is merged. A later session's `bd gate check` clears the
gates as merges land. **Epic closed always means the code is on `main`.**

**Which PRs get gates depends on the strategy.** A stacked run gates every PR as it opens, a later
round's PRs included. An epic-branch run gates none of its worker PRs — they target the epic branch
and the root merges them at the wave boundary; its one gate is created at integration, for the
aggregate PR, and a later round leaves it alone. A standalone bead gets no gate.

## Discovered work

Two forms, told apart by the `kind` on a worker's discovered item. Every discovered bead is named:
`<prefix>-<tracker ref>` (`gh-123`, `jira-ABC-42`), with `--external-ref` set to the same value,
where a tracker issue exists; `<prefix>-<slug>` otherwise. A taken ID is refused; append `-2` and
retry. Never `--force`. Workers report discoveries in their result JSON; the root files them — and
files its own, from the reconciliation at step 3 — and pushes as after every other write.

**Every discovered bead is a child of the epic**, created `deferred` so this round's `bd ready`
does not pick it up, with the parent set in a second call because `--id` and `--parent` do not
combine. The next round's go reopens it. At single-bead scope there is no epic: the edge alone, and
status `open`. `codefall-review` and `codefall-test` file what they find in the same two forms.

**`code`** — a tangent in the code: a bug, a missing test, a refactor. A task under the epic,
linked to the bead that found it.

```bash
bd create "Parser drops trailing comma" --id "$(bd config get issue_prefix)-parser-trailing-comma" \
  --deps discovered-from:<bead> -p 2 -s deferred
bd update "$(bd config get issue_prefix)-parser-trailing-comma" --parent <epic>
bd dolt push
```

**`design`** — the design's text and the code the task needed disagree, or the design and the spec
disagree, the task could still be finished, and the worker could not amend the document itself:
the fix would move a Task Plan row or a criterion a bead cites, or the document is frozen. Text the
worker could amend was amended in its branch instead, under *Amendments* below, and the root checks
each item's `blocks` before filing it, per *Results* in the workers reference beside this file; an
item that fails the check goes back to its worker, not into a bead. A revision request against the document: the same edge, plus the
design's path as `--spec-id` and the label `design-revision`. `codefall-design` lists exactly those
two markers at its start, and its Revise mode closes every one.

```bash
bd create "DESIGN-007 § Architecture names a StageStore the code replaced with StageContext" \
  --id "$(bd config get issue_prefix)-design-007-stagestore" \
  --deps discovered-from:<bead> --spec-id docs/designs/DESIGN-007-stage-context.md \
  -l design-revision -p 2 -s deferred
bd update "$(bd config get issue_prefix)-design-007-stagestore" --parent <epic>
bd dolt push
```

The body says what the document says, which document, what was found instead with file and line,
why it was not amended, and what the task did about it. A disagreement the task could not finish
under is a worker failure and an escalation, never a revision bead: the human decides, and the run
stops there.

**At session end, the epic carries the hand-off.** When any `design` bead was filed, one
`bd comment` on the epic — on the bead itself at single-bead scope — names every revision bead, so
the epic's own record shows the drift:

```bash
bd comment <epic> "design-revision: booking-design-007-stagestore, booking-design-007-retry — run /codefall-design DESIGN-007"
```

## Amendments

A worker's `amended` list names each upstream document it edited in its branch — a `Draft` or
`Ready` design or spec, text only, nothing that moves work. The root reads every list at the wave
boundary, before the next wave is claimed:

- **Two workers amended the same section of one document.** Keep the one whose branch is lower in
  the stack, or the first to open its PR when the branches are independent; revert the other on its
  branch with a commit that names the kept amendment; say so in the report. An overlap the root
  did not catch surfaces as a conflict at step 7's integration merge, which is reported and never
  resolved silently.
- **A spec was amended.** Regenerate the requirement's tracker issue, the existing-requirement case
  of the *Refreshing* sequence in the spec's tracker profile, as the mirror reference beside this
  file says. The amended text is on the worker's branch and nowhere else: read it with
  `git show origin/<branch>:<path>`, never from the primary checkout, which does not carry it.
- **The close reason** names the amendment beside what was verified, so the bead's record says the
  document moved with the work.
- **Every amendment is recorded as a closed bead.** One per `amended` entry, labelled
  `design-amended`, with `--spec-id` the document's path and the same `discovered-from` edge a
  revision bead carries; the body says what the document said and what the code needed, and the
  close reason names the branch and commit that carry the amendment. It adds no work to the graph
  and `codefall-design` never lists it; it exists so `bd list -l design-amended --spec <path>`
  answers which documents were wrong, where, and how often. It is parented under the epic like
  every other bead a delivery files, so `bd children <epic>` is complete; the chart counts every
  child the way `bd epic status` does, this one included.

```bash
bd create "DESIGN-007 § Architecture: StageStore renamed to StageContext" \
  --id "$(bd config get issue_prefix)-design-007-stagecontext-amended" \
  --deps discovered-from:<bead> --spec-id docs/designs/DESIGN-007-stage-context.md \
  -l design-amended
bd update "$(bd config get issue_prefix)-design-007-stagecontext-amended" --parent <epic>
bd close "$(bd config get issue_prefix)-design-007-stagecontext-amended" \
  -r "amended on feat/booking-DESIGN-007-T2-wire-context @ <sha>: § Architecture now names StageContext"
bd dolt push
```

Amendments are reported at close-out by document and PR, each with its `design-amended` bead, in
their own bucket beside the discovered work. Under single-bead scope the same reading happens once,
at the worker's return.

## Session end

**The epic's log.** At epic scope, each of `codefall-implement`, `codefall-review`, and
`codefall-test` appends one line to the epic's notes at its close-out, so `bd show <epic>` reads as
the delivery's history:

```bash
bd update <epic> --append-notes "round 2 (implement) [2026-10-01]: 3 tasks built, PRs #103 #104"
```

The round, the verb in parentheses, the date in brackets, a colon, then what happened in words:

```
round 1 (implement) [2026-09-30]: 7 tasks built, PRs #101 #102
round 1 (review) [2026-09-30]: 3 fixed, 2 deferred as children
round 1 (test) [2026-09-30]: FAIL 2/7, 1 child filed
```

Always `--append-notes`; plain `--notes` replaces the whole log. The round is `metadata.round` on
the epic, which `bd epic status --json` returns beside the child counts the chart prints.

Final `bd dolt push`. Every write already landed when its command ran; the database is the handoff,
and the next session opens with `bd ready`.
