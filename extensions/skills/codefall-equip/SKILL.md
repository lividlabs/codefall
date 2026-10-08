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
runs with and declares it; it runs it only to prove it.

Paths that start with `reference/`, `templates/`, or `../` are relative to this skill's directory,
not the user's project. A path through `../../../.codefall/` is the one that leaves the skills
directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## The four tracks

One run equips one of the four. Each has its own search, its own question, its own declaration,
its own proof, and its own pull request.

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

Read each when its track says to; none is loaded up front.

- `reference/local.md`, `reference/testing.md`, `reference/agent-setup.md`, `reference/landing.md` —
  one track each, in full: every step, what it searches for, the one question, the shape of every
  write, the proof, the report, and the rules that hold for that track alone. Read before step 1
  of the track.
- `reference/signals.md` — what in a repository says which tools it uses, the entry points it may
  already have, what each maps to, and consulting before the one question. Read at step 2 of the
  local track. Names `../../../.codefall/shared/running-agents.md`,
  `../../../.codefall/shared/run-agent.sh`, `../../../.codefall/shared/consult-prompt.md`,
  `../../../.codefall/shared/consult.schema.json`.
- `reference/contract.md` — what `start` and `update` promise, in full. Read at step 2 of the
  local track.
- `templates/local.sh` — the default script when the project has no task runner: one file, two
  subcommands. Read at step 3 of the local track.
- `templates/codefall-land-documents.yml` — the workflow the landing track installs.
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
| Revising either when the project's tools change | Deciding which tools the project uses | `codefall-plan` |
| Declaring `local`, `test.runners`, `harnessConfig`, and the `agents` lists in `.codefall/settings.json` | Per-branch databases, shared or otherwise | the project |
| Recording the runner's commands in the testing root's `AGENTS.md` | The testing root, its tree, and the `CODEFALL TESTING` section | `codefall init` |
| Proving a candidate meets the contract, or an agent answers | Any other settings field | `codefall init` |
| Installing the workflow and label that merge document pull requests | Changing a branch rule or any other repository setting | the owner |

**Every run does the same thing**, whether what it equips exists or not, first draft and revision
alike: read the project, propose the draft or the revision, confirm, write, declare. There is no
one-time mode.

## The contract

What `start` and `update` promise, in `reference/contract.md`: plain shell commands from the
project root, idempotent and cheap the second time, never destructive. The contract is not
negotiable: a candidate that does not keep it — one that drops, resets, or deletes — is reported
and never declared, however convenient it is, and a draft that would not keep it is not written.

## Search first, then ask with evidence

On an existing project the scripts usually exist under some name already. Read the repository
before asking anything, name what was found, and ask one question:

> I found `make dev-up`, which starts Postgres and Redis through compose, and `npm run db:migrate`.
> Are these the start and update commands, or should I draft new ones?

**Never ask whether something exists before looking.** Every track works this way: the testing
track finds the runner configuration and the test directory, the agents track reads the session's
own record, and the landing track reads `.github/workflows/` and the branch rule, each shown before
the question.

**What the search leaves ambiguous is consulted on once** before the question, per
`reference/signals.md` for the local track and `reference/testing.md` for the testing track. The
answer becomes the proposed option; the user picks it. A consult never declares and never installs.

## When another verb follows this skill

`codefall-scaffold` and `codefall-implement` follow the local track as a procedure, never through
the harness, and never the other tracks. What each does, and what it skips, is in
`reference/followers.md`.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process — the local scripts

`reference/local.md` carries this track in full: the searches, the one question and its three
answers, the draft or the revision, the declaration, the proof, and the report. Read it before
step 1 and follow it. The track ends with the `local` block declared and its pull request open.

1. **Read the project**: `.codefall/settings.json` and whether a `local` block is declared, the
   declared scripts, and the project's `AGENTS.md` files; no settings file means `codefall init`
   comes first.
2. **Find what is already there**, per `reference/signals.md`: the entry points and the signals,
   each candidate checked against `reference/contract.md` by reading it. Then **ask the one
   question** with the evidence beside it: declare what exists, draft new, or revise what is
   declared.
3. **Draft or revise**: two targets in the project's task-runner idiom, or `templates/local.sh`
   when it has none. Show the full draft or the diff, confirm, and branch to `equip/local` before
   the first write.
4. **Declare** the `local` block in `.codefall/settings.json`, keeping every other key.
5. **Prove them**, on a yes: `start`, then `update`, then `update` again; then `codefall doctor`.
6. **Report** what was declared, the proof, and the landing as its own pull request; end with the
   merge as the one step.

## Process — the test harness

`reference/testing.md` carries this track in full: the eight steps, what to search for, the runner
each surface takes, what setting one up means, and the shape of every write. Read it before step 1
and follow it. `<root>` is the testing root `test.dir` declares. The track ends with
`test.runners` declared, the Runners line recorded, and its pull request open.

1. **Read the declaration**: the `test` block, then the project's root `AGENTS.md` and
   `<root>/AGENTS.md`; no block, or no settings file, means `codefall init` declares the testing
   root first.
2. **Find what is already there**: a runner configuration and the end-to-end test directories, each
   hit read; name the surfaces.
3. **Ask the one question**: declare what exists, or set up the default runner for each surface. A
   surface this version has no spec runner for is refused before it is asked about.
4. **Set the runner up**: the dependency and a configuration collecting from `<root>/test-cases`.
   Show it, confirm, and branch to `equip/test-harness` before the first write.
5. **Declare and record**: `test.runners` in settings; the runner's line in `<root>/AGENTS.md`.
6. **Revise `update`** for the runner's own install, keeping [the contract](#the-contract).
7. **Prove it**: the runner's list command on the empty tree, then
   `../../../.codefall/shared/check-cases.sh` and, for Playwright,
   `../../../.codefall/shared/check-cases-playwright.sh`, then `codefall doctor`.
8. **Report**, land it as its own pull request, and end with the merge as the one step.

## Process — the agents

`reference/agent-setup.md` carries this track in full. Read it at step 1 and follow it. Every reply fits
on one screen: the configuration, one question, one change.

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
   the user does next: merge the pull request. A harness still not set up is named as a fact that
   this track settles once it is merged, not as a step after it.

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
   person's to merge, the one step; once it is merged, a document pull request merges on the
   `auto-merge` label or an approval.

## Rules

These hold on every track. A rule for one track alone is in that track's reference file.

- **Nothing is written without confirmation.** The full draft or the diff, shown first.
- **A bug in codefall is reported, never filed.** A failing shared script is codefall's; the issue
  text goes to the person, ready to paste. The one tracker a verb files on is the project's, per
  *Who is authoritative for what* in `../../../.codefall/shared/workflow.md`.
- **Revise the smallest thing.** A revision touches what the introduced tool needs and leaves the
  project's own lines alone.
- **Only the `local` block, the `test` block's `runners`, `harnessConfig`, and the `agents` lists.**
  No other settings field is this skill's to touch.
- **The merge is the one step a report ends on.** What follows it — the next verb, another
  harness, the label — is said as what happens once the pull request is merged, never as a step
  after it.
- **Refresh, test, and review are not this skill.** Say so and stop when the user wants the
  environment brought current, a suite run, or a review done, rather than equipped.
