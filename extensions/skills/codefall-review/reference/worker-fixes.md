# Fixes by a worker in another harness

The accepted fixes may be applied by an agent on another harness and model instead of this session:
a cheaper model for a set of mechanical fixes, a stronger one for hard ones. The worker runs
through `../../../../.codefall/shared/run-worker.sh` in a worktree this session made, with `git
push`, `gh`, and `bd` refused on its PATH. It applies the fixes and commits; this session checks
them and pushes. Read at step 6 when the person picks a worker.

## Contents

- When it is offered
- Before the run
- The run
- After the run

## When it is offered

When the fixes land on a branch — a branch, an open pull request, or a commit range's branch, per
*Where the fixes go* — and the entry's `work` list names an agent this machine can start other
than `current`. The list is the one `codefall-implement` uses, resolved the same way, per
*Resolving the list* in `../../../../.codefall/shared/running-agents.md`. Muse and agy have no
write mode and are passed over. Uncommitted work, a path on the default branch, and an epic's code
findings, which `codefall-implement` builds, are never given to a worker.

The triage question carries the choice in one line once findings are taken: "Apply them here, or
with `opencode:digitalocean/glm-5.3#max`?", every startable agent of the list named, here the
default. One worker applies every accepted fix; nothing is split.

The worktree is where the work is meant to land, not a wall. A Codex worker can write the worktree
and the repository's common git directory, which holds its config and hooks. A Claude Code or
OpenCode worker's shell can write anywhere the user can. The script checks the git config and the
hooks directories outside the worktree; nothing checks the rest.

## Before the run

1. **The worktree.** `git fetch origin`, then
   `git worktree add -b review-fixes/<stem> <path> origin/<branch>`, with `<stem>` the findings
   files' stem and `<path>` where this harness puts its own worktrees (`.claude/worktrees/` under
   Claude Code). The primary checkout does not move to the branch.
2. **The dependencies.** Install them in the worktree with the command the project's `AGENTS.md`
   gives, or its declared `update` script. A worker may have no network.
3. **The prompt**, written outside the worktree with the out-file beside it, so neither is
   committed. It holds:
   - the accepted findings from the findings JSON, each with its id, location, claim, conditions,
     and `patch`, which is a starting point and not a script: apply the intent, matching the
     surrounding code;
   - the rule that a code fix changing behaviour the plan or spec describes amends that document in
     the same commit, as text;
   - scope: the accepted findings and nothing else, no other refactor, no other finding;
   - the project's check commands, from its `AGENTS.md`, run until clean;
   - commit, with each message naming the finding ids it fixes; never push, never run `gh` or
     `bd`, never touch the findings files;
   - the final message, exactly one JSON object:
     `{"status": "success", "fixed": ["F1", "F3"], "notFixed": [{"id": "F2", "reason": "…"}]}`,
     or `{"status": "failure", "reason": "…"}`.

## The run

From the primary checkout, in the background:

```
../../../../.codefall/shared/run-worker.sh <harness>[:<model>] <worktree> <prompt-file> <out-file>
```

| Exit | Do |
| --- | --- |
| `0` | Read the result, the last JSON object in the out-file, and go on below |
| `70` | Apply the fixes here, as without a worker |
| `69`, `64` | The next startable agent the person named, else here; say so |
| `73`, `75`, `76` | Apply the fixes here, starting from what the worktree holds or from the branch; say why |
| `77` | Step 1 below first, then apply the fixes here |

## After the run

1. **Read the git-directory report.** On exit `77`, the script names each file it found added,
   removed, or changed in the repository's git config, the worktree's config, and the hooks
   directories outside the worktree. Before any further git command in the primary checkout,
   which would run a planted hook, read each one and restore it, and say in the report what was
   found and restored.
2. **Review the diff** — `git -C <worktree> diff origin/<branch>...HEAD` — against the accepted
   findings, one by one: each `fixed` id is fixed as its finding intended, and nothing outside them
   changed. A `notFixed` finding, or one fixed wrongly, is applied here, in the worktree. This is
   checking the fixes against what was accepted, not a second review.
3. **Re-run the project's checks** in the worktree.
4. **Break what each new test covers**, when the worker added tests: change the code the test
   covers so the behaviour is wrong, run the test, confirm it fails, restore the code. A test that
   still passes covers nothing; fix it here.
5. **Confirm the model**, as the script's `run-worker: opencode session` line or the transcript
   shows it: the `system` `init` event for Claude Code, the header for Codex.
6. **Write, commit, and push.** The findings files move into the worktree's `.codefall/reviews/`
   for their third write, every taken finding `fixed`, and are committed there with the fixes; the report and that commit's body
   name the agent that applied them, `fixed by <harness>:<model>`. Then
   `git -C <worktree> push origin HEAD:<branch>`, with the branch named, since the guard hook
   denies a push with no refspec while the primary checkout is on the default branch. A push the
   remote refuses because the branch moved is reported, not forced. The worktree is offered for
   removal in the report.
