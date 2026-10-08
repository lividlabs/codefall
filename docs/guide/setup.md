# Setting up a project

`codefall init` prepares a repository for Codefall. It installs the skills into each coding agent the
project uses, saves the project's answers in a settings file, and adds what the skills expect to find
in the repository. This page describes what `init` asks, what it writes, and how `codefall upgrade`
keeps all of it current.

A *skill* is a set of instructions a coding agent follows when you type its slash command, such as
`/codefall-specify`. A *coding agent* is an AI tool that reads and edits the code in your repository,
such as Claude Code or Codex. Codefall's settings and flags call coding agents *harnesses*.

## Set up an existing repository

Run `init` in the repository:

```sh
codefall init
```

`init` asks these questions, and each one has a flag that answers it without asking:

| Question | Flag | Notes |
| --- | --- | --- |
| Which coding agents should Codefall set up? | `--harness` | Repeat the flag, or separate names with commas, for several. |
| Which issue tracker should Codefall use? | `--tracker` | `github` (GitHub Issues) or `beads`. |
| Which GitHub repository holds the issues? | `--issues-repo` | Asked only for GitHub. It defaults to the repository the directory belongs to. |
| Which GitHub Project number? | `--issues-project` | Asked only for GitHub. Leave it blank for none. |
| May `/codefall-review` post its findings to a pull request? | `--review-post-to-pr` | Off unless you say yes. |
| Where should the project's test cases live? | `--test-dir` | Defaults to `testing`. |

`init` records the answers in `.codefall/settings.json`, which the skills read back.

`init` runs once per project. When the project already has `.codefall/manifest.json`, the record of
a finished install, `init` says so and stops; from then on, `codefall upgrade` maintains the install.
A project set up before the manifest existed runs `init` one more time, which writes the manifest,
and uses `upgrade` after that. [ADR-010](../adrs/ADR-010-upgrade.md) records why `init` runs once
and `upgrade` is its own command. An ADR (architecture decision record) is a short document that
records one hard-to-reverse decision and the reasons for it.

## Start a new project

`codefall create` makes the directory for a new project, then sets it up:

```sh
codefall create trip-planner --remote git@github.com:acme/trip-planner.git
```

It creates the directory and its git repository, commits a `README.md` and a `.gitignore`, runs
`init` there, and offers to push. `create` takes these flags of its own, along with every `init` flag
except `--location`:

| Flag | Does |
| --- | --- |
| `--description` | One sentence about the project, written under the title in `README.md`. |
| `--remote` | The URL of the git remote to add as `origin`. |
| `--push` | Pushes to the remote once `init` has finished. It needs `--remote`. |

## Install below the repository root

In a monorepo, one team may want Codefall in its own directory rather than at the repository root.
When you run `init` below the root, it first asks whether to install in the current directory or at
the root. `--location here` or `--location root` answers for a script. `upgrade` takes the same flag.

## Choose coding agents

A project can use more than one coding agent. `init` asks which ones to set up and never picks for
you. Each agent is named for its command-line binary, and that name is what `.codefall/settings.json`
records:

| Coding agent | Name |
| --- | --- |
| Claude Code | `claude` |
| Codex | `codex` |
| OpenCode | `opencode` |
| Antigravity | `agy` |
| Muse | `muse` |

Earlier releases called Claude Code `claude-code` and Antigravity `antigravity`. A project set up with
those names keeps working: `codefall doctor` warns about the old names, and the next
`codefall upgrade` rewrites them.

## What init writes

### The skills

Each coding agent finds skills in its own directory without being told where to look, so the skills
go there:

| Coding agent | Skills directory |
| --- | --- |
| Claude Code | `.claude/skills/` |
| Antigravity, Codex, Muse, OpenCode | `.agents/skills/` |

### The `.codefall/` directory

Everything else Codefall installs goes into `.codefall/`, once, whichever agents you chose. The skills
reach these files by paths Codefall writes, so they do not need to be in an agent's own directory:

| Path | Holds |
| --- | --- |
| `.codefall/settings.json` | what the project told `init`, which the skills read back |
| `.codefall/manifest.json` | what the last finished run wrote, for each agent and for `.codefall/` itself |
| `.codefall/hooks/shared/` | the scripts every agent's hooks run |
| `.codefall/shared/` | the files the skills read, and the scripts a skill runs |

A skill refers to a shared file as `../../../.codefall/shared/<file>`, which reaches the same file from
either skills directory. Nothing installed is a symlink, and none of this repository's maintainer
documents is installed into your project. [ADR-006](../adrs/ADR-006-install-layout.md) records why.

### Sections in `AGENTS.md`

`AGENTS.md` is the file coding agents read for project instructions. `init` adds four marked
sections to it: Codefall, Beads, Local environment, and Testing. A rerun replaces the text between
each section's markers and never touches a word outside them. When the project has no `CLAUDE.md`,
Claude Code gets a one-line `CLAUDE.md` that points at `AGENTS.md`.

The Codefall section frames the other three. It lists the skills in the order they run, the skills
outside that sequence, and which document or tool has the final say on what. The detail is in
`.codefall/shared/workflow.md`, a copy of [`docs/workflow.md`](../workflow.md) that CI keeps
identical to its source.

### The testing root

The directory you named for test cases is the *testing root*. `init` creates it with a `test-cases/`
directory and skeleton `AGENTS.md` and `README.md` files. Those files are yours as soon as they exist:
`init` writes each one only when it is missing, and a rerun never rewrites one.
[ADR-007](../adrs/ADR-007-test-cases.md) records what the tree is for.

### Ignore and attributes entries

`init` appends entries to three files and leaves any entries already there alone:

