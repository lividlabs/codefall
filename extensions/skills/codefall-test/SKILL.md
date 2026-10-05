---
name: codefall-test
description: Run what the project declares — every suite, the subset the changed files reach, a named subset — or one test case in one of its two modalities, or every case an epic's beads name on the epic's own branch before the merge. A spec case runs through the project's own runner and reports that runner's pass or fail. An agentic case is worked step by step through a driver the session already has, and each criterion is judged against what the run made observable. Every run writes a report under .codefall/tests/, with runner output, logs, and run-scoped state left git-ignored under the testing root's .artifacts/. Findings are triaged and become tracker issues and, in a delivery, children of the epic, only on the user's word; a criterion is never edited to make a run pass. Use when the user says /codefall-test, "run the tests", "run the suite for what changed", "test the epic", or "run <case> agentically".
argument-hint: "[suites | changed | <suite> | <area>/<slug> | <epic> | DESIGN-NNN] [modality=spec|agentic] [variant=<name>]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - Agent
  - Skill
---

# Test

Run what the project declares, judge what the run produced, and write down what happened.

One invocation is one run of one target. Nothing carries over: running the same case again is a new
run, with its own report.

**The case file is the whole specification.** Read the case, run it, judge it against the criteria
written there. The sibling spec and the application's code say what the product does, not what it
should do, and neither is read to decide a verdict.

Paths that start with `reference/` or `../` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills directory: it names
a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/case-file.md` — the case-file format: frontmatter, body sections, where a criterion may
  come from, variant-conditional and dropped criteria. Read when a case target is resolved.
  `codefall-implement` reads it to write one.
- `reference/agentic-run.md` — how an agentic run is driven and judged. Read at step 4 for an
  agentic case, before anything is driven.
- `reference/run-report.md` — the report's naming, its Markdown shape, its JSON record, and what is
  committed. Read at step 5.
- `reference/triage.md` — the four classes a finding falls into, and what happens to each, in and
  out of a delivery. Read at step 6 when a run found something.
- `run.schema.json` — the shape of the JSON record a report is written beside.
- `../codefall-review/reference/targets.md` — how an epic's branch is found. Read at step 2 for an
  epic target.
- `../codefall-implement/reference/beads.md` — the `code` and `design` forms a finding is filed in
  under the epic. Read at step 6 for an epic target.
- `../../../.codefall/shared/delivery.sh <epic>` — the delivery's one-line chart. Run at the
  confirmation and in the report for an epic target.

## Scope

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Running the declared suites | Installing or declaring a runner | `codefall-equip` |
| Running one case in either modality | Writing a case file | `codefall-implement` |
| Judging an agentic run against the case's criteria | Deciding which tasks need a case | `codefall-design` |
| Reporting a run and triaging what it found | Adding a criterion the spec is missing | `codefall-specify` |
| Recording a run's real side effects and cleaning them up | Editing a case, a spec, or application code | the work that follows |
| Naming a remedy a run needs | Running that remedy | the user |

## Targets

The argument's shape decides what is run.

| Argument | Target | Resolved by |
| --- | --- | --- |
| `suites`, or *(none)* | every declared suite | [the three sources](#suite-targets) |
| `changed` | the suites the changed files reach | `git diff` against the default branch, mapped onto the runner's own filter |
| a suite name | that suite alone | the three sources |
| `<area>/<slug>` | that case | `<root>/test-cases/<area>/<slug>.md` |
| a path under `<root>/test-cases/` | the case that path names | the file itself |
| an epic ID, or `DESIGN-NNN` | the epic's work: every case its children's criteria name, plus `changed`, on the epic's branch | `bd list --parent <epic> --all --json --limit 0`, each `<area>/<slug>` in the acceptance criteria; [Epic targets](#epic-targets) |

`<root>` is the testing root `test.dir` declares in `.codefall/settings.json`. The default branch is
what preflight's `default_branch` line names. An argument that resolves to nothing is a stop, not a
guess — say what was looked for and where.

**Modality.** `modality=spec` or `modality=agentic` picks one. With nothing said, a case declaring
one modality runs in it and a case declaring both is asked about. A case whose `modalities` does not
carry the requested modality is refused, naming what it does declare.

**Variants.** Every variant runs, one run each, unless `variant=<name>` names one. A name no variant
carries is a stop.

## Preflight

Every target starts here.

```bash
"../../../.codefall/shared/preflight.sh" .
```

Read the lines. The Beads lines matter to an epic target alone: `beads` not `ok` there is a stop
with the remedy `codefall-implement`'s preflight table names; every other target ignores them.

| Line | What it says | What happens |
| --- | --- | --- |
| `test=undeclared` | settings carry no `test` block | Say `codefall upgrade` declares the testing root, and stop |
| `test=unequipped` | no runner is declared | A case target stops and names `/codefall-equip`; a suite target carries on |
| `test=unknown` | there is a `test` block, and preflight had no jq to read it | Read the block from `.codefall/settings.json` and act on what it says; `runners` absent or empty is `unequipped` |
| `refresh=stale`, or `behind` above `0`, on the default branch | the checkout or the environment is behind | Say so, offer `/codefall-refresh`, and wait |
| `behind` above `0` on another branch | the branch is missing commits the default branch has | Say so and that the rebase is the user's; carry on. Never offer `/codefall-refresh`, which leaves the branch |
| `refresh=stale` on another branch | the environment may not match the branch | Say so, name the declared `local.update` command to run here, and wait |
| `refresh=undeclared` | no `local` block | Say so and name `/codefall-equip`; an agentic run stops, since it has no declared `start` |

A preflight that emits no `test=` line at all is an older install: read the `test` block from
`.codefall/settings.json` directly and read it the same way.

**Never run the remedy.** `codefall init`, `/codefall-equip`, and `/codefall-refresh` are the user's
to run.

## The confirmation

Nothing runs before the user sees what will. Name the target, the commands and where they came from,
the variants, the driver for an agentic run, and where the report will land:

```
checkout/place-order, agentic modality, 2 variants (card, saved-card).
Run command: npm run e2e -- checkout/place-order (testing/AGENTS.md, Runners).
Driver: the Playwright MCP browser tools in this session — proving with one call to
http://localhost:3000 before the run depends on them.
Report: .codefall/tests/<timestamp>-checkout-place-order.md and .json.

