# docs/plans — operative rules

- `archive/` is history. Do not read it unless the user asks about a superseded plan by name.
- A plan's identifier is permanent. `PLAN-007` means the same document after it is archived.
- Local task identifiers (`T1`, `T2`) are permanent within a plan and append-only. A retired one
  is never reused.
- Once a Tasks table says `Created in Beads`, Beads is authoritative for work state. The table keeps
  the tasks, edges, and plan refs; it never grows a status column.
- A plan bead's title, edges, and plan ref change only through `/plan` (Revise), which edits
  the row and the bead together — never with `bd` directly.
- A bead labelled `plan-revision` whose `spec_id` is a plan's path is a request to revise that
  plan. `/plan` lists them at its start and closes each one it settles.
- ADRs live in `docs/adrs/` and are never rewritten. A revision is a new, superseding ADR.
- Status describes the document, never the work. Beads holds work state.
- Status transitions are `/plan`'s to make, never a hand edit.
