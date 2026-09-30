# codefall-refresh — where it came from

Why `SKILL.md` looks the way it does: what was taken from elsewhere, what was rejected, and what was
tried and dropped. None of this is instruction — the skill is the instruction. This exists so nobody
re-adds something that was removed on purpose. The reasoning behind the contract it runs is
ADR-005 in the repository's `docs/adrs/`.

## Taken

**`git pull --ff-only`** — the one way the default branch is ever moved. A fast-forward cannot lose
anything and cannot produce a conflict, which is what makes it safe to run on a teammate's behalf.

**The squash-merge test from `git-delete-squashed`** — build the branch as one commit on its
merge-base and ask `git cherry` whether the default branch holds an equivalent patch. Every PR here
is squash-merged, so git's own ancestry check never sees a merged branch, and the remote branch is
not always gone, so a `[gone]` upstream is no signal either. The test needs no host API.

**`codefall-implement`'s preflight reading** — the checkout lines come from the same shared script
every verb runs; this skill reads them rather than asking git the same questions a second way.

**Doctor's remedies** — a failure carries what to do next, in the same sentence, or nothing. The
failures table follows that shape: what failed, what it means, what to do.

## Rejected

**Drafting the scripts on the first run.** A verb that runs daily and also does one-time inference
is two verbs, and it puts the drafting interview in front of the teammate least placed to answer
it. `codefall-equip` drafts; this skill sends an unequipped project there.

**Rebasing a feature branch.** A rebase can conflict, and a conflict in a skill that a teammate
runs to get going is the opposite of the point. Refresh leaves the branch instead, and the rebase
is the user's when they go back to it.

**Leaving a feature branch where it is.** The first version moved the checkout only when it was
already on the default branch, and brought the environment level with whatever branch it found.
Run after a PR merged, that left the user on the merged branch, `main` unpulled, the stamp on the
branch's commit, and the branch still there — none of which is what "get me set up for the next
task" means. Refresh now always ends on the default branch, and the branch it leaves loses nothing:
kept when unmerged, deleted only when the default branch holds its work.

**Stopping in a linked worktree.** `main` cannot be checked out in a worktree while the primary
checkout holds it, and the first answer was to stop and say so. That stopped the user in the one
place refresh is most often run from after `codefall-implement`. Refresh works in the primary
checkout instead, and removes the worktree it started in only when that work is merged and clean,
which stays inside implement's rule that cleanup is never automatic for open work.

**Deleting the remote branch.** It is a write to the team's remote, and a repository that deletes
branches on merge never needs it. Refresh names the command and leaves it to the user.

**Stashing a dirty tree to pull.** A stash is state the user did not ask for, in a stack other
sessions may be using. Uncommitted work in the primary checkout stops the run before anything
moves, and running `update` anyway would set the environment to a checkout the user is about to
leave.

**Adding the `.gitignore` entry when the stamp is not ignored.** That is `codefall init`'s line to
write, and a verb that writes it too is a second owner of the same entry. Refresh says so and
names init.

**Editing a script that fails.** The scripts are the project's, revised by `codefall-equip` at the
point a change makes them stale. A refresh that patched a script would put a change nobody
reviewed into a file the whole team runs.

**A hook that pulls at session start.** A hook cannot ask, and a pull moves the working tree under
whatever the session is doing. The `SessionStart` slot can carry a notice; the action stays a
verb someone invokes.

## Why the rules are shaped this way

**`start` always, `update` by the stamp.** `start` is cheap when everything is up and `update`
assumes it ran, so skipping `start` saves nothing and risks `update` reaching a service that is
down. `update` is skipped only on the stamp's say-so, because the stamp is the one record of the
environment having matched this commit.

**The stamp is written after `update` and never otherwise.** Written before, a failed `update`
would leave a record claiming an environment that does not exist, and the next refresh would skip
the work that was needed.

**The environment is brought level with the default branch, after the move.** The next task
starts from the default branch, so that is the checkout whose migrations and lockfile the
environment has to match, and the stamp records its commit. The half a pull leaves undone is the
whole reason the verb exists.
