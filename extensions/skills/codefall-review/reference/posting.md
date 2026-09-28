# Posting to a pull request

Off unless the project turned it on. Read at step 5 when the target is an open pull request.

`.codefall/settings.json` carries the switch as `postToPullRequest` in the `review` block. The key
is optional and absent means off, so a `review` block holding only its `agents` order posts nothing:

```json
{ "review": { "postToPullRequest": true } }
```

When it is `true` and the target is an open pull request, the findings post after triage as **one
review** — `event=COMMENT` with a `comments` array, so each finding becomes an inline thread on the
line it is about:

```bash
gh api --method POST repos/{owner}/{repo}/pulls/<number>/reviews --input <body.json>
```

`{owner}/{repo}` is filled by `gh` from the repository; `<number>` is the `number` field read at
resolution.

## Anchoring

GitHub rejects an inline comment on a line outside the diff, and rejects the whole call if any one
comment is out of bounds — so one stray finding loses the entire post. Sort them before sending.

**A finding anchors** when its `location.path` appears in `gh pr diff` and its `location.startLine`
falls inside one of that file's hunks, counted in the hunk's `+` numbering. Those get `path`, `line`
(the finding's `endLine`, or `startLine` when there is only one), and `side: "RIGHT"`. A finding
whose range straddles a hunk boundary anchors at the last line inside the hunk.

**Everything else goes in the review body** — a whole-file concern, a missing test, a `docs` finding
about a file the PR never touched, a finding on a line the diff only shows as context.

## What posts

**Only `fixed` and `deferred` findings.** A `dismissed` finding was judged wrong, and posting it
puts a rejected claim in front of people who will not see the reasoning.
