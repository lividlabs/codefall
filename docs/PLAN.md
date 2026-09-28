# Plan

The following items are outstanding in order to complete the first verion of the tool.

## Skills
- [ ] `report` — report/file a bug; requires a slightly different process as compared to specifiying a requirement
  - **Open Questions**
    - Does the bug report would take the place of a spec? Likely yes.
    - Does the bug report have a github issue and a markdown report? Probably.
- [ ] `fix` — (under consideration) fix could potentially be a combination of design+implement but specifically for bugs reports
- [x] `test` — run tests
  - unit, integration, and e2e
  - testing scope: full suite, a subset based on local/branch changes, a specific subset
  - a case in its `spec` modality through the project's runner, or in its `agentic` modality driven
    step by step; the format is `codefall-test/reference/case-file.md` and the run report is
    committed under `.codefall/tests/`
- [x] `equip` — build and rebuild the project's local scripts, `start` and `update`; followed as a
  reference by `scaffold` at code depth and by `implement` when a task introduces infrastructure,
  a dependency, a migration, or generated code
  - on an existing repo, search first and confirm with evidence: name the candidate scripts it
    found and ask whether to declare them or draft new ones; a candidate that fails the contract
    (not idempotent) is reported, never declared
  - [x] a second track for the test harness: search for a runner configuration, ask one question,
    set up the runner each surface takes, declare it in `test.runners`, record its commands in the
    testing root's `AGENTS.md`, and revise `update` for the runner's own install; setting a harness
    up is its own pull request
- [x] `refresh` — bring the checkout and the local environment current: start what is down, run
  `update`, record the stamp, explain failures in plain terms; never drafts a script
- [x] point-of-introduction rules — `design` names the script change in the task, `implement`
  counts it toward done, `review` carries a lens for it
- [x] test-case rules — `design` decides which tasks need a case and drafts its criteria into the
  bead, `implement` writes the case file from those criteria before the code and stops a bead whose
  harness is not equipped, `review`'s `tests` lens reads the case's citations
- [x] `scaffold` emits both scripts at code depth

## Command Line
- [ ] Set up `mise` config for the cli tool inside of init
- [ ] `create` command to start a new project
- [ ] `local` block in `.codefall/settings.json` declaring the `start` and `update` entry points;
  `doctor` checks they exist and are runnable
- [x] `test` block in `.codefall/settings.json` declaring the testing root and the runners; `init`
  asks for the root and creates the tree, `doctor` checks the declaration and the directory
- [ ] `init` writes the refresh rule into `AGENTS.md` through the marker mechanism, and names
  `equip` in its report when nothing is declared
- [ ] `preflight.sh` fetches and reports commits behind the default branch, a dirty tree, and the
  refresh stamp against `HEAD`; every verb reports it

### Landing order
Stacked PRs, in this order: settings schema plus doctor → init's `AGENTS.md` section → preflight
with the stamp → `equip` → `refresh` → the design/implement/review rules → scaffold at code depth.
The decision log entry for 2026-09-16 has the reasoning.

## Testing
- Test all of the skills against an actual project.
  - [x] `scaffold`
  - [x] `codefall-upgrade`
  - [ ] `envision`
  - [ ] `mock-up`
  - [ ] `specify`
  - [ ] `design`
  - [ ] `implement`
  - [ ] `review`
  - [ ] `report`
  - [ ] `test`
  - [ ] `equip`
  - [ ] `refresh`

## Completed

### Skills
- [x] `envision` — capture an idea as vision documents
- [x] `mock-up` — create or import mockups
- [x] `specify` — write a spec with acceptance criteria
- [x] `design` — write a design doc and create the task graph
- [x] `implement` — build ready tasks and open pull requests
- [x] `review` — review work, triage findings, apply fixes
- [x] `scaffold` — start a new project with ADRs and AGENTS.md files
- [x] `codefall-upgrade` — bring a project's installed codefall and its documents current
- [x] Shorten every SKILL.md to the ADR-004.2 guidelines

### Command Line
- [x] `init` — survey settings, install skills and hooks for each harness; `upgrade` alias
- [x] `doctor` — check the directory and tools are ready

### Release
- [x] Cross-platform binaries via GoReleaser (macOS and Linux)
- [x] release-please changelog and versioning
- [x] Signed packslip with each release
