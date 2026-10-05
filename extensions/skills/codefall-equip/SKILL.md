---
name: codefall-equip
description: Equip a project with the four things it has to have before the other verbs work. The local environment — start and update, declared under local in .codefall/settings.json and run by codefall-refresh. The test harness — a spec runner per surface, pointed at the testing root, declared in test.runners, its commands recorded in the testing root's AGENTS.md, which codefall-test runs cases through. The agents — how another harness is called, under harnessConfig, and who reviews and consults, in the agents lists — set up from any harness. The document landing — the GitHub Action that merges a document pull request once a person adds the land label to it. Each finds what the project already has, or drafts it from what the repository or the session shows, confirms, writes, and declares. Builds and rebuilds; codefall-scaffold and codefall-implement follow the local-scripts procedure.
argument-hint: "[local | test | agents | landing] [path]"
disable-model-invocation: true
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
---

# Equip

Equip a repository with what the other verbs need it to have. Four things: the **local
environment** — `start`, which brings its services up, and `update`, which makes the local
environment match the checkout; the **test harness** — the spec runner that collects the specs
beside the project's test cases; the **agents** — how another harness is called, and who reviews
and consults; and the **document landing** — the GitHub Action that merges a document pull request
once a person labels it `auto-merge` or approves it. The first three are declared in
`.codefall/settings.json`: `local`, which `codefall-refresh` runs; `test.runners`, which
`codefall-test` reads; `harnessConfig` and the `agents` lists, which `codefall-review` and every
consult read. The fourth is one file under `.github/workflows/` and one label.

Equipping is neither refreshing nor testing. This skill finds, drafts, or revises what a project
runs with and declares it; it runs it only to prove it. Bringing an environment current is
`codefall-refresh`; running a suite or a case is `codefall-test`.

Paths that start with `reference/`, `templates/`, or `../` are relative to this skill's directory,
not the user's project. A path through `../../../.codefall/` is the one that leaves the skills
directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## The four tracks

One run equips one of the four. Each has its own search, its own question, its own declaration,
and its own proof, and in the user's project each lands as its own pull request, which a person
merges.

