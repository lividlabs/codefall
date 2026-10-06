# How codefall works

codefall is a set of verbs — skills a coding harness runs on request — plus the CLI that installs
them into a project. The verbs chain from an idea to open pull requests, and every step leaves
something in the repository or in the task graph that the next step reads. Humans decide at each
gate; a person merges every code pull request, and a GitHub Action the project installs merges a
document pull request once a person adds the `auto-merge` label to it or approves it. This file is
the map; each verb's `SKILL.md` under [`extensions/skills/`](../extensions/skills/) holds the
procedure, and the [README](../README.md) argues for it. This file is edited here; the three
sections from *The chain*
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
  for the refresh stamp, the per-user `.codefall/user.json`, and a test run's output, and
  `.gitattributes` for a union merge of bd's append-only interaction log, each appended only when
  the file does not already name it.

`codefall doctor` checks that all of it is present and runnable. [ADR-006](adrs/ADR-006-install-layout.md)
records the layout. `codefall config` changes who reviews and consults in `settings.json` and the persona in
`user.json` afterwards, one command at a time and without prompting.

## The chain

`upgrade` and `equip` are invoked deliberately by a user (`disable-model-invocation: true`), because
each changes the install or the project's settings. Every other verb an agent may run too, and **a
verb runs the verb upstream of it** when the work needs that verb's judgment: `specify` runs
`mock-up` for a requirement with a visual surface, `design` runs `specify` to settle a `Draft` spec
and `mock-up` for a requirement still waiting on one, and `specify` runs `envision` when a vision
needs more than a text amendment, and `review` and `test` run `implement` on the epic to build the
problems the person took. It runs in the same session and on the same branch, under the
confirmation the verb already holds, and asks the person only for a product decision; nobody is
told to run a command and come back. `design` with technical decisions it set aside asks once
whether to settle them with sensible defaults now or leave them for an engineer, and on "now"
reaches `Ready` and beads in the same run. Whichever way a verb starts, it reports what it found,
offers, and applies only what the user takes.

The work on one epic from the design's graph to the human merge is a **delivery**, taken in
**rounds** of `implement`, `review`, and `test`. A delivery ends when the epic has no open children
and the last test run against it passed. A *run* is one invocation of one verb; a *session* is one
conversation. Every handoff between verbs is a bead, a document, a report, or a pull request, so a
`/clear` between verbs loses nothing, and that is the intended way to run a delivery: each verb's
report ends with one next command. In order:

| Verb | Reads | Writes | Hands to |
| --- | --- | --- | --- |
| `envision` | whatever the user arrived with: a sentence, a pitch document, a folder of mockups | `docs/visions/VISION-NNN-slug.md`, the *why*; sources kept verbatim under `docs/visions/sources/` | `scaffold` requires one; `specify` may draw on one |
| `scaffold` | a vision; an interview for what a template cannot decide; the project's consult agents when a stack stays open | ratified ADRs, scoped `AGENTS.md` files, optionally project files, boundary lint, and the `start` and `update` scripts | a project ready for `specify` |
| `specify` | the idea or vision, and an audit of what already exists; the project's consult agents for a question of fact the user cannot answer | `docs/specs/SPEC-NNN-slug.md`, the *what*: requirements with EARS acceptance criteria, mirrored to the tracker as a parent issue and one child per requirement; the mockups its requirements need, made through `mock-up` in the same run | `design` |
| `report` | the person who saw a bug, interviewed; the running product, driven through their steps; a test run's report or an issue they filed | `docs/bugs/BUG-NNN-slug.md`, what is wrong: the steps, the expected and actual result, evidence committed beside it, whether it reproduced, and EARS acceptance criteria, mirrored to one tracker issue | `design` |
| `mock-up` | a design-tool export, or nothing; run by `specify` or `design` for a requirement that needs one, or alone | `docs/mockups/<slug>/`, matching the app's own design system | `design`; an issue still labelled `requires-mockup` is one `design` runs `mock-up` for before it starts |
| `design` | the spec, the vision, the code; the project's consult agents for a technical point it cannot settle | `docs/designs/DESIGN-NNN-slug.md`, the *how*, scaled to the change; ADRs for hard-to-reverse choices; beads with dependency edges, each carrying its acceptance criteria and, where the task is verified through the wired product, the test case and its criteria; or, when the person chose to leave the decisions it set aside for an engineer, a `Draft` carrying them | `implement`; `design` again, for a `Draft` with decisions needed |
| `implement` | ready beads, an epic, or a design; the project's consult agents when a worker fails | a worktree per task, the test case before the code, verification against the bead's criteria and the project's checks, a pull request per task, walked as a serial stack or in waves on an epic branch until the frontier is empty | `review` and `test` on the epic's work; `implement` again, the next round, when they added children; the human, who merges when the graph is empty; `design`, for a disagreement that moves work, filed as a revision bead |
| `review` | anything live: uncommitted work, a branch, a PR, a commit range, a path, a document, an epic's work; the project's consult agents for what the reviewer could not settle | `.codefall/reviews/`, a JSON and Markdown pair per review; fixes on the target's branch for the findings the user takes; on an epic's work, children of the epic for the problems the person took, built by `implement` in the same session | the human; `implement`, which it runs itself on an epic's work; `design`, for a deferred finding that moves work |
| `test` | what the project declares: suites, the changed subset, one case in its `spec` or `agentic` modality, or an epic's work on its branch | `.codefall/tests/`, a report per run; findings triaged, never an edit that makes a run pass; on an epic's work, children of the epic for the problems the person took, built by `implement` in the same session | `implement`, which it runs itself on an epic's work; `fix` or `design` for work outside a delivery; `report`, when the user wants a bug document |

