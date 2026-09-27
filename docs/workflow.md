# How codefall works

codefall is a set of verbs — skills a coding harness runs on request — plus the CLI that installs
them into a project. The verbs chain from an idea to open pull requests, and every step leaves
something in the repository or in the task graph that the next step reads. Humans decide at each
gate, and a human performs every merge to `main`. This file is the map; each verb's `SKILL.md`
under [`extensions/skills/`](../extensions/skills/) holds the procedure, and the
[README](../README.md) argues for it. This file is edited here; the three sections from *The chain*
on are copied into [`extensions/shared/workflow.md`](../extensions/shared/workflow.md), which `init`
installs, by `extensions/scripts/workflow-sync.sh --write`, and CI fails when the copy drifts.

## What `init` puts in place

`codefall init` (or `codefall create` for a new directory) asks which harnesses the project uses and
installs for each:

- the skills, into the harness's own skills directory (`.claude/skills/`, `.agents/skills/`), because
  that is the one place a harness finds them by convention;
- `.codefall/`, once for all harnesses: `settings.json` (what the project told `init`, read back by
  the verbs), `manifest.json` (what the last run wrote), `hooks/shared/` (the guard scripts), and
  `shared/` (the files every skill reads and the scripts a verb runs);
- two hooks: a `PreToolUse` guard that denies merges and pushes to the default branch, and, where
  the harness has the event, a `SessionStart` prime on what Beads knows plus a notice naming what
  needs attention — `main` moved, the environment stale, a runner nobody declared, a Beads
  precondition blocking. The notice reports and never pulls or runs anything;
- marked sections in the project's `AGENTS.md` — Codefall, Beads, Local environment, Testing — and
  the testing root with its `test-cases/` directory. The Codefall section is the frame the other
  three sit inside, and it points at `.codefall/shared/workflow.md`, the installed copy of the
  chain, the verbs beside it, and who is authoritative for what;
- the entries other tools read: `.ignore` for what codefall commits and nobody greps, `.gitignore`
  for the refresh stamp and a test run's output, and `.gitattributes` for a union merge of bd's
  append-only interaction log, each appended only when the file does not already name it.

`codefall doctor` checks that all of it is present and runnable. [ADR-006](adrs/ADR-006-install-layout.md)
records the layout.

## The chain

Each verb is explicitly invoked (`disable-model-invocation: true`), reports what it found, offers, and
applies only what the user takes. `refresh` is the one exception: it carries no such line, and an
agent may run it when the environment is stale. In order:

| Verb | Reads | Writes | Hands to |
| --- | --- | --- | --- |
| `envision` | whatever the user arrived with: a sentence, a pitch document, a folder of mockups | `docs/visions/VISION-NNN-slug.md`, the *why*; sources kept verbatim under `docs/visions/sources/` | `scaffold` requires one; `specify` may draw on one |
| `scaffold` | a vision; an interview for what a template cannot decide; the project's consult agents when a stack stays open | ratified ADRs, scoped `AGENTS.md` files, optionally project files, boundary lint, and the `start` and `update` scripts | a project ready for `specify` |
| `specify` | the idea or vision, and an audit of what already exists; the project's consult agents for a question of fact the user cannot answer | `docs/specs/SPEC-NNN-slug.md`, the *what*: requirements with EARS acceptance criteria, mirrored to the tracker as a parent issue and one child per requirement | `design` |
| `mock-up` | a design-tool export, or nothing | `docs/mockups/<slug>/`, matching the app's own design system | `design`; an issue labelled `requires-mockup` blocks design until it exists |
| `design` | the spec, the vision, the code; the project's consult agents for a technical point it cannot settle | `docs/designs/DESIGN-NNN-slug.md`, the *how*, scaled to the change; ADRs for hard-to-reverse choices; beads with dependency edges, each carrying its acceptance criteria and, where the task is verified through the wired product, the test case and its criteria | `implement` |
| `implement` | ready beads, an epic, or a design; the project's consult agents when a worker fails | a worktree per task, the test case before the code, verification against the bead's criteria and the project's checks, a pull request per task, walked in parallel waves until the frontier is empty | the human, who merges; `design`, for a disagreement that moves work, filed as a revision bead |
| `review` | anything live: uncommitted work, a branch, a PR, a commit range, a path, a document; the project's consult agents for what the reviewer could not settle | `.codefall/reviews/`, a JSON and Markdown pair per review; fixes on the target's branch for the findings the user takes | the human; `design`, for a deferred finding that moves work |
| `test` | what the project declares: suites, the changed subset, or one case in its `spec` or `agentic` modality | `.codefall/tests/`, a report per run; findings triaged, never an edit that makes a run pass | tracker issues on the user's word |

