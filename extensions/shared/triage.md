# Triage

Shared reference. Read before anything found along the way is filed as a bead, whoever found it: a
worker in `codefall-implement`, a reviewer in `codefall-review`, a run in `codefall-test`, or the
person `codefall-report` interviewed, and before `codefall-report`'s triage mode walks the backlog.
Every item is triaged the same way and carries the same fields, so the backlog reads as one list
however its items reached it. The verb that found the item asks its own question and files on the
answer; this file says what the question proposes and what the bead carries. Nothing here is filed
unasked.

## Contents

- What triage takes
- Three questions
- Whose is it
- How bad is it
- Which area
- How big is the fix
- What a triaged bead carries
- Duplicates
- Followups

## What triage takes

Everything it is given. Most items are bugs, and this file says "bug" where only a bug can be meant,
but a cleanup, a missing test, a refactor, a request for something that never existed, and a bead
in the backlog typed `task` whose title says something is wrong all go through the same questions
and leave with the same fields. **What kind of thing an item is, is a field triage sets, never a
reason to skip the item, decline it, or hand it to another verb.** The type is proposed the way the
area is: from the types already in the graph, or any `bd create -t` accepts, with the reason in a
line, and the person's own word for it wins over yours. A bead someone typed `chore` stays `chore`.

Two things have routes of their own and are not beads this file files: a wrong expectation, a flake,
and agent variance, which `codefall-test`'s triage reference routes; and the `plan` form of a
discovered item, which `codefall-plan`'s Revise mode takes.

## Three questions

Every item is triaged by three questions, and the fix's size is recorded beside the answers:

1. **Whose is it** — the epic's, or beside it. Asked only inside a delivery.
2. **How bad is it** — the priority, from one scale.
3. **Which area** — the part of the product it belongs to, from the project's list.

The session proposes an answer to each, with a reason in a line, and the person confirms them in
the verb's own question: "I found N problems. Fix them all?" on an epic's work, the recap in
`codefall-report`, the bead offer in `codefall-review`, the one block in `codefall-report`'s triage
mode. A person may change any answer there. Nothing is asked per item.

## Whose is it

Inside a delivery, a bug belongs to the epic or sits beside it. An item that is not a bug, a missing
test or a refactor a worker met, is the epic's, per *Discovered work* in `codefall-implement`'s beads
reference: the work found it, and the next round builds it. For a bug, three classes, read in order:

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
in its description saying it was found near the epic; a round cannot absorb a plan. When the three
do not settle it, keep it with the epic and say why.

Outside a delivery there is no epic, and every item sits beside. An item beside any epic goes to the
open followup epic for its area when one exists, per *Followups* below, and has no parent otherwise.

How each verb answers the question:

| Verb | What it reads |
| --- | --- |
| `codefall-review` | A finding on a line the epic changed is the epic's. A finding the reviewer could only make by reading past the diff is the rest or beside. |
| `codefall-implement` | The worker's discovered item says whether its own change caused the bug or found it there, and the root applies the classes. |
| `codefall-test` | A failed case one of the epic's beads names is the epic's. A `changed` suite failure or an anomaly from the sweep is checked against the default branch: rerun that one suite or spec case there when the tree allows a checkout, and for an agentic case or an anomaly ask whether the epic's commits touched the code path. Reproduces on the default branch and fails no epic criterion is the rest or beside. |
| `codefall-report` | A bug a person reported sits beside, unless they say it belongs to a delivery in flight. |

## How bad is it

One scale for every item, whoever found it. It asks how bad the item is for a user.

| Severity | When | Priority |
| --- | --- | --- |
| `critical` | Data lost or exposed, or the product unusable for most users, with no workaround | P1 |
| `major` | A core flow broken for some users, or a workaround most would not find | P2 |
| `minor` | Wrong, with a workaround a user would find | P3 |
| `trivial` | Cosmetic; nothing is stopped | P4 |

P0 is a person's to set by hand on a bead they judge urgent; no verb assigns it.

An item that is not a bug sits on the same scale by what leaving it costs a user: a cleanup or a
missing test is `trivial` or `minor` unless leaving it blocks or breaks something, and a request
for something that never existed is placed by what its absence stops. `codefall-review` keeps its
words, and they are this scale: a blocker is P1, important is P2, minor is P3. `codefall-test`'s
classes are not priorities: a real bug takes its priority from this table like every other item, and
a wrong expectation, a flake, and agent variance keep the routes `codefall-test`'s triage reference
gives them.

