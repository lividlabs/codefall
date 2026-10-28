# Amendments and the beads offered at triage

What a finding does to the documents upstream of the work, and the bead a finding becomes when a
fix here cannot settle it. Read at step 5, when a finding names a document or is deferred.
Contents: *A finding that an upstream document is wrong*, *The bead filed at triage*.

## A finding that an upstream document is wrong

**It is fixed like any other document finding.** When the plan behind the work says one thing and
the code needed another, or the plan and its spec disagree, and the document is `Draft` or
`Ready`, the fix is text on the target's branch: the plan's section amended, or a criterion
appended to the spec with the requirement's tracker issue regenerated per
`../../codefall-specify/trackers/<name>/PROFILE.md`. Fix every document between the change and the
code that restates the point; when one of them fails these tests, fix none, and name each document
in the bead offered below, so the whole chain reaches `codefall-plan` together. Each document fix
is recorded as a closed `plan-amended` bead, per *Amendments* in
`../../codefall-implement/reference/beads.md`, with a `discovered-from` edge to the bead the branch
names when there is one.

## The bead filed at triage

**What that fix cannot do, and what is deferred, becomes a bead.** A finding left in the findings
file alone is one no verb reads again. At triage, file each in the form under *Discovered work* in
`../../codefall-implement/reference/beads.md`, under `../../../../.codefall/shared/decisions.md`:

- the `plan` form — `--spec-id` the plan's path, the label `plan-revision` — for a finding the
  plan caused that a fix here cannot settle because it moves work, the document is frozen, or the
  user deferred it;
- the `code` form for a `deferred` finding whose cause is the code.

Either carries a `discovered-from` edge to the bead the branch names when there is one. A `plan`
bead is a `deferred` child of that bead's epic, read from `bd show <bead> --json`, when there is
one. A `code` bead, a bug or not, is triaged per `../../../../.codefall/shared/triage.md`, which
sets its disposition, parent, status, priority, area, and the rest of its fields; a duplicate is
a comment on the bead that exists, and a finding that turns out not to be an issue is dismissed
with that reason rather than filed. Create it, `bd dolt push`, and record its ID as the finding's
`bead`. A `plan` bead is settled in the same run: `SKILL.md` runs `codefall-plan`'s Revise mode on
the document before the report.
