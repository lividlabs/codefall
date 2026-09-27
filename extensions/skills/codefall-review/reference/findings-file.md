# The findings file

Two files per invocation, sharing one stem. Read at step 4, before the first write.

## Contents

- Naming
- The Markdown shape
- Where they are committed
- `revision`

## Naming

```
.codefall/reviews/<YYYY-MM-DDTHHMMSSZ>-<target-key>.json
.codefall/reviews/<YYYY-MM-DDTHHMMSSZ>-<target-key>.md
```

`<target-key>` carries a slug wherever there is one to take: `pr-51-init-below-root` from a PR
title, `SPEC-004-trip-sharing` from a document, `feat-booking-DESIGN-007-T1-stage-context` from a branch,
`src-fulfillment` from a path, `range-a1b2c3d-e4f5a6b` from a range's two short SHAs, the resolved
common root for a prose target or `adhoc` when the files share none, and `uncommitted` when there
is no subject at all. In full:

```
.codefall/reviews/2026-09-14T081233Z-pr-51-init-below-root.json
```

The `.json` is the record; the `.md` is the same review written to be read. Both hold what was
reviewed and at which revision, who reviewed it and which agents were tried before it, which lenses
ran, what could not be checked, and every finding with its status. `../findings.schema.json` is the
shape of the JSON: `reviewer.name` is the configured agent when one answered, and `reviewer.tried`
the agents walked before it, each `skipped` or `failed` with the reason.

**The files are written three times** — after the review, after triage, after the fixes. An
interrupted session resumes from them rather than starting over.

## The Markdown shape

```markdown
# Review: <target-key>

**Reviewed:** <timestamp> · **Reviewer:** [<name> ]<harness>[/<model>] · **Revision:** <revision>
**Tried:** <name or harness> (skipped: not on PATH), <name> (failed: exited 76) — only when an agent was walked before the reviewer
**Lenses:** <the lenses that ran>

## Findings

### 1. <claim> — `blocker` · `correctness` · fixed
`path/to/file.go:120-134`

<conditions, when there are any, then the claim in full>

<the proposed patch, in a fenced diff block, when there is one>

## Not checked

- <what could not be reviewed, and why>
```

Findings are ordered most severe first, matching the triage list. A review with none says so under
the heading rather than dropping it. A deferred finding filed as a revision bead carries the bead's
ID after its status — `deferred · booking-design-007-stagestore` — and the JSON carries it as
`bead`. A finding promoted from `notChecked` after a consult carries a line after its location —
*consulted `architect` (codex): the retry loop re-reads the offset it just committed* — and the
JSON carries it as `consult`; an entry a consult left in `notChecked` carries the consult's view
appended to its text.

## Where they are committed

With the fixes, on whatever branch the fixes landed on — one commit carrying both.

**Except for uncommitted work**, where committing the findings would put them in the diff under
review. There the files are written and left unstaged, and the report says they are uncommitted and
where they are. The user commits them with their own work or not at all.

## `revision`

**`revision` is the target's, not the session's.** Standing on the default branch reviewing a pull
request, HEAD is the wrong answer.

| Target | `revision` |
| --- | --- |
| uncommitted | HEAD SHA, plus `dirty` |
| branch | the branch's tip SHA, plus `base` — the merge-base with the default branch |
| pull request | `headRefOid`, plus `base` — `baseRefOid` |
| range | `<to>`, plus `base` — `<from>` |
| document | the SHA of the last commit that touched the file; if it is modified in the working tree, HEAD plus `dirty` |
| path or prose | HEAD SHA of the checkout the review ran in, plus `dirty` when the tree is not clean |

**`dirty` covers everything the review actually read**, including untracked files. Take it over
staged, unstaged and untracked content together:

```bash
{ git diff HEAD; git ls-files --others --exclude-standard -z | xargs -0 -I{} git diff --no-index /dev/null {}; } | shasum -a 256
```

Short-form the result. It identifies a working tree; it is not meant to reconstruct one.