| File | Entry | Why |
| --- | --- | --- |
| `.ignore` | `.codefall/reviews/` | Review findings are committed, but searches through ripgrep skip them. |
| `.ignore` | `.codefall/tests/` | Test run reports are committed, and searches skip them too. |
| `.gitignore` | `.codefall/refresh.stamp` | The commit this machine's local environment was last brought current at. |
| `.gitignore` | `.codefall/user.json` | Your personal settings, such as your persona. |
| `.gitignore` | `<testing root>/.artifacts/` | Everything a test run produces besides its report. |
| `.gitattributes` | `.beads/interactions.jsonl merge=union` | Two branches that both appended to the Beads interaction log merge without a conflict. |

### Beads configuration

[Beads](https://github.com/gastownhall/beads) is a task tracker that stores its data in your git
repository; Codefall keeps its task graph there, and each task is a *bead*. `init` initializes the
Beads database and writes `audit.enabled: false` into `.beads/config.yaml`. Beads' interaction log
then stays off until the project turns it on.

## Hooks

A *hook* is a check a coding agent runs at a fixed point, such as before each command. `init`
registers Codefall's hooks in each agent's own hook file, beside any hooks the project already has:

| Coding agent | Hook file | Hooks |
| --- | --- | --- |
| Claude Code | `.claude/settings.json` | merge guard, session start |
| Codex | `.codex/hooks.json` | merge guard, session start |
| OpenCode | `.opencode/plugins/codefall.js` | merge guard, session start |
| Antigravity | `.agents/hooks.json` | merge guard only, because Antigravity has no session start event |
| Muse | none | Codefall has no hooks for Muse |

The **merge guard** runs before each shell command (the `PreToolUse` event). It denies any command
that would merge or push to the default branch, `gh stack merge` included, because a stack's trunk is
the default branch.

The **session start** hook (the `SessionStart` event) primes the session with what Beads knows, then
prints a notice about anything the project needs done:

- the checkout is behind the default branch;
- the local environment has not been refreshed since `HEAD` moved;
- nobody has declared a testing root or a test runner;
- a Beads precondition is blocking.

The notice names the skill that fixes each problem and prints nothing when everything is current. It
only reports: it never pulls, never runs the project's `update` script, and never fails a session
whose state it could not read.

## Check the setup

`codefall doctor` checks what Codefall needs and names a fix for each problem. It never runs the fix
itself. Its checks are grouped into categories: Settings, Harnesses, Agents, Local environment,
Testing, Beads, and GitHub CLI.

The Testing category, for example, looks at what the project declared about tests:

- It warns when no testing root is declared, and names `codefall upgrade` as the fix.
- It warns when no test runner is declared, and names `/codefall-equip`.
- It fails when the declared testing directory does not exist.

## Upgrade a project

After you install a newer `codefall`, run `upgrade` in each project to bring its install level with
the binary:

```sh
codefall upgrade
```

For the agents the settings record, `upgrade`:

- reinstalls the skills, shared files, and hooks;
- replaces the sections Codefall wrote into `AGENTS.md`;
- rewrites an agent name still spelled the old way;
- points a `$schema` URL that an earlier release wrote at the current one;
- records the run in the manifest.

It changes nothing it did not write. It reinstalls on every run, even when the binary's version
matches the project's, so a skill, shared file, ignore entry, `AGENTS.md` section, hook registration,
or testing file that has gone missing or been edited is put back, and the report says so. It reports
"already up to date" only when nothing changed.

### Breaking changes

`upgrade` asks before it moves a project to a different version, and `--yes` answers yes for a
script. Before it asks, and before it changes any file, it prints the breaking changes this
repository's changelog records between the version in the manifest and the binary's version, release
by release. `--yes` continues past those too. When either end of the range is a development build,
`upgrade` reports that it cannot tell which changes apply rather than reporting none.

### Files a release no longer ships

After installing, `upgrade` removes what the previous run wrote and this run did not. For example,
when a release drops a skill or ships it under a new name, the old skill comes out of every skills
directory `upgrade` installs into, along with any directory that leaves empty. The report names each
removal and says which ones were renames.

`upgrade` compares only the agents the settings name, only the files the manifest lists, and nothing
outside the install directories. A manifest with no file lists gives it nothing to compare, and the
report says so.

### Add a coding agent

`--harness` adds an agent the project did not choose at `init`. `upgrade` installs for it and records
it in the settings:

```sh
codefall upgrade --harness codex
```

### Upgrade from the layout before `.codefall/`

Run `codefall upgrade`. It removes the files the old install recorded that the new layout does not
write. A manifest from before file lists existed records none, so you delete the old copies by hand;
the pull request that introduced the new layout lists what to delete.

## CLI commands

| Command | What it does |
| --- | --- |
| `codefall create <dir>` | Makes the directory and its git repository, commits a `README.md` and a `.gitignore`, runs `init` there, and offers to push. |
| `codefall init` | Sets up a directory for Codefall, once: the settings, the skills for each agent, Beads, the hooks, the `AGENTS.md` sections, the testing root, and the manifest that records the run. |
| `codefall upgrade` | Brings an installed project level with the binary for the agents the settings record. It warns about breaking changes in between, reinstalls, removes what the release no longer ships, and touches nothing it did not write. |
| `codefall update` | Replaces a binary the install script installed with the latest release, or with a version given as an argument. `--check` reports how the binary was installed and changes nothing. See [Installing codefall](install.md#update-codefall). |
| `codefall config` | Opens an editor for the review and consult agents, review posting, and your persona. Its subcommands set the same values from a script, and both refuse a value the settings would not accept. See [Configuration](configuration.md). |
| `codefall doctor` | Reports whether a project has what Codefall needs, with a fix for each failed check, and repairs nothing. |
