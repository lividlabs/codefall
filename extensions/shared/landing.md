# Landing what a verb wrote

The shared procedure for putting a verb's files into git: a branch of their own, a commit of those
files and nothing else, a push, and a pull request. Every verb that writes to the repository follows
it in full — `codefall-envision`, `codefall-scaffold`, `codefall-specify`, `codefall-report`,
`codefall-mock-up`, `codefall-plan`, `codefall-equip`, `codefall-upgrade` — except
`codefall-implement`, which lands one branch and one pull request per task on its own terms, and
`codefall-review`, whose fixes land on the branch under review.

No verb merges to the default branch, and the guard hook denies the attempt. Everything short of
that — branching, committing, pushing, opening the pull request — is the verb's to do, and a
document never sits uncommitted on the default branch.

**Two kinds of pull request leave this procedure, and a person decides when each one merges.**

- **A document pull request** comes from `codefall-envision`, `codefall-specify` (its mockups
  included), `codefall-report`, `codefall-plan`, and `codefall-mock-up` run on its own. It is
  opened as an ordinary pull request, and a GitHub Action the project installs with `/codefall-equip
  landing` merges it when a person adds the `auto-merge` label to it or approves it, provided it
  touches only document paths. No verb adds the label or approves. A project without the Action gets
  the same pull request, and a person merges it by hand.
- **A pull request from `codefall-equip`, `codefall-scaffold`, or `codefall-upgrade`** changes
  scripts, settings, or templates, so a person merges it; the Action never touches it.

Document pull requests do not stack. Each one branches from the default branch and targets it. The
one verb that stacks pull requests is `codefall-implement`, for code; the stacks reference beside
this file (stacks.md) is its.

## Contents

- Before the first file: the branch
- After the last file: the commit
- Then: push and pull request
- A draft pull request for a `Draft` document
- What to report

## Before the first file: the branch

Read the checkout before writing anything — `branch` and `dirty` from the preflight output, or
`git branch --show-current` and `git status --short`. The default branch is
`git symbolic-ref --short refs/remotes/origin/HEAD` without the `origin/` prefix, else `main`.

| Checkout | Do |
| --- | --- |
| On the default branch | `git switch -c <branch>` and say so — it is not a question |
| On another branch | Ask once: write onto this branch, or branch from the default branch? |
| Detached HEAD | Stop. Say where the checkout is and ask for a branch to work on |
| Run from another verb | No branch of your own; the files go on the calling verb's branch and in its commit |

Branch names carry the verb and the identifier:

| Verb | Branch |
| --- | --- |
| `envision` | `vision/VISION-NNN-slug` |
| `specify` | `spec/SPEC-NNN-slug` |
| `report` | `bug/BUG-NNN-slug` |
| `mock-up` | `mockup/<slug>` |
| `plan` | `plan/PLAN-NNN-slug` |
| `scaffold` | `scaffold/<project-or-surface>` |
| `equip` | `equip/local`, `equip/test-harness`, `equip/agents`, or `equip/landing` |
| `upgrade` | `upgrade/<YYYY-MM-DD>` |

A dirty tree does not stop the branch: `git switch -c` carries uncommitted changes along untouched.
It decides what the commit holds, which is only the files this run wrote.

## After the last file: the commit

Once every file of the run is written — the document, an `AGENTS.md` the directory was missing, the
back-link in another document — commit by path:

```bash
git add <every file this run wrote or edited>
git commit -m "<type>(<scope>): <what>"
```

- **By path, never `git add -A` or `git commit -a`.** Anything else in the tree is the user's.
- **A Conventional Commit line naming the document** — `docs(specs): add SPEC-003 trip export`,
  `docs(plans): add PLAN-002 booking history`. Where the project has its own commit convention,
  that wins.
- **A run that wrote beads includes `.beads/interactions.jsonl`** when it changed: the Beads section
  says the log lands in the next commit after a bead write, and this is that commit.
- The commit is the record of what the user already confirmed. Nothing is written without that
  confirmation, so nothing here asks for it again.

## Then: push and pull request

After the commit, push, open the pull request, and say so. This holds for every verb that follows
this procedure, `codefall-equip`, `codefall-scaffold`, and `codefall-upgrade` included. The files
were confirmed before they were written, and the pull request is how they reach the next verb or
the person who merges them; neither the push nor the pull request is a question.

```bash
git push -u origin <branch>
gh pr create --base <default-branch> --title "<the commit line>" --body-file <tempfile>
```

The base is always the default branch. The pull request is an ordinary one, not a draft, with one
exception: a document whose status is `Draft` gets `--draft`, as the next section says.

The body says what the document is and its status, and carries `Relates to #<issue>` when the run
created or refreshed a tracker issue, so the issue and the pull request find each other. The title
is the commit line: a squash merge takes it as the commit message.

- **A branch that already has a pull request** (`gh pr view <branch> --json url`) is pushed, and the
  pull request is named rather than opened again.
- **No remote** (`git remote` prints nothing): say so, skip the push, and report the branch. The
  commit is the deliverable on this machine.
- **Never merge, never push the default branch, and never add the `auto-merge` label.** The label
  is a person's way of saying a document pull request may merge; a verb that added it would be
  merging. An approving review is the other way, and GitHub does not let the pull request's author
  give one.
- **Changes the person asks for after the pull request is open** are new commits on the same
  branch, pushed the same way.

## A draft pull request for a `Draft` document

A document whose status is `Draft` — the person is stopping and coming back, or a plan still
carries decisions it set aside for an engineer — is opened as a draft pull request:

```bash
gh pr create --base <default-branch> --title "<the commit line>" --body-file <tempfile> --draft
```

The draft state follows the document's status and nothing else. When a later run promotes the
document to `Ready`, that run marks the pull request ready for review with `gh pr ready <number>`
before it pushes the commit, because GitHub does not merge a draft and because the push is what
makes the Action look at the pull request again if the `auto-merge` label is already on it. Marking
the pull request ready is not the signal to merge; the `auto-merge` label or an approving review is,
and a person gives it.

## What to report

The branch, the commit's subject line, and the pull request's URL. Then one sentence saying how it
merges, chosen by the kind of pull request and by whether the project has the Action, which is the
file `.github/workflows/codefall-land-documents.yml`:

| Pull request | Project has the Action | Say |
| --- | --- | --- |
| document, `Ready` | yes | the pull request is open at <url>; add the `auto-merge` label when you want it merged, or have someone approve it; either one merges it |
| document, `Ready` | no | the pull request is open at <url> and waits for a person to merge it; `/codefall-equip landing` installs the Action that merges a labelled or approved document pull request |
| document, `Draft` | either | the pull request is a draft at <url> because the document is `Draft`, and what promotes it |
| `equip`, `scaffold`, or `upgrade` | either | the pull request is open at <url>, and a person merges it because it changes code or settings |

Then the one next command the verb's own report names. When there was no remote, say that the work
is committed locally and where.
