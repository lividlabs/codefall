---
name: codefall-refresh
description: Take the user from wherever they are to the default branch, current and ready for the next task — leave a feature branch or an implement worktree for the primary checkout, switch it to the default branch and fast-forward it, clean up the branch and the worktree whose work has merged, sync the beads, run the project's declared start and update commands, and record the commit the environment now matches. The routine before starting new work.
argument-hint: "[path]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Bash
  - ExitWorktree
---

# Refresh

Take the user from wherever they are to the default branch in the primary checkout, current, with
the Beads database and the local environment level with it, and say what happened in plain words.
This is the thing to run before starting new work, and the thing to run instead of pulling by hand:
a git pull moves the code and leaves the team's beads, the database, the dependencies, and the
generated code where they were.

Refreshing is not equipping. This skill runs the `start` and `update` commands the project declared
under `local` in `.codefall/settings.json`; it never drafts or edits them. A project with nothing
declared is sent to `codefall-equip`.

Paths that start with `reference/` or `../` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills directory: it names
a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

- `reference/failures.md` — the failures refresh's commands produce most often, what each means,
  and the sentence to say. Read when `git`, `start`, or `update` exits non-zero in the primary
  checkout.

## Scope — run, never write

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Switching the primary checkout to the default branch, and fast-forwarding it | Rebasing, merging, or resetting any branch | the user |
| Leaving the worktree the run started in, and removing it and deleting a local branch once its work is merged | Any other worktree, and any remote branch | `codefall-implement`'s cleanup offer, the user |
| Syncing the beads with their Dolt remote | Wiring the remote, or resolving a conflict the sync halts on | the user |
| Running the declared `start` and `update` | Drafting, editing, or declaring them | `codefall-equip` |
| Recording the stamp after a clean `update` | Any other file in the project | — |
| Saying what failed and what to do | Fixing the script that failed | `codefall-equip` |
| Reporting uncommitted work, a stash, a diverged default branch | Touching any of them | the user |

## Where refresh takes the checkout

Every run ends in the primary checkout, on the default branch. The primary checkout is the first
`worktree` entry of `git worktree list --porcelain`, written `<primary>` below; a linked worktree is
any other.

| Found | Action |
| --- | --- |
| Run from a linked worktree | Leave it at step 3; the rest runs against `<primary>` |
| `<primary>` has uncommitted changes | **Stop**: list the files and say to commit or stash them, then rerun. Nothing moves and nothing runs |
| `<primary>` on another branch | `git -C <primary> switch <default>`; the branch is handled at step 5 |
| `<primary>` on a detached HEAD | Switch when a branch or remote ref contains `HEAD`; otherwise **stop** and name the hash, since switching would strand it |
| The switch refused, such as the default branch held by another worktree | **Stop**: quote git's line and name the worktree that holds it |
| On the default branch, `origin/<default>` is a fast-forward | `git -C <primary> pull --ff-only origin <default>` |
| On the default branch, diverged | Not pulled: say how many local commits the remote does not have; carry on |
| No remote, or the fetch failed | Not pulled: say so; carry on |

A pull that is refused is reported, never retried with a merge, a rebase, or a reset.

**Every command names `<primary>`**: `git -C <primary> …`, and `cd <primary> && <command>` as one
command for everything else. A `cd` on its own does not carry into the next command in most
harnesses, and never out of a worktree session.

## Merged

A branch is merged when merging it into `origin/<default>` would change nothing:

```bash
git -C <primary> merge-tree --write-tree origin/<default> <branch>
git -C <primary> rev-parse "origin/<default>^{tree}"
```

Merged when the first exits `0` and its first line equals the second. Anything else — a different
tree, a conflict, a git older than 2.38 that has no `--write-tree` — is not merged, and the branch
is kept.

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

### 2. Find where the user is

```bash
git rev-parse --show-toplevel
git worktree list --porcelain
```

When the top level is not the first entry, the run started in a linked worktree: note its path,
its branch, and `git status --porcelain` there. Then run the shared check against the primary
checkout:

```bash
"../../../.codefall/shared/preflight.sh" <primary>
```