| Argument | Track | Procedure |
| --- | --- | --- |
| `local`, or a path alone | the two local-environment scripts | [Process — the local scripts](#process--the-local-scripts) |
| `test` | the test harness | [Process — the test harness](#process--the-test-harness) |
| `agents` | how this harness is called, and who reviews and consults | [Process — the agents](#process--the-agents) |
| `landing` | the Action that merges document pull requests | [Process — the document landing](#process--the-document-landing) |

A path may follow any word; it is the project directory, and the working directory is the default.
With no word at all, read `.codefall/settings.json` and `.github/workflows/`, say what `local`,
`test`, `harnessConfig`, `agents`, and the landing declare today, and ask which of the four to
equip before anything else.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/signals.md` — what in a repository says which tools it uses, the entry points it may
  already have, what each maps to, and consulting before the one question. Read at step 2 of the
  local track. Names `../../../.codefall/shared/running-agents.md`,
  `../../../.codefall/shared/run-agent.sh`, `../../../.codefall/shared/consult-prompt.md`, `../../../.codefall/shared/consult.schema.json`.
- `templates/local.sh` — the default script when the project has no task runner: one file, two
  subcommands. Read at step 3 of the local track.
- `reference/testing.md` — the whole testing procedure: what to search for, the runner each surface
  takes, what installing each one means, the exact shape of every write, and how the result is
  proven. Read at step 1 of the testing track.
- `reference/agents.md` — the whole agents procedure: the display, the one question, the writes,
  the proof. Read at step 1 of the agents track.
- `reference/landing.md` — the whole document-landing procedure: what the workflow does, the
  branch-rule reading, the one question, the write, the proof. Read at step 1 of the landing track.
- `templates/codefall-land-documents.yml` — the workflow the landing track installs.
- `reference/contract.md` — what `start` and `update` promise, in full. Read at step 2 of the
  local track.
- `reference/followers.md` — what `codefall-scaffold` and `codefall-implement` do when they follow
  the local track. Read by those verbs, not by a run of this one.
- `../../../.codefall/shared/landing.md` — the shared procedure for the branch, the commit, the
  push, and the pull request. Read before the first write on any track.

## Scope — what a project runs with, not the running

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Finding the scripts and the runner a project already has | Running the scripts on an ordinary day, and pulling or comparing the checkout | `codefall-refresh` |
| Drafting the scripts when there are none | Running a suite or a case | `codefall-test` |
| Installing or declaring a spec runner per surface | Writing a test case | `codefall-implement` |
| Revising either when the project's tools change | Deciding which tools the project uses | `codefall-design` |
| Declaring `local`, `test.runners`, `harnessConfig`, and the `agents` lists in `.codefall/settings.json` | Per-branch databases, shared or otherwise | the project |
| Recording the runner's commands in the testing root's `AGENTS.md` | The testing root, its tree, and the `CODEFALL TESTING` section | `codefall init` |
| Proving a candidate meets the contract, or an agent answers | Any other settings field | `codefall init` |
| Installing the workflow and label that merge document pull requests | Changing a branch rule or any other repository setting | the owner |

**Every run does the same thing**, whether what it equips exists or not: read the project, propose
the draft or the revision, confirm, write, declare. There is no first-time mode.

## The contract

What `start` and `update` promise, in `reference/contract.md`: plain shell commands from the
project root, idempotent and cheap the second time, never destructive. A candidate that does not
keep it is reported and never declared; a draft that would not keep it is not written.

## Search first, then ask with evidence

On an existing project the scripts usually exist under some name already. Read the repository
before asking anything, name what was found, and ask one question:

> I found `make dev-up`, which starts Postgres and Redis through compose, and `npm run db:migrate`.
> Are these the start and update commands, or should I draft new ones?

**Never ask whether something exists before looking**: the repository answers that faster than the
person, and the person may not know. Every track works this way: the testing track finds the
runner configuration and the test directory, the agents track reads the session's own record, and
the landing track reads `.github/workflows/` and the branch rule, each shown before the question.

**What the search leaves ambiguous is consulted on once** before the question, per
`reference/signals.md` for the local track and `reference/testing.md` for the testing track, and the
answer becomes the proposed option. The user still chooses; a consult never declares or installs.

## When another verb follows this skill

`codefall-scaffold` and `codefall-implement` follow the local track as a procedure, never through
the harness, and never the other tracks: **a test harness is never set up inside another verb's
pull request.** What each does, and what it skips, is in `reference/followers.md`.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process — the local scripts

### 1. Read the project

The target is the path argument, or the working directory. Read `.codefall/settings.json` and note
whether a `local` block is declared; read the declared scripts when it is. Read the project's
`AGENTS.md`, root and scoped, for verify commands and conventions. If there is no
`.codefall/settings.json` at all, stop and say `codefall init` comes first.

### 2. Find what is already there

Read `reference/signals.md`. Two searches:

- **Entry points** the project already has — Makefile and justfile targets, package scripts, a
  `scripts/` directory, a Taskfile or mise tasks, a `bin/setup` — whose names or bodies say start,
  up, setup, bootstrap, migrate, sync, dev.
- **Signals** of the tools the environment needs — lockfiles, a compose file, a migrations
  directory, a codegen config, a dotenv sample file — which decide what a draft has to do and what
  a candidate has to cover.

Read `reference/contract.md`. Check each candidate against it by reading the candidate, not by
running it, and say which of the two kinds a candidate that does not keep it is — **wrong for any
caller**, or **written for a different caller** — in the words *What fails the contract* in
`reference/signals.md` gives. Never tell the user their scripts are broken when they are not.

Then ask the one question, with what was found beside it. Three answers:

- **Declare what exists.** Go to step 4 with the candidates as they are.
- **Draft new.** Go to step 3 with the signals.
- **Revise what is declared.** The block already exists and the user says the tools changed. Go to
  step 3 with the declared scripts open.

When no candidate keeps the contract, the first answer is not offered. Ask draft or revise only —
revise only when a `local` block is declared — and name the steps of the existing scripts the
draft reuses verbatim, so the user sees what they keep.

### 3. Draft or revise

Read `templates/local.sh`. A project with a **task runner idiom** — a Makefile, a justfile,
package scripts everyone runs — gets two targets in that idiom; a project with **none** gets
`scripts/local.sh` from the template, with `start` and `update` as its subcommands. Each step in
`update` is the tool's own idempotent form, guarded where it has no cheap no-op, per *Drafting* in
`reference/signals.md`. A **revision** changes only what the introduced tool needs and leaves the
rest as the project wrote it; show the diff, not the whole file.

Show the full draft, or the diff, and confirm before writing anything. Then, before the first
write, take the branch step of `../../../.codefall/shared/landing.md`: standing on the default
branch, `git switch -c equip/local`; on another branch, ask once which to use.

### 4. Declare

Write the `local` block into `.codefall/settings.json`, keeping every other key and the file's
formatting:

```json
"local": {
  "start": "scripts/local.sh start",
  "update": "scripts/local.sh update"
}
```

Commands are relative to the project root. A script of the project's own is made executable.

### 5. Prove them

Offer to run them; running `start` brings real services up on the user's machine, so it is a
yes-or-no rather than a step. On yes: `start`, then `update`, then `update` again. The second
`update` must exit `0` quickly, and that is the proof of the contract. A failure is fixed in the
script and the proof re-run; a failure that is the environment's — Docker not running — is
reported with what to do.

Then run `codefall doctor` when the CLI is on `PATH` and read its **Local environment** section:
both checks pass, or the declaration is wrong and step 4 is repeated. A warning about the refresh
stamp not being git-ignored is `codefall upgrade`'s to fix; name it in the report.

### 6. Report

- What was declared, and whether it was found, drafted, or revised.
- Whether the scripts were proven, and by which run.
- The landing, per `../../../.codefall/shared/landing.md`: the scripts and the settings committed by
  path on the branch, pushed, and its pull request opened without asking — its own, never another
  verb's. A person merges it.
- **Last, what the user does next**: merge the pull request, run `/codefall-refresh` once so the
  stamp exists, and `codefall upgrade` if doctor warned about `.gitignore`.

## Process — the test harness

`reference/testing.md` carries this track in full — what to search for, the runner each surface
takes, what installing one means, and the shape of every write. Read it at step 1 and follow it.
`<root>` is the testing root `test.dir` declares.

### 1. Read the declaration

The target is the path argument, or the working directory. Read the `test` block in
`.codefall/settings.json`. No block, or no `.codefall/settings.json` at all: stop and say
`codefall init` declares the testing root first — this skill never declares one. Then read
`reference/testing.md`, the project's root `AGENTS.md`, and `<root>/AGENTS.md`.

### 2. Find what is already there

Search for a runner configuration and for the directories the project keeps end-to-end tests in,
and read each hit rather than counting it. Name the surfaces the repository shows.

### 3. Ask the one question

One question, with the evidence beside it: declare what exists, or set up the default runner for
each surface. A surface this version has no spec runner for — React Native, Tauri, Flutter — is
refused for the `spec` modality here, with the reason, and the refusal is said before it is asked
about. Agentic cases stay available for it wherever `codefall-test` has a driver.

### 4. Set the runner up

Install or declare it per `reference/testing.md`: the dependency, and a configuration whose test
directory is `<root>/test-cases` and whose match is the runner's suffix, with retries off and one
worker. Show the configuration and confirm before writing anything. Then, before the first write,
take the branch step of `../../../.codefall/shared/landing.md`: standing on the default branch,
`git switch -c equip/test-harness`; on another branch, ask once which to use.

### 5. Declare and record

Write `test.runners` into `.codefall/settings.json`, keeping every other key and the file's
formatting. Write the runner's line into the Runners section of `<root>/AGENTS.md`: its name, the
command that runs every spec, and the command that runs one case. Names go to settings because
programs read them; commands go to `<root>/AGENTS.md` because agents and people run them.

### 6. Revise `update`

The runner's own install is part of making the environment match the checkout — Playwright's
browsers, a Go tool the specs build with. Revise the declared `update` by step 3 of the local
track, keeping [the contract](#the-contract), and say what changed.

### 7. Prove it

Run the runner's list command against the empty tree: Playwright exits `0` and lists no specs,
which shows the configuration collects from where it says, and `go test` answers
`matched no packages`, which is that runner's expected answer until a spec exists. Then run
`../../../.codefall/shared/check-cases.sh`, and `../../../.codefall/shared/check-cases-playwright.sh`
as well when the runner is Playwright; no cases yet is a pass. Then `codefall doctor`, whose
**Testing** checks must all pass.

### 8. Report

What was declared and whether it was found or installed; where the runner's line went; what
`update` gained; what the proof showed. Then the landing, per `../../../.codefall/shared/landing.md`:
the configuration, the settings, `<root>/AGENTS.md`, and the `update` revision committed by path on
the branch, pushed, and its pull request opened without asking — its own, never another verb's. A
person merges it. **End with what the user does next**: merge the pull request; the first case is
written by `/codefall-implement` when a bead names one.

## Process — the agents

`reference/agents.md` carries this track in full. Read it at step 1 and follow it. Any harness
can be set up from any other; the track finds the parameters. Every reply fits on one screen: the
configuration, one question, one change.

1. **Read** `harnesses`, `harnessConfig`, and `agents`; no settings file means `codefall init`
   comes first. Answer which harness this is per
   `../../../.codefall/shared/running-agents.md`.
2. **Show the configuration** in the reference's shape, then **ask what to change**: set up a
   harness, change the lists, or nothing. A file `codefall doctor` says does not match the schema
   is not shown: say so in one line and rebuild it.
3. **Do the one thing chosen**, showing what will be written and asking `Write this?`. Nothing
   stops.
4. **Write** on `equip/agents` through `codefall config`; no CLI means stop and say to install it.
   A file out of date with the schema is written directly, then checked by `codefall doctor`.
5. **Prove it**: one run through `../../../.codefall/shared/run-agent.sh`, exit `0` and a
   non-empty out-file; then `codefall doctor`.
6. **Report** the configuration as it now stands, land it as its own pull request, and end with what
   the user does next: merge, and rerun it for any harness not yet set up.

## Process — the document landing

`reference/landing.md` carries this track in full. Read it at step 1 and follow it.

1. **Read** `.codefall/settings.json`, the default branch, and the GitHub repository; no settings
   file means `codefall init` comes first, and no GitHub remote is a stop.
2. **Find what is already there**: the workflow file, any other merging workflow, and the branch
   rule, read with `gh api` and never changed.
3. **Ask the one question**: install the workflow, shown in full, with the branch-rule finding in a
   sentence. A rule that would block the workflow's token is the owner's to change.
4. **Write** `.github/workflows/codefall-land-documents.yml` from the template, on `equip/landing`,
   and create the `auto-merge` label.
5. **Prove** that the file parses, the allowlist accepts a spec path and rejects a source path, and
   the label exists.
6. **Report** what was written, what the owner changes, and the landing: its own pull request, a
   person's to merge; then a document pull request merges on the `auto-merge` label or an approval.

## Rules

- **Search before asking.** Name what was found; never open with "do scripts exist?"
- **A consult proposes the option; the user picks it.** Never a declaration, never an install.
- **The contract is not negotiable.** A candidate that drops, resets, or deletes is reported and
  never declared, however convenient it is.
- **Nothing is written without confirmation.** The full draft or the diff, shown first.
- **Every run is the same procedure**, first draft and revision alike. No one-time mode.
- **Revise the smallest thing.** A revision touches what the introduced tool needs and leaves the
  project's own lines alone.
- **Declare relative to the project root.** Commands, not paths; a task runner's target is a
  command.
- **Proving runs real services, so it is offered.** Never started on the user's machine unasked.
- **Only the `local` block, the `test` block's `runners`, `harnessConfig`, and the `agents` lists.**
  No other settings field is this skill's to touch — the testing root, the tree under it, and the
  `CODEFALL TESTING` section are `codefall init`'s.
- **The runner follows the surface.** Playwright for a browser front end, an Electron shell, or an
  HTTP API; `go test` for a Go surface.
- **Refuse the `spec` modality where this version has no runner** — React Native, Tauri,
  Flutter — and say which modality remains.
- **Names in settings, commands in `<root>/AGENTS.md`.** Never the other way around, and never both.
- **A harness is equipped in its own pull request.** It never rides along in a task's.
- **The landing track changes no branch rule.** It writes one workflow file, creates the
  `auto-merge` label, and tells the owner what the branch rule has to allow.
- **Cases are not this skill's.** An empty tree is what proves a harness; writing the first case is
  `/codefall-implement`.
- **Refresh, test, and review are not this skill.** Say so and stop when the user wants the
  environment brought current, a suite run, or a review done, rather than equipped.
- **The agents track sets up any harness from any harness.** Parameters come from that harness's
  own records and config, and are written once the user confirms them.
