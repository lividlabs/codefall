# Landing a document stack without a merge click

A recipe for a repository owner. Codefall does not ship any of it, and nothing in the skills
assumes it exists.

## What it does

A delivery's documents form one stack of pull requests: the spec at the bottom, the mockups on it,
the design on top. By default a person merges that stack once, at the design pull request, when the
design is done. This recipe removes that click. When the design pull request is marked ready for
review, a GitHub Action checks that the stack touches only document paths and merges it. The person
approved each document inside the session; the merge is the consequence.

Code pull requests are not touched. They stack too, and a person merges them.

GitHub's own auto-merge does not work here: "Auto-merge is not supported for stacked pull
requests." The Action does what auto-merge would have done, through the stack merge API.

## The pieces

**1. The trigger.** GitHub fires a `pull_request` event with the type `ready_for_review` when a
draft is marked ready. The Action listens for that event on the design pull request. Marking a lower
layer ready does nothing, because the stack merges from the top.

**2. The path check.** The Action diffs the stack from the trunk to the design branch,
`git diff --name-only origin/main...HEAD`, and fails unless every path is under one of:

```
docs/visions/
docs/specs/
docs/bugs/
docs/mockups/
docs/designs/
docs/adrs/
.codefall/reviews/
.codefall/tests/
.beads/interactions.jsonl
```

A path outside the list means code or configuration is in the stack, and the stack waits for a
person. The list is an allowlist of paths, never an inspection of content: that is what keeps the
check deterministic.

**3. The merge.** On a passing check the Action runs the stack merge up to the design pull
request, `gh stack merge <pr> --squash --yes`, with the `gh-stack` extension installed on the
runner. It is one all-or-nothing operation: spec, mockups, and design land together or not at all.

**4. The branch rule.** Branch protection runs on every layer of a stack. If `main` requires a
human review, the Action's token cannot merge. The rule has to let this Action through for pull
requests that pass the path check: a ruleset with a bypass for the GitHub App or token the Action
runs as, scoped to that check. The default `GITHUB_TOKEN` cannot bypass a required review, so the
Action needs an App installation token or a fine-grained token stored as a secret. This is the one
part of the recipe that is a repository setting rather than a file, and the owner decides whether
the trade is acceptable: documents land on `main` with the in-session approval as the only review.

**5. The customization.** One line in `.codefall/skills/shared/CUSTOMIZE.md`, the landing
customization every document verb reads, tells the design verb to mark the top pull request ready
for review and stop, because the stack lands itself. Without it the verb leaves the merge to a
person, as it does everywhere else.

## A starting point for the workflow

Not a finished Action; the owner's agent fills it in for the repository.

```yaml
name: land-document-stack
on:
  pull_request:
    types: [ready_for_review]
permissions:
  contents: write
  pull-requests: write
jobs:
  land:
    if: startsWith(github.head_ref, 'design/')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - name: Only document paths
        run: |
          set -euo pipefail
          allow='^(docs/(visions|specs|bugs|mockups|designs|adrs)/|\.codefall/(reviews|tests)/|\.beads/interactions\.jsonl$)'
          git diff --name-only "origin/${{ github.event.repository.default_branch }}...HEAD" \
            | grep -vE "$allow" && { echo "non-document paths in the stack"; exit 1; } || true
      - name: Merge the stack up to this pull request
        env:
          GH_TOKEN: ${{ secrets.DOCS_STACK_TOKEN }}
        run: |
          gh extension install github/gh-stack
          gh stack merge "${{ github.event.pull_request.number }}" --squash --yes
```

The branch-name condition assumes codefall's `design/DESIGN-NNN-slug` branches; a repository that
names them differently changes it. The token in `DOCS_STACK_TOKEN` is the one the branch rule lets
through.

## A prompt for the owner's agent

Paste this into your own coding agent in the repository, and answer its questions:

> Set this repository up so a stack of document pull requests lands itself when its top pull
> request is marked ready for review. Read `docs/recipes/landing-document-stacks.md` in the codefall
> repository first; it has the pieces and a starting workflow. Then: (1) add the workflow under
> `.github/workflows/`, with the path allowlist exactly as the recipe lists it and the branch-name
> condition matching how this repository names design branches; (2) tell me what the current branch
> rule on the default branch requires, and propose the smallest ruleset change that lets this
> workflow's token merge a pull request that passed the path check, naming the bypass actor and what
> it can and cannot do; (3) tell me which token or GitHub App to create for `DOCS_STACK_TOKEN`, with
> the minimum permissions, and where to store it; (4) add one line to
> `.codefall/skills/shared/CUSTOMIZE.md` so the design verb marks the top pull request ready for
> review and stops. Ask me before changing any repository setting, and do not merge anything
> yourself.

## Later: `equip` could do this

`codefall-equip` builds what the other verbs need a project to have, one track per run, each as its
own pull request. This recipe is the shape of a fourth track: the workflow file and the
customization line are ordinary files it can write, and the ruleset change is a `gh api` call it can
make with the owner confirming and holding admin rights. It is not built, because the ruleset bypass
is a repository decision the owner should make in the open, and the technique should prove itself by
hand first.
