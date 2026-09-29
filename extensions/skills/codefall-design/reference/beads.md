# Beads: creating and verifying the graph

Read before creating the graph (step 9).

## Contents

- What gets created
- The test case in a bead's criteria
- The plan file
- Name the beads
- Verify the graph
- Publish the graph

## What gets created

| Bead | One per | Type | Body |
| --- | --- | --- | --- |
| Epic | design document | `epic` | The Overview, and the path to the document |
| Task | Task Plan row | `task`, or `bug` where it is one | What to do, the design ref, and acceptance criteria |

**Every task bead carries acceptance criteria** — two to five checkable statements, drawn from the
spec requirements the task serves and the Hard Constraints that bind it. A criterion that traces to
a spec requirement cites it (`R3: context survives a restart`). Write them as checks, not restated
task text; `codefall-implement` verifies them before closing the bead.

Tier 0 has no epic. One or two beads are created directly, with the reproduction, the cause, and
acceptance criteria in the body — the bead must be self-sufficient. Each is created with `--id`:
`<prefix>-<tracker ref>` (`gh-123`, `jira-ABC-42`) where a tracker issue exists, with
`--external-ref` set to the same value, else `<prefix>-<slug>`. A taken ID is refused; append `-2`
and retry. `<prefix>` is what `bd config get issue_prefix` prints.

**A fix for a bug report** is type `bug`. Its body names the report's path, its reproduction is the
report's steps, and its criteria are the report's, each citing its identifier in full —
`BUG-012-AC-01`. The tracker ref in its ID is the report's `**Issue:**` row.

## The test case in a bead's criteria

A task whose work is verified through the wired product — the real interface, against the real
services, as a user meets it — carries a test case, and the bead's acceptance criteria are where it
is written. Nothing is created here: `codefall-implement` writes the case file from these criteria
before it writes the code, and `codefall-test` runs it.

Three things go in the criteria.

- **The case**, as `<area>/<slug>` — a product area and a hyphenated slug. That is the path the file
  takes under the testing root's `test-cases/`, and it is the case's identifier.
- **The modalities**, `spec`, `agentic`, or both. `agentic` only where verifying an outcome needs
  judgement — generated prose, conversational behaviour, visual wrongness a locator cannot assess.
  `spec` otherwise. Neither where unit tests verify the task, and a driver being available is not a
  reason to add one.
- **Every criterion the case will hold**, each carrying its source. A criterion that restates a spec
  criterion cites it in full — `SPEC-003-REQ-01-AC-01`, the form `codefall-specify` writes and the
  form `grep` finds; one that restates a bug report's criterion cites `BUG-012-AC-01` the same way.
  A criterion that goes further is marked `derived`, cites the requirement it elaborates —
  `SPEC-003-REQ-01` — and says in one line what it adds: a boundary, a negative path, error handling. A case covers every criterion of the requirement it tests, or records why one was
  dropped.

```
Case: checkout/place-order · modalities: spec, agentic
1. An order placed with a valid card reaches a confirmation naming the order id.
   (SPEC-003-REQ-01-AC-01)
2. A declined card leaves the cart intact and states why, without charging.
   (derived from SPEC-003-REQ-01 — the negative path; the spec names only the accepted card.)
```

**The implementation is never a source.** Not the code, not clicking through the application to see
what it does, not a pull request's own text. Reading code for mechanics — a control's role, its
accessible name — is allowed; reading it to decide what should happen is not.

**A gap the criteria expose is appended to the spec, in this run.** Where the spec is silent or
ambiguous, draft the criterion in the spec's own EARS form under the requirement it belongs to,
taking the next `AC` number — numbering is append-only, so nothing already cited moves — and show
it at step 7 beside the task plan as a spec amendment. On yes, the bead's criterion cites the new
identifier in full; the spec is edited on this run's branch at step 8, and the requirement's tracker
issue is regenerated per the *Refreshing* sequence in
`../../codefall-specify/trackers/<name>/PROFILE.md`, the existing-requirement case alone. On no, the
criterion is written marked `derived`, which from here on means the spec was offered it and
declined. Nothing is patched into the case.

