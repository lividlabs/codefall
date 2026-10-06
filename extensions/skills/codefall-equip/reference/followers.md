# When another verb follows this skill

`codefall-scaffold` and `codefall-implement` read `codefall-equip`'s local track as the procedure;
nothing invokes it through the harness. Both follow the local track only: **a test harness is
never set up inside another verb's pull request.** When a task needs a case and no runner is
declared, `codefall-implement` says so, names `/codefall-equip`, and leaves those beads unstarted.

- **`codefall-scaffold`, at runnable-skeleton depth.** Nothing exists to search for; skip step 2.
  Draft from what scaffold emitted — its manifest, its compose file if any, its migration tool —
  declare, and prove the scripts as part of scaffold's own verification. At docs-only depth write
  nothing and name `codefall-equip` in scaffold's report as owed work.
- **`codefall-implement`, on a task that introduces infrastructure, a dependency, a migration, or
  generated code.** The bead's acceptance criteria name the script change; the bead is the
  confirmation. Read the declared scripts, revise them per step 3 of the local track for what the
  task introduced, keep the contract, and change the declaration only when an entry point moved.
  The change lands in the task's own pull request and is named in its body.
- **Neither follows the agents track or the landing track.**
