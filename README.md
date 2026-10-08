<p align="center">
  <img src="assets/codefall-logo.svg" alt="Codefall" width="460">
</p>

<p align="center">
  A workflow for building software with coding agents, from an idea to a merged pull request.
</p>

<p align="center">
  <a href="https://github.com/lividlabs/codefall/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/lividlabs/codefall"></a>
  <a href="https://github.com/lividlabs/codefall/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/lividlabs/codefall/actions/workflows/ci.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/github/license/lividlabs/codefall"></a>
</p>

<p align="center">
  <a href="#installation">Install</a> · <a href="#how-it-works">How it works</a> · <a href="docs/guide/">Guide</a>
</p>

## What is Codefall?

A coding agent such as Claude Code or Codex can build a feature in an afternoon. Left to itself,
though, it forgets what it decided when the session ends, starts building before anyone has agreed
on what to build, and reviews its own work. Codefall gives the agent a process that addresses all
three.

Codefall has two parts. The `codefall` command-line tool installs a set of *skills* into your
project: instructions the coding agent follows when you type a slash command such as
`/codefall-specify`. The skills do the work, and each one leaves its result in your repository,
where the next one reads it.

With Codefall, you get:

- Requirements and plans written as Markdown files in your repository, reviewed and merged like code.
- Tasks tracked in a dependency graph stored in git, so any session can pick up where the last one
  stopped.
- One branch and one pull request per task.
- Reviews by a different agent from the one that wrote the code.
- Tests written from the requirements, not from the code they test.

## Installation

### Prerequisites

Codefall needs these tools on your `PATH`:

