<!-- BEGIN CODEFALL LOCAL -->
## Local environment

`codefall-refresh` runs `start` and `update`, declared under `local` in `.codefall/settings.json`.

- Run `/codefall-refresh` before new work and after a pull request merges; it ends on the
  default branch, wherever it starts. Never pull by hand.
- A change that adds infrastructure, a dependency, a migration, or generated code changes `start`
  or `update` in the same pull request; `codefall-equip` is the procedure.
- No `local` block declared: run `/codefall-equip` first.

Instructions from the user or elsewhere in this file take precedence over this section.
<!-- END CODEFALL LOCAL -->
