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

- `reference/failures.md` — the failures `start` and `update` produce most often, what each means,
  and the sentence to say. Read at step 7 when a command exits non-zero.

## Scope — run, never write

| In scope | Out of scope | Whose |
| --- | --- | --- |
| Switching the primary checkout to the default branch, and fast-forwarding it | Rebasing, merging, or resetting any branch | the user |
| Removing the worktree it was run from, and deleting a local branch, once its work is merged | Any other worktree, and any remote branch | `codefall-implement`'s cleanup offer, the user |
| Syncing the beads with their Dolt remote | Wiring the remote, or resolving a conflict the sync halts on | the user |
| Running the declared `start` and `update` | Drafting, editing, or declaring them | `codefall-equip` |
| Recording the stamp after a clean `update` | Any other file in the project | — |
| Saying what failed and what to do | Fixing the script that failed | `codefall-equip` |
| Reporting uncommitted work, a stash, a diverged default branch | Touching any of them | the user |

## Where refresh takes the checkout

Every run ends in the primary checkout, on the default branch. The primary checkout is the first
`worktree` entry of `git worktree list --porcelain`; a linked worktree is any other.

| Found | Action |
| --- | --- |
| Run from a linked worktree | Work moves to the primary checkout; the worktree is handled at step 4 |
| Primary checkout has uncommitted changes | **Stop**: list the files and say to commit or stash them, then rerun. Nothing moves and nothing runs |
| Primary checkout on another branch | `git switch <default>`; the branch is handled at step 4 |
| Primary checkout on a detached HEAD | `git switch <default>` when a branch or remote ref contains `HEAD`; otherwise **stop** and name the hash, since switching would strand it |
| `git switch` refused, such as the default branch held by another worktree | **Stop**: quote git's line and name the worktree that holds it |
| On the default branch, `origin/<default>` is a fast-forward | `git pull --ff-only origin <default>` |
| On the default branch, diverged | Not pulled: say how many local commits the remote does not have; carry on |
| No remote, or the fetch failed | Not pulled: say so; carry on |

A pull that is refused is reported, never retried with a merge, a rebase, or a reset.

## Merged

A branch is merged when `origin/<default>` holds its work, by either test:

```bash
git merge-base --is-ancestor <branch> origin/<default>        # merged or fast-forwarded
base=$(git merge-base origin/<default> <branch>)
squash=$(git commit-tree "<branch>^{tree}" -p "$base" -m squash)
git cherry origin/<default> "$squash"                          # "-" first: squash-merged
```

The second test builds the branch as one commit and asks whether the default branch already carries
an equivalent patch; a line starting with `-` is a yes, `+` is a no. A squash that was edited at
merge time reads as a no, and the branch is kept.

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
"../../../.codefall/shared/preflight.sh" <primary checkout>
```

Note `branch`, `dirty`, `fetch`, `default_branch`, `behind`, `ahead`, `refresh`, and `beads`.

### 3. Go to the default branch

Apply [the table](#where-refresh-takes-the-checkout), in the primary checkout. Every command from
here runs from the primary checkout's root: `cd` there, so the session stays there when the
harness keeps a directory between commands.

### 4. Clean up what was left

Test each with [Merged](#merged), against the `origin/<default>` step 2 fetched.

- **The branch the primary checkout left.** Merged: `git branch -D <branch>`, and note its old
  tip. Not merged: keep it, and note how many of its commits are not on its upstream, or that it
  has none.
- **The worktree the run started in.** Merged and clean: `git worktree remove <path>`, then
  `git branch -D <branch>`, and note the old tip. Not merged, uncommitted work, or a detached HEAD:
  keep it, and note which and the path.
- **A merged branch still on the remote.** Note it, with `git push origin --delete <branch>` as the
  user's to run.

Never pass `--force` to `git worktree remove`, and never touch a worktree the run did not start in.

### 5. Sync the beads

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

### 6. Bring the environment level

Run the declared `start`, always: it is idempotent and cheap when everything is up, and `update`
may assume it ran.

Then run the declared `update` when the stamp does not match `HEAD`, or when the user asked for
it regardless. A stamp that matches is the skip: say the environment was already current at
`<short hash>` and do not run `update`.

Run both from the primary checkout's root, as declared, through the shell. Show the output as it
arrives when it is short; summarize it when it is long, and keep the last twenty lines of stderr
for step 7.

### 7. Say what happened

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

### 8. Report

One block, short:

- Where the user is now: the primary checkout's path, on the default branch at `<short>`, and
  whether it moved from a branch, from `<short>`, or was not pulled and why.
- What was left behind: each branch and worktree from step 4, removed with its old tip or kept and
  why, and a merged branch still on the remote.
- The beads: synced, no remote, skipped because beads is blocked, or halted and why.
- The environment: `start` ran; `update` ran or was skipped as current.
- The stamp: written at `<short>`, or not, and why.
- **Last, what the user does next**: nothing, or the commit or stash a stop asked for, the
  `codefall upgrade` for `.gitignore`, the `/codefall-equip` for a broken script. When the run
  started in a worktree and the harness does not keep a directory between commands, the next step
  is to open the next session in the primary checkout.

## Rules

- **Every run ends on the default branch in the primary checkout**, or stops before anything
  moves and says why.
- **Uncommitted work in the primary checkout stops the run.** Never stashes, never commits.
- **Moves the checkout with `git switch` and `git pull --ff-only` only.** Never merges, never
  rebases, never resets.
- **Deletes a local branch or removes a worktree only when it is merged**, only the one the run
  left, and never with `--force`. Never deletes a remote branch.
- **Syncs the beads with `bd sync` and never settles what it halts on.** No force push, no
  strategy flag, no remote wired from here.
- **Never drafts, edits, or declares a script.** A project with nothing declared is sent to
  `codefall-equip`; a script that fails is reported to it.
- **`start` always, `update` when the stamp says so** or the user asks.
- **The stamp is written after a clean `update` and never otherwise**, and only here.
- **A failure is a sentence, not a stack trace**: what failed, what it means, what to do.
- **Never adds the `.gitignore` entry**; names `codefall init`.
- **Runs only what is declared**, from the primary checkout's root, as written.
