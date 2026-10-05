# How codefall lands documents

A page for a repository owner. It explains what happens to a document a codefall verb writes, from
the first file to the default branch, and names the one repository setting that is yours to judge.

## What happens

Five verbs write documents: `envision` (`docs/visions/`), `specify` (`docs/specs/`, with the
mockups it makes under `docs/mockups/`), `report` (`docs/bugs/`), `design` (`docs/designs/` and
`docs/adrs/`), and `mock-up` run on its own (`docs/mockups/`). Each one does the same thing:

1. branches from the default branch;
2. writes the document and works with you until you sign it off;
3. commits the files it wrote, pushes, and opens an ordinary pull request;
4. tells you the pull request's address, and that you add the `auto-merge` label when you want it
   merged, or have someone approve it; either one merges it.

The label is your decision reaching GitHub. A GitHub Action in your repository runs when the
`auto-merge` label is added to a pull request or when someone approves it, checks that every path in
the pull request's diff is a document path, and merges it. No verb adds the label or approves, and
nothing else a verb does is read as a signal to merge. GitHub does not let a pull request's author
approve it, and the verbs open pull requests as the person running them, so an approval always comes
from a second person. You can add the label the moment the verb's report ends, or after a colleague
has read the document, or never.

A document you left as `Draft` — you stopped and are coming back, or a design still carries
technical decisions you chose to leave for an engineer — gets a draft pull request. It blocks
nothing. The run that later promotes the document to `Ready` marks the pull request ready for
review, because GitHub does not merge a draft; that mark is not a signal to merge, and the pull
request still waits for the `auto-merge` label or an approval.

Document pull requests do not stack. Each one branches from the default branch and lands on its
own. Code pull requests are different: `implement` opens one per task as a GitHub stack, and a
person merges those. Pull requests from `equip`, `scaffold`, and `upgrade` are also a person's to
merge, because they change scripts and settings.

## The Action

`/codefall-equip landing` installs it as `.github/workflows/codefall-land-documents.yml`, from a
template the extension ships, and creates the `auto-merge` label in your repository. The file does
three things:

**It runs when the `auto-merge` label is added to a pull request, or when a review approving it is
submitted**, and again on every commit pushed to a pull request that already carries the label, so a
late change is checked again before it merges. The path check and the draft check run the same way
for every trigger. A draft pull request is never merged, whatever labels or approvals it has.

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
sign-off in the session and your label on the pull request as its only review. Code pull requests
are untouched by this; they still need whatever your rule requires.

A repository with no rule on its default branch needs nothing.

## Without the Action

A project that has not run `/codefall-equip landing` gets the same pull requests. Each verb's report
says the pull request is open and waits for a person to merge it, and names `/codefall-equip
landing` as the way to merge document pull requests with a label or an approval instead.

## What a report looks like

> SPEC-006 is written and `Ready`. The pull request is open at <url>. Add the `auto-merge` label
> when you want it merged, or have someone approve it; either one merges it. Next: run
> `/codefall-design SPEC-006`.

> DESIGN-003 is `Draft`: you left two technical decisions for an engineer. Pull request #1158 is a
> draft at <url> until they are settled. Next: have an engineer run `/codefall-design DESIGN-003`.

## Related

- [ADR-014](adrs/ADR-014-upstream-verbs-and-document-landing.md) — the decision.
- `extensions/shared/landing.md` — the procedure every verb follows, as installed into a project at
  `.codefall/shared/landing.md`.
- `extensions/skills/codefall-equip/reference/landing.md` — equip's landing track in full.
