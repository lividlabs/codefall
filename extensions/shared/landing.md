# Landing what a verb wrote

The shared procedure for putting a verb's files into git: a branch of their own, a commit of those
files and nothing else, and an offered pull request. Every verb that writes to the repository
follows it — `codefall-envision`, `codefall-scaffold`, `codefall-specify`, `codefall-report`,
`codefall-mock-up`, `codefall-design`, `codefall-equip`, `codefall-upgrade` — except
`codefall-implement`, which lands one branch and one pull request per task on its own terms, and
`codefall-review`, whose fixes land on the branch under review.

A human performs every merge to the default branch, and the guard hook denies the alternative.
Everything short of that — branching, committing, pushing, opening the pull request — is the verb's
to do or to offer, and a document never sits uncommitted on the default branch.

## Contents

- Before the first file: the branch
- After the last file: the commit
- Then: push and pull request, one offer
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

Branch names carry the verb and the identifier:

| Verb | Branch |
| --- | --- |
| `envision` | `vision/VISION-NNN-slug` |
| `specify` | `spec/SPEC-NNN-slug` |
| `report` | `bug/BUG-NNN-slug` |
| `mock-up` | `mockup/<slug>` |
| `design` | `design/DESIGN-NNN-slug` |
| `scaffold` | `scaffold/<project-or-surface>` |
| `equip` | `equip/local`, `equip/test-harness`, or `equip/agents` |
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
  `docs(designs): add DESIGN-002 booking history`. Where the project has its own commit convention,
  that wins.
- **A run that wrote beads includes `.beads/interactions.jsonl`** when it changed: the Beads section
  says the log lands in the next commit after a bead write, and this is that commit.
- The commit is the record of what the user already confirmed. Nothing is written without that
  confirmation, so nothing here asks for it again.

## Then: push and pull request, one offer

One question, after the commit:

> SPEC-003 is committed on `spec/SPEC-003-trip-export`. Push it and open a pull request?

On yes:

```bash
git push -u origin <branch>
gh pr create --base <default-branch> --title "<the commit line>" --body-file <tempfile>
```

The body says what the document is and its status, and carries `Relates to #<issue>` when the run
created or refreshed a tracker issue, so the issue and the pull request find each other. The title
is the commit line: a squash merge takes it as the commit message.

- **A branch that already has a pull request** (`gh pr view <branch> --json url`) is pushed, and the
  pull request is named rather than opened again.
- **No remote** (`git remote` prints nothing): say so, skip the push, and report the branch. The
  commit is the deliverable on this machine.
- **On no**: report the branch and stop. The commit is there when they want it.
- **Never merge, and never push the default branch.** Report the merge as the user's, as
  `codefall-implement` does.

## What to report

The branch, the commit's subject line, the pull request URL when one was opened, and — when the
user declined the push — that the work is committed locally and where.
