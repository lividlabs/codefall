# codefall-fix — where it came from

What this took from elsewhere, and what it deliberately did not, so nobody re-adds it. None of this
is instruction — the skill is the instruction.

## What fix restates

Fix is `codefall-design` at tier 0 followed by `codefall-implement` on one bead, written as one
procedure. It links the reference files it can use as they are, so those reach it without anyone
editing it: design's `reference/bugs.md` and `reference/beads.md`, and implement's
`reference/done.md`, `reference/beads.md`, and `reference/workers.md`.

The rest it restates, because it lives in a `SKILL.md`. **A change to any of these checks fix in
the same pull request** (`extensions/skills/AGENTS.md` holds the rule):

| Source | What fix restates | Where in fix |
| --- | --- | --- |
| design, *Scale the artifact to the work* | The tier-0 conditions | Scope |
| design, *ADRs* | The ADR trigger | Scope |
| design, step 2 | The bug report as a target | Step 2 |
| design, step 3 | Reading the ADRs, the code, and the graph | Step 2 |
| design, step 7 | The tier-0 confirmation: the bead, its body, its criteria | Step 4 |
| design, *Rules* | A ticket must not change under someone holding it | Rules |
| implement, step 1 | The Beads stop, the checkout lines, the `test=` line | Steps 1 and 3 |
| implement, *The go gate* | The single-bead plan approval and the permissions condition | Step 4 |
| implement, steps 5, 6, and 8 | Claim, one worker, verify, close, discovered work, the report | Steps 5 to 7 |
| implement, *Rules* | Merges, scope, tests, the harness, verifying workers, pushing | Rules |
| implement, `reference/mirror.md` | The pull request's `Fixes` and `Relates to` lines | Step 6 |

## Taken

**The single confirmation.** Running design at tier 0 and then implement on its bead asked twice —
the beads, then the plan — with a preflight and a report each time. Fix shows both in one block.

## Dropped

**Following design's and implement's `SKILL.md` files with overrides.** The first version did this
to keep one copy of every rule. It asked the agent to hold about eight hundred lines of two
procedures and to know which of their stops fix replaced, and a compaction restores only the invoked
skill's own `SKILL.md`, so the procedures were lost exactly where a long build runs. A procedure of
its own is restored whole.

**A CI check that the restated parts have not moved.** Whether a change to design or implement
matters to fix is a judgment, so the rule is in `AGENTS.md` and this table is the map.

**Model invocation.** "Fix this" is said in every session; an agent that could run fix would turn a
passing edit into a bead, a worktree, and a pull request.

**The offers of `/codefall-specify` and `/codefall-report`.** A small change proceeds from what it
was given; the bead carries the reproduction.