Run now?
```

A run that will create real side effects says so here, and says what cleans them up. An epic target
opens with the chart line and names the branch the run will check out.

## Suite targets

**The commands come from the same three sources `codefall-implement` resolves its own from**, in the
same order — its *Verification and done* reference states them — reading
`.codefall/skills/codefall-test/CUSTOMIZE.md` in place of implement's own customization file. Say at
the confirmation which source answered, and offer to record an inferred command in `AGENTS.md`.

**The changed-files subset** is `git diff --name-only` against the default branch plus the working
tree's own changes, mapped onto the runner's filter: paths for a runner that takes paths, packages
for one that takes packages. Where the runner cannot filter — or the changed files map onto nothing
the runner recognises — say so and run the full suite rather than silently running less.

A suite run reports the runner's own pass and fail. The agentic vocabulary is not used for it.

## Case targets

Read the case file in full before anything runs, and `reference/case-file.md` beside it.

**The run command comes from `<root>/AGENTS.md`**, its Runners section, which carries one line per
runner with its run-all and run-one commands. No Runners line for the case's runner is a stop that
names `/codefall-equip`.

- **`spec`** runs the runner's run-one command against the case's sibling spec, once, and reports
  the runner's own pass or fail. Its output, traces, and HTML reports land under
  `<root>/.artifacts/`, which is git-ignored.
- **`agentic`** follows `reference/agentic-run.md`: the application is brought up through the
  declared `start` command, setup runs as the scripts the Preconditions name, the `## Steps` are
  worked through a driver, and each criterion gets a verdict from what the run made observable.

**Where an agentic run happens.** One variant runs in this session. A case with several variants
runs each variant in its own subagent, one at a time, so that nothing one variant saw reaches the
next; each returns its verdicts, attempt counts, side effects, and anomalies, and this session
writes the report and triages. A subagent that cannot reach the driver is a stop: say so, and run
the variants here instead.

## Epic targets

An epic's work is tested where it stands, before the merge: the round's end-to-end run, and the
"last test passed" half of the delivery's done condition. The cases are every `<area>/<slug>` the
epic's children name in their acceptance criteria, each in the modality it declares, plus the
`changed` suites against the default branch; a child whose criteria name no case contributes
nothing.

**The branch** is the epic branch, or the serial stack's top, found as
`../codefall-review/reference/targets.md` says. It is checked out in the primary checkout the way
`codefall-review` checks a branch out for its fixes: `git fetch origin`, then `git checkout
<branch>`, when the tree is clean. **A dirty tree stops the run** before anything is checked out;
say so and leave the tree as it is. Then run the preflight again on the branch: `refresh=stale`
there means the environment matches `main` and not this branch, so run the declared `local.update`
once — the one place this verb runs it, because the user asked for this branch — and proceed;
`behind` on the branch is reported and carries on, as the table says.

The report names the branch left checked out. `/codefall-refresh` is the way back to the default
branch once the run is over.

## The report

**A report is written for every run**, including one that passed and one that ended in an error.
Naming, the Markdown shape, the JSON record, and what is committed are in `reference/run-report.md`.
Reports are committed under `.codefall/tests/`; everything a runner or a driver produced stays
git-ignored under `<root>/.artifacts/`.

### Kept out of codebase search

A `.ignore` file beside `.codefall/` holds the line `.codefall/tests/`, so ripgrep-backed harnesses
skip old run reports. `codefall init` writes it and `codefall doctor` warns when it is missing.

**Check it before the run, and offer:**

> `.ignore` doesn't list `.codefall/tests/`, so this run's report will show up in codebase searches.
> Add the line?

On yes, append it — never replace the file. On no, carry on and say nothing further. Say nothing at
all when the line is already there.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Check preconditions

