# Triage

Shared reference. Read before a bug is filed as a bead, whoever found it: a worker in
`codefall-implement`, a reviewer in `codefall-review`, a run in `codefall-test`, or the person
`codefall-report` interviewed. Every bug is triaged the same way and carries the same fields, so
the backlog reads as one list however the bugs reached it. The verb that found the bug asks its
own question and files on the answer; this file says what the question proposes and what the bead
carries. Nothing here is filed unasked.

## Contents

- Three questions
- Whose is it
- How bad is it
- Which area
- How big is the fix
- What a bug bead carries
- Duplicates
- Sweeps

## Three questions

Every bug is triaged by three questions, and the fix's size is recorded beside the answers:

1. **Whose is it** — the epic's, or beside it. Asked only inside a delivery.
2. **How bad is it** — the priority, from one scale.
3. **Which area** — the part of the product it belongs to, from the project's list.

The session proposes an answer to each, with a reason in a line, and the person confirms them in
the verb's own question: "I found N problems. Fix them all?" on an epic's work, the recap in
`codefall-report`, the bead offer in `codefall-review`. A person may change any answer there.

## Whose is it

Inside a delivery, a bug belongs to the epic or sits beside it. Three classes, read in order:

**The epic's**, when any one of these holds:

- It stops the epic from shipping: a criterion one of the epic's beads cites fails because of it.
- The epic's changes caused it. It is not there on the default branch.
- It is in code the epic is changing: a file or a function the epic's pull requests touch.

Who caused the bug does not matter. A bug that was there before the epic and stops it from shipping
is the epic's.

**Beside the epic**, when it is in a feature or a component the epic never touched and the epic
ships fine with it still there.

**The rest** is judged by three things together:

- Is it near the work? The same feature, the same user flow, or files next to the ones the epic is
  changing.
- Is it P1 to P3?
- Does fixing it leave the delivered work better than leaving it?

Yes to all three keeps it with the epic, whether the bug is old or new. A P4 goes beside the epic.
A fix big enough to need its own plan goes beside the epic too, whatever its priority, with a line
in its description saying it was found near the epic; a round cannot absorb a plan.

Outside a delivery there is no epic, and every bug sits beside. A bug beside any epic goes to the
open sweep epic for its area when one exists, per *Sweeps* below, and has no parent otherwise.

How each verb answers the question:

| Verb | What it reads |
| --- | --- |
| `codefall-review` | A finding on a line the epic changed is the epic's. A finding the reviewer could only make by reading past the diff is the rest or beside. |
| `codefall-implement` | The worker's discovered item says whether its own change caused the bug or found it there, and the root applies the classes. |
| `codefall-test` | A failed case one of the epic's beads names is the epic's. A `changed` suite failure or an anomaly from the sweep is checked against the default branch: rerun that one suite or spec case there when the tree allows a checkout, and for an agentic case or an anomaly ask whether the epic's commits touched the code path. Reproduces on the default branch and fails no epic criterion is the rest or beside. |
| `codefall-report` | A bug a person reported sits beside, unless they say it belongs to a delivery in flight. |

## How bad is it

One scale for every bug, whoever found it. It asks how bad the bug is for a user.

| Severity | When | Priority |
| --- | --- | --- |
| `critical` | Data lost or exposed, or the product unusable for most users, with no workaround | P1 |
| `major` | A core flow broken for some users, or a workaround most would not find | P2 |
| `minor` | Wrong, with a workaround a user would find | P3 |
| `trivial` | Cosmetic; nothing is stopped | P4 |

P0 is a person's to set by hand on a bead they judge urgent; no verb assigns it.

`codefall-review` keeps its words, and they are this scale: a blocker is P1, important is P2, minor
is P3. `codefall-test`'s classes are not priorities: a real bug takes its priority from this table
like every other bug, and a wrong expectation, a flake, and agent variance are not bugs and keep the
routes `codefall-test`'s triage reference gives them.

Inside a delivery, every bug that is the epic's is built next round, and priority orders the round.
Beside the epic, priority is what decides whether a bug is picked up at all, and it feeds the
middle class above.

## Which area

Every bug carries one label, `area:<name>`, naming the part of the product it belongs to: `checkout`,
`search`, `billing`. An area is a part of the product as a person meets it, not a layer and not a
runner, the same word the testing root uses for `test-cases/<area>/`.