A contained fix skips the documents: `design` writes beads only when a change stays inside one
component and comes to a task or two, and `specify` is for features, not every change. A bug a
person saw starts at `report` instead of `specify`; a bug a test run found starts as the issue `test`
files on the user's word. Either way `design` finds its cause before it chooses a tier. `fix` runs
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
one: amended into the document, turned into a task row, or rejected with why. An amendment
`implement` or `review` makes is also recorded as a closed bead labelled `design-amended`, with the
same `spec_id`, so which documents were wrong, and where, can be listed later.
[ADR-008](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-008-upstream-amendments.md)
records the rule.

Every bead a verb files during a delivery — a code discovery, a deferred review finding, a bug a
test run found, a revision bead — is a child of the epic, created `deferred` so the round still
running does not pick it up, and reopened at the next round's go. The round number is metadata on
the epic, written by `implement` alone. Each verb prints the delivery's one-line chart from
`.codefall/shared/delivery.sh` at its start and in its close-out — the epic, the round, and the
children closed over the children in all, counted as `bd epic status` counts them — appends one
line to the epic's notes (`round N (verb) [date]: what happened`), records in a `bd comment`
anything decided in conversation that no artifact holds, and then says it is safe to `/clear`.
[ADR-013.4](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-013.4-deliveries.md)
records the rule.

## Keeping the project current

Four verbs sit beside the chain rather than in it:

- **`equip`** builds and rebuilds what the other verbs need the project to have. One track is the
  local environment: `start` and `update`, declared under `local` in `.codefall/settings.json`
  ([ADR-005](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-005-local-environment-scripts.md)).
  The other is the test harness: a spec runner per surface, declared in `test.runners`, its
  commands recorded in the testing root's `AGENTS.md`
  ([ADR-007](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-007-test-cases.md)).
  The third is the agents: how another harness is called, under `harnessConfig`, and who reviews
  and consults, in the `agents` lists, set up from any harness
  ([ADR-009.4](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-009.4-agents.md)).
  The fourth is the document landing: the GitHub Action that merges a document pull request once
  a person adds the `auto-merge` label to it or approves it, and the label itself, with the branch
  rule read and
  reported, never changed
  ([ADR-014](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-014-upstream-verbs-and-document-landing.md)).
  One run equips one track, and each lands as its own pull request, which a person merges. What a
  search leaves ambiguous is consulted on once before the one question.
- **`refresh`** is what to run instead of pulling by hand, and it ends on `main` wherever it
  starts: leave a feature branch or an implement worktree for the primary checkout, switch it to
  `main` and fast-forward it, delete the branch and remove the worktree it left when their work is
  merged, `bd sync` the beads with their Dolt remote, run `start`, run `update` when the commit
  moved, record the commit in a git-ignored stamp. Uncommitted work in the primary checkout stops
  it. It never rebases, stashes, deletes a remote branch, or settles a conflict the sync halts on.
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
  still spelled the old way, points a `$schema` URL an earlier release wrote at the current one,
  removes what the previous install wrote that this one does not ship, and touches nothing else.
  `codefall init` runs once and refuses a project that has a manifest, naming
  `upgrade`
  ([ADR-010](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-010-upgrade.md)).

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
- **A task bead closes at done** — criteria verified, checks green, PR open — not at merge, because
  the close is what releases the next link in a stack. Gates carry the merge seam: every PR gates a
  "landed" bead inside the epic, and the next session's `bd gate check` turns merges into bead
  state. **The epic closes when the graph is empty and the code is on `main`**, and the graph is
  not empty while a child `review` or `test` filed is open.
- **Done is read from the graph.** `bd children <epic>` with nothing open but the landed bead, and
  the last test run against the epic passed. No verb declares a delivery done; the chart shows it.
- **No verb merges to `main`.** A person merges every code pull request: `implement` ends at open
  PRs and the link to the top of the stack, which lands every layer when merged on GitHub, `review`
  and `test` point at the stack the same way, and the guard hook denies the alternative in every
  harness. A GitHub Action the project installs with `equip landing` merges a document pull request
  once a person adds the `auto-merge` label to it or approves it and its diff holds only document
  paths; no verb adds the label or approves, and a project without the Action merges those by hand.
- **Everything short of the merge is the verb's.** A verb that writes to the repository branches
  from `main` before its first file, commits what it wrote by path, pushes, and opens the pull
  request without asking; a document never sits uncommitted on `main`. A document verb opens an
  ordinary pull request, a draft only while the document is `Draft`, and its report says to add the
  `auto-merge` label when the person wants it merged or to have someone approve it; `scaffold`,
  `equip`, and `upgrade` open one a
  person merges. `implement` does this per task, as one GitHub stack per epic; documents do not
  stack. All of them follow the shared `landing.md` beside this file's installed copy.
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
