# How codefall works

codefall is a set of verbs — skills a coding harness runs on request — plus the CLI that installed
them into this project. The verbs chain from an idea to open pull requests, and every step leaves
something in the repository or in the task graph that the next step reads. Humans decide at each
gate, and a human performs every merge to `main`. This file is the map; each verb's `SKILL.md` in
the harness's skills directory holds the procedure.

This is a copy. The source is `docs/workflow.md` in the
[codefall repository](https://github.com/lividlabs/codefall-cli/blob/main/docs/workflow.md), which
is where it is edited; `codefall init` installs the copy and replaces it on a rerun.

## Contents

- The chain
- Keeping the project current
- Who is authoritative for what

## The chain

`envision`, `specify`, `report`, `fix`, `mock-up`, `scaffold`, `upgrade`, and `equip` are invoked
deliberately by a user (`disable-model-invocation: true`). `design`, `implement`, `test`, `review`, and `refresh`
carry no such line, so an agent may run these too, and a session can carry a design through
implementation, review, and test without a person typing each verb. Whichever way a verb starts, it
reports what it found, offers, and applies only what the user takes. In order:

| Verb | Reads | Writes | Hands to |
| --- | --- | --- | --- |
| `envision` | whatever the user arrived with: a sentence, a pitch document, a folder of mockups | `docs/visions/VISION-NNN-slug.md`, the *why*; sources kept verbatim under `docs/visions/sources/` | `scaffold` requires one; `specify` may draw on one |
| `scaffold` | a vision; an interview for what a template cannot decide; the project's consult agents when a stack stays open | ratified ADRs, scoped `AGENTS.md` files, optionally project files, boundary lint, and the `start` and `update` scripts | a project ready for `specify` |
| `specify` | the idea or vision, and an audit of what already exists; the project's consult agents for a question of fact the user cannot answer | `docs/specs/SPEC-NNN-slug.md`, the *what*: requirements with EARS acceptance criteria, mirrored to the tracker as a parent issue and one child per requirement | `design` |
| `report` | the person who saw a bug, interviewed; the running product, driven through their steps; a test run's report or an issue they filed | `docs/bugs/BUG-NNN-slug.md`, what is wrong: the steps, the expected and actual result, evidence committed beside it, whether it reproduced, and EARS acceptance criteria, mirrored to one tracker issue | `design` |
| `mock-up` | a design-tool export, or nothing | `docs/mockups/<slug>/`, matching the app's own design system | `design`; an issue labelled `requires-mockup` blocks design until it exists |
| `design` | the spec, the vision, the code; the project's consult agents for a technical point it cannot settle | `docs/designs/DESIGN-NNN-slug.md`, the *how*, scaled to the change; ADRs for hard-to-reverse choices; beads with dependency edges, each carrying its acceptance criteria and, where the task is verified through the wired product, the test case and its criteria; or, with decisions the person could not settle, a `Draft` carrying them for an engineer's run | `implement`; `design` again, for a `Draft` with decisions needed |
| `implement` | ready beads, an epic, or a design; the project's consult agents when a worker fails | a worktree per task, the test case before the code, verification against the bead's criteria and the project's checks, a pull request per task, walked in parallel waves until the frontier is empty | the human, who merges; `design`, for a disagreement that moves work, filed as a revision bead |
| `review` | anything live: uncommitted work, a branch, a PR, a commit range, a path, a document; the project's consult agents for what the reviewer could not settle | `.codefall/reviews/`, a JSON and Markdown pair per review; fixes on the target's branch for the findings the user takes | the human; `design`, for a deferred finding that moves work |
| `test` | what the project declares: suites, the changed subset, or one case in its `spec` or `agentic` modality | `.codefall/tests/`, a report per run; findings triaged, never an edit that makes a run pass | `report`, for a real bug the user wants filed |

A contained fix skips the documents: `design` writes beads only when a change stays inside one
component and comes to a task or two, and `specify` is for features, not every change. A bug starts
at `report` instead of `specify`, and `design` finds its cause before it chooses a tier. `fix` runs
the two steps for one such change in a single run: `design` at tier 0 for one bead, then `implement`
on it, behind one confirmation, stopping to name `design` when the work needs a document, an ADR, or
a second bead. It takes a bead, a bug report, an issue, or a description.

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
  The third is the agents: how another harness is called, under `harnessConfig`, and who reviews
  and consults, in the `agents` lists, set up from any harness
  ([ADR-009.4](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-009.4-agents.md)).
  One run equips one track, and each lands as its own pull request. What a search leaves ambiguous
  is consulted on once before the one question.
- **`refresh`** is what to run instead of pulling by hand: fetch, fast-forward `main` when safe,
  `bd sync` the beads with their Dolt remote, run `start`, run `update` when the commit moved,
  record the commit in a git-ignored stamp. It never rebases a feature branch, stashes a dirty
  tree, or settles a conflict the sync halts on.
- **`upgrade`** brings the install and the documents current, in that order. When the manifest
  records an older version than the binary's, it offers `codefall upgrade`, runs it only on a yes,
  and puts the command's breaking changes and its question to the user. Then it brings a scaffolded
  project's documents up to the current templates, reporting each difference with its provenance
  and applying only what the user takes. It also handles first-time adoption of the stance on an
  existing repo. A missing template the user declines is recorded and not offered again, and
  `/codefall-upgrade adopt` offers the declined ones again.
- **`codefall upgrade`** prints the breaking changes recorded between the installed version and the
  binary's and asks to continue, then reinstalls the skills, shared files, and hooks for the harnesses
  the settings record, replaces its own marked sections and registrations, rewrites a harness name
  still spelled the old way, removes what the previous install wrote that this one does not ship, and
  touches nothing else. `codefall init` runs once and refuses a project that has a manifest, naming
  `upgrade`
  ([ADR-010](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-010-upgrade.md)).

The scripts stay current at the point of introduction: a task that adds infrastructure, a
dependency, a migration, or generated code changes `start` or `update` in the same pull request.
`design` names it in the task, `implement` counts it toward done, and `review` carries a lens for it.

## Who is authoritative for what

- **Documents in the repository** are canonical for the why (vision), the what (spec), what is
  wrong (bug report), and the how (design). Each carries a `Status` that describes the document only.
- **The tracker** (GitHub Issues in this version) mirrors specs and bug reports so people can see
  what is ready, in progress, and done; the document stays canonical.
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
  `scaffold`, `equip`, and `upgrade` follow the shared `landing.md` beside this file's installed copy.
- **The context that finds a problem never fixes it.** `review` runs in the first agent this machine
  can run from the review list of the project's entry for the harness the session is in, else its
  `default` entry, a subagent of the current harness with nothing configured, and `via=` overrides
  that for one run. Wherever it runs, that context reviews
  and this session triages and applies. A test criterion is never written from the
  implementation it verifies, and never edited to make a run pass.
- **A skill refuses only what it cannot do.** A missing runner, tool, or tracker profile is an exit;
  disagreement about size or fit is said aloud and then the user's call is followed.
- **A consult proposes; the session decides.** A run that cannot settle a technical question puts it
  once to the consult list of the project's entry for the harness the session is in, else its
  `default` entry, a subagent of the current harness when nothing is configured, and reads the answer as analysis: never a write, never an ADR, never a stand-in for a
  preference the user has stated, and every consult named in the report.
