# Landing strategies

How work reaches `main`. Read at step 4, when classifying the run.

## Contents

- The two strategies
- Integration
- Later rounds
- The diagrams

## The two strategies

The graph's shape and the user's wish pick one; the user can overrule it at the go gate. The
project's `AGENTS.md` constrains the choice before either speaks, `CUSTOMIZE.md` tunes it within
those constraints, and a conflict between the two is drift — show the difference and ask, never
silently pick. A `CUSTOMIZE.md` that still names *parallel stacks*, a strategy this skill no longer
has, is read as the epic branch, and the report says so.

| Strategy | When | Shape |
| --- | --- | --- |
| **Serial stack** (default) | Any graph, when nothing below applies | Every bead in topological order, one branch atop the previous, one PR each; the bottom PR targets `main`; workers run one at a time |
| **Epic branch** | Work that must not land on `main` in increments, or the user wants parallel waves | Workers branch off `epic/<id>-<slug>` in waves, PRs target it, the root merges them at each wave boundary, one aggregate PR to `main` |

- **There is no depth cap.** A deep stack's cost is disclosed at the go gate, not capped against.
- **Any DAG serializes into a single stack.** Topological order makes one branch-atop-branch chain
  out of any graph, fan-in included, so the serial stack is always available.
- **Parallelism is the epic branch's.** A wave is every bead the graph has ready at once, built by
  concurrent workers off the same base; the wave boundary is where the root merges and the next
  wave starts.

## Integration

**Prediction is verified at integration, never trusted.** When the frontier is empty, merge the
stack's top branch, or the epic branch, onto current `main` in a scratch worktree
(`git worktree add <dir> origin/main`, then `git merge --no-commit origin/<branch>`), run the
project's verification there, and remove the worktree. Nothing is pushed. A clean merge with green
checks is reported with the merge order and the run stops.

A conflict, or a check that fails only on the merged tree, is reported with the files it touches,
and the offer is to **merge forward**: `git merge origin/main` into the lowest branch the conflict
reaches, resolved in the open, committed as an ordinary merge commit, and pushed to that branch.
The links above it need nothing; git merges identical content cleanly when their turn comes, and
the merge commit disappears under squash merge. **Nothing is ever rebased or force-pushed.** A
branch whose PR is open keeps every commit it had, so a reviewer's comments stay anchored and a
later round builds on what is there.

## Later rounds

A delivery's later round files its beads as children of the epic and builds them the same way,
on top of what round one left:

- **Serial stack:** the new bead branches from the stack's one open top — the open PR from
  `feat/<epic>-*` that no other open PR bases on, read from
  `gh pr list --state open --json headRefName,baseRefName --limit 200` at the go gate — or from
  `main` when every PR has merged. Later-round beads stack on each other in claim order, so the
  stack keeps one top.
- **Epic branch:** the new bead branches from `epic/<id>-<slug>` as any wave does. The aggregate PR
  that already exists is reused and its gate left alone; the report says its diff moved.

Nothing is force-pushed to a branch whose PR has merged, and nothing below the new link changes
shape.

## The diagrams

The go gate renders the plan as a branch diagram built from the actual graph. Serial stack:

```
main ── T1 ── T2 ── T3 ── T4 ── T5 ── T6
         one stack of 6 · merges drain bottom-up
```

Epic branch:

```
main ── epic/booking-DESIGN-007-stage-context
          ├── T1 ─┐
          ├── T2 ─┼── T4    fan-in: T4 needs T1 + T2
          └── T3 ─┘         wave 1: T1 T2 T3 · wave 2: T4
```
