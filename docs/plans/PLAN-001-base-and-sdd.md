# PLAN-001: A base tool, with the skills under `codefall sdd`

**Status:** Draft — 2026-11-16

## Overview

Codefall has been its skills plus the CLI that installs them. This plan makes the CLI the tool and
the skills its default content. A project can be set up for codefall without the skills, and every
command that installs or serves them sits under one group, `codefall sdd`, for spec-driven
development.

The work lands in two stages. The first moves the commands and changes nothing they do. The second
separates what the tool writes for itself, the *base*, from what it writes for the skills, the
*skill set*, in three places: the install steps, `.codefall/manifest.json`, and the settings files.

Installing a skill set that does not ship in the binary is not in this plan. It needs the manifest
and the settings to record an owner first, which this plan does, and it gets a plan of its own.

- **`init`, `upgrade`, and `create` move under `codefall sdd` with their flags unchanged** — each
  one installs the skill set today. `create` is no exception: it runs the `bd` and `gh` checks,
  takes every `init` flag, and ends by running `init`.
- **The old root names refuse, and name the new command** — the root `init` comes back in the
  second stage as the base setup. An alias that ran `sdd init` in between would change what the
  name does without saying so.
- **`config`, `doctor`, and `update` stay at the root** — `config` is one editor over one file
  (ADR-012), and `doctor` is one report on one project. Each shows the skill set as a section.
  `update` replaces the binary and has nothing to do with a project.
- **A setting has an owner, and the file shows it** — the skill set's settings move under an `sdd`
  key in `.codefall/settings.json` and `.codefall/user.json`. A person reading the file sees whose
  each setting is, removing the skill set deletes one block, and a later skill set takes a block
  under its own name.
- **`harnesses`, `harnessConfig`, and `agents` belong to the base** — they say which harnesses a
  project uses, how each is called, and which agent a session hands work to. A tool that runs
  agents without the skills needs all three. The lists in an `agents` entry keep the names
  `review`, `consult`, and `work`, fixed in Go.
- **The rename lands before the split** — `sdd init` first does everything `init` does today, the
  base parts included. The breaking change is then one release, and the split that follows renames
  no command.

## Architecture

The command tree once both stages have landed:

```
codefall
├── init                 the base setup (Decisions needed, 1)
├── create <directory>   a new directory and repository, then the base setup (Decisions needed, 1)
├── config               one editor; the skill set's settings under their own heading
├── doctor               one report; the skill set's checks as their own section
├── update               replaces the binary
└── sdd
    ├── init             adds the skill set, and runs the base setup first when it is missing
    ├── upgrade          brings the install level with the binary, and migrates the settings
    ├── create           a new directory and repository, the base setup, and the skill set
    ├── status           the installed version and the harnesses it is installed for
    └── remove           deletes what the manifest records for the skill set
```

Between the two stages the root `init`, `upgrade`, and `create` exist only to refuse. Each prints
one sentence that names the command under `sdd`.

`init` and `upgrade` are one use case in `internal/initcmd/`, a list of steps run in order. The
second stage gives each step an owner. A command names the owners it runs, and `sdd init` runs the
base's steps first when the manifest records no base.

| Step | Owner |
| --- | --- |
| settings | the base writes its own keys; the skill set writes the `sdd` block |
| extension | the skill set: the skills, and the files under `.codefall/shared/` |
| cleanup | each owner, over its own record in the manifest |
| beads | the skill set |
| hook | the base registers the merge guard; the skill set registers the session notice, which reads the skill set's preflight script |
| agents | the skill set: every section it writes into `AGENTS.md` |
| testing | the skill set |
| ignore | the base writes the line for `.codefall/user.json`; the skill set writes the rest |

The manifest records each file under the owner that wrote it, and the cleanup step compares a
record only with what the same owner wrote in this run. That comparison is what lets a second skill
set be installed later without the first one's upgrade deleting its files.

`run-agent.sh` and `run-worker.sh`, the scripts that read `agents` and `harnessConfig`, stay in the
skill set. The base stores and checks those two keys, and nothing in the base runs an agent yet.

