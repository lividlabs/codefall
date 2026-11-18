# PLAN-001: The project commands move under `codefall project`

**Status:** Ready — 2026-11-18
**Related:** adr: ADR-018

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
  already a file of its own.
- **Each settings file is named for what owns it** — the project's file becomes
  `.codefall/project.json`, beside `.codefall/user.json`. The plan that builds the IDE names the
  IDE's file and says where it lives, which may be outside a repository.

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
steps, and the manifest are untouched.

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

## Data Models

`.codefall/settings.json` is renamed `.codefall/project.json`. What the file holds does not change,
and its `version` stays 1. `.codefall/user.json` keeps its name.

`codefall project upgrade` moves the file, in the step that already rewrites a harness name and the
`$schema` URL, and reports the move. The JSON schema becomes `cli/schemas/project.schema.json`, and
the URL of the old one joins `FormerSchemaIDs`, so the same step repoints a file's `$schema`.

The file is named in 37 files under `extensions/`: skills, shared scripts, `AGENTS.md` sections, and
hook notices. They change in the pull request that renames it, with the Go code that reads and
writes it.

## Error Handling

- A root `init`, `upgrade`, or `create` — the command exits non-zero with one sentence naming the
  command under `project`, whatever flags it was given, and writes nothing.
- A project that holds only `.codefall/settings.json`, read by a binary that expects
  `project.json` — `doctor` reports a failed check whose remedy is `codefall project upgrade`, and
  `config` refuses with the same remedy. The skills installed there still read the old name, so
  they keep working until the upgrade replaces them and moves the file in one run.
- A project that holds both files — the upgrade keeps `project.json` and says `settings.json` was
  left where it is, for a person to remove.

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
| The project's file stays `.codefall/settings.json`, and the IDE's file takes a new name | The project's file would be the one not named for the project |
| The project's file becomes `project.json`, and the IDE takes `.codefall/settings.json` | A name that meant the project's settings would come to mean the IDE's, and it settles where the IDE's file lives before the IDE is planned |
| `config` and `doctor` move under `project` | Both are run day to day, and `doctor` also checks the machine; they stay at the root and take the prefix as a second spelling |

## Tasks
> Staged. Not in Beads: this repository does not run its own verbs yet, so each task is a GitHub
> issue, T1 as #260, T2 as #261, T3 as #262, and T4 as #264.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | Write ADR-018: what a project is, the `codefall project` group, the root names that refuse, and the settings files and their names; it names ADR-010 as the record it changes | — | Overview |
| T2 | Move `init`, `upgrade`, and `create` under `codefall project`; the root names refuse; every reference in the Go messages, `extensions/`, the README, `docs/guide/`, and `cli/AGENTS.md` changes with them; add the test that holds the installed text to the command tree | T1 | Architecture |
| T3 | Accept `codefall project config` and `codefall project doctor` | T2 | Components and Interfaces |
| T4 | Rename the project's settings file to `.codefall/project.json`: the Go code that reads and writes it, the JSON schema, the move in `codefall project upgrade`, and every file under `extensions/` that names it | T2 | Data Models |

T2 and T4 are each one task and not several. Each renames something other code and the installed
text name, a command in the first and a file in the second, and no part of either builds and passes
on its own before every user of the old name has changed. T4 follows T2 because both edit the same
installed text, and the two are released together, so a project crosses one breaking release.
