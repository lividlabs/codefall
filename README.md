<p align="center">
  <img src="assets/codefall-logo.svg" alt="Codefall" width="460">
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

Codefall is an open source toolkit/meta-harness/software factory that gives coding agents a
software development process. It provides the process as skills the agent runs, a template for
each document the process produces, and an architecture to start a new project on, recorded as
architecture decision records (ADRs).
Each skill leaves its result in your repository as a document, a task, or a pull request, where the
next skill, and the next session, reads it.

The `codefall` command installs the skills into your project for the coding agents it uses: Claude
Code, Codex, OpenCode, Antigravity, or Muse. A *skill* is a set of instructions the agent follows
when you type its slash command, such as `/codefall-specify`. The skills take a feature from an idea
to a pull request you merge, and other skills start a new project, write up a bug, draw a mockup,
and keep the install current. [How it works](#how-it-works) walks one feature through them;
[Skills](#skills) lists them all.

### Why spec-driven development?

Codefall is a kit for *spec-driven development*: you and the agent agree in writing on what to build
before the agent builds it, and the written record drives every step after. Working this way:

- **The record prevents context decay.** A coding agent forgets what it decided when its context
  ends. Codefall writes each decision to a document, a task in a dependency graph stored in git, or
  a pull request, so the next session starts from the record rather than from memory.
- **A lifecycle replaces vibe coding.** A feature goes through requirements, a design, tasks, code,
  review, and test, in that order. You approve the requirements, the design, and the start of the
  build, and the rest runs on its own.
- **Decision records catch architectural drift.** Scaffold records a new project's architecture as
  ADRs and a linter that enforces its boundaries. Plan reads the ADRs before it designs a change and
  records each new hard-to-reverse choice as another, and review reads the code against them.
- **Every round catches and records bugs.** A second agent reviews each change, tests are written
  from the acceptance criteria rather than from the code, and what either finds is filed as a task,
  triaged, and fixed in the next round.
- **The skills guide the agent without constraining it.** A skill says what to produce and in what
  order, and leaves how to the agent's judgment. A project adds its own procedure to any skill, and
  the agent runs a skill on its own when the work calls for it.

Codefall is one of several spec-driven kits. GitHub's [Spec Kit](https://github.com/github/spec-kit)
takes a feature from a project constitution through a spec, a plan, and tasks to code, in the agent
you already use. [Kiro](https://kiro.dev/), from AWS, builds specs into its own IDE and command line
as requirements, design, and tasks files. [Cascadia](https://github.com/docraticmethod/cascadia)
compiles requirements into a strategy, an architecture, and instructions for the agent to build
from. Codefall keeps the whole lifecycle in the repository: the architecture and its ADRs, the task
graph, the reviews, the test results, and the bug reports, beside the specs and plans.

## Installation

### Prerequisites

Codefall needs these tools on your `PATH`:

- [git](https://git-scm.com/).
- [Beads](https://github.com/gastownhall/beads), a task tracker that stores its data in your git
  repository. Codefall keeps its task graph there. The command is `bd`; on macOS, install it with
  `brew install beads`.
- The [GitHub CLI](https://cli.github.com/), `gh`, logged in to github.com. Codefall opens pull
  requests and issues with it, so your project needs a GitHub repository.
- At least one supported coding agent: Claude Code, Codex, OpenCode (version 2 or later), Antigravity,
  or Muse.

### Install script

```sh
curl -fsSL https://install.codefall.dev/sh | sh
```

The script downloads the latest release, verifies it against the release's checksums, and installs
it to `~/.local/bin`. If that directory is not on your `PATH`, the script adds it in your shell's
startup file and tells you to open a new terminal; set `CODEFALL_NO_MODIFY_PATH=1` to have it print
the line instead. To install a specific version, pass the version as an argument:

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

`init` asks which coding agents the project uses, where its tests should live, and whether to copy
specs and bug reports to GitHub Issues. It then installs
the skills for each agent, saves Codefall's settings in `.codefall/settings.json`, sets up Beads,
and adds a Codefall section to `AGENTS.md`, the file coding agents read for project instructions.
The settings are meant to be checked in, so everyone on the team works from the same ones.

The skills work on your project's existing architecture. To adopt Codefall's architecture as well,
run `/codefall-upgrade` in your coding agent. It offers each architecture decision as a document
you can take or decline; `/codefall-upgrade adopt` offers the ones you declined again.

### A new project

A new project takes three steps. First, create the repository:

```sh
codefall create todo
```

`create` makes the directory and a git repository with a first commit, then runs `init` there.

Next, open your coding agent in the new directory and describe what you are building:

```
/codefall-envision Todo, a to-do list where you add tasks and check them off.
```

The skill asks what is wrong, who it affects, and what is still undecided, then writes a *vision*,
a short document that says what the project is for and what nobody has decided yet, to
`docs/visions/VISION-001-todo.md`.

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

Codefall's skills take a feature through three steps, and you merge the result. Each skill writes
something that the next one reads:

```
specify → plan → implement → you merge
```

Here is one feature going through it. Suppose your product is Todo, a to-do list, and you want
users to be able to export their list as a file.

**1. Say what to build.** In your coding agent, type:

```
/codefall-specify Users can export their to-do list as a file.
```

The skill interviews you about the feature: who uses it, what should happen, and what should happen
when something goes wrong. It then writes a *spec*, a document that lists the feature's
requirements, to `docs/specs/SPEC-003-list-export.md`. Each requirement carries *acceptance
criteria*: testable statements of what the finished feature must do. They are written in
[EARS](https://alistairmavin.com/ears/) (Easy Approach to Requirements Syntax), which limits each
criterion to a few fixed sentence patterns:

```
SPEC-003-REQ-01-AC-01  WHEN the user selects export, the system SHALL produce a file listing every task and whether it is done.
SPEC-003-REQ-01-AC-02  IF the list is empty, THEN the system SHALL say so instead of producing an empty file.
```

In a project that uses GitHub Issues, each requirement also becomes an issue. When a requirement
needs a screen, the skill runs `/codefall-mock-up` itself and puts the mockup beside the spec. The
spec arrives as its own pull request for you to merge.

**2. Decide how to build it.** Type:

```
/codefall-plan SPEC-003
```

The skill reads the spec and the code, then writes a *plan*, a short software design document, to
`docs/plans/PLAN-007-list-export.md`. When the feature looks like more than one delivery's worth of
work, it says so and offers to write one plan per shippable increment instead, each built and
merged on its own. The plan ends with a table of tasks and the order they depend on:

```
| ID | Task                                 | Depends on |
|----|--------------------------------------|------------|
| T1 | Add the list export format           | —          |
| T2 | Add the export button and download   | T1         |
```

When you approve the table, each row becomes a *bead*, which is a task in Beads. The beads record
which task waits on which, so the agent always knows what is ready to start. A small fix skips the
plan document and gets beads alone. Beads holds the tasks in every project; the issue tracker only
holds copies of specs and bug reports, and a project can choose to keep none.

**3. Build it.** Type:

```
/codefall-implement PLAN-007
```

The skill reads the task graph and sorts the tasks into *waves*: every task whose dependencies are
met goes into the same wave, and the tasks in a wave are built at the same time. It shows you the
waves, and after you approve, it runs to the end without asking again. Each task is built in its
own worktree with its own tests, checked against its acceptance criteria, and opened as a pull
request. A bug or a loose end a task turns up is filed as a bead when the task reports it, never
fixed in passing. The skill reports progress in one line:

```
todo-PLAN-007 · round 1 · 2/3 ███████████░░░░░░
```

When the last wave is built, the skill runs a *round* of checks on the whole delivery. It calls
`/codefall-review`, which has a second agent, in another coding harness when the project has one
configured, read the changes and file each problem it finds as a bead. It then calls
`/codefall-test`, which runs the project's test suites and the test cases written from the
acceptance criteria, and files what fails the same way. Then it triages everything filed under the
delivery, picks the blockers and the important findings, and any minor one it judges worth the
round, as the next batch, and loops: it builds the batch in waves, reviews, tests, and triages
again. The loop ends when a round leaves nothing worth another, with every finding fixed or set
aside with a reason and the last test run passed, or when the skill judges that another round
would not bring the delivery closer to done, and says why. A delivery takes at least three rounds,
and ten is the ceiling, which you can raise when you approve the build.

**4. Merge it.** The skills stop at open pull requests. You review and merge them on GitHub.

### Fixing a bug

A bug takes a shorter path with the same shape: a document, a task, and a pull request.

```
report → fix → you merge
```

**1. Write it up.** Type:

```
/codefall-report Checking off the last task still shows one task left.
```

The skill interviews you for the steps, what you expected, what happened instead, and any
screenshots or output you have, then offers to reproduce the bug in the running app. It writes a
*bug report*, with the evidence beside it, to `docs/bugs/BUG-012-last-task-count.md`, including
acceptance criteria for the fix, and files one bead for it, triaged by area, priority, and size. In
a project that uses GitHub Issues, the report becomes an issue too. The report arrives as its own
pull request for you to merge.

**2. Fix it.** Type:

```
/codefall-fix BUG-012
```

The skill finds the cause, confirms the one task with you, and builds it in the same run: a
worktree, a test that covers the bug, the fix, and a pull request. When the change turns out to
need a design document, an ADR, or more than one task, it stops and names `/codefall-plan`
instead. You review and merge the pull request.

Some skills work outside both paths. `/codefall-envision` captures an idea before anyone specifies
or builds it. From a sentence, a pitch document, or a folder of mockups, it writes a vision: what is
wrong, who it affects, the rough shape of an answer, and what nobody has decided yet, kept as open
questions rather than invented. `/codefall-mock-up` adds a mockup of a screen to the repository,
drawn to match the app's existing design. `/codefall-report triage` sorts the backlog by area and
priority and cuts an epic from the items you pick. `/codefall-review` and `/codefall-test` also run
on their own, on a pull request, a document, or the whole project, and end by asking which problems
to fix.

Two rules hold throughout. First, every step's result is a file, a bead, or a pull request, so you
can clear the agent's context between steps and lose nothing. Second, no skill merges code, and in
coding agents that support hooks, a hook (a check the agent runs before each command) refuses any
merge or push to `main`.
[`docs/workflow.md`](docs/workflow.md) shows what each skill reads and writes.

## Skills

The workflow is opinionated about what each step leaves behind and flexible about where you start.
Every skill runs on its own, at any point. A change too small for a spec goes through plan and
implement, or through fix in one run, and a small plan is a task table with no document. To have a
pull request, a branch, or a set of files looked at, call review. A project adds its own procedure
to any skill, such as a question it always asks or a step that runs after writing, in
`.codefall/skills/<skill>/CUSTOMIZE.md`. [`customizations.md`](extensions/shared/customizations.md)
says what a customization may change.

| Skill | What it does | Writes to |
| --- | --- | --- |
| **Ideas and requirements** | | |
| [`/codefall-envision`](docs/guide/envision.md) | Turns an idea into a vision before anyone specifies or builds it: what is wrong, who it affects, the rough shape of an answer, and what nobody has decided yet. Takes a sentence, a pitch document, or a folder of mockups. | `docs/visions/` |
| [`/codefall-specify`](docs/guide/specify.md) | Turns a feature into requirements with acceptance criteria, copied to GitHub Issues when the project uses them. | `docs/specs/` |
| [`/codefall-report`](docs/guide/report.md) | Writes a bug report, tries to reproduce the bug, and copies it to GitHub Issues when the project uses them. With `triage`, sorts the backlog and cuts an epic from the items you pick. | `docs/bugs/`, Beads |
| [`/codefall-mock-up`](docs/guide/mock-up.md) | Imports or draws a mockup of a screen that matches the app's design. | `docs/mockups/` |
| **Building** | | |
| [`/codefall-plan`](docs/guide/plan.md) | Writes a software design document for the change, breaks the work into tasks with dependencies, and records hard-to-reverse decisions as ADRs. | `docs/plans/`, Beads |
| [`/codefall-implement`](docs/guide/implement.md) | Builds the tasks in waves, each with its tests and a pull request, then reviews, tests, and triages the result in rounds until the delivery is done. | pull requests |
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
  `docs/`. A project that uses GitHub Issues gets them generated from the documents, so the two
  never disagree about what was decided.
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
| `dart-flutter` | Flutter apps on mobile, desktop, and the web | supported |
| `rust-native`, `kotlin-native`, `swift-native`, `python`, `java` | | planned |

When a project needs a profile that is not supported yet, scaffold stops instead of guessing.
[The architecture guide](docs/guide/architecture.md) explains the rules and how scaffold matches a
project to its profiles.

## Where to go next

| To | Read |
| --- | --- |
| See every install option | [Installing codefall](docs/guide/install.md) |
| Find out what `init` adds to a repository | [Setting up a project](docs/guide/setup.md) |
| Choose which agents review, consult, and build, or set your persona | [Configuration](docs/guide/configuration.md) |
| Learn how one skill works | [The guide](docs/guide/) |
| See what each skill reads and writes | [`docs/workflow.md`](docs/workflow.md) |
| Merge document pull requests automatically | [`docs/landing-documents.md`](docs/landing-documents.md) |
| Understand the architecture that scaffold sets up | [Architecture](docs/guide/architecture.md) |
| See what is planned | [Roadmap](extensions/docs/ROADMAP.md) |

## Terms

- **Acceptance criterion:** a testable statement of what a feature must do, written in EARS.
- **ADR:** architecture decision record, a short document that records one hard-to-reverse decision
  and the reasons for it.
- **Bead:** one task in Beads.
- **Beads:** a task tracker that stores its data in your git repository. Codefall keeps its task
  graph there in every project.
- **Coding agent:** an AI tool that reads and edits the code in your repository, such as Claude Code
  or Codex. Codefall's settings and flags call these *harnesses*.
- **EARS:** Easy Approach to Requirements Syntax, a format that limits each requirement to a few
  fixed sentence patterns.
- **Hook:** a check the coding agent runs at a fixed point, such as before each command.
- **Mirror:** the copy of a spec or bug report that Codefall keeps in the issue tracker, in a
  project that chose one. The document is the record, and the mirror is regenerated from it.
- **Persona:** whether the skills talk to you as an engineer or as a product manager.
- **Profile:** the templates and decisions scaffold applies to one kind of code, such as Go.
- **Skill:** a set of instructions a coding agent follows when you type its slash command.
- **Spec:** a document that lists a feature's requirements and their acceptance criteria.
- **Spec-driven development:** agreeing in writing on what to build before an agent builds it, with
  the written record driving each step after.
- **Vision:** a short document about what a project or feature is for: what is wrong, who it
  affects, the rough shape of an answer, and what is still undecided.

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
