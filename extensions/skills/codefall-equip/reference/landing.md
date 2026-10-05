# The document landing

Read at step 1 of the landing track and follow it. The track installs one file,
`.github/workflows/codefall-land-documents.yml`, from `../templates/codefall-land-documents.yml`,
and tells the owner about the one repository setting the file cannot change.

## Contents

- What the workflow does
- 1. Read the project
- 2. Find what is already there
- 3. Ask the one question
- 4. Write
- 5. Prove it
- 6. Report

## What the workflow does

Every codefall verb that writes a document — `codefall-envision`, `codefall-specify`,
`codefall-report`, `codefall-design`, and `codefall-mock-up` run on its own — opens its pull request
as a draft and marks it ready for review when the person has signed the document off. The workflow
listens for that (`ready_for_review`, and `synchronize` so a late commit is checked again), checks
that every path in the pull request's diff is one of these:

```
docs/visions/   docs/specs/   docs/bugs/   docs/mockups/   docs/designs/   docs/adrs/
.codefall/reviews/   .codefall/tests/   .beads/interactions.jsonl
```

and merges it with `gh pr merge --squash` under the default `GITHUB_TOKEN`. One path outside the
list, and the pull request waits for a person. The check is a list of paths, never a reading of
content. Code pull requests are not touched; `codefall-implement`'s are a person's to merge.

## 1. Read the project

The target is the path argument, or the working directory. Read `.codefall/settings.json`; none
means `codefall init` comes first. Read the default branch from
`git symbolic-ref --short refs/remotes/origin/HEAD`, and the repository from `gh repo view --json
nameWithOwner`. No remote on GitHub is a stop: the workflow has nowhere to run.

## 2. Find what is already there

- `.github/workflows/codefall-land-documents.yml` exists: read it and compare it with the template.
  Identical means nothing to do; say so and stop. Different means the project edited it: show the
  difference and ask whether to keep theirs or take the template, never overwrite it unasked.
- Another workflow under `.github/workflows/` that merges pull requests (`gh pr merge`,
  `pull_request_target`, `automerge`): name it, so the owner knows two things would merge.
- **The branch rule.** Read it, never change it:

  ```bash
  gh api "repos/{owner}/{repo}/branches/<default>/protection" 2>/dev/null   # 404 means none
  gh api "repos/{owner}/{repo}/rulesets?includes_parents=true"
  ```

  A required pull request review, or a ruleset with `pull_request` rules, means the workflow's
  token cannot merge until the owner lets it through: in a ruleset, a bypass for the
  `github-actions` app or for repository admins scoped to this workflow; in classic branch
  protection, the "allow specified actors to bypass required pull requests" list. Say which one
  applies and what documents landing without a second reader means: the person's sign-off in the
  session is the review. A repository with no rule needs nothing.

## 3. Ask the one question

Show the workflow file as it will be written, the branch-rule finding from step 2 in one or two
sentences, and ask whether to install it. When a rule would block the token, the question says so:
"Install the workflow now, and change the rule yourself afterwards?" Equip changes no repository
setting.

## 4. Write

Take the branch step of `../../../.codefall/shared/landing.md`: standing on the default branch,
`git switch -c equip/landing`; on another branch, ask once which to use. Write the file from the
template, byte for byte unless the owner asked for a different allowlist, in which case change the
`allow` line only.

## 5. Prove it

The file has nothing to run until it is on the default branch. Prove what can be proved now:
the YAML parses (`python3 -c 'import yaml,sys; yaml.safe_load(open(sys.argv[1]))' <file>` where
Python with PyYAML is available, else `gh workflow list` after the merge is the check), and the
allowlist regular expression accepts `docs/specs/SPEC-001-x.md` and rejects `src/a.js`:

```bash
printf 'docs/specs/SPEC-001-x.md\nsrc/a.js\n' | grep -vE '<the allow line>'   # prints src/a.js only
```

The full proof is the next document pull request a verb marks ready: it merges within a couple of
minutes, and `gh run list --workflow codefall-land-documents` shows the run.

## 6. Report

What was written and why, the branch-rule finding and what the owner has to change if anything,
and the landing per `../../../.codefall/shared/landing.md`: the file committed by path on
`equip/landing`, pushed, and its pull request opened ready for review, a person's to merge. **End
with what the user does next**: merge the pull request, change the branch rule if step 2 said so,
and then the next document verb's pull request lands on its own.
