# Landing what a verb wrote

The shared procedure for putting a verb's files into git: a branch of their own, a commit of those
files and nothing else, a push, and a pull request. Every verb that writes to the repository follows
it — `codefall-envision`, `codefall-scaffold`, `codefall-specify`, `codefall-report`,
`codefall-mock-up`, `codefall-design`, `codefall-equip`, `codefall-upgrade` — except
`codefall-implement`, which lands one branch and one pull request per task on its own terms, and
`codefall-review`, whose fixes land on the branch under review.

No verb merges to the default branch, and the guard hook denies the attempt. Everything short of
that — branching, committing, pushing, opening the pull request — is the verb's to do, and a
document never sits uncommitted on the default branch.

**Two kinds of pull request leave this procedure.**

- **A document pull request** comes from `codefall-envision`, `codefall-specify` (its mockups
  included), `codefall-report`, `codefall-design`, and `codefall-mock-up` run on its own. It is
  opened as a draft, marked ready for review when the person has signed the document off, and
  merged by a GitHub Action the project installs with `/codefall-equip landing`, which merges a
  ready pull request that touches only document paths. A project without the Action gets the same
  pull request, and a person merges it.
- **A pull request from `codefall-equip`, `codefall-scaffold`, or `codefall-upgrade`** changes
  scripts, settings, or templates, so it is opened ready for review and a person merges it.

Document pull requests do not stack. Each one branches from the default branch and targets it. The
one verb that stacks pull requests is `codefall-implement`, for code; the stacks reference beside
this file (stacks.md) is its.

## Contents

- Before the first file: the branch
- After the last file: the commit
- Then: push and pull request
- Marking a document pull request ready
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
| `design` | `design/DESIGN-NNN-slug` |
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
  `docs(designs): add DESIGN-002 booking history`. Where the project has its own commit convention,
  that wins.
- **A run that wrote beads includes `.beads/interactions.jsonl`** when it changed: the Beads section
  says the log lands in the next commit after a bead write, and this is that commit.
- The commit is the record of what the user already confirmed. Nothing is written without that
  confirmation, so nothing here asks for it again.

## Then: push and pull request

After the commit, push and open the pull request, and say so. The document was confirmed before it
was written, and the pull request is how it reaches the next verb; neither is a question.

```bash
git push -u origin <branch>
gh pr create --base <default-branch> --title "<the commit line>" --body-file <tempfile> --draft   # a document pull request
gh pr create --base <default-branch> --title "<the commit line>" --body-file <tempfile>           # equip, scaffold, upgrade
```

The base is always the default branch. A document pull request is opened as a draft every time,
whatever the document's status; marking it ready is the last step of the run, below, so that the
Action never merges a run that is still writing.

The body says what the document is and its status, and carries `Relates to #<issue>` when the run
created or refreshed a tracker issue, so the issue and the pull request find each other. The title
is the commit line: a squash merge takes it as the commit message.

- **A branch that already has a pull request** (`gh pr view <branch> --json url`) is pushed, and the
  pull request is named rather than opened again.
- **No remote** (`git remote` prints nothing): say so, skip the push, and report the branch. The
  commit is the deliverable on this machine.
- **Never merge, and never push the default branch.**

## Marking a document pull request ready

When every file of the run is committed and pushed and the document's status is `Ready`, mark the
pull request ready for review:

```bash
gh pr ready <number>
```

This is the sign-off reaching GitHub: the person confirmed the document before it was written, and
marking the pull request ready is what tells the Action to merge it. A document whose status is
`Draft` — the person is stopping and coming back, or a design still carries decisions it set
aside — keeps a draft pull request, and the run that later promotes the document marks it ready.

Then read the state once before reporting:

```bash
gh pr view <number> --json state,mergedAt
```

Which of three things to say is decided by what comes back and by whether the project has the
Action, which is the file `.github/workflows/codefall-land-documents.yml`:

| State | Project has the Action | Say |
| --- | --- | --- |
| `MERGED` | either | the pull request is merged |
| `OPEN`, ready | yes | the pull request is ready and the Action is merging it; it takes a minute or two |
| `OPEN`, ready | no | the pull request is ready and waits for a person to merge it; `/codefall-equip landing` installs the Action |
| `OPEN`, draft | either | the pull request is a draft because the document is `Draft`, and what promotes it |

## What to report

The branch, the commit's subject line, the pull request URL, and the pull request's state in one of
the four sentences above. For an `equip`, `scaffold`, or `upgrade` pull request: that it is open
and a person merges it. When there was no remote, that the work is committed locally and where.
