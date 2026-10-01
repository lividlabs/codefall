# Resolving the harder targets

Read at step 1 when the argument is an epic, several pull requests, a document identifier, or
prose. The simple shapes — nothing, a branch, a PR, a range, a path — are resolved by the table in
`SKILL.md`.

## Contents

- An epic's work
- Several pull requests
- Document identifiers
- A prose argument

## An epic's work

An epic ID (`<prefix>-DESIGN-NNN`) or a design identifier (`DESIGN-NNN`, which names the epic
`<prefix>-DESIGN-NNN`, `<prefix>` being `bd config get issue_prefix`) is the work
`codefall-implement` built for it, wherever it stands:

```bash
gh pr list --state open --json headRefName,baseRefName,number --limit 200
```

- **An epic branch** — an open PR from `epic/<epic>-*` — is the target as a branch: its diff against
  the default branch holds every wave the root merged into it.
- **A serial stack** — open PRs from `feat/<epic>-*` — has one top: the branch that is no other open
  PR's `baseRefName`. The top is the target, as a branch; its three-dot diff against the default
  branch holds every link's change.
- **Nothing open** means every PR has merged or none was opened: say which, and stop. A merged PR
  is not reviewable.

The findings file's `target.epic` carries the epic ID, and the chart line
(`../../../../.codefall/shared/delivery.sh <epic>`) opens the confirmation and the report. Fixes
land on the top branch or the epic branch as new commits; a fix on the epic branch is pushed
directly and moves the aggregate PR's diff, which the report says.

## Several pull requests

`codefall-implement` leaves one per task. A stack, where each pull request is based on the one
below it, is one target: the top branch, or the range from the merge-base of its tip with the
default branch to its tip, holds every pull request's diff. Pull requests against the default branch
share nothing and are one invocation each, in the order the implement report listed them.

## Document identifiers

Document identifiers are the ones the other verbs define. Resolve one by globbing its directory and
stop if it matches nothing or more than one. Never guess at a near miss, and never invent a form
those verbs do not define.

## A prose argument

A prose argument is a scope and sometimes a narrowing. Resolve the scope by searching — the
description names components, behaviours, or domain terms, and those map to files.

**An ambiguous scope is interviewed, not guessed.** Search, show what you found, and ask what the
search could not settle — which of two components was meant, whether the boundary includes its
callers, whether a second subsystem matching the same terms is in or out. Search again with the
answer.

**Two rounds, then stop.** If the second round has not narrowed the set to something the user
recognises, say so and ask for a path or a document identifier instead.

Do not review until the user confirms the file list. Stop if the search finds nothing — say so and
ask for a different description rather than widening on your own.
