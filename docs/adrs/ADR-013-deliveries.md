# ADR-013: Deliveries

## Status

Accepted — 2026-09-30

## Context

The chain runs spec → design → implement → review → test, and each verb hands what it finds forward
in a different form. Three of those forms left the epic's graph. `implement` filed a code discovery
as a task beside the graph, with a `discovered-from` edge but no parent. `review` applied accepted
findings in the session and left a deferred finding whose cause was the code in the findings file
alone, under a directory `.ignore` keeps out of search. `test` filed a tracker issue on the user's
word and stopped, and `fix` later opened a standalone bead for it. `test` was also written to run
after the merge: implement's scope said `codefall-test` owns what comes after the work lands.

So the task beads burned down to zero, the epic looked finished, and the work review and test had
found sat where no verb reads at its start. In a real run the person lost track of the second of two
findings. Nothing named the unit between design and merge either, so "session" meant both one
conversation and the whole effort, and the context cost of running every verb in one conversation
was treated as the price of not losing state.

Two further things surfaced while settling this. The default landing strategy, *parallel stacks*,
had never been picked in practice, and the file-scope prediction and hotspot rule that served it were
the largest part of the landing reference. Its integration step rebased each stack bottom onto
`main`, which in a stack leaves every link above based on commits that no longer exist, so the
stack above the bottom stopped merging cleanly. That step arrived with the plugin import on
2026-09-10 and was never recorded in the skill's notes.

The principle that decides all of it: **the work is not done until it is working, and done is read
from the graph.** Beads already refuses to close an epic while a child is open, and `implement` on
an epic already walks `bd ready --mol <epic>` until the ready set is empty and reconciles a re-run.
The burndown existed. The findings were not on it.

## Decision

### Vocabulary

A **delivery** is the work on one epic from the design's graph to the human merge, taken in
**rounds** of implement, review, and test. It ends when the epic has no open children and the last
test run against it passed. A **run** is one verb invocation; a **session** is one conversation.
A `/clear` between verbs is the intended way to run a delivery: every handoff is a bead, a document,
a report, or a pull request, so a fresh session loses nothing.

### Every finding is a child of the epic

Every finding that survives triage becomes a bead with `--parent <epic>` and a `discovered-from`
edge to what found it: implement's code discoveries, review's deferred findings whose cause is the
code, test's real bugs, and `design-revision` beads. It is created with status `deferred`, so
`bd ready` does not hand it to the round that is still running, and the next round's go reopens and
claims it. Nothing is filed unasked; the offers stay where they were. A bead with no epic gets the
edge alone. `design-amended` beads are parented too, for a complete `bd children`, and excluded
from every count because they record an amendment and are never work.

### What closes when

A task bead still closes at done — criteria verified, checks green, pull request open — because the
close is what releases the next link in a stack. The epic closes when the graph is empty and the
code is on `main`, and the graph is not empty while a child review or test filed is open. Done is a
query: `bd children <epic>` with nothing open but the landed bead, and the last test run against the
epic passed.

### Test runs before the merge

`test` takes an epic as a target: the cases the epic's children name, plus the suites the changed
files reach, run on the epic's branch or the stack's top, checked out in the primary checkout the way
`review` already checks a branch out for its fixes. Every other target behaves as before, so an ad
hoc case or a full suite before a release runs at any time.

### The round and the chart

The round number is metadata on the epic, written only by `implement` at the go gate of a re-entry
run, so no bounce produces two bumps. Every verb in a delivery prints one line from the shared
`delivery.sh` at its start and in its close-out: the epic, the round, closed over total children,
a bar, open children, and the merge gates. A verb's close-out also records in a `bd comment`
anything decided in the conversation that no artifact holds, and then says it is safe to `/clear`.

### Two landing strategies

*Parallel stacks* is removed, with its file-scope prediction, hotspot rule, and two-stack diagram.
The **serial stack** is the default: every bead in topological order, one branch atop the previous,
workers one at a time. The **epic branch** is the other: parallel waves off `epic/<id>-<slug>`,
merged at wave boundaries, one aggregate pull request. A later round's bead branches from the
stack's one open top or from the epic branch; nothing below it changes shape.

### Integration never rewrites a branch

In place of the rebase, integration merges the stack top or the epic branch onto current `main` in
a scratch worktree, runs verification there, and pushes nothing. On a conflict it offers to merge
`main` forward into the lowest affected branch as an ordinary merge commit, resolved in the open.
Nothing is rebased or force-pushed.

## Consequences

- The epic and the spec's mirrored tracker issue stay open while any finding is open. That is the
  intent: a finding review or test took seriously is part of the feature.
- `bd children <epic>` answers whether a delivery is done, and the SessionStart prime already
  reports it.
- `design`'s Revise mode compares only beads named `<prefix>-DESIGN-NNN-Tn` against the table and
  lists the other children as the delivery's discovered work, never as dropped rows.
- Review's findings record and test's run record gain optional fields for the epic, the branch,
  and the bead; older records still validate.
- `fix` refuses a bead whose epic still has open merge gates and names `implement`, because a fix
  lands on `main` and a delivery's work does not.
- A worker's result and prompt are unchanged; the parent is the root's to set.
- `implement` writes one piece of bead metadata, the epic's `round`, as the stated exception to its
  rule against writing any.

## Related

- [ADR-007](ADR-007-test-cases.md) — test cases named in a bead's criteria.
- [ADR-008](ADR-008-upstream-amendments.md) — amendments and `design-revision` beads; this ADR
  parents those beads under the epic and leaves the rest of ADR-008 standing.
- [ADR-004.2](ADR-004.2-skill-length-guidelines.md) — the length budget the skill edits here had to
  fit under.
