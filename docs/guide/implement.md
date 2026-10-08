# /codefall-implement

`/codefall-implement` builds the tasks that [`/codefall-plan`](plan.md) put into Beads. It writes
the code and the tests for each task, checks the result against the task's acceptance criteria, and
opens a pull request. Run it once a plan's beads exist.

```
/codefall-implement PLAN-007
```

The examples on this page continue the trip planner from the [README](../../README.md#how-it-works).
PLAN-007 created an *epic*, `trips-PLAN-007`, which is the bead that groups the plan's tasks, and two
task beads, `trips-PLAN-007-T1` and `trips-PLAN-007-T2`. A *bead* is one task in Beads, the tracker
that stores its data in your git repository.

## What can you point it at?

The argument decides how much work the run takes on:

| Argument | What the run builds |
| --- | --- |
| A bead, such as `trips-PLAN-007-T1` | That bead alone |
| An epic, such as `trips-PLAN-007` | Every task under the epic, until none is left to build |
| A plan, such as `PLAN-007` | The plan's epic, the same as naming the epic |
| Nothing | Nothing yet: the run shows the work that is ready, grouped by epic, and asks |

An epic means working through the whole task graph. Each bead records which beads must finish before
it can start, and a bead is *ready* when all of those have finished. Closing one task makes the
next ones ready. The run builds the ready tasks in *waves*, where a wave is the set of tasks that
are ready at the same time. Each task is built by a background *worker*, a separate agent working in
its own *worktree*, which is a separate working copy of the repository. The run continues until no
task is left to start.

A plan whose task table still says `Staged. Not yet in Beads` has no graph to build, and the run
refuses it and points at `/codefall-plan`.

## What you approve before it starts

One approval starts the run. Before anything is built, the run shows you a single summary, called
the go gate, that lists everything it will do:

- the landing strategy, described below, with the reason it was chosen;
- a diagram of the branches it will create;
- the waves, and the model proposed for each task;
- what it will claim in Beads;
- the consult order a failure will be put to, described below;
- when a vision sits behind the work, that approving will set the vision's status to `Active`.

After you approve, only a failure stops the run. Because the run asks nothing in the middle, it can
build a large plan overnight.

## What happens when a task fails?

A failed task gets one second opinion and one retry before it reaches you. The *root*, which is the
session coordinating the run, sends the failure, the bead, and the options it sees to the project's
*consult* agents. Those are the second-opinion readers listed under `consult` in
`.codefall/settings.json`; with nothing configured, the consult is a fresh subagent of your current
coding agent. The root then starts a new worker at a higher effort or with a stronger model, with
the failure reason and the consult's answer added to its instructions.

When the task fails a second time, the root consults once more and then stops and brings the
failure to you, with the analysis in front of you. You can fix it by hand, skip the bead, or abort
the run. The consult keeps a run from stopping on a problem that a second reading would have
solved.

## How work lands

Work reaches `main` in one of two ways, and the go gate tells you which the run chose.

**A serial stack is the default.** The run builds every bead in dependency order, one at a time. Each
task gets its own branch, created on top of the previous task's branch, and its own pull request.
The run links the pull requests into a GitHub stack as they open. A *stack* is a chain of pull
requests in which each one targets the branch of the one below it. For PLAN-007, T1's pull request
targets `main`, and T2's targets T1's branch.

**An epic branch is the alternative.** The run uses it for work that must not reach `main` in pieces,
or when you want workers to run in parallel. The run creates one branch for the epic, workers branch
off it in waves, and the root merges each worker's pull request into the epic branch at the end of
each wave. One combined pull request then carries the whole epic to `main`.

A stack has no depth limit. Each layer's pull request shows only its own changes, and merging the
top of the stack on GitHub lands every layer below it. When you merge one layer alone, GitHub rebases
the layers above it. Nothing is rebased or force-pushed by hand: a fix to a lower layer is carried up
the stack with `gh stack`.

Document pull requests never stack. Each one branches from `main`, and the project's GitHub Action
merges it on its own, as [`docs/landing-documents.md`](../landing-documents.md) describes.

## When is a task done?

A task bead closes when it is done, and done does not mean merged. A task is done when three things
are true:

- its acceptance criteria hold;
- the project's checks pass;
- its pull request is open.

This is how Beads itself defines a closed bead, and it lets the next task in a stack start as soon
as the previous task's branch is pushed.

Merging is tracked separately, by *gates*. The run creates one extra bead under the epic, the
*landed bead*, named `trips-PLAN-007-MERGED`. Every pull request the run opens adds a gate to it,
and the landed bead cannot close until every gated pull request has merged. Because the epic cannot
close while the landed bead is open, the epic stays open until you have merged everything. At the
start of the next session, `bd gate check` turns your merges into bead state.

Tests are part of done. The worker writes the tests the plan named and any others the work turns out
to need. The end-to-end run across the epic, regression testing, and retesting in a fresh context
belong to [`/codefall-test`](test.md).

## When is the whole delivery done?

A *delivery* is all the work on one epic, from the plan to the merge. It proceeds in *rounds*. Each
round is a run of `implement`, then [`/codefall-review`](review.md), then
[`/codefall-test`](test.md), all against the epic's work before you merge.

Review and test each end with one question: "I found N problems. Fix them all?" Each problem you
take becomes a new bead under the epic, and the same session then runs `/codefall-implement` on the
epic to build them. You type one command, and the next round starts.

Every skill in the delivery prints the same progress line when it starts and when it finishes. The
line shows the epic, the round, and how many of the epic's beads are closed:

```
trips-PLAN-007 · round 1 · 2/3 ███████████░░░░░░
```

Here both tasks are closed, and the third bead is the landed bead, which closes when you merge. Each
skill also adds one line to the epic's notes when it finishes, so `bd show trips-PLAN-007` reads as
the history of the delivery:

```
round 1 (implement) [2026-10-08]: 2 tasks built, PRs #101 #102
round 1 (review) [2026-10-08]: 3 fixed, 2 deferred as children
round 1 (test) [2026-10-08]: FAIL 2/7, 1 child filed
```

The delivery is done when nothing under the epic is open except the landed bead and the last test
run passed. At that point, only the merge remains.
[ADR-013.4](../adrs/ADR-013.4-deliveries.md) records the rule.

## Who merges?

A person merges every code pull request. The run ends at open pull requests and gives you one link,
to the top of the stack, because merging that pull request on GitHub lands every layer. The run never
hands you a merge command or a list of pull requests to merge in order. Codefall also installs a
hook, a check the coding agent runs before each command, that refuses any merge or push to `main`.

## What happens when the plan turns out to be wrong?

A worker that finds the plan or the spec disagreeing with the code fixes the document instead of
working around it. Three cases are possible:

- **The worker can still finish its task.** It amends the plan's or the spec's text in its own
  branch and names the amendment in its pull request. When the spec changed, the root regenerates
  the spec's GitHub issue.
- **The worker cannot amend the document.** This is the case when the fix would move work, such as
  a task row or a criterion a bead cites; when the document is frozen; or when you declined the
  amendment. The disagreement is filed as a `plan-revision` bead. The final report lists these
  beads separately from code follow-ups and tells you to run `/codefall-plan` on that document.
- **The task cannot be finished under the disagreement.** The run stops.

[ADR-008](../adrs/ADR-008-upstream-amendments.md) records the rule, which applies to any document
further up the chain, not only the plan.

## When are test cases written?

A worker writes a task's test case before it writes the code. A *test case* is a Markdown file that
describes how to check a behavior through the running product; [`/codefall-plan`](plan.md) names it
in the bead's acceptance criteria. The worker writes the case from those criteria and nothing else:
not the spec beside it, not the code it is about to write, and not its own pull request text. Then,
when the case's modality calls for one, it writes the generated test that the project's runner will
execute.

The case counts toward done, but running it belongs to [`/codefall-test`](test.md). A bead whose
case needs a test runner the project has not set up does not start. The run says so and names
`/codefall-equip`, because setting up a runner is its own pull request and never rides along inside
a task's pull request. Beads that need only unit tests carry on.
