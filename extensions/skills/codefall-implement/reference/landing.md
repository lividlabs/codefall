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

- **There is no depth cap.** A deep stack costs nothing a reviewer sees: each layer's PR shows only
  its own diff.
- **Any DAG serializes into a single stack.** Topological order makes one branch-atop-branch chain
  out of any graph, fan-in included, so the serial stack is always available.
- **Parallelism is the epic branch's.** A wave is every bead the graph has ready at once, built by
  concurrent workers off the same base; the wave boundary is where the root merges and the next
  wave starts.
- **A serial stack is a GitHub stack.** The worker opens its PR with `--base` on the layer below;
  the root links it with `gh stack link <bottom-pr> <this-branch>` when the second PR opens and
  `gh stack link <stack-number> <this-branch>` after that, per
  `../../../../.codefall/shared/stacks.md`. One PR is not a stack; the second link creates it.

## Integration

**Prediction is verified at integration, never trusted.** When the frontier is empty, ask GitHub:
`gh pr view <number> --json mergeable,mergeStateStatus,statusCheckRollup` for every open layer.
Branch protection and the checks run on each layer, so a clean answer on every one is the
verification. Report the stack per *Pointing at a stack* in `../../../../.codefall/shared/stacks.md`
— the link to the top PR, and that merging it on GitHub lands every layer — and stop.

A layer GitHub reports as conflicting, or whose checks fail, is fixed on the branch that owns the
change, and the fix is cascaded with `gh stack rebase` then `gh stack push`, which pushes each
branch with a lease. **Nothing is rebased or force-pushed by hand.** The extension does the cascade,
GitHub does the rebase on merge, and a reviewer's view of each layer stays its own diff throughout.

## Later rounds

A delivery's later round files its beads as children of the epic and builds them the same way,
on top of what round one left:

- **Serial stack:** the new bead branches from the stack's one open top — the open PR from
  `feat/<epic>-*` that no other open PR bases on, read from
  `gh pr list --state open --json headRefName,baseRefName --limit 200` at the go gate — or from
  `main` when every PR has merged. Later-round beads stack on each other in claim order, so the
  stack keeps one top, and each is linked into the stack as it opens.
- **Epic branch:** the new bead branches from `epic/<id>-<slug>` as any wave does. The aggregate PR
  that already exists is reused and its gate left alone; the report says its diff moved.

Nothing below the new link is touched by hand.

## The diagrams

The go gate renders the plan as a branch diagram built from the actual graph. Serial stack:

```
main ── T1 ── T2 ── T3 ── T4 ── T5 ── T6
         one stack of 6 · merging T6 lands all six
```

Epic branch:

```
main ── epic/booking-DESIGN-007-stage-context
          ├── T1 ─┐
          ├── T2 ─┼── T4    fan-in: T4 needs T1 + T2
          └── T3 ─┘         wave 1: T1 T2 T3 · wave 2: T4
```