## Components and Interfaces

- `codefall sdd init`, `codefall sdd upgrade`, `codefall sdd create <directory>` — the three
  commands under their new names, with the flags and the output they have today.
- `codefall init`, `codefall upgrade`, `codefall create` at the root — a refusal in one sentence
  that names the command under `sdd`, until T7 gives `init` and `create` back a meaning.
- `codefall sdd status` — prints whether the skill set is installed, the version each harness has,
  and what `sdd upgrade` would change.
- `codefall sdd remove` — deletes the files the manifest records for the skill set, its sections of
  `AGENTS.md`, its hook registrations, and the `sdd` block in both settings files. It leaves the
  documents the skills wrote under `docs/`, which are the project's.
- `doctor.Prerequisites` — becomes two checks. The check on `git` runs before a base command. The
  checks on `bd` and `gh` run before a command under `sdd`.

## Data Models

`.codefall/settings.json` goes from version 1 to version 2. The keys the skill set owns move under
`sdd`: `tracker`, the tracker's own block (`beads` or `github`), `review`, `local`, and `test`.

Version 1:

```json
{
  "version": 1,
  "tracker": "github",
  "github": { "issuesRepo": "acme/todo" },
  "harnesses": ["claude", "codex"],
  "agents": [{ "activeAgent": "default", "review": [{ "harness": "current" }] }],
  "review": { "postToPullRequest": false },
  "local": { "start": "scripts/start.sh", "update": "scripts/update.sh" },
  "test": { "dir": "testing" }
}
```

Version 2:

```json
{
  "version": 2,
  "harnesses": ["claude", "codex"],
  "agents": [{ "activeAgent": "default", "review": [{ "harness": "current" }] }],
  "sdd": {
    "tracker": "github",
    "github": { "issuesRepo": "acme/todo" },
    "review": { "postToPullRequest": false },
    "local": { "start": "scripts/start.sh", "update": "scripts/update.sh" },
    "test": { "dir": "testing" }
  }
}
```

A version 2 file with no `sdd` block is valid, and describes a project set up without the skills.
`tracker` is required inside the block and no longer at the top level.

`.codefall/user.json` goes from version 1 to version 2 the same way: `persona` moves to
`sdd.persona`.

`codefall sdd upgrade` migrates both files, in the step that already rewrites a harness name and
the `$schema` URL, and reports what it moved. Both JSON schemas under `cli/schemas/` change with
it.

`.codefall/manifest.json` gains an owner on every entry: the base's files in one entry, and each
skill set's harness entries and shared entry under the set's name. A manifest written before the
change names no owner and is read as the skill set's, which is what every file in it was.

## Error Handling

- A root `init`, `upgrade`, or `create` run between the two stages — the command exits non-zero
  with one sentence naming the command under `sdd`, and writes nothing.
- A version 1 settings file read by a binary that writes version 2 — `doctor` reports a failed
  check whose remedy is `codefall sdd upgrade`, and `config` refuses with the same remedy. Neither
  reads the old paths.
- A file that holds a key at both the top level and under `sdd` — the migration keeps the one under
  `sdd` and reports the other as removed, the way the manifest keeps the entry under a harness's
  current name when it holds both spellings.
- A file the old manifest lists under the skill set and the base writes in this run, such as the
  merge guard — cleanup never removes a path this run wrote for any owner, so the file stays and is
  recorded under the base.

## Testing Strategy

Two tests are worth more than the unit tests each task brings.

A test in `extensions/` reads every `codefall <command>` the installed text names, in the skills,
the `AGENTS.md` sections, and the hook notices, and fails on one the command tree does not hold.
T2 changes 68 of those references, and the test keeps the next rename from leaving one behind.

The migration is tested against version 1 files kept as fixtures: one per tracker, one with every
optional block, one with none, and one that already holds an `sdd` block.

## Alternatives Considered

