# ADR-018: The project and its commands

## Status

Accepted — 2026-11-18

## Context

The commands at the root of the CLI were named when the install was the whole product. `codefall
init` sets up a directory for codefall, `codefall upgrade` keeps what it wrote current, and
`codefall create` runs init in a directory it makes first. Beside them, `codefall config` edits
the settings init writes, `codefall doctor` reports on the directory and the tools on the machine,
and `codefall update` replaces the binary (ADR-015). What the three set up has never had a name of
its own: ADR-010 calls it "an installed project" in passing, and the settings file is named
`settings.json` for what it holds, not for what owns it.

Codefall is growing from the skills and their installer into a tool with an *IDE*, an integrated
development environment, of its own. What the installer sets up gets a name in this record: a
*project* is the skills, and the config they read, set up together and never apart. The IDE will
read settings of its own, and those settings do not depend on a project. Commands will arrive
that act on the IDE, and the root gives them no way to say what they act on. The two verbs that
bring things current already share a word: `codefall upgrade` brings a project's install current
and `codefall update` replaces the binary, and nothing at the root says they act on different
things.

Seven alternatives were weighed.

**`codefall sdd` as the group's name.** `sdd` names a method, spec-driven development, and a
group's name should name what its commands act on. A person who runs another method on their
project still has a project. `project` says so whatever the method is, and `sdd` does not.

**The root names run the `project` commands as aliases.** Every script keeps working, and the new
name stays hidden: a person who never types it never learns it. A refusal tells a person the new
name at the moment they reach for the old one. An alias has nothing to say.

**A project set up without the skills, with the install split between a base and a skill set.** A
project is the skills and the config they read, so there is nothing to set up without them: the
settings record the harnesses and the tracker the skills use, and an install without the skills
would be settings waiting for skills that never load. The split would also give the install two
owners where it has one.

**One settings file, with a section for the project and a section for the IDE.** The IDE's
settings do not depend on a project, and the two change for different reasons, which is why
`.codefall/user.json` is already a file of its own beside the project's. A file of the IDE's own
leaves the project's file as it is, and no project migrates for a section it does not use.

**The project's file stays `.codefall/settings.json`, and the IDE's file takes a new name.** The
rule this record settles is that each settings file is named for what owns it, and under this
option the project's file is the one not named for the project.

**The project's file becomes `project.json`, and the IDE takes `.codefall/settings.json`.** A
name that meant the project's settings would come to mean the IDE's, and a person who learned
`settings.json` as the project's file would meet it meaning something else. It also settles
where the IDE's file lives before the IDE is planned.

**`config` and `doctor` move under `project`.** A person runs both day to day, and `doctor`
checks the machine's tools as well as the project, so neither acts only on a project. They stay
at the root and take the prefix as a second spelling.

## Decision

### A project is the skills and the config they read

Init sets up the skills and the config together, never apart. There is no project without the
skills, so no flag leaves them out, and the install keeps one owner. What works without a project
is the rest of the tool: `codefall update` replaces the binary, and whatever the IDE brings acts
on the IDE.

### Every command that sets a project up or keeps it current moves under `codefall project`

`init`, `upgrade`, and `create` move under the group with their flags and their output unchanged.
The group names what the commands act on, and it separates `codefall project upgrade`, which
brings a project current, from `codefall update`, which replaces the binary. The command tree:

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

`cli/cmd/codefall/main.go` builds the group and mounts on it what it mounts on the root today:
the commands `initcmd.Commands` returns and `create.Command`. The use cases, the steps, and the
manifest are untouched. A refusing command belongs to the component that held the name:
`initcmd` exports the two for `init` and `upgrade`, and `create` exports its own.

### The old root names refuse, and name the command under `project`

`codefall init`, `codefall upgrade`, and `codefall create` exit non-zero with one sentence that
names the command under `project`, whatever flags they were given, and write nothing. The rename
is a breaking change: the repository defines one as a change a project has to answer by hand
after upgrading, and every script that runs a root command is that. The ADRs and the
decision log name the old commands too, and stay as they were written, because a record is never
rewritten.

### `config` and `doctor` stay at the root and take `project` as a second spelling

`codefall project config` and `codefall project doctor` run the same commands as the root's,
subcommands and flags included, limited to the project. Every setting and every check is about
the project today, so the two spellings do the same thing until a root command covers something
outside one. One editor over one file stays one editor (ADR-012): a second spelling is a second
mount, not a second editor, and each mount is a command built for it alone, as the CLI rules
require of a command that appears in two places.

### Each settings file is named for what owns it

A project's settings and the IDE's settings are two files, and each is named for what owns it.
The project's file becomes `.codefall/project.json`, beside `.codefall/user.json`. What the
file holds does not change, and its `version` stays 1. `codefall project upgrade` moves the file
in the step that already rewrites a harness spelling and the `$schema` URL, and reports the
move. The JSON schema becomes `cli/schemas/project.schema.json`, and the old schema's URL joins
`FormerSchemaIDs`, so the same step repoints a file's `$schema`. The plan that builds the IDE
names the IDE's file and says where it lives, which may be outside a repository.

### What is left for later records

Three things are left to later records:

- `codefall project remove`, which deletes what a project's install wrote.
- Installing a skill set that does not ship in the binary.
- The IDE and its settings file.

## Consequences

- **Every project and script that runs a root name changes it once.** The refusal names the
  command under `project`, so a person changing a script has the new name in the error. The
  command rename and the settings-file rename release together, so a project crosses one
  breaking release, not two. Both renames reach a project through the warning ADR-010 puts in
  upgrade, which prints a release's breaking changes before it applies anything.
- **The installed text changes in the same pull requests as the code.** The skills, the
  `AGENTS.md` sections, and the hook notices under `extensions/` name `init` and `upgrade` 68
  times, and the settings file in 37 files, and all of it is installed into every project. The
  command names change in the pull request that moves the commands, and the file name in the
  one that moves the file, each with the Go code that reads and writes it. A test reads every
  `codefall <command>` the installed text names and fails on one the command tree does not hold,
  which keeps the next rename from leaving one behind.
- **A project's own script that reads `.codefall/settings.json` changes its path.** `codefall
  project upgrade` moves the file, reports the move, and replaces the installed skills in the
  same run, so no installed skill reads a file that is not there. A script the project wrote
  against the old path is the project's to change.
- **`codefall project upgrade` and `codefall update` no longer share a verb.** A person reading
  the root's help finds `update` beside a group whose name says what it acts on, and a person
  looking for what brings a project current finds it under `project`.
- **Two spellings of `config` and `doctor` must say the same thing until one of them covers more
  than a project.** The two mounts are two commands, and every subcommand and every check one
  gains, the other gains with it. They part ways when a root command covers something outside a
  project, and not before.

## Related

- [ADR-010](ADR-010-upgrade.md) — the record this one changes, and only in its command names:
  `init` and `upgrade` move under `project`, and nothing about what the two do changes with
  them. Upgrade still refuses a project with no manifest and names init, still removes what the
  previous install wrote and this one did not, and still warns about breaking changes.
- [ADR-006](ADR-006-install-layout.md) — the install layout the group's commands write; the
  layout itself is unchanged.
- [ADR-012](ADR-012-ui-composition.md) — one editor over one settings file, which is why
  `config` stays whole under a second spelling instead of gaining an editor per place.
- `docs/plans/PLAN-001-codefall-project.md` — the plan this record is written from: the command
  tree, the settings file's rename, and the tasks.
- Issues #261, #262, and #264 — the work that carries this record out: the move and the
  refusals, the two spellings of `config` and `doctor`, and the settings file's rename.
