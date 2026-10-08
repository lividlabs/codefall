# Legacy designs — the customization sections

Three marked sections, one per verb, for a project that holds plans written by `codefall-design`
under `docs/designs/` before the verb became `codefall-plan` and its documents moved to
`docs/plans/`. Each section goes between its own markers into the file its heading names; nothing
outside the markers is written. The documents themselves are never moved or renamed: their bead
identifiers cannot follow, so the identity stays with the file.

## `.codefall/skills/codefall-plan/CUSTOMIZE.md`

<!-- codefall:legacy-designs -->
## Plans written before the rename

This project holds plans under `docs/designs/` as `DESIGN-NNN-slug.md`, written by `codefall-design`
before it became `codefall-plan`. Each is a plan under an earlier name, and nothing about it moves.

- Read `docs/designs/` alongside `docs/plans/` when checking whether an existing plan already covers
  the work. `docs/designs/archive/` is history, like `docs/plans/archive/`.
- Take the next number from the highest across both directories: `DESIGN-007` is followed by
  `PLAN-008`, never by `PLAN-001`.
- Revise one of these in place under its `DESIGN-NNN` identity. Its epic is `<prefix>-DESIGN-NNN`,
  its task table is headed `Task Plan`, and the revision requests filed against it carry the label
  `plan-revision` or, from before the rename, `design-revision`: list both. Never cut a new plan to
  replace one that only needs revising.
- Archive one to `docs/designs/archive/` under the same name.
- A `DESIGN-NNN` on a vision's `Related` line or in a spec resolves to `docs/designs/`.

Remove this section once `docs/designs/` holds only archived plans whose beads are closed.
<!-- /codefall:legacy-designs -->

## `.codefall/skills/codefall-implement/CUSTOMIZE.md`

<!-- codefall:legacy-designs -->
## Plans written before the rename

This project holds plans under `docs/designs/` as `DESIGN-NNN-slug.md`, written by `codefall-design`
before it became `codefall-plan`. A bead whose `spec_id` points there is read exactly as one pointing
at `docs/plans/`: the document's `Task Plan` section is its Tasks table, and a bead's `Design ref`
is its Plan ref.

- `/implement DESIGN-NNN` resolves to the epic `<prefix>-DESIGN-NNN`. Bead identifiers are never
  renamed.
- A revision request or amendment record filed against such a document carries the current labels,
  `plan-revision` and `plan-amended`, with the document's path as `--spec-id`.

Remove this section once `docs/designs/` holds only archived plans whose beads are closed.
<!-- /codefall:legacy-designs -->

## `.codefall/skills/codefall-review/CUSTOMIZE.md`

<!-- codefall:legacy-designs -->
## Plans written before the rename

This project holds plans under `docs/designs/` as `DESIGN-NNN-slug.md`, written by `codefall-design`
before it became `codefall-plan`.

- A target that resolves to `docs/designs/DESIGN-NNN-slug.md` is a `plan` target, reviewed with the
  plan lenses; its task table is headed `Task Plan`.
- When an ADR is the target, search `docs/designs/` as well as `docs/plans/` for the plans that cite
  it.

Remove this section once `docs/designs/` holds only archived plans whose beads are closed.
<!-- /codefall:legacy-designs -->
