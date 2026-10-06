# Setting up the local scripts

The local track in full: what to read, the two searches, the one question and its three answers,
the draft or the revision, the declaration, the proof, and the report. Read before step 1 of the
local track and follow it. `signals.md` and `contract.md` are read at the steps that name them.

## Contents

- [1. Read the project](#1-read-the-project)
- [2. Find what is already there](#2-find-what-is-already-there)
- [3. Draft or revise](#3-draft-or-revise)
- [4. Declare](#4-declare)
- [5. Prove them](#5-prove-them)
- [6. Report](#6-report)
- [Rules](#rules)

## 1. Read the project

The target is the path argument, or the working directory. Read `.codefall/settings.json` and note
whether a `local` block is declared; read the declared scripts when it is. Read the project's
`AGENTS.md`, root and scoped, for verify commands and conventions. If there is no
`.codefall/settings.json` at all, stop and say `codefall init` comes first.

## 2. Find what is already there

Read `signals.md`. Two searches:

- **Entry points** the project already has — Makefile and justfile targets, package scripts, a
  `scripts/` directory, a Taskfile or mise tasks, a `bin/setup` — whose names or bodies say start,
  up, setup, bootstrap, migrate, sync, dev.
- **Signals** of the tools the environment needs — lockfiles, a compose file, a migrations
  directory, a codegen config, a dotenv sample file — which decide what a draft has to do and what
  a candidate has to cover.

Read `contract.md`. Check each candidate against it by reading the candidate, not by running it,
and say which of the two kinds a candidate that does not keep it is — **wrong for any caller**, or
**written for a different caller** — in the words *What fails the contract* in `signals.md` gives.
Never tell the user their scripts are broken when they are not.

Then ask the one question, with what was found beside it. Three answers:

- **Declare what exists.** Go to step 4 with the candidates as they are.
- **Draft new.** Go to step 3 with the signals.
- **Revise what is declared.** The block already exists and the user says the tools changed. Go to
  step 3 with the declared scripts open.

When no candidate keeps the contract, the first answer is not offered. Ask draft or revise only,
and name the steps of the existing scripts the draft reuses verbatim, so the user sees what they
keep.

## 3. Draft or revise

Read `../templates/local.sh`. A project with a **task runner idiom** — a Makefile, a justfile,
package scripts everyone runs — gets two targets in that idiom; a project with **none** gets
`scripts/local.sh` from the template, with `start` and `update` as its subcommands. Each step in
`update` is the tool's own idempotent form, guarded where it has no cheap no-op, per *Drafting* in
`signals.md`. A **revision** changes only what the introduced tool needs and leaves the rest as the
project wrote it; show the diff, not the whole file.

Show the full draft, or the diff, and confirm before writing anything. Then, before the first
write, take the branch step of `../../../../.codefall/shared/landing.md`: standing on the default
branch, `git switch -c equip/local`; on another branch, ask once which to use.

## 4. Declare

Write the `local` block into `.codefall/settings.json`, keeping every other key and the file's
formatting:

```json
"local": {
  "start": "scripts/local.sh start",
  "update": "scripts/local.sh update"
}
```

Commands are relative to the project root. A script of the project's own is made executable.

## 5. Prove them

Offer to run them; `start` brings real services up, so it is a yes-or-no, not a step. On yes:
`start`, then `update`, then `update` again. The second `update` must exit `0` quickly, and that is
the proof of the contract. A failure is fixed in the script and the proof re-run; a failure that is
the environment's — Docker not running — is reported with what to do.

Then run `codefall doctor` when the CLI is on `PATH` and read its **Local environment** section:
both checks pass, or the declaration is wrong and step 4 is repeated. A warning about the refresh
stamp not being git-ignored is `codefall upgrade`'s to fix; name it in the report.

## 6. Report

- What was declared, and whether it was found, drafted, or revised.
- Whether the scripts were proven, and by which run.
- The landing, per `../../../../.codefall/shared/landing.md`: the scripts and the settings committed
  by path on the branch, pushed, and its pull request opened without asking. A person merges it.
- **Last, what the user does next**: merge the pull request. Once it is merged, the first
  `/codefall-refresh` writes the stamp; that is a fact, not a second step.

## Rules

The rules that hold for every track are in `../SKILL.md`. These hold for this one:

- **Declare relative to the project root.** Commands, not paths; a task runner's target is a
  command.
- **Proving runs real services, so it is offered.** Never started on the user's machine unasked.
