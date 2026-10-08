# codefall-implement — where it came from

What this took from elsewhere, what it deliberately did not, and why the rules are shaped the way
they are. None of this is instruction — the skill is the instruction. This exists so nobody re-adds
something that was removed on purpose.

## Taken

**From the dev-implement skill pair** that preceded this extension: the worker-prompt pattern with
strategy encoded in `BASE_REF`/`PR_TARGET`, the go-block, wave execution with verified worker
results, the recovery table, and the humans-merge-`main` rule with its hook.

**From Kiro**: the one-task-at-a-time worker discipline and read-everything-first; drift
reconciliation at resume, which became the recovery table.

**From beads' own docs**: close-at-done with gates carrying the merge seam.

## Dropped

**From the dev-implement pair**: `MAX_STACK_DEPTH = 4` — the depth cap solved a cosmetic problem
and a 12-PR stack works; the two-skill split — one skill decides scope at run time; and the board
scripts — beads replaced the project board.

**Parallel stacks**, the original default, with spec-kit's rule that parallel means file-disjoint
and the file-scope prediction and hotspot rule that served it. In practice the classifier never
picked it, the prediction machinery was the largest part of the landing reference, and two stacks
each targeting `main` doubled every question about where a later round's bead goes. Two strategies
remain: the serial stack, and the epic branch for parallel waves (ADR-013).

**Restacking the stack bottoms onto `main` at integration.** It arrived with the import and was
never recorded here. Rebasing the bottom alone left every link above based on commits that no
longer existed, so the stack above it stopped merging cleanly. A scratch-worktree trial merge
replaced it for a week; then GitHub's own stacked pull requests (public preview, 2026-07-30) made
both unnecessary. GitHub rebases the layers above a merged one, and `gh stack rebase` cascades a
lower-layer fix. Integration now asks GitHub whether each layer is mergeable and nothing is rebased
or force-pushed by hand (ADR-014).

**From Kiro and spec-kit**: checklist gates and the converge audit — review is another verb.

## Adapted

**Gates attach to a landed child bead**, not to dependents and not to the epic. A stacked dependent
deliberately builds on its parent's branch, so gating it would block exactly what stacking allows.
`bd gate create` rejects an epic, and an epic already refuses to close while children are open, so
one extra child task carries the gates and the epic follows it.

**Workers do not self-claim with `bd ready --claim`**, diverging from beads' multi-agent docs. That
pattern arbitrates races between peer agents pulling a shared queue; this skill's topology is
dispatcher and workers, the root assigns work top-down, and the default embedded database is
single-writer besides. That is why every `bd` write is the root's: architecture, not etiquette.

## Why there is no depth cap

A deep stack used to cost a muddy three-dot diff on the top PR until the chain drained. GitHub's
stacked pull requests show each layer its own diff, so that cost is gone; what remains is
wall-clock, disclosed at the go gate. Any DAG serializes into a single stack by topological order,
so the single stack is always available and costs nothing in structure.

## Why workers are verified, never trusted

A worker can run its whole verification and stop without pushing, and the harness still reports it
completed. So link N+1 launches only after the root has confirmed link N's branch is on the remote
and its PR exists.

## Why the go gate is the only stop

After go, waves proceed on their own and failures are the only mid-run stop. That absence is what
makes an overnight run possible; the user can always interrupt. Background workers cannot answer
permission prompts, which is why the gate states the permissions condition and offers single-task
mode when it fails.

## Why closed means done, not merged

That is beads' own semantics — "closing a beads issue means 'work is done' but the code may still
be on a feature branch" — and it is what lets a stacked dependent start the moment its parent's
branch is pushed. The merge is tracked separately by the gates. The epic is different: it closes
when the graph is empty and the code is on `main`, and the graph is not empty while a child review
or test filed is open. Done for the delivery is read from the graph, never declared (ADR-013).

## Why discovered work is a child of the epic

A finding that lived beside the graph, in a findings file, or in a tracker issue alone was invisible
to the next verb's start, and the person running the chain had to remember it. As a child of the
epic it holds the epic open, shows on the chart, and is claimed by the next round like any task.
It is created `deferred` so the round still running does not pick it up: `bd ready` skips
`deferred`, and the round boundary is where it opens. `plan-amended` beads are parented too, for
a complete `bd children`, and excluded from the counts because they are records, not work.

## Why the epic's round is the one metadata implement writes

The round has to live somewhere every verb can read and the chart can print, and only one verb may
bump it or a review that files both a code child and a revision bead produces two bumps. Implement
owns the moment a round starts — the go gate of a re-entry run — so it writes `round` there and
nowhere else; plan's Revise mode adds children and bumps nothing.

## Why the mirror is coarse

Requirement sub-issues close with the parent, never individually, because beads carry plan refs,
not requirement IDs, and a mirror that guesses is worse than one that is coarse.
