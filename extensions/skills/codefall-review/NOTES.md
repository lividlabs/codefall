# codefall-review — where it came from

Why `SKILL.md` looks the way it does: what was taken from elsewhere, what was rejected, and what was
tried and dropped. None of this is instruction — the skill is the instruction. This exists so nobody
re-adds something that was removed on purpose.

## Taken

**OpenCode's built-in `/review`** — target inference from the argument's shape, the rule that diffs
alone are not enough, and most of Calibration: be certain, no hypothetical edge cases, state the
conditions, do not be a zealot about style, no flattery. That template is the best short statement
of review discipline in the open and there was no reason to write a worse one.

What it does not have, and this skill added: document targets, the lens decomposition, structured
findings, and everything after the review — triage, fixes, a file.

**Anthropic's `pr-review-toolkit`** — one pass per question rather than one reviewer looking for
everything, and the lenses its agents cover: silent failures, test coverage, type design, comment
accuracy, simplification. Dropped from it: the "Strengths" and "Positive Observations" sections,
which are flattery with a heading.

**Cross-model review wrappers** — shelling out to another harness's headless mode rather than trying
to switch providers inside the host. No harness offers a per-subagent provider switch, and a
session-wide base URL override changes the host model too, so a subprocess is the only way.

**`codefall-upgrade`** — the report as a deliverable, and stopping after it.

## Rejected

**An external reviewer that edits.** Codex and the rest can write, and some cross-model tools let the
second model fix what it found. A subprocess writing files bypasses the `PreToolUse` hooks that are
codefall's enforcement and the checkpoints that make a session reversible, so a change it made would
be neither guarded nor undoable. Every external reviewer runs read-only.

**The reviewer fixing what it found.** The point of a second reading is different blind spots. A
model that finds and fixes grades its own work on the next pass. The reviewer is a subagent or
another harness; the session triages and fixes.

**A lens filter as a skill parameter.** `focus=<lens,…>` meant getting the argument right before
seeing what it would have done. The confirmation step names the lenses that will run and asks what
to drop, which puts the choice where the information is.

## Tried and dropped

**Rounds.** An earlier draft numbered reviews per target — `round-001`, `round-002` — reconciled the
previous round's findings against the current code, and marked them `resolved` or `stale`. It bought
stable finding ids, a `status` lifecycle, staleness rules, and reconciliation logic. All of that
only matters when findings sit unresolved between reviews, and they do not: a review ends with every
finding fixed, dismissed, or deferred. One invocation, one file, and a later review of the same
thing is a new review.

**Review as a read-only verb.** The first draft changed nothing at all and left every fix to the
user. It kept the reviewer and the fixer apart by construction, and it meant a review of twelve typos produced
twelve things to type by hand. The subagent boundary gets the same separation without that cost.

**Inline `> [!REVIEW]` callouts in documents.** Document findings were going to be written into the
document beside the passage they were about. That is an edit to the reviewed artifact made before
anyone agreed to it. Findings go in the findings file, and accepted ones become real edits after
triage.

**Filing findings as beads unasked.** A finding that outlives the review is offered as a bead and
never filed on the review's own initiative: the graph is `codefall-design`'s, and a review that
quietly adds work to it is making a decision that is not its own. An earlier edition kept deferred
findings in the file alone, with `design-revision` beads the one exception. That left a deferred
finding whose cause was the code in a file `.ignore` hides from search, where no verb read it again
and the person running the chain had to remember it. Under ADR-013 every finding the user defers is
offered as a `deferred` child of the epic — a revision bead for one the design caused, a code bead
for one the code caused — so the next round of `codefall-implement` claims it and the chart shows
it. The offer is the same; what changed is where a taken offer lands.

**Specific commits, merged pull requests, merged branches, superseded ADRs.** All were targets in the
first draft. Reviewing history produces findings about code that has moved on, and there is nowhere
for a fix to land. Every target is something live.

**A self-contained review packet.** The external reviewer was going to be handed a copy of
everything it needed — the diff, the full files, the conventions, the upstream document — because a
subprocess starts with no context. It runs inside the repository instead and reads what it needs,
which is less to build and less to go stale.

## Why the rules are shaped this way

**Four subagents in parallel.** Several passes each looking for one thing find more than one pass
looking for everything. Each reads the material once and asks one coherent set of questions.

**An external harness gets one call.** Each invocation pays full process startup and re-reads the
repository; four subprocesses asking four questions about the same files buys the decomposition at
several times its worth.

**Only `fixed` and `deferred` findings post to a pull request.** A `fixed` one tells a reviewer what
changed and why; a `deferred` one is real work someone chose not to do now, which is what a PR
thread is for. A `dismissed` one was judged wrong.

**The target decides where fixes land.** Reviewing PR #51 from a feature branch puts the fixes on
#51's branch. The new-worktree case exists because a document or path on the default branch has
nothing to land on, not because of where the session started. Uncommitted work is never moved
because a new worktree cannot contain the changes in this one.

**Diffs alone are not enough.** Code that looks wrong in isolation is often correct given what
surrounds it, and code that looks fine in a diff is often wrong given what it replaced.

**`.ignore` rather than `.gitignore`.** Ripgrep honours `.ignore` and git does not, so the reviews
directory is committed while every harness that searches through ripgrep skips it. The objective is
keeping old findings out of unrelated searches, not secrecy. The offer to add the line lives in this
skill because this is where a missing line costs something.

**The default branch is resolved once.** Branch diffs and `revision.base` must agree, and they will
not if each derives its own.