Note `branch`, `dirty`, `fetch`, `default_branch`, `behind`, `ahead`, `refresh`, and `beads`.
`dirty=true` is the stop in [the table](#where-refresh-takes-the-checkout), before step 3.

### 3. Leave the worktree

Only when the run started in one. Test its branch with [Merged](#merged), then, when the harness
has `ExitWorktree`, call it: `remove` when the branch is merged and the worktree has no
uncommitted changes, `keep` otherwise.

| `ExitWorktree` answers | Then |
| --- | --- |
| Done | The session is back where it entered the worktree |
| No worktree session is active | The harness does not manage this one: the session stays, and a merged worktree is removed at step 9 |
| Refuses `remove` over commits not on the original branch | That is the squash-merge. Ask once, citing the merged test; on yes call it with `discard_changes: true`, on no call `keep` |
| Refuses `remove` for a worktree entered by path | Call `keep`; a merged worktree is removed at step 5 |

With no `ExitWorktree`, the session stays in the worktree, and a merged worktree is removed at
step 9.

### 4. Go to the default branch

Apply [the table](#where-refresh-takes-the-checkout) to `<primary>`. A command the harness refuses
— `Operation not permitted`, a permission request rejected — is a stop; `reference/failures.md`
has the row.

### 5. Clean up the branch

Test with [Merged](#merged), against the `origin/<default>` step 2 fetched.

- **The branch `<primary>` left.** Merged: `git -C <primary> branch -D <branch>`, and note its
  old tip. Not merged: keep it, and note how many of its commits are not on its upstream, or that
  it has none.
- **A worktree step 3 kept by path**, merged and clean: `git -C <primary> worktree remove <path>`,
  then delete its branch the same way.
- **A merged branch still on the remote.** Note it, with `git push origin --delete <branch>` as the
  user's to run.

Never pass `--force` to `git worktree remove`, and never touch a worktree the run did not start in.

### 6. Sync the beads

Skip this step when preflight said `beads=blocked`: say so in the report, and leave the remedy to
the verb that needs beads. Otherwise:

```bash
cd <primary> && bd sync
```

One command: pull the team's claims and closes from the Dolt remote, halt on a conflict, repair the
blocked flags the merged edges changed, and push whatever this machine wrote and never published.
Nothing in git moves. A worktree shares the primary checkout's database. Read the exit code:

| Exit | Meaning | Say |
| --- | --- | --- |
| `0`, output says no remote is configured | No Dolt remote is wired | The database is this machine's. `bd dolt push --yes` adopts the git origin as the remote, and is the user's to run |
| `0` | Pulled and pushed | Synced, in the report |
| `2` | A conflict bd could not settle; nothing pushed | Which issues, from bd's output, and that `bd conflicts` resolves it by hand |
| `3` | Another writer won the push race; nothing pushed | Transient: run `/codefall-refresh` again |
| `4` | Uncommitted changes are stuck in the working set; nothing pushed | Quote the line; the user clears it, and nothing here retries |
| `1` | Transport, authentication, or storage | Quote the line; the fix is the environment's |

Never run `bd dolt push --force`, `bd dolt pull --strategy`, or `bd conflicts resolve` from here.

### 7. Bring the environment level

Run the declared `start`, always: it is idempotent and cheap when everything is up, and `update`
may assume it ran.

Then run the declared `update` when the stamp does not match `HEAD`, or when the user asked for
it regardless. A stamp that matches is the skip: say the environment was already current at
`<short hash>` and do not run `update`.

Run each as `cd <primary> && <command>`, as declared, through the shell. Show the output as it
arrives when it is short; summarize it when it is long, and keep the last twenty lines of stderr
for step 8.

### 8. Say what happened

A clean exit from both: write the stamp.

```bash
git -C <primary> rev-parse HEAD > <primary>/.codefall/refresh.stamp
```

Then confirm the stamp is ignored — `git -C <primary> check-ignore -q .codefall/refresh.stamp` —
and, when it is not, say so and name `codefall upgrade` as the fix. Never add the entry from here.

A non-zero exit: read `reference/failures.md`, match the stderr, and say three things — what
failed, what it means, and what to do — in one short paragraph a teammate who does not read
stack traces can act on. Quote the one line of stderr that says it, not the twenty. A failure that
is the environment's, such as Docker not running, ends with "run `/codefall-refresh` again once
that is done". A failure that is the script's — a command not found, a wrong path — names
`/codefall-equip` to fix the script, and this skill does not edit it.

No stamp is written after a failure.

### 9. Remove the worktree the session is in

Only when step 3 left the session inside a worktree whose branch is merged and which has no
uncommitted changes. This is the last command of the run, since every command after it would run
in a directory that is gone:

```bash
git -C <primary> worktree remove <path> && git -C <primary> branch -D <branch>
```

### 10. Report

One block, short:

- Where the user is now: `<primary>`, on the default branch at `<short>`, and whether it moved
  from a branch, from `<short>`, or was not pulled and why.
- What was left behind: each branch and worktree from steps 3, 5, and 9, removed with its old tip
  or kept and why, and a merged branch still on the remote.
- The beads: synced, no remote, skipped because beads is blocked, or halted and why.
- The environment: `start` ran; `update` ran or was skipped as current.
- The stamp: written at `<short>`, or not, and why.
- **Last, what the user does next**: nothing, or the commit or stash a stop asked for, the
  `codefall upgrade` for `.gitignore`, the `/codefall-equip` for a broken script. When the session
  is still in a worktree directory, removed or not, the next step is to open the next session in
  `<primary>`.

## Rules

- **Every run ends on the default branch in the primary checkout**, or stops before anything
  moves and says why.
- **Uncommitted work in the primary checkout stops the run.** Never stashes, never commits.
- **Every command names the primary checkout**, with `git -C` or `cd <primary> &&` in the same
  command. A `cd` on its own is never relied on.
- **Moves the checkout with `git switch` and `git pull --ff-only` only.** Never merges, never
  rebases, never resets.
- **Deletes a local branch or removes a worktree only when it is merged**, only the one the run
  left, and never with `--force`. `discard_changes` goes to `ExitWorktree` only after the merged
  test and the user's yes. Never deletes a remote branch.
- **Syncs the beads with `bd sync` and never settles what it halts on.** No force push, no
  strategy flag, no remote wired from here.
- **Never drafts, edits, or declares a script.** A project with nothing declared is sent to
  `codefall-equip`; a script that fails is reported to it.
- **`start` always, `update` when the stamp says so** or the user asks.
- **The stamp is written after a clean `update` and never otherwise**, and only here.
- **A failure is a sentence, not a stack trace**: what failed, what it means, what to do.
- **Never adds the `.gitignore` entry**; names `codefall init`.
- **Runs only what is declared**, from the primary checkout's root, as written.
