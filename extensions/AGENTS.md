# codefall-extension — operative rules

Terse on purpose: the why lives in the linked docs. This is the same convention the extension
installs into scaffolded projects — an `AGENTS.md` holds the operative rules and links back, and
never restates the reasoning.

## ADRs are history

- A ratified ADR is never rewritten — not in a user's project, not in this repo. A revision lands
  as a **new, superseding ADR**; the only in-place edit ever made to an existing ADR is flipping
  its Status line to `Superseded by <id> — <date>`.
- Successor naming: the current template's name when a rename is involved, an edition suffix
  (`ADR-BASE-02.2`) when it isn't. Mechanics: `skills/codefall-upgrade/SKILL.md`, step 6.
- This binds every verb that touches ADRs — `codefall-upgrade` today, `codefall-design` and the rest as they land.

## Skills

- The rules for writing and changing a skill — shape, length, what goes where, supporting files,
  templates — live in [`skills/AGENTS.md`](skills/AGENTS.md). Length is governed by
  [ADR-004.2](../docs/adrs/ADR-004.2-skill-length-guidelines.md).

## Hooks

- `hooks/shared/` holds the script every harness's guard runs, and `hooks/<harness>/` holds the
  definition `codefall init` registers — never copied as files. Hooks are the one place a skill's
  relative-path rule does not apply.
- **Every definition names the guard under `.codefall/hooks/shared/`**, which is where `init`
  installs it for every harness (ADR-006). Claude Code's and Codex's reach it from
  `$(git rev-parse --show-toplevel)`, Antigravity's from the workspace, and the OpenCode plugin's
  from `import.meta.dir`; the destination is the same file whichever way a definition gets there.
- **Every script in `hooks/shared/` is named `codefall-<what-it-does>.sh`.** The prefix is not
  decoration: `init` recognises its own registration in a file it shares with the project's hooks by
  the script a command names, and replaces that entry on an upgrade rather than appending beside it.
  A script renamed out of the prefix would leave every project's old registration running for ever.

## AGENTS.md sections

- `agents/sections/` holds the four sections `codefall init` writes into a project's `AGENTS.md` —
  `codefall.md`, `beads.md`, `local.md`, `testing.md` — and `agents/testing/` the two skeletons it
  writes at the testing root. Like the hook definitions, they are read from the binary one file at
  a time and never copied: `init` splices a section between its markers and writes a skeleton only
  where none exists.
- **A section is its markers and nothing outside them.** The opening marker is the first line, the
  closing marker is the last, the file ends with one newline, and neither marker appears twice.
  `init` refuses a file that breaks this, and the facade test pins the shipped ones. The markers
  themselves are constants in `cli/internal/initcmd/internal/domain/sections.go`; a section's words
  change here, its markers never do.
- `{{TESTING_ROOT}}` is the one placeholder, filled with the project's declared testing root. A
  section is read from the project root, so it names an installed file as `.codefall/shared/<file>`,
  not by a skill's `../../../` path.
- Keep each section short: every harness reads all four at session start. The decision log holds
  the token counts the last rewrite settled on.

## Prose

- Banned jargon, everywhere — docs, skills, ADRs, commit messages, PR bodies: *arm*,
  *load-bearing*, *honest* and *honestly*, and *fork* unless it means a GitHub fork. Plain words
  instead: option, important, accurate, explicit.

## Workflow

- All work happens on a branch or worktree, never directly on `main` — a ruleset forbids pushing
  it anyway. Pull `main` before branching. Every change lands through a PR.
- Stacked branches are fine, and the right way to break large work into smaller reviewable
  pieces. Check `gh pr list` before basing new work.
- Documentation is part of the work, not a follow-up. Before calling a change done, check
  `README.md`, `AGENTS.md`, `docs/`, and every skill the change touches, and land the updates as
  their own commit in the same PR.
- Conventional commits, with a body that says *why*. release-please reads the types
  (`release-please-config.json` maps them to changelog sections).
- release-please owns `CHANGELOG.md` → the repo-root CHANGELOG it releases alongside — and the
  extension's version markers come from the release-please PR alone.

## Testing a skill locally

- Codefall copies the embedded tree from the `extensions/` directory — point `init` at a sandbox
  checkout with the CLI binary built from the same working tree.
