# Rounds

The round boundary: what happens between one round of a delivery and the next, and when a run
stops. Read at the go step of a later round, before anything is claimed, and in a continuous run
after `codefall-test` returns.

**The purpose of stopping is to avoid spending rounds that will not bring the epic closer to
done.** When the signals are mixed, run the round. The ceiling bounds the cost of being wrong in
that direction, and a round that should not have run costs compute; a stop that should not have
happened costs a person coming back to restart it, which is the one thing a continuous run exists
to remove.

## Contents

- The boundary, in order
- The batch
- Three exits
- Signals
- The ceiling
- The ledger lines

## The boundary, in order

1. `bd dolt pull`, `bd gate check`, the chart line.
2. **Revision beads.** `bd list -l plan-revision --spec <plan path> -s open,deferred`. When any is
   open, run the `codefall-plan` skill on the plan through the `Skill` tool, in this session, for
   its Revise mode; its three outcomes follow `../../../../.codefall/shared/decisions.md`, and a
   row it adds is a child of the epic like every row.
3. **Triage everything under the epic that is open or deferred**, the landed bead and the human
   beads aside: `bd list --parent <epic> -s open,deferred --limit 0 --json`. Each item gets one of
   the dispositions in `../../../../.codefall/shared/triage.md`, with its reason in a line: the
   epic's, this round, beside the epic or in the area's followup, a duplicate, or not an issue; and
   its priority and size are checked against the scale. A bead that already exists and is not an
   issue closes with `-r "not an issue: …"`; a duplicate closes with `-r "duplicate of <id>"` and
   its evidence commented on the one kept; one that belongs beside the epic is reparented with
   `bd update <id> --parent <followup>` or `--no-parent` and set `open`.
4. **Pick the batch**, below, and reopen it: `bd update <id> -s open` per bead.
5. **Answer the three exits and read the signals**, below. On continue: set the round, claim the
   first wave, `bd dolt push`, and go, as step 5 of `SKILL.md` says. On stop: the stop line below,
   the report, and `/codefall-implement <epic>` as the way to resume.
6. **Write the boundary line** on the epic's notes either way.

## The batch

Every open or deferred child that is P1 or P2 and can be built now is in the batch. A P3 joins it
when the run judges it worth the round, near the files the batch already touches, cheap, and
leaving the delivered work better, with the reason stated. Everything else stays deferred for a
later round or leaves the epic at step 3. A human bead is never in a batch, and a bead that depends
on one is left out with a line saying which question holds it.

The batch is what the gate shows on a later round, and what the boundary line records.

## Three exits

These are mechanical, because each is something the run can do nothing about:

| Exit | When | What happens |
| --- | --- | --- |
| **Done** | Nothing is open under the epic but the landed bead and any human beads, and this round's test run passed | Point at the stack per *Pointing at a stack* in `../../../../.codefall/shared/stacks.md`; the delivery ends. Human beads are named for the person to answer before the epic closes |
| **Nothing can be built** | Every bead in the batch is blocked on a human bead, or has already failed twice and been skipped | Stop; name each bead and what holds it |
| **The ceiling** | The epic's `round` metadata has reached the ceiling | Stop; say the ceiling was reached and can be raised at the gate |

## Signals

Everything else is a signal the run weighs, and the default is to continue. A signal never stops
the run on its own.

| Signal | Trouble when | Not trouble when |
| --- | --- | --- |
| The last round closed nothing | Its beads failed or were skipped | Its work was Revise mode turning revision beads into rows, or a retry is queued for this round |
| The batch is not smaller than the last round's | The same kind of finding keeps arriving from new work, two rounds running | The new items carry `caused-by` edges from this round; `test` ran cases for the first time; triage pulled P3s in by choice |
| Review found the same class of finding three rounds running | A convention or the plan is wrong and nobody has changed it | The project's tests are thin and the finding is "missing test", which is the project, not the delivery |

Before stopping on signals, consult once, per *Consulting* in
`../../../../.codefall/shared/running-agents.md`: `QUESTION` is whether another round is likely to
move the epic closer to done; `FILES` are the epic's children and this round's review and test
reports; `OPTIONS` are continue and stop, each with what the run sees. A yes, or no answer, runs
the round. A no stops it, with the consult's reasoning in the stop line. A repeated finding class
that points at the plan runs Revise mode first, at step 2 of the next boundary, and a convention it
points at becomes a human bead naming the convention, filed as `decisions.md` says, with the round
continuing.

## The ceiling

Ten rounds per delivery, read from the epic's `metadata.round`, so a resumed run does not reset
it. The gate shows it, and the person changes it there or sets a standing value in
`.codefall/skills/codefall-implement/CUSTOMIZE.md`. Reaching it is an exit above; raising it at the
gate continues.

## The ledger lines

All on the epic's notes with `--append-notes`, never `--notes`, in the form *Session end* in
`beads.md` gives; a decision about one bead goes on that bead per `decisions.md`.

```
round 2 (implement) [2026-10-28]: 5 tasks built, PRs #103 #104, 3 decisions on T2 T4 T7, 1 question (booking-human-stage-context)
round 2→3 boundary [2026-10-28]: batch 4 (P1 1, P2 3); moved 6 to booking-FOLLOWUP-checkout-001; closed 2 not an issue; batch 9→4; continue
round 3 stop [2026-10-29]: nothing can be built — T9 and T11 wait on booking-human-stage-context; resume with /codefall-implement booking-PLAN-007
```

The round line counts the decisions and names the beads they are on, so the final report finds
every one without a search. The boundary line records the signals seen and what was decided, and
after a few deliveries those lines are the evidence for changing the signals or the ceiling.
