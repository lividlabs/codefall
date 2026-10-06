# The document landing

Read at step 1 of the landing track and follow it. The track installs one file,
`.github/workflows/codefall-land-documents.yml`, from `../templates/codefall-land-documents.yml`,
creates the `auto-merge` label in the repository, and tells the owner about the one repository setting
the file cannot change.

## Contents

- What the workflow does
- 1. Read the project
- 2. Find what is already there
- 3. Ask the one question
- 4. Write
- 5. Prove it
- 6. Report
- Rules

## What the workflow does

Every codefall verb that writes a document — `codefall-envision`, `codefall-specify`,
`codefall-report`, `codefall-design`, and `codefall-mock-up` run on its own — opens its pull request
and tells the person to add the `auto-merge` label when they want it merged, or to have someone
approve it; either one merges it. No verb adds the label or approves. The workflow runs when the
label is added (`labeled`, label name `auto-merge`), again on a commit pushed to a pull request that
already carries it (`synchronize`), so a late change is checked again, and when a review is
submitted with the state `approved` (`pull_request_review`). GitHub does not let a pull request's
author approve it, and a verb opens the pull request as the person running it, so an approval is a
second person's. It checks that every path in the pull request's diff is one of these:

```
docs/visions/   docs/specs/   docs/bugs/   docs/mockups/   docs/designs/   docs/adrs/
.codefall/reviews/   .codefall/tests/   .beads/interactions.jsonl
```

and merges it with `gh pr merge --squash` under the default `GITHUB_TOKEN`. One path outside the
list, and the pull request waits for a person. A draft pull request is never merged. The check is a
list of paths, never a reading of content. Code pull requests are not touched; `codefall-implement`'s
are a person's to merge, and so are this verb's own.

## 1. Read the project

The target is the path argument, or the working directory. Read `.codefall/settings.json`; none
means `codefall init` comes first. Read the default branch from
`git symbolic-ref --short refs/remotes/origin/HEAD`, and the repository from `gh repo view --json
nameWithOwner`. No remote on GitHub is a stop: the workflow has nowhere to run.

## 2. Find what is already there

- `.github/workflows/codefall-land-documents.yml` exists: read it and compare it with the template.
  Identical means nothing to do for the file; say so. Different means the project edited it: show
  the difference and ask whether to keep theirs or take the template, never overwrite it unasked.
- The `auto-merge` label exists (`gh label list --json name --jq '.[].name'` names it): say so.
  Missing: the write creates it.
- Another workflow under `.github/workflows/` that merges pull requests (`gh pr merge`,
  `pull_request_target`, `automerge`): name it, so the owner knows two things would merge.
- **The branch rule.** Read it, never change it:

  ```bash
  gh api "repos/{owner}/{repo}/branches/<default>/protection" 2>/dev/null   # 404 means none
  gh api "repos/{owner}/{repo}/rulesets?includes_parents=true"
  ```

  A required pull request review, or a ruleset with `pull_request` rules, means the workflow's token
  cannot merge until the owner lets it through: in a ruleset, a bypass for the `github-actions` app
  or for repository admins scoped to this workflow; in classic branch protection, the "allow
  specified actors to bypass required pull requests" list. Say which one applies and what documents
  landing without a second reader means: the person's sign-off in the session and their label on the
  pull request are the review, unless a colleague approves it instead. A repository with no rule
  needs nothing.

When the file is identical to the template and the label exists, there is nothing to do; say so and
stop.

## 3. Ask the one question

Show the workflow file as it will be written, say that the `auto-merge` label will be created, give the
branch-rule finding from step 2 in one or two sentences, and ask whether to install it. When a rule
would block the token, the question says so: "Install the workflow now, and change the rule
yourself afterwards?" Equip changes no branch rule.

## 4. Write

Take the branch step of `../../../../.codefall/shared/landing.md`: standing on the default branch,
`git switch -c equip/landing`; on another branch, ask once which to use. Write the file from the
template, byte for byte unless the owner asked for a different allowlist, in which case change the
`allow` line only.

Then create the label, which lives in the repository and not in the file:

```bash
gh label create auto-merge --description "merge this document pull request" --color 0E8A16
```

A label that already exists is left as it is; `gh label create` reports that and the run says so.

## 5. Prove it

The file has nothing to run until it is on the default branch. Prove what can be proved now:
the YAML parses (`python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' <file>` where
Python with PyYAML is available, else `gh workflow list` after the merge is the check), the
allowlist regular expression accepts `docs/specs/SPEC-001-x.md` and rejects `src/a.js`, and the
label is listed:

```bash
printf 'docs/specs/SPEC-001-x.md\nsrc/a.js\n' | grep -vE '<the allow line>'   # prints src/a.js only
gh label list --json name --jq '.[].name' | grep -x auto-merge                  # prints auto-merge
```

The full proof is the next document pull request a person labels `auto-merge` or approves: it merges
within a couple of minutes, and `gh run list --workflow codefall-land-documents` shows the run.

## 6. Report

What was written and why, that the `auto-merge` label exists, the branch-rule finding and what the owner
has to change if anything, and the landing per `../../../../.codefall/shared/landing.md`: the file
committed by path on `equip/landing`, pushed, and its pull request opened and named by URL, none of
it a question, a person's to merge. **End with what the user does next**: merge the pull request,
and nothing after it. The branch-rule change, when step 2 found one, is said above as the owner's;
and once the pull request is merged, a document pull request merges when a person adds the
`auto-merge` label to it or approves it, which the report states as a fact, not as a step.

## Rules

The rules that hold for every track are in `../SKILL.md`. This one holds for this track:

- **The landing track changes no branch rule.** It writes one workflow file, creates the
  `auto-merge` label, and tells the owner what the branch rule has to allow.
