# Designing a fix for a bug

What changes when the input is a bug rather than a feature. Read at step 2 when the input names or
describes one.

## The report

Read `docs/bugs/` — not `archive/` — for the `BUG-NNN` the input names or describes. The report is
the target the way a spec is:

- **Its acceptance criteria are what the fix is verified against.** They go into the bead as they
  are, each citing its identifier in full — `BUG-012-AC-01`.
- **Its Spec row names the requirement** the expected behaviour belongs to. Read that requirement's
  criteria too; a fix that breaks one is not a fix.
- **`Status: Draft` is a stop**, the same gate as a `Draft` spec.

When the input describes a bug and no report exists, offer `/codefall-report` once. On no, continue
from the description, and the bead carries the reproduction you establish below.

## Reproduce it when the report could not

Read the `**Reproduced:**` row. On `Yes`, the report's steps are the reproduction. On `No` or
`Not attempted`, try it before looking for a cause, the way `codefall-report` does: the local
environment through the declared `start` and nothing else, a driver proved by one live call, the
report's steps as written, and nothing changed by hand or mocked. Offer what the attempt shows as an amendment to the report's Reproduction section and row — text
that moves no work, shown at step 7 with everything else.

A bug that still does not reproduce is said plainly. The user chooses between one investigation bead,
whose criteria are that the bug is reproduced and its cause is written down, and stopping.

## Find the cause

Read the code the steps pass through, from the surface the reporter used inward, until you can say
why the product does what the Actual result says. **The cause is written into the bead** in two or
three sentences, with the file and line it sits at.

A cause the reporter suspected, under the report's Open questions, is a lead to check, not a finding.

A cause you cannot find is said out loud with where you looked, and the same choice follows: one
investigation bead, or stopping.

## The tier

The cause decides it, through the same table as any other work. A cause inside one component with a
fix of one or two beads is tier 0. A cause that sits across a component boundary, or a fix that
changes a public interface, is tier 1, and the design's `Related` row carries `bug: BUG-NNN`.

**A bug fix carries a regression test**, whatever the tier: a unit test where one reaches the cause,
and a test case, drafted into the bead as for any task, where the bug was seen through the wired
product and only the wired product shows it.
