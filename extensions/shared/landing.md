# Landing what a verb wrote

The shared procedure for putting a verb's files into git: a branch of their own, a commit of those
files and nothing else, and an offered pull request. Every verb that writes to the repository
follows it — `codefall-envision`, `codefall-scaffold`, `codefall-specify`, `codefall-report`,
`codefall-mock-up`, `codefall-design`, `codefall-equip`, `codefall-upgrade` — except
`codefall-implement`, which lands one branch and one pull request per task on its own terms, and
`codefall-review`, whose fixes land on the branch under review.

A human performs every merge to the default branch, and the guard hook denies the alternative.
Everything short of that — branching, committing, pushing, opening the pull request — is the verb's
to do, and a document never sits uncommitted on the default branch.

**The documents of one delivery stack.** The spec's pull request is the bottom layer, the mockups
stack on it, the design on top, and nobody is asked to merge until the design is done: merging the
design pull request lands the whole stack. The stacks reference beside this file (stacks.md) says
how GitHub stacks work; this file says where each verb's layer goes.

## Contents

- The shared customization
- Before the first file: the branch
- Stacking on the layer below
- After the last file: the commit
- Then: push and pull request
- What to report

## The shared customization

Landing is the one behaviour every document verb shares, so its customization is shared too:
`.codefall/skills/shared/CUSTOMIZE.md` in the user's project, read before the branch step by every
verb that follows this file, beside the verb's own `CUSTOMIZE.md`. It is where a repository that
lands its document stacks itself says so — for example, that the design verb marks the top pull
request ready for review and stops, because an Action merges the stack. The customizations rule
holds: it extends this procedure and never relaxes it.

## Before the first file: the branch

Read the checkout before writing anything — `branch` and `dirty` from the preflight output, or
`git branch --show-current` and `git status --short`. The default branch is
`git symbolic-ref --short refs/remotes/origin/HEAD` without the `origin/` prefix, else `main`.

| Checkout | Do |
| --- | --- |
| The layer below has an open pull request | `git fetch origin` and `git switch -c <branch> origin/<layer-branch>`, per *Stacking on the layer below*, and say so |
| On the default branch, no layer below | `git switch -c <branch>` and say so — it is not a question |
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
| `equip` | `equip/local`, `equip/test-harness`, or `equip/agents` |
| `upgrade` | `upgrade/<YYYY-MM-DD>` |

A dirty tree does not stop the branch: `git switch -c` carries uncommitted changes along untouched.
It decides what the commit holds, which is only the files this run wrote.

## Stacking on the layer below

The layer below is the open pull request of the document this one builds on:

| Verb | Layer below |
| --- | --- |
| `specify` | the vision's, `vision/VISION-NNN-*`, when the spec draws on one |
| `mock-up` | the spec's, `spec/SPEC-NNN-*`; else the vision's |
| `design` | the mockups', `mockup/*`, when a mockup run landed its own layer; else the spec's; else the bug report's, `bug/BUG-NNN-*` |

Find it with `gh pr list --state open --json number,headRefName,baseRefName --limit 200` and the
branch prefix. A pull request that has merged is no layer: branch from the default branch as the
table above says. A document with no upstream — a vision, a bug report, a spec with no vision —
starts a stack of its own at the default branch.

Branching from `origin/<layer-branch>` is what makes the diff of this layer show only this
document. The pull request below is never edited, rebased, or force-pushed by this run.

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
gh pr create --base <layer-branch-or-default> --title "<the commit line>" --body-file <tempfile> [--draft]
gh stack link <layer-pr-or-stack-number> <branch>     # when there is a layer below
```

The base is the layer below's branch when there is one, else the default branch. `--draft` when the
document's status is `Draft`; `gh pr ready` when a later run promotes it. With a layer below,
`gh stack link` adds this pull request to the stack: the layer's pull request number when the stack
does not exist yet (two pull requests make one), the stack number GitHub shows when it does. The
`gh-stack` extension is `gh extension install github/gh-stack`; a machine without it gets the pull
request and a line in the report saying the stack was not linked and how to.

The body says what the document is and its status, and carries `Relates to #<issue>` when the run
created or refreshed a tracker issue, so the issue and the pull request find each other. The title
is the commit line: a squash merge takes it as the commit message.

- **A branch that already has a pull request** (`gh pr view <branch> --json url`) is pushed, and the
  pull request is named rather than opened again.
- **No remote** (`git remote` prints nothing): say so, skip the push, and report the branch. The
  commit is the deliverable on this machine.
- **Never merge, and never push the default branch.** The merge is the person's, at the top of the
  stack, when the design is done — unless the shared customization says the repository lands the
  stack itself.

## What to report

The branch, the commit's subject line, the pull request URL, and its place in the stack: which layer
it is, and that nothing merges until the design pull request is done. The design verb's report says
instead that merging its pull request lands the stack. When there was no remote, that the work is
committed locally and where.
