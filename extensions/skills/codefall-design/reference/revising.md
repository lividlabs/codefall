# When a design changes after its beads exist

Read for the Revise mode. A row and its bead share a number — `Tn` is `<prefix>-DESIGN-NNN-Tn` —
so a later run edits the graph the table already describes rather than duplicating it. The user
edits the table; this mode brings the beads level with it. A revision can also be asked for from
downstream, as a bead — the last section.

## Contents

- Row by row
- When a task row changes
- Revision beads

## Row by row

**Read the current beads first.** `bd show <prefix>-DESIGN-NNN-Tn --json` for every row, and
`bd show <prefix>-DESIGN-NNN --children` for anything the table no longer names. The comparison is
against the graph itself, not against a record of what the document used to say.

**Only beads named `<prefix>-DESIGN-NNN-Tn` are rows.** The epic's other children are the
delivery's work: `<prefix>-DESIGN-NNN-MERGED`, the landed bead; the `code` discoveries, deferred
review findings, and test-found bugs that `codefall-implement`, `codefall-review`, and
`codefall-test` filed under the epic; closed `design-amended` records; and the revision beads the
last section settles. List them under their own heading in the report, with the chart line from
`../../../../.codefall/shared/delivery.sh <epic>`, and never treat one as a row the table dropped.

A design that still carries a mapping line in place of a table was created before the table stayed.
Rebuild the table from the beads — title, blockers, and design ref from each `bd show` — and rewrite
the callout to the current form before going row by row.

Then, row by row:

| Case | What happens |
| --- | --- |
| The row changed | Edited or replaced, decided by whether anyone is holding the bead — below |
| The row is new | A new local ID, the next after the highest ever used, and a new bead named `<prefix>-DESIGN-NNN-Tn` |
| An edge changed | `bd dep add` or `bd dep remove`, then re-verify with `bd ready` and `bd dep cycles` |
| The row is gone | Its local ID goes on the `Retired:` line under the table. **Report the bead and let the user choose** — close it, or leave it open because work already happened against it |

**A removed row is never closed silently.** Somebody may be holding that ticket, and a design edit
is not evidence that its work stopped mattering.

## When a task row changes

Read the bead first — `bd show <id> --json` reports its status, assignee, and comment count.

**Untouched** — `open`, unassigned, no comments — is edited, whatever changed. Nobody has acted on
the old wording, so the size of the change does not matter.

**Touched** — claimed, in progress, commented on, or closed — asks one question: would the work done
against the old wording still be correct and sufficient under the new wording? Yes, edit it. No,
replace it: `bd rename <id> <id>-superseded` to move the old bead out of the way, create the new
bead under the plain `<id>`, and `bd supersede <id>-superseded --with <id>`. The row does not
change: the current bead always holds the plain ID.

A ticket must not change under someone holding it. Where nobody is holding it, editing costs
nothing.

Worked, so two sessions apply it the same way:

| Old row | New row | Bead state | Result |
| --- | --- | --- | --- |
| Wire context load into `/scaffold` | same, with a Design ref added | any | **edit** — nothing about the work changed |
| Add `StageContext` type | Add `StageContext` type + serde | open, unclaimed | **edit** — nobody started; the bead becomes the bigger task |
| Add `StageContext` type | Add `StageContext` type + serde | closed | **replace** — the type exists, the serde does not, and editing a closed bead leaves it unbuilt |
| Emit context on `/envision` exit | Emit context on every verb's exit | claimed | **replace** — they are building one thing and would silently owe five |

`bd supersede <old> --with <new>` closes the old bead with a reference to its replacement, so the
record of what the task used to be survives. **The local ID does not change** — `T2` still means
`T2`; only the bead behind it moves, and the plain ID names the new one.

**A row that has become two separate tasks is a removal plus two additions**, not a replacement, and
the removal goes through the report-don't-close rule above.

**Then date the callout and push.** Add today to the callout — `Created in Beads <date>, revised
<today>, as …` — and `bd dolt push` once, after every edit, replacement, and edge change, so the
graph a teammate reads is the one the table now describes.

Say per row which you did and why, in the report.

## Revision beads

`bd list -l design-revision --spec <this design's path>` is the input from outside the document:
a bead `codefall-implement` or `codefall-review` filed because the design's text and the code, or
the design and the spec, disagree in a way the finder could not amend itself — the fix moves a task
row or a criterion a bead cites, the document was frozen, or the user declined the amendment there.
Its body says what the design says, what was found instead, and which document it names, and a
`discovered-from` edge names the bead or the review that found it. A bead labelled `design-amended`
is different: `codefall-implement` and `codefall-review` file one, already closed, for each
amendment they made, so the record of what was wrong is queryable. It asks nothing of this run and is not listed here. Read
the revision beads before the row-by-row pass, show them to the user, and settle every one in one
of three ways:

| The user decides | What happens |
| --- | --- |
| The document is wrong | The text is amended — the design's, or the spec's by appending, with the requirement's tracker issue regenerated per `../../codefall-specify/trackers/<name>/PROFILE.md` and the bead's criteria updated in the same pass — and the bead closes with the amendment as its reason — `bd close <id> -r "amended: § Architecture now names StageContext"` |
| The work is real | A new row on the table, per the row-is-new case above, and the bead closes with the new task's ID as its reason — the task carries the work, the request does not |
| The request is wrong | The bead closes with why, in the user's words |

**Every revision bead ends closed.** An open one after a revision means the loop is not finished,
and the next run reads it again. Push after the closes, with the rest, and say per bead which way
it went in the report. A revision bead filed during a delivery is already a child of the epic; a
new row it becomes is a child too, as every row is, and the closed request stays where it is. This
mode writes no round: the next round starts when `codefall-implement` claims the epic again, which
is what the report's last line names — `/codefall-implement DESIGN-NNN`.