| Option | Why it was set aside |
| --- | --- |
| The root names run the `sdd` commands as aliases | The root `init` takes the base meaning in the second stage, and the alias would change what it does without a word |
| `create` stays at the root | It cannot be used without the skill set today, and its flags are `init`'s |
| A `codefall sdd config` beside the root `config` | `config` is one editor over one file, and `agents` and `harnessConfig` are the base's |
| The settings file stays flat, and Go records each key's owner | The file does not show whose a key is, and removing the skill set means deleting keys one at a time |
| A second file, `.codefall/sdd.json` | Every skill reads two files, and a person has two documents to find and edit |
| Skill sets that declare their own settings, survey, hooks, and sections as data | The survey, the editor, and the checks in `initcmd`, `config`, and `doctor` would all be read from data; defaults a project can leave out do not need it |

## Hard Constraints

- THE SYSTEM SHALL remove a file only when the manifest records it under the owner being upgraded
  or removed, and this run wrote it for no owner.
- THE SYSTEM SHALL leave a settings file unchanged when a migration fails partway, so a second run
  starts from the file the first one read.

## Decisions needed

### 1. What the root `init` and `create` do once the base exists
- **Affects:** T7
- **Options:** the base only — each root name has one meaning, and a project that wants the skills
  runs `codefall sdd init` or starts with `codefall sdd create`; the base and the skill set unless
  a flag leaves the skills out — one command gives the opinionated default, and the root `init`
  and `sdd init` then do the same thing on a new project
- **Needs an ADR:** yes — ADR-018 records the command tree, and this is the last part of it

### 2. Whether the base needs the tracker
- **Affects:** T6, T9
- **Options:** the tracker is the skill set's — `tracker` and its block move under `sdd`, and `bd`
  and `gh` are checked only before a command under `sdd`; the tracker is the base's — the keys stay
  at the top level, T6 moves three blocks and not five, and every command keeps the checks. The
  second is right only if the interface the base grows shows work items to a project that has no
  skills. Changing the answer after T6 lands is a second migration.
- **Needs an ADR:** yes — ADR-018 records which settings each owner holds

## Tasks
> Staged. Not yet in Beads.

| ID | Task | Depends on | Plan ref |
|----|------|-----------|------------|
| T1 | Write ADR-018: the base and the skill set, the `sdd` command group, and the owner of each setting; it names ADR-006 and ADR-010 as the records it changes | — | Overview |
| T2 | Move `init`, `upgrade`, and `create` under `codefall sdd`; the root names refuse; every reference in the Go messages, `extensions/`, the README, `docs/guide/`, and `cli/AGENTS.md` changes with them; add the test that holds the installed text to the command tree | T1 | Architecture |
| T3 | Record an owner on every manifest entry, read a manifest without one as the skill set's, and have cleanup compare within one owner | T1 | Data Models |
| T4 | Give each install step an owner, run the base's steps apart from the skill set's, and have `sdd init` run the base's first when no base is recorded | T2, T3 | Architecture |
| T5 | Group the checks in the `settings` and `userfile` modules by owner, with the file's shape unchanged | T1 | Data Models |
| T6 | Settings version 2: the `sdd` block in both files, both JSON schemas, the migration in `sdd upgrade`, `config` and `doctor` on the new paths, and every skill and shared script that reads a moved key | T2, T5 | Data Models |
| T7 | Give the root `init` and `create` the meaning decision 1 settles | T4, T6 | Decisions needed |
| T8 | Group the skill set's settings under their own heading in the `config` editor, shown only when the skill set is installed | T3, T6 | Components and Interfaces |
| T9 | Report the skill set as its own section in `doctor`, skip it when the skill set is not installed, and split the prerequisite check in two | T3, T4 | Components and Interfaces |
| T10 | `codefall sdd status` | T3 | Components and Interfaces |
| T11 | `codefall sdd remove` | T4, T6 | Components and Interfaces |

T2 and T6 are each one task and not several. Each renames something other code uses, a command name
in the first and a settings path in the second, and no part of either builds and passes on its own
before every user of the old name has changed.
