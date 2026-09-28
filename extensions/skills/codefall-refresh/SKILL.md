---
name: codefall-refresh
description: Bring the checkout, the Beads database, and the local environment current — fetch, fast-forward the default branch when that is safe, sync the beads, run the project's declared start and update commands, and record the commit the environment now matches. Safe to run at any time; the routine before starting new work.
argument-hint: "[path]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Bash
---

# Refresh

Bring the checkout, the Beads database, and the local environment current, in that order, and say
what happened in plain words. This is the thing to run before starting new work, and the thing to
run instead of pulling by hand: a git pull moves the code and leaves the team's beads, the database,
the dependencies, and the generated code where they were.

Refreshing is not equipping. This skill runs the `start` and `update` commands the project declared
under `local` in `.codefall/settings.json`; it never drafts or edits them. A project with nothing
declared is sent to `codefall-equip`.

Paths that start with `reference/` or `../` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills directory: it names
a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

- `reference/failures.md` — the failures `start` and `update` produce most often, what each means,
  and the sentence to say. Read at step 5 when a command exits non-zero.

## Scope — run, never write

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Fetching, and fast-forwarding the default branch when safe | Rebasing or merging a feature branch | the user |
| Syncing the beads with their Dolt remote | Wiring the remote, or resolving a conflict the sync halts on | the user |
| Running the declared `start` and `update` | Drafting, editing, or declaring them | `codefall-equip` |
| Recording the stamp after a clean `update` | Any other file in the project | — |
| Saying what failed and what to do | Fixing the script that failed | `codefall-equip` |
| Reporting a stash, a dirty tree, a detached HEAD | Touching any of them | the user |

## What refresh moves, and when

The checkout is moved in exactly one case. Everything else brings the environment level with the
checkout as it stands and says why the checkout was left alone.

| Checkout | Action on the checkout |
| --- | --- |
| On the default branch, clean tree, `origin/<default>` is a fast-forward | `git pull --ff-only origin <default>` |
| On the default branch, clean tree, diverged | Left alone: say the branch has local commits the remote does not, and how many |
| On the default branch, dirty tree | Left alone: say what is uncommitted, in files, and that a pull would land on top of it |
| On a feature branch | Left alone: report `behind` and leave the rebase to the user |
| Detached HEAD | Left alone: report it |
| No remote, or the fetch failed | Left alone: report it, and carry on with the environment |

**The environment is always brought level with the checkout it finds**, whatever the row above.
That is the half a hand pull leaves undone, and the reason this skill exists.

## The stamp

`.codefall/refresh.stamp` holds the commit `update` last exited `0` at, on this machine. It is
written here and nowhere else, after a clean `update`, as one line. It is per machine and
git-ignored — `codefall init` adds the entry — and a stamp matching `HEAD` is what lets a session
that starts three times on the same commit run `update` once.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Read the declaration

The target is the path argument, or the working directory. Read `.codefall/settings.json`.

- No file: stop and say `codefall init` comes first.
- No `local` block, or a block missing `start` or `update`: stop and say the project is not
  equipped yet, and that `/codefall-equip` is what does it.

### 2. Read the checkout

Run the shared check. The checkout lines decide step 3, and the `beads` line decides step 4.

```bash
"../../../.codefall/shared/preflight.sh" .
```

Note `branch`, `dirty`, `fetch`, `default_branch`, `behind`, `ahead`, `refresh`, and `beads`.

### 3. Move the checkout, if it is safe

Apply [the table](#what-refresh-moves-and-when). The one moving case is
`git pull --ff-only origin <default_branch>`; a pull that is refused is reported, never retried
with a merge or a rebase. After a move, `HEAD` has changed, so the stamp is compared again.

### 4. Sync the beads

Skip this step when preflight said `beads=blocked`: say so in the report, and leave the remedy to
the verb that needs beads. Otherwise:

```bash
bd sync
```

One command: pull the team's claims and closes from the Dolt remote, halt on a conflict, repair the
blocked flags the merged edges changed, and push whatever this machine wrote and never published.
Nothing in git moves. Read the exit code:

| Exit | Meaning | Say |
| --- | --- | --- |
| `0`, output says no remote is configured | No Dolt remote is wired | The database is this machine's. `bd dolt push --yes` adopts the git origin as the remote, and is the user's to run |
| `0` | Pulled and pushed | Synced, in the report |
| `2` | A conflict bd could not settle; nothing pushed | Which issues, from bd's output, and that `bd conflicts` resolves it by hand |
| `3` | Another writer won the push race; nothing pushed | Transient: run `/codefall-refresh` again |
| `4` | Uncommitted changes are stuck in the working set; nothing pushed | Quote the line; the user clears it, and nothing here retries |
| `1` | Transport, authentication, or storage | Quote the line; the fix is the environment's |

Never run `bd dolt push --force`, `bd dolt pull --strategy`, or `bd conflicts resolve` from here.

### 5. Bring the environment level

Run the declared `start`, always: it is idempotent and cheap when everything is up, and `update`
may assume it ran.

Then run the declared `update` when the stamp does not match `HEAD`, or when the user asked for
it regardless. A stamp that matches is the skip: say the environment was already current at
`<short hash>` and do not run `update`.

Run both from the project root, as declared, through the shell. Show the output as it arrives
when it is short; summarize it when it is long, and keep the last twenty lines of stderr for
step 6.

### 6. Say what happened

A clean exit from both: write the stamp.

```bash
git rev-parse HEAD > .codefall/refresh.stamp
```

Then confirm the stamp is ignored — `git check-ignore -q .codefall/refresh.stamp` — and, when it
is not, say so and name `codefall upgrade` as the fix. Never add the entry from here.

A non-zero exit: read `reference/failures.md`, match the stderr, and say three things — what
failed, what it means, and what to do — in one short paragraph a teammate who does not read
stack traces can act on. Quote the one line of stderr that says it, not the twenty. A failure that
is the environment's, such as Docker not running, ends with "run `/codefall-refresh` again once
that is done". A failure that is the script's — a command not found, a wrong path — names
`/codefall-equip` to fix the script, and this skill does not edit it.

No stamp is written after a failure.

### 7. Report

One block, short:

- The checkout: moved from `<short>` to `<short>`, or left alone and why.
- The beads: synced, no remote, skipped because beads is blocked, or halted and why.
- The environment: `start` ran; `update` ran or was skipped as current.
- The stamp: written at `<short>`, or not, and why.
- **Last, what the user does next**: nothing, or the rebase the user has to do, the
  `codefall upgrade` for `.gitignore`, the `/codefall-equip` for a broken script.

## Rules

- **Never drafts, edits, or declares a script.** A project with nothing declared is sent to
  `codefall-equip`; a script that fails is reported to it.
- **Moves the checkout in one case only**: default branch, clean tree, fast-forward. Never
  merges, never rebases, never stashes.
- **Syncs the beads with `bd sync` and never settles what it halts on.** No force push, no
  strategy flag, no remote wired from here.
- **Always brings the environment level with the checkout it finds**, even when the checkout is
  left alone.
- **`start` always, `update` when the stamp says so** or the user asks.
- **The stamp is written after a clean `update` and never otherwise**, and only here.
- **A failure is a sentence, not a stack trace**: what failed, what it means, what to do.
- **Never adds the `.gitignore` entry**; names `codefall init`.
- **Runs only what is declared**, from the project root, as written.
