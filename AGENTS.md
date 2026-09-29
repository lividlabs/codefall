# codefall

Opinionated skills for the software development lifecycle, plus the CLI that installs them. The repo
holds two components; each scopes its own operative rules:

- `cli/` — the Go installer. Rules: [cli/AGENTS.md](cli/AGENTS.md)
- `extensions/` — the skills, hooks, and shared scripts that `codefall init` installs, and the
  sections it writes into a project's `AGENTS.md`. Rules: [extensions/AGENTS.md](extensions/AGENTS.md)

Decisions for the whole repository live in [`docs/adrs/`](docs/adrs/); the in-flight scratchpad is
[`docs/decision-log.md`](docs/decision-log.md). Provenance for the scaffolded documents is
`.codefall/scaffold.json`.

## How codefall works

`codefall init` installs the verbs into each harness a project uses, and `.codefall/` beside them
with the settings, shared scripts, and hooks the verbs read. The verbs chain from an idea to open
pull requests, and each leaves something the next one reads: `envision` → `docs/visions/`, `specify`
→ `docs/specs/` mirrored to the tracker, `report` → `docs/bugs/` mirrored the same way, `mock-up` →
`docs/mockups/`, `design` → `docs/designs/` and beads with dependency edges, `implement` → a
worktree, a test case, and a pull request per task, `review` → `.codefall/reviews/`, `test` →
`.codefall/tests/`. Beside the chain, `fix` runs `design` at tier 0 and `implement` on one bead in
one run, `scaffold` starts a project, `upgrade` brings its install and documents current, and
`equip` and `refresh` keep the local environment level with the checkout. Bead state travels over
the git remote as `refs/dolt/data`: a verb runs `bd dolt push` after every bead write, and `refresh`
runs `bd sync`.
`envision`, `specify`, `report`, `fix`, `mock-up`, `scaffold`, `upgrade`, and `equip` are invoked
deliberately by a user; `design`, `implement`, `test`, `review`, and `refresh` may also be run by an
agent. Every verb applies only what the user takes; a human performs every merge to `main`, and a
hook denies the alternative.
[`docs/workflow.md`](docs/workflow.md) holds the full chain, what each verb reads and writes, and
who is authoritative for what.

## Workflow

These hold for both components; each component's own file adds to them.

- Every change lands through a PR, and a PR is **squash-merged**. Merge commits and rebase merges
  are turned off on GitHub, and the branch is deleted on merge.
- A change is **breaking** when a project that ran an earlier codefall has to do something by hand
  after upgrading: a flag, skill, or hook renamed or removed, a settings field or file layout
  changed, a hook command changed. A new one of any of those is not.
- **The PR title becomes the commit message** on a branch of more than one commit; a single-commit
  branch keeps that commit's own title. So the title is a Conventional Commit line naming the
  highest-impact change on the branch, and it carries `!` when any commit on the branch is
  breaking. Adding a commit means reading the title again — a branch that starts as `feat:` and
  gains a breaking rename needs a new title before it merges.
- release-please reads that one header, not the commits underneath it: the squashed body holds
  every commit message, and none of them is parsed. A breaking commit also carries a
  `BREAKING CHANGE:` footer, so it reaches the changelog by a second path if the title fails to.
