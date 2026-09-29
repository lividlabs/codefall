codefall
--------

**Opinionated skills for the software development lifecycle.**

This directory is the extension. Everything above it is packaging.

| Skill | Does |
| --- | --- |
| [`codefall-envision`](skills/codefall-envision/SKILL.md) | Get an idea onto paper before anyone specifies or scaffolds it: a numbered vision document under `docs/visions/` carrying the problem, the rough shape of an answer, and what nobody has decided yet. |
| [`codefall-scaffold`](skills/codefall-scaffold/SKILL.md) | Start a new project on the Clean + package-by-component stance: ratified ADRs, scoped `AGENTS.md`, optionally project files and boundary lint. |
| [`codefall-upgrade`](skills/codefall-upgrade/SKILL.md) | Bring a project's install and docs current: offer `codefall upgrade` when the manifest is behind the binary, then report what changed in the templates since the project's version, with per-file provenance, and apply only what the user takes. Also handles first-time adoption of the stance. |
| [`codefall-specify`](skills/codefall-specify/SKILL.md) | Turn a feature idea into a specification another session can implement: a spec document under `docs/specs/` holding requirements with EARS acceptance criteria, mirrored to the issue tracker. |
| [`codefall-report`](skills/codefall-report/SKILL.md) | Turn a bug into a report another session can fix: interview the person who saw it for the steps, the expected and actual result, screenshots, and the environment, try to reproduce it on the spot, and write a bug report under `docs/bugs/` with acceptance criteria, mirrored to the issue tracker. |
| [`codefall-fix`](skills/codefall-fix/SKILL.md) | Fix something small in one run: `design` at tier 0 for one bead, then `implement` on it, behind one confirmation instead of two. Takes a bead, a bug report, an issue, or a description; stops and names `design` when the work needs a design document, an ADR, or more than one bead. Never merges to `main`. |
| [`codefall-mock-up`](skills/codefall-mock-up/SKILL.md) | Get the visual surface of a feature into the repository under `docs/mockups/`: import what a design tool exported, or make the mockup here, matching the app's own design system. |
| [`codefall-design`](skills/codefall-design/SKILL.md) | Decide how a feature gets built and put the work into the graph: a design document under `docs/designs/` scaled to the change, ADRs for hard-to-reverse choices, and the tasks in Beads with their dependency edges, each carrying the acceptance criteria it is verified against and the test case where one is called for. |
| [`codefall-implement`](skills/codefall-implement/SKILL.md) | Execute the graph: claim ready beads, build each in an isolated worker worktree with tests as part of done, write the test case a bead's criteria name before the code, verify against acceptance criteria, open PRs, and walk the waves until the frontier is empty. Never merges to `main`, and never sets a test harness up. |
| [`codefall-review`](skills/codefall-review/SKILL.md) | Review something and fix what the user accepts: uncommitted work, a branch, an open PR, a path, a document, or a description of what to look at. A subagent or another harness reviews, the session triages with the user and applies what they take, and every finding is committed under `.codefall/reviews/`. |
| [`codefall-test`](skills/codefall-test/SKILL.md) | Run what the project declares: every suite, the subset the changed files reach, a named subset, or one test case in its `spec` or `agentic` modality. A spec case runs through the project's own runner; an agentic case is driven step by step and judged against the case's criteria. Every run is reported under `.codefall/tests/`, and findings are triaged rather than turned into edits that make a run pass. |
| [`codefall-equip`](skills/codefall-equip/SKILL.md) | Equip a project with the three things the other verbs need it to have: the local-environment scripts `codefall-refresh` runs — `start` and `update`, declared under `local`; the test harness `codefall-test` runs cases through — a spec runner per surface, its configuration pointed at the testing root, declared in `test.runners` and with its commands recorded in the testing root's `AGENTS.md`; and the agents `codefall-review` and every consult reach for — how a harness is called, under `harnessConfig`, and the `agents` lists, set up from any harness. Finds what the project already has or drafts it from what the repository or the session shows. One track per run; scaffold and implement follow the local one as the procedure. |
| [`codefall-refresh`](skills/codefall-refresh/SKILL.md) | Bring the checkout, the beads, and the local environment current: fetch, fast-forward the default branch when that is safe, sync the Beads database with its Dolt remote, run the declared `start` and `update`, record the commit the environment now matches, and turn a failure into a sentence that says what to do. Safe to run at any time. |

`codefall-envision`, `codefall-specify`, `codefall-report`, `codefall-fix`, `codefall-mock-up`,
`codefall-scaffold`, `codefall-upgrade`, and `codefall-equip` are explicitly invoked and carry
`disable-model-invocation: true`, so none of them fire on their own. `codefall-design`,
`codefall-implement`, `codefall-test`, `codefall-review`, and `codefall-refresh` carry no such line,
so an agent may run these too, and a session can carry a design through implementation, review, and
test without a person typing each verb. The extension also ships hooks per harness, defined under
[`hooks/`](hooks/): a `PreToolUse` guard that denies merges and pushes to the default branch
everywhere, plus, for the harnesses that have the event (Claude Code, Codex, and OpenCode), a
`SessionStart` prime on what Beads knows and a notice naming what the project needs done — the
checkout behind the default branch, an environment that has not been refreshed since `HEAD` moved, a
testing root or a runner nobody has declared, a Beads precondition that is blocking. The notice
reads the shared preflight, prints nothing when everything is current, and reports only: no hook
pulls or runs the project's `update`. The shared scripts both hooks run live in `hooks/shared/`;
`codefall init` merges the definitions or copies the plugin.

`codefall init` copies three of the directories here and leaves the rest. `skills/` goes into each
chosen harness's own skills directory, because that is the only part a harness finds by convention.
`hooks/shared/` and [`shared/`](shared/) go into the project's `.codefall/`, once per install
whatever harnesses were chosen, because every path that reaches them is codefall's at both ends — a
skill names a shared file `../../../.codefall/shared/<file>`, and every harness's hook definition
names `.codefall/hooks/shared/…`. The per-harness definitions under `hooks/<harness>/` are read
straight from the binary and never copied, and so are the documents under [`agents/`](agents/): the
four sections `init` writes into a project's `AGENTS.md` and the two skeletons it writes at the
testing root. This file, [`AGENTS.md`](AGENTS.md),
[`skills/AGENTS.md`](skills/AGENTS.md), [`docs/`](docs/), and each skill's `NOTES.md` are written for
someone working on codefall and are installed nowhere.
[ADR-006](../docs/adrs/ADR-006-install-layout.md) records the layout.

See the [repository README](../README.md) for the architectural stance, the surface catalog, and
installation instructions.

## License

[MIT](../LICENSE) — Copyright (c) 2026 Livid Labs, LLC, authored by Dave Jensen.

The templates under `skills/codefall-scaffold/templates/`, and everything `codefall-scaffold` copies
from them into your project, are additionally available under [0BSD](../LICENSE): no attribution, no
notice, no obligation. Your architecture documents are yours.
