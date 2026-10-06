# Stacked pull requests on GitHub

Shared reference. Read before opening a pull request whose base is another pull request's branch,
before merging or changing one, and before a report points a person at one. Everything here is
from GitHub's own documentation of the feature, except the last section, which is codefall's one
rule about stacks: how a report points at one. One verb stacks pull requests: `codefall-implement`,
whose task pull requests form one stack per epic, and whose landing reference says how it links
them; `codefall-review` and `codefall-test` point at that stack when their run on an epic's work
ends with the merge. Document pull requests do not stack; each targets the default branch, per the
landing procedure beside this file.

**Written 2026-10-05 against the public preview.** GitHub shipped stacked pull requests on
2026-07-30 and still marks them "subject to change". A model's training may predate the feature,
so this file, not memory, is the source.

## Contents

- What a stack is
- Creating and growing one
- What a reviewer sees
- Merging
- Changing a lower layer
- What is not supported
- Pointing at a stack
- Sources

## What a stack is

"Two or more pull requests in the same repository" where "the first, or bottom, pull request
targets the stack's trunk — usually your repository's default branch", and each pull request above
targets the branch of the one below it. GitHub links them, shows a stack map on every member, and
treats them as one unit for merging. One pull request is not a stack; the stack exists from the
second.

The trunk is `main` unless a stack is deliberately rooted elsewhere, such as a release branch.

## Creating and growing one

The `gh stack` extension does it from the command line:

```bash
gh extension install github/gh-stack
```

**Without local tracking**, which is how a verb that runs in its own session does it:

```bash
gh stack link <bottom-branch-or-pr> <next> [<next>...]   # bottom to top
gh stack link <stack-number> <new-branch-or-pr>          # append to an existing stack
```

`gh stack link` pushes any branch named, creates a pull request for any branch that has none "with
the correct base branch chaining", reuses an open pull request where one exists, creates the stack
when none of the pull requests is in one, and extends the stack when some are. "Existing PRs are
never removed." The stack number is the one GitHub shows in its stack UI; stack numbers and pull
request numbers never overlap.

**With local tracking**, for a person working a stack in one checkout: `gh stack init <branch>`
starts one on the trunk, `gh stack add <branch>` adds a layer on top, `gh stack submit` pushes every
branch and creates or updates the pull requests and the stack. With `--auto` new pull requests are
drafts unless `--open` is passed.

**On the website**: open a pull request whose base is another pull request's branch and choose
**Create stack**; on a member, the stack icon offers **Add to stack**. GitHub also recognises a
chain of open pull requests whose bases line up and shows a banner offering to make them a stack.

A pull request may join a stack only from the same repository; forks cannot.

## What a reviewer sees

"Each pull request in a stack shows only the diff for its layer." A reviewer reads one layer at a
time, in parallel with others, and the stack map shows where each sits. Branch protection "is
enforced on every pull request in the stack, even mid-stack pull requests", and the checks that run
for pull requests against the trunk "run for all pull requests in the stack".

## Merging

- **Merging a layer merges every layer below it.** "You cannot merge a mid-stack pull request in
  isolation, the pull requests below it will always merge with it." A layer can merge once "all pull
  requests below it are approved and have passing checks".
- **Merging the top lands the whole stack.** One merge, one operation, all or nothing: "if any PR
  cannot be merged, none are."
- **GitHub rebases what is above.** "When you merge a pull request at the bottom of the stack, the
  remaining branches are automatically rebased so the next pull request targets the default base
  branch." Nobody rebases by hand after a merge.
- **Merge methods**: merge commit, squash, and rebase are all supported.
- **From the command line**: `gh stack merge [<pr>]` merges the stack up to and including that pull
  request, atomically; `--squash`, `--merge`, `--rebase` pick the method, `--yes` skips the prompt.
  With no argument it merges the current branch's whole stack.
- **Merge queue**: "Stacks fully support merge queues. All pull requests in the stack are added to
  the queue in the correct order." With a queue on the trunk, `gh stack merge` adds the stack to
  the queue instead of merging directly.

## Changing a lower layer

Fix on "the branch that owns the change", then `gh stack rebase` to cascade the change through
the layers above, then `gh stack push`, which pushes each branch with `--force-with-lease`. "Each
pull request above the changed one reflects the update, and any CI checks are re-triggered."
`gh stack sync` does the fetch, rebase, push, and pull-request update in one command. The cascade
is the extension's job, not a hand-run `git rebase`.

## What is not supported

- **Auto-merge.** "Auto-merge is not supported for stacked pull requests." The auto-merge checkbox
  and `gh pr merge --auto` do nothing useful on a stack member. A merge queue is the supported way
  to land layers without a click each.
- **Cross-fork stacks.**
- **Merging a mid-stack layer alone.**

## Pointing at a stack

Codefall's rule, the one in this file. A report whose next step is the merge of a stack gives the
link to the **top pull request** — the most recent one, the one no other open pull request bases
on — and says that merging it on GitHub lands the whole stack. GitHub's stack view merges every
layer from that pull request, atomically, per *Merging* above. That is the whole instruction:

> The work is in a stack of five pull requests. Merging the top one, <url>, lands all five.

Never a `gh` command, and never a list of pull requests to merge in order from the bottom. Both
work on GitHub and both are the wrong instruction: the command adds nothing the link does not, and
the bottom-up list turns one click into five. The rule holds for every persona; an engineer gets
the same link. A single pull request — a one-task run, or an epic branch's aggregate pull request —
gets the same sentence with its one link.

## Sources

- [Stacked pull requests are now in public preview](https://github.blog/changelog/2026-07-30-stacked-pull-requests-are-now-in-public-preview/)
- [About stacked pull requests](https://docs.github.com/en/pull-requests/get-started/about-stacked-prs)
- [Creating stacked pull requests](https://docs.github.com/en/pull-requests/how-tos/create-pull-requests/creating-stacked-pull-requests)
- [Reviewing stacked pull requests](https://docs.github.com/en/pull-requests/how-tos/review-pull-requests/reviewing-stacked-pull-requests)
- [Merging stacked pull requests](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/merging-stacked-pull-requests)
- [Stacked pull requests CLI commands](https://docs.github.com/en/pull-requests/reference/stacked-prs-cli-commands)
- [Stacked pull requests APIs and webhooks](https://docs.github.com/en/pull-requests/reference/stacked-pull-requests-apis-and-webhooks)
