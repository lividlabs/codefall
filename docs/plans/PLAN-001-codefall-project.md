# PLAN-001: The project commands move under `codefall project`

**Status:** Draft — 2026-11-17

## Overview

Codefall is growing from the skills and their installer into a tool with an IDE of its own. What
`codefall init` sets up today gets a name, a *project*: the skills, and the config they read. Every
command that sets a project up or keeps it current moves under one group, `codefall project`. The
commands do what they do today, under new names.

- **`init`, `upgrade`, and `create` move under `codefall project` with their flags unchanged** —
  the group names what the commands act on. It also separates `codefall project upgrade`, which
  brings a project current, from `codefall update`, which replaces the binary.
- **A project is the skills and the config** — there is no project without the skills, so no flag
  leaves them out, and the install keeps one owner. What works without a project is the rest of
  the tool.
- **The old root names refuse, and name the new command** — `codefall init`, `codefall upgrade`,
  and `codefall create` each exit with one sentence. The rename is released as a breaking change.
- **`config` and `doctor` stay at the root, and take the `project` prefix too** — `codefall project
  config` and `codefall project doctor` run the same commands, limited to the project. Every
  setting and every check is about the project today, so the two spellings do the same thing until
  a root command covers something outside one.
- **A project's settings and the IDE's settings are two files** — the IDE's settings do not depend
  on a project, and the two change for different reasons, which is why `.codefall/user.json` is
  already a file of its own. What each file is called is not settled (Decisions needed, 1).

Three things are left for later plans: `codefall project remove`, which deletes what a project's
install wrote; installing a skill set that does not ship in the binary; and the IDE.

## Architecture

The command tree:

```
codefall
├── config               the editor over the project's settings
├── doctor               the report on this directory and the tools on this machine
├── update               replaces the binary
├── init                 refuses, and names `codefall project init`
├── upgrade              refuses, and names `codefall project upgrade`
├── create               refuses, and names `codefall project create`
└── project
    ├── init             today's `codefall init`
    ├── upgrade          today's `codefall upgrade`
    ├── create           today's `codefall create`
    ├── config           the root `config`, limited to the project
    └── doctor           the root `doctor`, limited to the project
```

`cli/cmd/codefall/main.go` builds the `project` group and mounts on it what it mounts on the root
today: the commands `initcmd.Commands` returns, and `create.Command`. `config` and `doctor` are
mounted twice, once on the root and once on the group, each mount with a command built for it
alone, as `cli/AGENTS.md` requires of a command that appears in two places. The use cases, the
steps, the manifest, and both settings files are untouched.

A refusing command belongs to the component that held the name: `initcmd` exports the two for
`init` and `upgrade`, and `create` exports its own.

The names appear in more than the command tree. Doctor's remedies name `codefall upgrade`, init
refuses a project that has a manifest by naming it, and upgrade refuses one that has none by naming
`codefall init`. The skills, the `AGENTS.md` sections, and the hook notices under `extensions/`
name the two commands 68 times, and that text is installed into every project. The README,
`docs/guide/`, and `cli/AGENTS.md` name them as well. All of it changes in the pull request that
moves the commands. The ADRs and the decision log are records and stay as they were written.

## Components and Interfaces

- `codefall project init`, `codefall project upgrade`, `codefall project create <directory>` — the
  three commands under their new names, with the flags and the output they have today.
- `codefall init`, `codefall upgrade`, `codefall create` — a refusal in one sentence that names the
  command under `project`.
- `codefall project config`, `codefall project doctor` — the root commands under a second spelling,
  subcommands and flags included.

## Error Handling

- A root `init`, `upgrade`, or `create` — the command exits non-zero with one sentence naming the
  command under `project`, whatever flags it was given, and writes nothing.

## Testing Strategy

A test in `extensions/` reads every `codefall <command>` the installed text names, in the skills,
the `AGENTS.md` sections, and the hook notices, and fails on one the command tree does not hold.
T2 changes 68 of those references, and the test keeps the next rename from leaving one behind.

## Alternatives Considered

| Option | Why it was set aside |
| --- | --- |
| `codefall sdd` as the group's name | It names a method, spec-driven development, where `project` names what the commands act on |
| The root names run the `project` commands as aliases | The rename is released as a breaking change, and the refusal tells a person the new name |
| A project set up without the skills, with the install split between a base and a skill set | A project is the skills and the config, so there is nothing to set up without them |
| One settings file, with a section for the project and a section for the IDE | The IDE's settings do not depend on a project; a file of their own leaves the project's file as it is, and no project has to migrate |
| `config` and `doctor` move under `project` | Both are run day to day, and `doctor` also checks the machine; they stay at the root and take the prefix as a second spelling |

## Decisions needed

### 1. What the project's settings file and the IDE's settings file are called
- **Affects:** T1, and a task this decision may add
- **Options:** the project's file becomes `.codefall/project.json`, and the IDE's file is named
  when the IDE is planned — each file is named for what owns it, `codefall project upgrade` renames
  the file, and every skill and shared script that reads it changes one path; the project's file
  becomes `.codefall/project.json` and the IDE takes `.codefall/settings.json` — the same rename,
  and a name that meant the project's settings comes to mean the IDE's; the project's file stays
  `.codefall/settings.json` and the IDE's file takes a new name — nothing moves, and the project's
  file is the one not named for the project
- **Needs an ADR:** yes — ADR-018 records the two files

## Tasks
> Staged. Not in Beads: this repository does not run its own verbs yet, so each task is a GitHub
> issue, T1 as #260, T2 as #261, and T3 as #262.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | Write ADR-018: what a project is, the `codefall project` group, the root names that refuse, and the two settings files; it names ADR-010 as the record it changes | — | Overview |
| T2 | Move `init`, `upgrade`, and `create` under `codefall project`; the root names refuse; every reference in the Go messages, `extensions/`, the README, `docs/guide/`, and `cli/AGENTS.md` changes with them; add the test that holds the installed text to the command tree | T1 | Architecture |
| T3 | Accept `codefall project config` and `codefall project doctor` | T2 | Components and Interfaces |

T2 is one task and not several. It renames commands that other code and the installed text name,
and no part of it builds and passes on its own before every user of the old names has changed.