The list of areas is the `area:` labels in use, which `bd label list-all` prints; no file holds
it, so filing a bug mid-run changes nothing in the checkout. When no `area:` label exists yet,
propose names from what the project already has: the directories under `<root>/test-cases/`, the
directories that hold a scoped `AGENTS.md`, and the specs under `docs/specs/`. Pick the area that
fits the bug, or propose a new name when none does, and say so in the question. Never invent a
second spelling of a name already in use; `codefall-report`'s triage mode merges one that slipped
through.

`bd list -t bug -l area:<name>` is the view of one area.

## How big is the fix

Two sizes, recorded as metadata so `codefall-fix` and a sweep can read them without opening the bug:

- `size=fix`: the change stays inside one component, changes no public interface, is one bead, and
  calls for no ADR. `codefall-fix` can take it alone.
- `size=plan`: anything else. `codefall-plan` takes it.

The judgment is `codefall-fix`'s scope test, made once at triage from what the finder saw. A later
run that finds the size wrong changes the metadata and says so.

## What a bug bead carries

Every bug bead is created with these, and nothing is left for a later verb to fill in:

| Field | Value |
| --- | --- |
| ID | `<prefix>-gh-<n>` where a tracker issue exists, `<prefix>-BUG-NNN` for a bug report under `tracker: beads`, `<prefix>-<slug>` otherwise. `<prefix>` is `bd config get issue_prefix`. A taken ID is refused: add `-2` and retry, never `--force` |
| Type | `bug` |
| Priority | P1 to P4, from the scale above |
| Status | `deferred` when the bug is the epic's, so the running round leaves it alone; `open` otherwise |
| Parent | The epic when the bug is the epic's; the open sweep epic for its area when one exists and the bug is beside; none otherwise. Set in a second call, because `--id` and `--parent` do not combine |
| Area | `-l area:<name>` |
| Source edge | `caused-by:<bead>` when the epic's change caused the bug; `discovered-from:<bead>` when the work found a bug that was already there. `<bead>` is the task whose work found it, or the epic for a suite failure that names no case. A bug a person reported has no edge |
| Spec | `--spec-id`: the bug report's path when one exists, since `codefall-plan` and `codefall-fix` read it; otherwise the spec the bug falls under, when there is one |
| External reference | `--external-ref gh-<n>` when the project mirrors to a tracker and an issue was filed |
| Size | `size=fix` or `size=plan`, in metadata |
| Found by | `found=test`, `found=review`, `found=implement`, or `found=report`, in metadata |
| Description | What is wrong, with the evidence the finder has: the case, variant, and criterion for a test run, with evidence paths under `<root>/.artifacts/`; the file and line for a review or a worker; the report's path for `codefall-report`. A bug in the middle class carries the line that says why it stayed with the epic, or that it was found near it |

```bash
prefix=$(bd config get issue_prefix)
bd create "Declined card empties the cart" --id "$prefix-gh-212" \
  -t bug -p 2 -s deferred -l area:checkout \
  --deps caused-by:booking-PLAN-007-T3 --spec-id docs/specs/SPEC-003-checkout.md \
  --external-ref gh-212 --metadata '{"size":"fix","found":"test"}' \
  -d "checkout/place-order, variant card, criterion AC-02 failed: …"
bd update "$prefix-gh-212" --parent booking-PLAN-007
bd dolt push
```

The parent line is left out when the bug has none. `-s open` replaces `-s deferred` beside an epic.

## Duplicates

Search before anything is classified. A bug already filed is a comment on that bead or issue, with
the new evidence, never a second one.

- The graph: `bd list -t bug --all --title-contains "<keywords>"`, and `bd search "<keywords>"`.
- The tracker, when the project has one, with the search command in
  `codefall-report`'s profile for it.
- The reports under `docs/bugs/`, not `archive/`.

A match that is closed and reproduces again is reopened with the new evidence, not refiled.

## Sweeps

A bug beside every epic waits in the backlog with its fields set, where `codefall-fix` picks one up
or a sweep takes a batch. A **sweep** is an ordinary epic cut from the open bugs of one area when a
person wants that area burned down: `codefall-report`'s triage mode creates it as
`<prefix>-SWEEP-<area>-NNN`, type `epic`, titled for the area, and reparents the bugs the person
picked under it. It has no plan document; `codefall-implement` takes it as a delivery, reads each
bug bead as a tier-0 bead, and `codefall-review` and `codefall-test` run on its work. It closes
when its bugs have merged, like any epic.

A sweep that is open is where a new bug beside an epic in that area goes, so the sweep's next round
picks it up. No container that never closes is ever created: the organization is the fields on the
bug, and an epic exists to be emptied.
