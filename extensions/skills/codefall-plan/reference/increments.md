# When one spec is more than one delivery

Read at step 4, with the tier, when the work could be more than one delivery; again at step 6, once
the tasks are staged; and for the Split mode. These are guidelines: the size is a judgment the run
states with its reason, and the cut is the person's.

## Contents

- What a delivery costs
- Saying it
- The cut
- The documents
- The graph
- Split mode
- The report

## What a delivery costs

A plan is one delivery: one epic, run by `codefall-implement` through rounds of build, review,
test, and triage until a person merges it. A delivery goes best when it finishes in a few hours of
agent time, four to six as a starting point, with one round or two: the person reads one ledger
and one stack of pull requests, and what review and test find is small enough to build in the next
round. A project that knows its own figure states it in
`.codefall/skills/codefall-plan/CUSTOMIZE.md`.

What drives the cost, in the order it matters:

| Signal | What it drives |
| --- | --- |
| How many tasks | Agent time: every task is a worker run, a test case where it carries one, and a pull request |
| How many components the tasks touch, and how many tasks are verified through the wired product | The rounds after the first: review and test find more across more surfaces, and each finding is a bead the next round builds |
| How deep the dependency chain is, the waves | Wall-clock: waves run one after another, however many workers there are |

The project's own deliveries are the reference, not a number in this file. Beads records when
every epic was created and closed, its `round` metadata, and its children:

```bash
bd list -t epic --all --json                      # created_at, closed_at, metadata.round
bd list --parent <epic> --all --limit 0 --json    # its children, Tasks rows and the rest
```

Read the closed epics and say what they ran: tasks, rounds, and elapsed time each. A project with
no closed delivery has nothing to compare against: say so, judge from the signals, and say the
figure the judgment assumed. No count here or in `CUSTOMIZE.md` is a threshold the run applies;
each is a reference point for a judgment the run states.

## Saying it

At step 4, in the sentence that names the tier, with a specific cut and the reason:

> Fourteen tasks in five waves across the booking store, the API, and the web app, four of them
> verified through the product. The three deliveries this project has closed ran five to eight
> tasks in two rounds each. This looks like two deliveries: REQ-01 to REQ-03 first, which ships
> the export on its own, then REQ-04 and REQ-05, which share it.

Then do what the person says. Which requirements ship together is a product decision, per
`../../../../.codefall/shared/decisions.md`: a person present answers it; with nobody present it is
consulted once, and the answer taken is recorded as that file says. A declined cut is recorded the
same way, and the one plan proceeds. Say it again at step 6 only when the staged count changes the
judgment either way: a cut that no longer looks needed, or one delivery that now looks like two.

## The cut

- **Whole requirements, in ship order.** A requirement is what a consumer observes, so it is the
  smallest thing that ships; `codefall-specify` cut it that way, and a plan does not cut one in two.
- **Every prefix of the order leaves the product whole.** After the first increment merges, and
  after each one after it, a consumer can use what shipped, nothing visible does nothing, and no
  surface is half-changed. The first increment carries what the later ones depend on.
- **Cut at build boundaries across increments as across tasks.** A later increment may delete what
  an earlier one left beside the old thing; an earlier one never deletes what a later one's code
  still uses. *Cut at build boundaries* in `beads.md` has the two fixes.
- **A foundation-only first increment**, a data model, a migration, an interface nothing calls yet,
  is one the person takes knowingly: name it as one that ships nothing a consumer sees.
- **When no order passes**, because nothing in the spec works until all of it does, there is no
  split. Say so, say how long the one delivery looks, and plan it as one.

## The documents

One plan per increment, written in this run, numbered consecutively in ship order: `PLAN-007`,
`PLAN-008`. Each is a plan like any other, with its own tier judged on its own work, its own
sections, and its own Tasks table, and:

- its `Related` line names the requirements it covers on the `spec` label and the plan before it
  on `after`, in the form `document.md` gives: `spec: SPEC-004 (REQ-04, REQ-05) · after: PLAN-007`;
- the first plan holds the Architecture that spans the feature, and a later plan's Architecture
  holds only what is its own and points at the first for the rest: "as PLAN-007 § Architecture,
  plus …";
- an ADR is written once, on the plan whose work decides it;
- every plan is shown at step 7 and confirmed together. One still carrying **Decisions needed** is
  `Draft` per `lifecycle.md`, has no epic yet, and gets its graph and its edge in Promote mode.

The branch is `plan/PLAN-NNN-slug` of the first plan, and every plan lands in the one pull request
the run opens, per `../../../../.codefall/shared/landing.md`.

## The graph

One epic per plan, created at step 9 as `beads.md` says, first plan first. Then one edge per later
increment, from its epic to the epic before it:

```bash
bd dep add <prefix>-PLAN-008 <prefix>-PLAN-007    # PLAN-008's epic waits on PLAN-007's
```

The epic before it closes when its graph is empty and its code is on `main`, so the later epic and
everything under it stays out of `bd ready` until then, and `codefall-implement` on it says which
epic holds it and stops. The rows' **Depends on** column is unchanged by this: the edge is on the
epic, not the rows.

Verify a later increment with `bd ready --mol <its epic>`, which checks its rows against each
other, and `bd dep cycles`. Plain `bd ready` lists none of a later increment while the epic before
it is open, which is the edge working. `--mol` lists its roots, because it does not see a blocker
outside the epic; that is the check of the rows wanted here, and it is why `codefall-implement`
reads plain `bd ready` before it claims an epic. Push once, after every epic and edge.

## Split mode

For a plan whose delivery has turned out, or looks about to turn out, bigger than one. The person
asks for it, on the plan, and the rows move under `revising.md`'s rules for a changed row:

1. **Read the beads first**, `bd show` for every row, and choose with the person which rows move.
   **Only an untouched bead moves**: `open`, unassigned, no comments. A claimed, commented, or
   closed bead stays in this plan, and the cut goes around it; when the rows that would move all
   depend on one, say the split cannot be made yet and why.
2. **The new plan** takes the next number and `after: PLAN-NNN` naming this one. Its rows take `T1`
   onward in its own table, with their plan refs rewritten to its sections. In this plan's table
   the moved rows come out, and their local IDs go on the `Retired:` line.
3. **The beads**: `bd rename <prefix>-PLAN-007-T5 <prefix>-PLAN-009-T1`, then
   `bd update <id> --parent <prefix>-PLAN-009 --spec-id <the new plan's path>`, and the plan ref
   in its body. An edge from a moved row to a row that stayed comes out with `bd dep remove`,
   replaced by the one edge from the new epic to this one as *The graph* says; edges among the
   moved rows stay.
4. **Date both callouts**, verify both graphs, push, and land both documents in one pull request.

A split never closes a bead and never changes one someone holds.

## The report

Beside what step 11 reports for one plan: every plan of the split, with its epic, the requirements
it covers, and the plan it follows; the delivery figures the judgment was made against; a declined
cut and where it was recorded. The last line is `/codefall-implement PLAN-NNN` on the first plan.