**The user's approval of the task plan is the sign-off** the derived criteria need, so every such
criterion is shown at step 7 and nothing later asks again.

The case-file format `codefall-implement` writes to is
`../../codefall-test/reference/case-file.md`.

Every bead gets `--spec-id` set to the design document's path, so `bd list --spec <path>` finds a
design's beads from the graph side, as `bd list --parent <prefix>-DESIGN-NNN` does from the epic.

## The plan file

`bd create --graph` takes a plan file and creates every node and edge in one call, then prints the
mapping the document needs. The plan's node keys **are** the Task Plan's local IDs.

```json
{
  "nodes": [
    { "key": "EPIC", "type": "epic",
      "title": "DESIGN-007: Stage context handoff",
      "description": "<the Overview, and docs/designs/DESIGN-007-stage-context.md>" },
    { "key": "T1", "type": "task", "parent_key": "EPIC",
      "title": "Add StageContext type + serde",
      "description": "<what to do> · Design ref: DESIGN-007 § Components and Interfaces" },
    { "key": "T2", "type": "task", "parent_key": "EPIC",
      "title": "Wire context load into /scaffold",
      "description": "<what to do> · Design ref: DESIGN-007 § Architecture" }
  ],
  "edges": [
    { "from_key": "T2", "to_key": "T1", "type": "blocks" }
  ]
}
```

```bash
bd create --graph <plan.json> --dry-run    # validates the graph, creates nothing
bd create --graph <plan.json>
```

```
Created 3 issues
  EPIC -> booking-a2g
  T1 -> booking-unz
  T2 -> booking-s58
```

**The edge direction is the trap.** Despite the type name, `from_key` is the **dependent** and
`to_key` is the **blocker**: `{"from_key": "T2", "to_key": "T1", "type": "blocks"}` means T2 is
blocked by T1. It maps straight off the staging table — the row's own ID is `from_key`, and each
entry in its **Depends on** column is a `to_key`. Getting it backwards produces a graph that runs in
reverse, and nothing but the ready set will tell you.

**The plan file carries only these fields.** `key`, `title`, `type`, `description`, `labels`,
`priority`, `parent_key` on a node; `from_key`, `to_key`, `type` on an edge. Anything else is
**silently dropped** with a warning. Neither `--spec-id` nor `--acceptance` is among them, and
there is no ID field, so all three are set afterwards, in one pass over the creation output: the
rename below, then the update.

```bash
bd update <id> --spec-id docs/designs/DESIGN-007-stage-context.md \
  --acceptance $'R3: context survives a restart\nHard constraint: one open write txn per booking'
```

## Name the beads

The graph is created with the hash IDs `bd` picks and renamed to the document's own numbering,
behind the project's prefix, before anything cites them:

| Bead | ID |
| --- | --- |
| Epic | `<prefix>-DESIGN-NNN` |
| Task `Tn` | `<prefix>-DESIGN-NNN-Tn` |

`<prefix>` is what `bd config get issue_prefix` prints. Never write a project's prefix into a
skill or a template; `bd create` and `bd rename` refuse an ID under another prefix. `bd rename`
carries the parent, the edges, and every reference with it.

```bash
bd rename booking-a2g booking-DESIGN-007
bd rename booking-unz booking-DESIGN-007-T1
bd rename booking-s58 booking-DESIGN-007-T2
```

A rename refused because the ID exists means this design's beads were created before, whatever
the callout says. Reconcile per `revising.md`; never create a second graph beside the first.

## Verify the graph

Creation is not verification, and a reversed edge is invisible in the creation output.

```bash
bd ready          # the unblocked set
bd dep cycles     # must find none
```

**The ready set must be exactly the rows whose Depends on column is `—`**, plus the epic. If a task
with prerequisites is ready, or a root task is not, the edges went in backwards — fix them before
the callout is rewritten, while the table still says it is staged.

## Publish the graph

```bash
bd dolt push
```

A bead write is this machine's until it is pushed: the database is Dolt, and the team's copy is
`refs/dolt/data` on the git remote. Push once the graph verifies and before the callout is
rewritten, so a teammate's `bd ready` shows the same roots the table records. A project with no Dolt
remote says it skipped and exits 0: say once that the graph is this machine's, and carry on.
