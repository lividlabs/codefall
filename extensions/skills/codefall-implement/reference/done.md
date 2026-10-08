# Verification and done

When a bead is done, where the commands that check it come from, and what they check. Read at
step 3, before the approach is formed, and again at step 6. `codefall-fix` reads it for its one bead.

A bead is done when three things are true: **its acceptance criteria hold, the project's checks are
green, and its PR is open.** Done is not merged.

**Where the commands come from.** Implement hardcodes no build, lint, or test invocation:

1. `.codefall/skills/codefall-implement/CUSTOMIZE.md` — verb-specific tuning, such as a fast subset per bead
   with the full suite reserved for pre-PR;
2. the project's `AGENTS.md` — scaffolded projects carry the command list in their verification
   section;
3. inference from the repo (`package.json` scripts, `Makefile`, `go.mod`) — stated at the go gate,
   with an offer to record the inferred commands in `AGENTS.md`.

The resolved list is passed into worker prompts. Workers re-derive nothing.

**The checks:**

- The project's own verification commands, run until clean.
- The harness's built-in passes on the bead's own diff, where the harness provides them:
  `simplify` always; `code-review` and `security-review` when available. These are checks inside
  implement, not review — implement renders no verdict on its own work.
- The bead's acceptance criteria, checked one by one. What passed goes into the close reason. A
  bead with no acceptance field falls back to the plan's Hard Constraints plus the spec's
  criteria, and the close reason still records what was verified.

**Tests are part of done, not a follow-up.** So are the local scripts: a bead whose criteria name
the `start` and `update` change, or whose diff adds infrastructure, a dependency, a migration, or
generated code, changes the declared scripts in the same PR, following the local track of
`codefall-equip` as that skill's followers reference says. The bead is the confirmation; the PR
body names the change.

**A test case the criteria name is written before the code**, from those criteria and from nothing
else — never the sibling spec, the application's code, or a pull request's own text. Its format is
`../../codefall-test/reference/case-file.md` and it lands at `<root>/test-cases/<area>/<slug>.md`
under the testing root; the spec follows it where the modality calls for one. The case counts
toward done. **Running it does not** — that is `codefall-test`'s. What is checked here is that
`../../../../.codefall/shared/check-cases.sh` passes and that the runner's run-one command in
`<root>/AGENTS.md` collects the spec.