A contained fix skips the documents: `design` writes beads only when a change stays inside one
component and comes to a task or two, and `specify` is for features, not every change.

The chain runs backward at the moment a step finds an earlier document wrong. A verb amends any
upstream document, at any distance, in its own run and its own pull request, when the document's
lifecycle allows the edit (a `Draft` or `Ready` spec, design, or vision; never an `Active` vision
or a ratified ADR), the amendment is text that moves no work (no task row changed, no criterion a
bead cites retired or reworded; a spec is amended by appending), and the user takes it at the
confirmation the verb already holds. Every document between the change and the step is amended
together, or none is. What fails those tests is filed as a bead labelled `design-revision` with the
design's path as its `spec_id`; `design` lists those at its start, and its Revise mode closes each
one: amended into the document, turned into a task row, or rejected with why.
[ADR-008](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-008-upstream-amendments.md)
records the rule.

## Keeping the project current

Four verbs sit beside the chain rather than in it:

- **`equip`** builds and rebuilds what the other verbs need the project to have. One track is the
  local environment: `start` and `update`, declared under `local` in `.codefall/settings.json`
  ([ADR-005](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-005-local-environment-scripts.md)).
  The other is the test harness: a spec runner per surface, declared in `test.runners`, its
  commands recorded in the testing root's `AGENTS.md`
  ([ADR-007](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-007-test-cases.md)).
  One run equips one track, and setting a harness up is its own pull request. What its search
  leaves ambiguous is consulted on once before the one question.
- **`refresh`** is what to run instead of pulling by hand: fetch, fast-forward `main` when safe,
  `bd sync` the beads with their Dolt remote, run `start`, run `update` when the commit moved,
  record the commit in a git-ignored stamp. It never rebases a feature branch, stashes a dirty
  tree, or settles a conflict the sync halts on.
- **`graft`** brings a scaffolded project's documents up to the current templates, reporting each
  difference with its provenance and applying only what the user takes. It also handles first-time
  adoption of the stance on an existing repo.
- **`codefall init`** rerun (`upgrade`) reinstalls the skills and hooks for the harnesses already
  recorded, replacing its own marked sections and registrations and touching nothing else.

The scripts stay current at the point of introduction: a task that adds infrastructure, a
dependency, a migration, or generated code changes `start` or `update` in the same pull request.
`design` names it in the task, `implement` counts it toward done, and `review` carries a lens for it.

## Who is authoritative for what

- **Documents in the repository** are canonical for the why (vision), the what (spec), and the how
  (design). Each carries a `Status` that describes the document only.
- **The tracker** (GitHub Issues in this version) mirrors specs so people can see what is ready, in
  progress, and done; the spec document stays canonical.
- **Beads** is authoritative for task state from the moment a design's staged task plan is approved
  and becomes beads. A design keeps its task table — the tasks, edges, and design refs it decided,
  under the epic's ID — and never a copy of work state.
  The database on a machine is a local Dolt copy; the team's is `refs/dolt/data` on the git remote,
  and only `bd dolt pull` and `bd dolt push` move it. Every verb that writes a bead pushes after the
  write, and `refresh` syncs before work starts, so `bd ready` answers for the team and not for one
  checkout. A project with no Dolt remote is told so once and works on this machine alone.
- **A bead closes at done** — criteria verified, checks green, PR open — not at merge. Gates carry
  the merge seam: every PR gates a "landed" bead inside the epic, and the next session's `bd gate
  check` turns merges into bead state.
- **A human performs every merge to `main`.** `implement` ends at open PRs and a reported bottom-up
  merge order, and the guard hook denies the alternative in every harness.
- **Everything short of the merge is the verb's.** A verb that writes to the repository branches
  before its first file, commits what it wrote by path, and offers the push and the pull request;
  a document never sits uncommitted on `main`. `implement` does this per task; the document verbs,
  `scaffold`, `equip`, and `graft` follow the shared `landing.md` beside this file's installed copy.
- **The context that finds a problem never fixes it.** `review` runs in the first agent of the
  project's configured order that this machine can run, a subagent of the current harness with
  nothing configured, and `via=` overrides that for one run. Wherever it runs, that context reviews
  and this session triages and applies. A test criterion is never written from the
  implementation it verifies, and never edited to make a run pass.
- **A skill refuses only what it cannot do.** A missing runner, tool, or tracker profile is an exit;
  disagreement about size or fit is said aloud and then the user's call is followed.
- **A consult proposes; the session decides.** A run that cannot settle a technical question puts it
  once to the project's `consult` order of agents, a subagent of the current harness when nothing is
  configured, and reads the answer as analysis: never a write, never an ADR, never a stand-in for a
  preference the user has stated, and every consult named in the report.