- [git](https://git-scm.com/).
- [Beads](https://github.com/gastownhall/beads), a task tracker that stores its data in your git
  repository. Codefall keeps its task graph there. The command is `bd`; on macOS, install it with
  `brew install beads`.
- The [GitHub CLI](https://cli.github.com/), `gh`, logged in to github.com. Codefall opens pull
  requests and issues with it, so your project needs a GitHub repository.
- At least one supported coding agent: Claude Code, Codex, OpenCode, Antigravity, or Muse.

### Install script

```sh
curl -fsSL https://install.codefall.dev/sh | sh
```

The script downloads the latest release, verifies it against the release's checksums, and installs
it to `~/.local/bin`. To install a specific version, pass the version as an argument:

```sh
curl -fsSL https://install.codefall.dev/sh | sh -s -- 0.30.0
```

To update later, run `codefall update`. The [install guide](docs/guide/install.md) covers the
script's options.

### mise

[mise](https://mise.jdx.dev/) installs and manages development tools. To install Codefall with it:

```sh
mise use -g packslip:github.com/lividlabs/codefall
```

Leave out `-g` to install Codefall for the current project only. When mise manages the binary,
`codefall update` prints the mise command that updates it.

### Release downloads

Each [release](https://github.com/lividlabs/codefall/releases) has prebuilt binaries for macOS and
Linux, on Intel and ARM.

## Set up a project

### An existing project

In the repository, run:

```sh
codefall init
```

`init` asks which coding agents the project uses and where its tests should live. It then installs
the skills for each agent, saves Codefall's settings in `.codefall/settings.json`, sets up Beads,
and adds a Codefall section to `AGENTS.md`, the file coding agents read for project instructions.
The settings are meant to be checked in, so everyone on the team works from the same ones.

The skills work on your project's existing architecture. To adopt Codefall's architecture as well,
run `/codefall-upgrade adopt` in your coding agent. It offers each architecture decision as a
document you can take or decline.

### A new project

A new project takes three steps. First, create the repository:

```sh
codefall create trips
```

`create` makes the directory and a git repository with a first commit, then runs `init` there.

Next, open your coding agent in the new directory and describe what you are building:

```
/codefall-envision A trip planner where travelers build an itinerary and share it.
```

The skill asks about the problem, who has it, and what is still undecided, then writes a *vision*,
a short document about what the project is for, to `docs/visions/VISION-001-trip-planner.md`.

Last, set up the architecture:

```
/codefall-scaffold
```

Scaffold reads the vision to find the project's parts, such as a React web app and a Go API, and
asks you to confirm them and choose a stack for each. It then records the project's architecture
decisions in `docs/adrs/` and writes an `AGENTS.md` for each part. You choose how much more it
writes:

| Depth | What scaffold writes |
| --- | --- |
| Docs only (the default) | The architecture decisions and `AGENTS.md` files |
| Docs and project files | Also the build configuration, a linter with rules that enforce the architecture, a formatter, a test runner, and CI |
| Runnable skeleton | Also a folder for each component and an app that starts with no features in it |

Scaffold stops once the project is set up and empty. The first feature starts with
`/codefall-specify`, as [How it works](#how-it-works) shows. [Design
principles](#architecture-for-new-projects) describes the architecture scaffold sets up.

### Check and update a project

To check a project's setup at any time, run:

```sh
codefall doctor
```

`doctor` lists everything Codefall needs, marks what is missing, and says how to fix each problem.
It never changes anything itself.

After you install a newer version of `codefall`, run `codefall upgrade` in each project to bring its
skills up to date. [Setting up a project](docs/guide/setup.md) lists everything `init` writes.

## How it works

Codefall's skills take a feature through a fixed sequence. Each skill writes something that the
next one reads:

```
specify → plan → implement → review and test → you merge
```

Here is one feature going through it. Suppose your product is a trip planner, and you want
travelers to be able to export their itinerary.

**1. Say what to build.** In your coding agent, type:

```
/codefall-specify Travelers can export a trip itinerary as a file.
```

The skill interviews you about the feature: who uses it, what should happen, and what should happen
when something goes wrong. It then writes a *spec*, a document that lists the feature's
requirements, to `docs/specs/SPEC-003-itinerary-export.md`. Each requirement carries *acceptance
criteria*: testable statements of what the finished feature must do. They are written in
[EARS](https://alistairmavin.com/ears/) (Easy Approach to Requirements Syntax), which limits each
criterion to a few fixed sentence patterns:

```
SPEC-003-REQ-01-AC-01  WHEN a traveler selects export, the system SHALL produce a file containing the itinerary.
SPEC-003-REQ-01-AC-02  IF the trip is missing a departure date, THEN the system SHALL name the missing field.
```

Each requirement also becomes a GitHub issue, and the spec arrives as its own pull request for you
to merge.

**2. Decide how to build it.** Type `/codefall-plan`. The skill reads the spec and the code, then
writes a plan to `docs/plans/PLAN-007-itinerary-export.md`. The plan ends with a table of tasks and
the order they depend on:

```
| ID | Task                                 | Depends on |
|----|--------------------------------------|------------|
| T1 | Add the itinerary export format      | —          |
| T2 | Add the export button and download   | T1         |
```

When you approve the table, each row becomes a *bead*, which is a task in Beads. The beads record
which task waits on which, so the agent always knows what is ready to start. A small fix skips the
plan document and gets beads alone.

**3. Build it.** Type `/codefall-implement`. The skill shows you how it will build the plan, and
after you approve, it works through the tasks without stopping. Each task gets its own branch, its
own tests, a check against its acceptance criteria, and a pull request. The skill reports progress
in one line:

```
trips-PLAN-007 · round 1 · 2/3 ███████████░░░░░░
```

**4. Check it.** `/codefall-review` has a second agent read the changes and list the problems it
finds. `/codefall-test` runs the project's tests, including test cases the agent works through step
by step in a browser or a terminal. Both end by asking which problems to fix, and the fixes you pick
go back through `implement`.

**5. Merge it.** The skills stop at open pull requests. You review and merge them on GitHub.

Some skills work outside this sequence. `/codefall-envision` writes down a rough idea before it is
ready to specify: the problem, who has it, and the questions nobody has answered yet.
`/codefall-report` writes a bug report, tries to reproduce the bug, and files it as an issue.
`/codefall-fix` handles a small change in one run by planning a single task and building it.
`/codefall-mock-up` adds a mockup of a screen to the repository, drawn to match the app's existing
design.

Two rules hold throughout. First, every step's result is a file, a bead, or a pull request, so you
can clear the agent's context between steps and lose nothing. Second, no skill merges code. A hook,
a check the coding agent runs before each command, refuses any merge or push to `main`.
[`docs/workflow.md`](docs/workflow.md) shows what each skill reads and writes.

## Skills

| Skill | What it does | Writes to |
| --- | --- | --- |
| **Ideas and requirements** | | |
| [`/codefall-envision`](docs/guide/envision.md) | Writes down an idea: the problem, who has it, and the open questions. | `docs/visions/` |
| [`/codefall-specify`](docs/guide/specify.md) | Turns a feature into requirements with acceptance criteria, mirrored to GitHub issues. | `docs/specs/` |
| [`/codefall-report`](docs/guide/report.md) | Writes a bug report, tries to reproduce the bug, and mirrors it to a GitHub issue. | `docs/bugs/` |
| [`/codefall-mock-up`](docs/guide/mock-up.md) | Imports or draws a mockup of a screen that matches the app's design. | `docs/mockups/` |
| **Building** | | |
| [`/codefall-plan`](docs/guide/plan.md) | Breaks the work into tasks with dependencies, and records hard-to-reverse decisions as ADRs. | `docs/plans/`, Beads |
| [`/codefall-implement`](docs/guide/implement.md) | Builds each ready task with its tests and opens a pull request for it. | pull requests |
| [`/codefall-fix`](docs/guide/fix.md) | Plans and builds one small change in a single run. | a pull request |
| **Checking** | | |
| [`/codefall-review`](docs/guide/review.md) | Has a second agent review code or a document, then applies the fixes you accept. | `.codefall/reviews/` |
| [`/codefall-test`](docs/guide/test.md) | Runs the project's test suites and test cases, and reports the results. | `.codefall/tests/` |
| **Project upkeep** | | |
| [`/codefall-scaffold`](docs/guide/architecture.md) | Starts a new project on Codefall's architecture. | `docs/adrs/`, `AGENTS.md` |
| [`/codefall-upgrade`](docs/guide/upgrade.md) | Brings a project's installed skills and documents up to date after a new release. | the project's Codefall files |
| [`/codefall-equip`](docs/guide/local-environment.md) | Sets up what the other skills need: local environment scripts, a test runner, review agents, and automatic merging for document pull requests. | `.codefall/settings.json` |
| [`/codefall-refresh`](docs/guide/local-environment.md) | Brings the checkout, the beads, and the local environment up to date with `main`. | nothing |

You run `/codefall-upgrade` and `/codefall-equip` yourself, because they change the project's setup.
The agent may run any of the others on its own when the work calls for it.

## CLI commands

| Command | What it does |
| --- | --- |
| `codefall create <dir>` | Makes a new directory and git repository, then runs `init` in it. |
| `codefall init` | Sets up Codefall in an existing repository. It runs once per project. |
| `codefall upgrade` | Updates a project's skills, hooks, and settings to match the installed `codefall`. |
| `codefall update` | Replaces the `codefall` binary with the latest release. |
| `codefall config` | Opens an editor for the project's review agents and your persona. |
| `codefall doctor` | Checks that a project has everything Codefall needs, and says how to fix what is missing. |

Your *persona* sets how the skills talk to you: as an engineer, the default, or as a product manager.
It never changes what they are allowed to do. [Configuration](docs/guide/configuration.md) covers
the settings in full.

## Design principles

- **Documents in the repository are the record.** Specs, plans, and bug reports are files under
  `docs/`. GitHub issues are generated from them, so the two never disagree about what was decided.
- **People merge code.** Skills open pull requests and stop. A person reviews and merges every one
  that changes code.
- **The agent that finds a problem does not fix it.** An agent reviewing its own work misses the
  same things it missed while writing it, so reviews run in a separate agent.
- **Tests come from the requirements.** A test written by reading the code confirms what the code
  does, bugs included. Codefall writes test cases from the acceptance criteria instead.
- **Every step leaves a handoff.** A skill finishes by writing a file, a bead, or a pull request, and
  its report ends by naming the next step.

### Architecture for new projects

`/codefall-scaffold` starts every new project on one architecture: Clean Architecture, a layout that
keeps business rules independent of frameworks and databases, organized by component. A component
is one area of the application, such as bookings or payments, with its own code and its own data.
The result is a single deployable application whose components stay cheap to split into separate
services later. Scaffold records each of these choices in the new project as an ADR (architecture
decision record).

Scaffold applies a *profile* for each kind of code in the project, which supplies the templates and
decisions for that language:

| Profile | Covers | Status |
| --- | --- | --- |
| `typescript-react` | TypeScript and Node backends; React front ends for the web and React Native | supported |
| `go` | Go services, APIs, workers, and command-line tools | supported |
| `rust-native`, `kotlin-native`, `swift-native`, `dart-flutter`, `python`, `java` | | planned |

When a project needs a profile that is not supported yet, scaffold stops instead of guessing.
[The architecture guide](docs/guide/architecture.md) explains the rules and how scaffold matches a
project to its profiles.

## Where to go next

| To | Read |
| --- | --- |
| See every install option | [Installing codefall](docs/guide/install.md) |
| Find out what `init` adds to a repository | [Setting up a project](docs/guide/setup.md) |
| Choose which agents review and consult, or set your persona | [Configuration](docs/guide/configuration.md) |
| Learn how one skill works | [The guide](docs/guide/) |
| See what each skill reads and writes | [`docs/workflow.md`](docs/workflow.md) |
| Merge document pull requests automatically | [`docs/landing-documents.md`](docs/landing-documents.md) |
| Understand the architecture scaffold sets up | [Architecture](docs/guide/architecture.md) |
| See what is planned | [Roadmap](extensions/docs/ROADMAP.md) |

## Terms

- **Acceptance criterion:** a testable statement of what a feature must do, written in EARS.
- **ADR:** architecture decision record, a short document that records one hard-to-reverse decision
  and the reasons for it.
- **Bead:** one task in Beads.
- **Beads:** a task tracker that stores its data in your git repository. Codefall keeps its task
  graph there.
- **Coding agent:** an AI tool that reads and edits the code in your repository, such as Claude Code
  or Codex. Codefall's settings and flags call these *harnesses*.
- **EARS:** Easy Approach to Requirements Syntax, a format that limits each requirement to a few
  fixed sentence patterns.
- **Hook:** a check the coding agent runs at a fixed point, such as before each command.
- **Persona:** whether the skills talk to you as an engineer or as a product manager.
- **Profile:** the templates and decisions scaffold applies to one kind of code, such as Go.
- **Skill:** a set of instructions a coding agent follows when you type its slash command.
- **Spec:** a document that lists a feature's requirements and their acceptance criteria.

## Contributing

Codefall's source has two parts. [`cli/`](cli/) is the `codefall` command-line tool, written in Go,
and [`extensions/`](extensions/) holds the skills and hooks it installs. The rules for working on
each are in [`AGENTS.md`](AGENTS.md), and the repository's own decisions are in
[`docs/adrs/`](docs/adrs/). Changes land as squash-merged pull requests with
[Conventional Commit](https://www.conventionalcommits.org/) titles.

## License

[MIT](LICENSE). Copyright (c) 2026 Livid Labs, LLC, authored by Dave Jensen.

The templates under `extensions/skills/codefall-scaffold/templates/`, and everything `scaffold`
copies from them into your project, are also available under [0BSD](LICENSE): no attribution, no
notice, no obligation. Your architecture documents are yours.
