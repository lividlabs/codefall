# /codefall-equip and /codefall-refresh

`/codefall-equip` sets a project up with what the other skills need it to have. `/codefall-refresh`
brings your checkout and your local environment up to date before you start new work. Run equip once
for each thing a project lacks, and again when its tools change. Run refresh at the start of each
piece of work, in place of `git pull`.

```
/codefall-equip local
/codefall-refresh
```

Codefall's skills are instructions your coding agent follows when you type a slash command; the
[README](../../README.md) introduces them.

## Why isn't `git pull` enough?

Pulling `main` brings in changes that need more work on your machine: a migration the local
database needs, a dependency to install, code to regenerate, or a container to rebuild. Each
arrives as part of a diff. A teammate who does not read every diff has no way to know which of them
apply. Codefall closes that gap with two scripts the project owns, `start` and `update`.
`/codefall-equip` writes them, and `/codefall-refresh` runs them.

## What does /codefall-equip set up?

Equip has four *tracks*, and one run sets up one of them. Name the track after the command. With no
track named, equip says what the project declares today and asks which to set up.

| Track | What it sets up | Details |
| --- | --- | --- |
| `local` | The `start` and `update` scripts | below |
| `test` | A test runner for each part of the project, pointed at the project's test cases | [/codefall-test](test.md) |
| `agents` | Which coding agents review and give second opinions, and how each one is started | [Configuration](configuration.md) |
| `landing` | A GitHub Action that merges document pull requests | [Landing documents](../landing-documents.md) |

Every track works the same way. Equip searches the repository first, shows you what it found, and
asks one question. It shows the full draft or the diff before it writes anything. For the first three
tracks it then declares the result in `.codefall/settings.json`. It runs the result once to prove it
works, and opens a pull request for it on its own branch, such as `equip/local`. When the search leaves something unclear, equip asks
the project's *consult* agents once before it asks you. A consult is a second opinion from the
agents your settings list for that purpose, and its answer appears as the proposed option for you
to accept or change.

You run `/codefall-equip` yourself. An agent never starts it on its own, because it changes the
project's setup.

## What are the start and update scripts?

`start` brings up what the project needs running locally, such as a database. `update` makes the
local environment match the checkout: it installs dependencies, runs migrations, and regenerates
code. Both are declared under `local` in `.codefall/settings.json` as plain shell commands:

```json
"local": {
  "start": "make dev-up",
  "update": "make update"
}
```

Because they are plain commands, a Makefile target or a package script works as well as a script
of the project's own, and anyone can run them from a terminal.

Both scripts must be safe to run at any time. Running them a second time is cheap and changes
nothing, and neither one ever drops, resets, or deletes data. Equip will not declare a script that
breaks those rules, and it will not write one.

In an existing project, the scripts usually exist under another name. Equip finds them and asks:

> I found `make dev-up`, which starts Postgres and Redis through compose, and `npm run db:migrate`.
> Are these the start and update commands, or should I draft new ones?

When you choose a draft, equip writes two targets in the project's task runner, or
`scripts/local.sh` when the project has none. In a new project, `/codefall-scaffold` writes the
scripts as part of setting the project up.

## What does the landing track install?

The landing track writes `.github/workflows/codefall-land-documents.yml` and creates an `auto-merge`
label in the repository. Once that pull request is merged, a document pull request merges itself
when you add the `auto-merge` label to it or a colleague approves it. Equip also tells you what your
branch rule has to allow. It never changes a branch rule itself.
[Landing documents](../landing-documents.md) explains the workflow in full.

## What does /codefall-refresh do?

Refresh takes you from wherever you are to the default branch, up to date, with the beads and the
local environment level with it. A *bead* is one task in
[Beads](https://github.com/gastownhall/beads), the task tracker Codefall keeps in your git
repository. In order, refresh:

1. Leaves the worktree it started in, if any, for the primary checkout. A worktree is a separate
   working copy of the repository; `/codefall-implement` builds each task in one.
2. Switches the primary checkout to the default branch, and fast-forwards it when that is safe.
3. Deletes the local branch and worktree it left, but only when their work has been merged.
4. Syncs the Beads database with its Dolt remote by running `bd sync`, so the task graph it reads is
   the team's.
5. Runs `start`, every time.
6. Runs `update` when the commit has moved since the last clean run.
7. Records that commit in `.codefall/refresh.stamp`, so the next session on the same commit skips
   `update`.

The stamp is per machine, and `codefall init` adds it to `.gitignore`.

Refresh moves the checkout only with `git switch` and `git pull --ff-only`. It never merges,
rebases, or resets a branch. When the primary checkout has uncommitted changes, refresh lists the
files, asks you to commit or stash them, and stops before anything moves or runs. It never stashes
or commits for you. When the default branch has diverged from the remote, refresh says so and
carries on without pulling.

When something fails, refresh tells you in one short paragraph what failed, what it means, and what
to do. When the cause is your environment, such as Docker not running, it asks you to run
`/codefall-refresh` again once that is fixed. When the cause is a script, it names
`/codefall-equip`, because refresh never edits a script. A project with no scripts declared is sent
to `/codefall-equip local`.

## How do the scripts stay current?

A change that adds infrastructure, a dependency, a migration, or generated code updates the scripts
in the same pull request. Each skill plays a part:

- `/codefall-plan` names the script change in the task.
- `/codefall-implement` counts it as part of finishing the task.
- `/codefall-review` has a `local` lens that checks for it.

`codefall init` writes this rule into the project's `AGENTS.md`. `codefall doctor` checks that the
scripts are declared and that the stamp is git-ignored. Every skill that reads the repository tells
you when `main` has moved or the environment is out of date. The notice at the start of each
session reports the same two things, so you hear about a moved `main` before it causes a merge
conflict.