Run [the preflight](#preflight) and act on the lines. Read `.codefall/settings.json` for the `test`
block, the project's `AGENTS.md` (root and scoped), and `<root>/AGENTS.md`. Check the `.ignore`
entry and offer to add it when it is missing.

### 2. Resolve the target

By [the table](#targets). For a case target, read the case file and `reference/case-file.md`, refuse
a modality the case does not declare, and resolve the variants. A case file that does not match the
format is reported, never repaired here: run `../../../.codefall/shared/check-cases.sh` and hand the
user what it lists. For a suite target, resolve the commands and say which source answered. For an
epic target, list the children and their cases, find the branch, and check it out per
[Epic targets](#epic-targets); then resolve each case and the `changed` suites as above.

### 3. Confirm

Present [the confirmation](#the-confirmation) and wait. For an agentic run, the driver is named here
and proved before the run depends on it, per `reference/agentic-run.md`.

### 4. Run

A suite or a spec case runs its command and the runner reports itself. An agentic case runs per
`reference/agentic-run.md`, once per variant, with per-criterion verdicts and attempt counts taken
as the run goes rather than reconstructed at the end.

### 5. Write the report

Read `reference/run-report.md`. Write both files under `.codefall/tests/`. A run interrupted part way
still gets a report, marked as the run it was, with every side effect it created recorded.

### 6. Triage

Read `reference/triage.md` when anything failed, was unreachable, or turned up in the anomaly sweep.
Classify each, write the working notes it names, and stop there: a finding becomes a tracker issue,
and in a delivery a child of the epic, only on the user's explicit word. **For an epic target the
question is one line** — "I found N problems. Fix them all? (I recommend yes.)" — and the person
may take all, some, or none; each one taken is filed as the triage reference says, an issue and a
`deferred` child of the epic.

### 7. Report

The chart line for an epic target; the verdicts, the report's path, the driver that ran, the side
effects and their disposition, the attempt counts worth seeing, the anomaly sweep, the triage classes,
and the beads filed. No summary of what went well. For an epic target, one line goes on the epic's
notes — `round N (test) [date]: the verdict and what was filed` — per *Session end* in
`../codefall-implement/reference/beads.md`, and anything decided here that no bead, report, or
issue holds goes in a `bd comment` on the epic; the report then says it is safe to `/clear`.
**End with what the user does next.** For an epic target: when children were filed, say this run
is now running `codefall-implement <epic>` to fix them, then run that skill with the epic as its
argument, through the `Skill` tool, in this session; its go gate reopens the children, and its
report ends the session with its own one command. When nothing was filed, this run is the last
test passed, and the merge order from the implement report is what remains. Otherwise per class: a real bug with an issue filed goes
to `/codefall-fix #<issue>` when the change is contained and `/codefall-design #<issue>` otherwise,
or to `/codefall-report <run report path>` when the user wants a bug document first; a real bug not
filed waits for the user's word; a wrong expectation goes to the decision that supersedes the
criterion; a flake or agent variance is recorded and waits for the next run; when every case passed,
nothing.

## Rules

- **The case file is the specification.** Never read the sibling spec, the application's code, or a
  pull request's text to decide what a criterion means.
- **Never edit a criterion to make a run pass.** A criterion that is wrong is triaged as a wrong
  expectation and changed as its own work.
- **A run never edits a case, a spec, or application code.** It produces a report and triage notes.
- **Nothing is mocked, faked, or intercepted**, at any layer. A state that cannot be forced through
  the product's own interfaces is `unreachable`, never simulated.
- **Refuse a modality the case does not declare**, and say which it does.
- **Refuse the agentic modality when no driver works**, and say how a checked-in MCP server entry
  supplies one.
- **A driver failure is reported, never worked around.**
- **Fixed messages only.** Every message a run sends is a string from the case's `messages` map,
  never reworded and never composed.
- **Setup may retry, bounded; an assertion never does.** Attempts are recorded even when the step
  eventually succeeded.
- **Record a pass when you see one.** An observation made in passing is still an observation.
- **A report is written for every run**, and committed under `.codefall/tests/`. Runner output,
  traces, logs, and run-scoped state stay under `<root>/.artifacts/`.
- **Side effects are recorded the moment they exist**, and cleaned up the way `<root>/AGENTS.md`
  says.
- **A finding becomes a tracker issue, and in a delivery a `deferred` child of the epic, only on
  the user's word**, and the existing issues are searched first. In a delivery the word is the
  answer to one question, "Fix them all?", and the children it files are built by
  `codefall-implement`, which this session runs after the report; the person types nothing between
  the test and the fix.
- **Name the remedy, never apply it.** This verb does not run `codefall init`, `/codefall-equip`,
  or `/codefall-refresh`, and runs the declared `update` only on the epic's branch it checked out
  itself.
- **`/codefall-refresh` is offered on the default branch only, during a run.** It ends on the
  default branch, so on any other branch it takes the user away from what is under test; once an
  epic run is over, the report names it as the way back.
- **An argument that resolves to nothing is a stop**, not a guess at the nearest case.
