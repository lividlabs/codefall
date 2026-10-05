# How codefall lands documents

A page for a repository owner. It explains what happens to a document a codefall verb writes, from
the first file to the default branch, and names the one repository setting that is yours to judge.

## What happens

Five verbs write documents: `envision` (`docs/visions/`), `specify` (`docs/specs/`, with the
mockups it makes under `docs/mockups/`), `report` (`docs/bugs/`), `design` (`docs/designs/` and
`docs/adrs/`), and `mock-up` run on its own (`docs/mockups/`). Each one does the same thing:

1. branches from the default branch;
2. writes the document and works with you until you sign it off;
3. commits the files it wrote, pushes, and opens a pull request **as a draft**;
4. marks the pull request **ready for review** when the document's status is `Ready`.

That last step is your sign-off reaching GitHub. A GitHub Action in your repository listens for it
and merges the pull request when every path in its diff is a document path. You are never asked to
merge a document. The verb's report says the pull request is merged, or that it is ready and the
Action is merging it, and ends with the one command that comes next.

A document you left as `Draft` — you stopped and are coming back, or a design still carries
technical decisions you chose to leave for an engineer — keeps a draft pull request. It blocks
nothing, and the run that later promotes the document marks the pull request ready.

Document pull requests do not stack. Each one branches from the default branch and lands on its
own. Code pull requests are different: `implement` opens one per task as a GitHub stack, and a
person merges those.

## The Action

`/codefall-equip landing` installs it as `.github/workflows/codefall-land-documents.yml`, from a
template the extension ships. The file does three things:

**It fires when a pull request is marked ready for review**, and again on every push to a pull
request that is already ready, so a late commit is checked again. A draft never fires it.

**It checks the paths.** The pull request's diff against its base must contain only paths under:

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

One path outside the list, and the job fails with the paths named; the pull request waits for a
person. This is a list of paths, never a reading of content, which is what keeps the check
predictable.

**It merges** with `gh pr merge --squash` under the default `GITHUB_TOKEN`. The squash takes the
pull request's title as the commit message, which is the verb's Conventional Commit line.

The template is `extensions/skills/codefall-equip/templates/codefall-land-documents.yml` in the
codefall repository. Equip writes it unchanged, shows it to you first, and never overwrites a copy
you have edited.

## The one setting that is yours

Branch protection and rulesets apply to the Action's merge like any other. If your default branch
requires a pull request review, the default token cannot merge, and the Action fails at its last
step until you let it through:

- in a **ruleset**, add a bypass for the `github-actions` app, or for repository admins, scoped to
  pull requests;
- in classic **branch protection**, add the actor to "allow specified actors to bypass required
  pull requests".

Equip reads the rule with `gh api` and tells you which of these applies. It changes no repository
setting itself. The trade is yours to judge: a document lands on the default branch with your
sign-off inside the session as its only review. Code pull requests are untouched by this; they
still need whatever your rule requires.

A repository with no rule on its default branch needs nothing.

## Without the Action

A project that has not run `/codefall-equip landing` gets the same pull requests. Each verb's report
says the pull request is ready and waits for a person to merge it, and names `/codefall-equip
landing` as the way to stop being asked.

## What a report looks like

> SPEC-006 is written and `Ready`. Pull request #1152 is ready for review and the Action is merging
> it; it takes a minute or two. Next: run `/codefall-design SPEC-006`.

> DESIGN-003 is `Draft`: you left two technical decisions for an engineer. Pull request #1158 stays
> a draft until they are settled. Next: have an engineer run `/codefall-design DESIGN-003`.

## Related

- [ADR-014](adrs/ADR-014-upstream-verbs-and-document-landing.md) — the decision.
- `extensions/shared/landing.md` — the procedure every verb follows, as installed into a project at
  `.codefall/shared/landing.md`.
- `extensions/skills/codefall-equip/reference/landing.md` — equip's landing track in full.
