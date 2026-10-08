# codefall-plan — where it came from

What this took from elsewhere, and what it deliberately did not, so nobody re-adds it. None of this
is instruction — the skill is the instruction.

## Taken

**From Kiro**, the primary model: the plan document's section set, and the principle that the
plan document is architecture only, with tasks living somewhere else. Kiro requires six sections —
Overview, Architecture, Components and Interfaces, Data Models, Error Handling, Testing Strategy —
and three of those are conditional here, because a required section with nothing behind it teaches
readers to skim. Also from Kiro: research inline rather than in sibling files, and bug fixes routed
away from the full artifact, which is tier 0.

**From spec-kit**: the Technical Context block, which forces concreteness where prose hand-waves;
the shape of the Alternatives table; and numbered-identifier traceability, which here is the local
task IDs and the Plan ref column.

**From `codefall-envision`**: the `Related` row as a labeled list rather than a set of rows. A
new kind of upstream artifact costs a label rather than an edit to the template and the skill; when
a bug-report verb lands, its plans carry `bug: BUG-012` and nothing else moves.

**From `codefall-specify`**: status describes the document, never the work. It diverges from
`codefall-envision`, which carries an `Active` state because a vision has no tracker
representation to carry it. A plan has one — the beads it created — so there is no `Active` here.

## Dropped

**From spec-kit**: the constitution check, the phase gate and its re-check ceremony, the Project
Structure section, and the `research.md` / `data-model.md` / `quickstart.md` / `contracts/` file
sprawl. Codefall's stance is that what gets proposed gets built, so the gating and negotiation loops
have nothing to gate — what is kept is the structure that keeps documents scannable and
machine-parseable.

**A per-row hash in the Tasks table.** Beads holds the content that was created, so a later run
compares the document against the graph itself rather than against a record of what the document
used to say.

**A bead ID column in the Tasks table.** Every row's bead is `<prefix>-PLAN-NNN-Tn`; a column would
repeat that rule once per row.

**A "may be out of date" disclaimer over the table.** It teaches readers to distrust the table and
says nothing about how a difference is resolved. The callout instead dates the plan, says Beads is
authoritative, and names Revise as the reconciliation.

**Recording both the spec and its vision on `Related`.** A spec already names its vision in its
own `**Vision:**` row; a plan that carries both keeps a second copy of a link nothing keeps in
sync.

## Why the Tasks table stays

The table used to collapse to a line mapping each local ID to the hash `bd` had picked, on the
reasoning that a task list left in a git-tracked document diverges from the graph and nobody updates
it. Two things changed. Beads took the plan's own numbering, so the line carried nothing a reader
could not derive; and Revise works row by row, editing the document first and the graph after, so
the table is the input to the one path that changes a plan's tasks. What can diverge is work
state, which the table never held. So the table stays, dated, with Beads authoritative for state and
for anything filed under the epic since, and the plans `AGENTS.md` forbids editing a plan bead's
title, edges, or ref with `bd` directly, which is what keeps the two level.