Inside a delivery, every item that is the epic's is built next round, and priority orders the round.
Beside the epic, priority is what decides whether an item is picked up at all, and it feeds the
middle class above.

## Which area

Every item carries one label, `area:<name>`, naming the part of the product it belongs to:
`checkout`, `search`, `billing`. An area is a part of the product as a person meets it, not a layer
and not a runner, the same word the testing root uses for `test-cases/<area>/`.

The list of areas is the `area:` labels in use, which `bd label list-all` prints; no file holds
it, so filing an item mid-run changes nothing in the checkout. When no `area:` label exists yet,
propose names from what the project already has: the directories under `<root>/test-cases/`, the
directories that hold a scoped `AGENTS.md`, and the specs under `docs/specs/`. Pick the area that
fits the item, or propose a new name when none does, and say so in the question. Never invent a
second spelling of a name already in use; `codefall-report`'s triage mode merges one that slipped
through.

`bd list -s open,deferred --exclude-type epic -l area:<name>` is the view of one area.

## How big is the fix

Two sizes, recorded as metadata so `codefall-fix` and a followup can read them without opening the
item:

- `size=fix`: the change stays inside one component, changes no public interface, is one bead, and
  calls for no ADR. `codefall-fix` can take it alone.
- `size=plan`: anything else. `codefall-plan` takes it.

The judgment is `codefall-fix`'s scope test, made once at triage from what the finder saw. A later
run that finds the size wrong changes the metadata and says so.

## What a triaged bead carries

Every triaged bead is created with these, and nothing is left for a later verb to fill in:

| Field | Value |
| --- | --- |
| ID | `<prefix>-gh-<n>` where a tracker issue exists, `<prefix>-BUG-NNN` for a bug report under `tracker: beads`, `<prefix>-<slug>` otherwise. `<prefix>` is `bd config get issue_prefix`. A taken ID is refused: add `-2` and retry, never `--force` |
| Type | `bug` when the product is wrong; otherwise the type proposed under *What triage takes*, or the one the person set |
| Priority | P1 to P4, from the scale above |
| Status | `deferred` when the item is the epic's, so the running round leaves it alone; `open` otherwise |
| Parent | The epic when the item is the epic's; the open followup epic for its area when one exists and the item is beside; none otherwise. Set in a second call, because `--id` and `--parent` do not combine |
| Area | `-l area:<name>` |
| Source edge | `caused-by:<bead>` when the epic's change caused the bug; `discovered-from:<bead>` when the work found something that was already there. `<bead>` is the task whose work found it, or the epic for a suite failure that names no case. An item a person reported has no edge |
| Spec | `--spec-id`: the bug report's path when one exists, since `codefall-plan` and `codefall-fix` read it; otherwise the spec the item falls under, when there is one |
| External reference | `--external-ref gh-<n>` when the project mirrors to a tracker and an issue was filed |
| Size | `size=fix` or `size=plan`, in metadata |
| Found by | `found=test`, `found=review`, `found=implement`, or `found=report`, in metadata |
| Description | What is wrong, or what is wanted, with the evidence the finder has: the case, variant, and criterion for a test run, with evidence paths under `<root>/.artifacts/`; the file and line for a review or a worker; the report's path for `codefall-report`. An item in the middle class carries the line that says why it stayed with the epic, or that it was found near it |

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

The parent line is left out when the item has none. `-s open` replaces `-s deferred` beside an
epic.

## Duplicates

Search before anything is classified. An item already filed is a comment on that bead or issue,
with the new evidence, never a second one.

- The graph: `bd list --all --title-contains "<keywords>"`, and `bd search "<keywords>"`.
- The tracker, when the project has one, with the search command in
  `codefall-report`'s profile for it.
- The reports under `docs/bugs/`, not `archive/`.

A match that is closed and reproduces again is reopened with the new evidence, not refiled.

## Followups

An item beside every epic waits in the backlog with its fields set, where `codefall-fix` picks one
up or a followup takes a batch. A **followup** is an ordinary epic created from the open items of
one area when a person wants that area burned down: `codefall-report`'s triage mode creates it as
`<prefix>-FOLLOWUP-<area>-NNN`, type `epic`, titled for the area, and reparents the items the
person picked under it. It has no plan document; `codefall-implement` takes it as a delivery, reads
each child as a tier-0 bead, and `codefall-review` and `codefall-test` run on its work. It closes
when its children have merged, like any epic.

A followup that is open is where a new item beside an epic in that area goes, so the followup's
next round picks it up. No container that never closes is ever created: the organization is the
fields on the item, and an epic exists to be emptied.
