# The spec's tracker issue

When the spec behind the work has a GitHub mirror, its **parent issue** walks the work's state, at
the granularity GitHub can express. Read at step 6, at the first claim, and at step 8.

| Moment | With a Project board | Without one |
| --- | --- | --- |
| First claim | Status → **In Progress** | — |
| Work built, PRs open | Status → **In Review** | — |
| Every PR merged, epic closed | Status → **Done**, issue closed, children closed with it | Issue closed, children with it |

- Requirement sub-issues close **with the parent, never individually**.
- **A spec a worker amended** has one requirement whose criteria grew. At the wave boundary, the
  root regenerates that requirement's issue body per the *Refreshing* sequence in
  `../../codefall-specify/trackers/<name>/PROFILE.md`, the existing-requirement case alone, from
  the spec as it stands on the worker's branch — `git show origin/<branch>:<path>`, since the
  primary checkout does not carry the amendment — and nothing else in the mirror moves for it.
- Every PR body carries one line — `Relates to #<spec-issue>` — so the mirror cross-links the work
  as it happens.
- **A fix for a bug report** carries `Fixes #<bug-issue>` instead, from the report's `**Issue:**`
  row — reached through the design's `bug` label, or the bead's body at tier 0 — on the pull request
  that reaches the default branch: the task's own at tier 0, the epic's aggregate otherwise. GitHub
  closes the issue at the human's merge, and nothing else in this table moves for it.
- Board IDs are per-installation and never stored in this skill: discover them at run time
  (`gh project list`, `gh project field-list`), or read them from `CUSTOMIZE.md` when the project
  has pinned them there. A pinned ID that fails means the board changed — rediscover, show the
  difference, and ask before updating the file.
