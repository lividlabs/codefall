# Workers

One background agent per bead, each in its own worktree. Read at step 6, before rendering the first
prompt.

## Contents

- Launching
- Chains
- Results
- Failure handling
- Consulting
- Worktrees survive the run

## Launching

`Agent` with `isolation: 'worktree'`, prompted from `../worker-prompt.md`, rendered by substituting
`{{BEAD_ID}}`, `{{TITLE}}`, `{{BODY}}`, `{{ACCEPTANCE}}`, `{{DESIGN_REF}}`, `{{BRANCH}}`,
`{{BASE_REF}}`, `{{PR_TARGET}}`, `{{VERIFY_COMMANDS}}`, `{{RELATES_LINE}}`, `{{REPO}}`,
`{{TESTING_ROOT}}`, and `{{CASE_FILE_FORMAT}}`.

**The two testing placeholders are resolved by the root, not the worker.**

| Placeholder | What the root renders |
| --- | --- |
| `{{TESTING_ROOT}}` | `test.dir` from `.codefall/settings.json`, as the worker sees it from the repository root — `testing` |
| `{{CASE_FILE_FORMAT}}` | the absolute path of `codefall-test`'s `reference/case-file.md` in the skills directory this run was invoked from — `<repo>/.claude/skills/codefall-test/reference/case-file.md`, or `.agents/skills/…` under a harness that mirrors the tree |

A worker reads the format by that absolute path because it cannot resolve a skill-relative one: it
runs in the project, not in the skills directory. Render both on every prompt; a bead whose criteria
name no case simply never reads them.

- **Single-bead scope also gets a worktree.** The primary checkout stays free for the user.
- **The worker is strategy-blind.** Stacked versus epic is fully encoded in `BASE_REF` and
  `PR_TARGET`. Branch names are root-supplied and deterministic — `feat/<bead>-<slug>` — so a
  stacked link's `BASE_REF` can name its predecessor's branch before that worker exists.
- **Worktree isolation seeds from the repo's default HEAD**, not from `BASE_REF`. The worker's
  first act is its own `git fetch` followed by `git checkout -b {{BRANCH}} origin/{{BASE_REF}}` —
  off the remote-tracking ref, because git's one-branch-one-worktree rule makes a plain checkout
  fail when another worktree holds that branch.
- **Workers never run `bd`.** Everything a worker needs from the tracker travels in its prompt, and
  every tracker write is the root's.

## Chains

**Chains are strictly sequential; parallelism is across chains.** Link N+1 launches only after the
root has verified two facts about link N: the branch is on the remote, and the PR exists. The root
checks, never trusts.

## Results

The worker's final message is exactly one JSON object:

```json
{"bead": "booking-DESIGN-007-T2", "status": "success", "pr": 102,
 "branch": "feat/booking-DESIGN-007-T2-wire-context",
 "discovered": [{"kind": "code", "title": "Parser drops trailing comma", "context": "…",
                 "from": "booking-DESIGN-007-T2"},
                {"kind": "design", "title": "§ Architecture names a StageStore the code replaced",
                 "context": "…", "from": "booking-DESIGN-007-T2"}],
 "amended": [{"document": "docs/designs/DESIGN-007-stage-context.md", "section": "Architecture",
              "summary": "StageStore renamed to StageContext, matching the code"}]}
```

or `{"bead": "…", "status": "failure", "reason": "…"}`. The `discovered` list is how tangent work
reaches the root, which files it — the worker's diff stays scoped to its bead. `kind` is `code` or
`design` and picks the form under *Discovered work* in `beads.md`, so the root files each item
without re-reading the worker's prose. The `amended` list names each upstream document the worker
amended in its branch; the root reads it at the wave boundary per *Amendments* in `beads.md`.

## Failure handling

A worker returning `{"status": "failure"}` gets **one consult, then one automatic retry**. The root
consults per *Consulting* below on the failure reason and the bead, and launches a fresh worker at
higher effort or a stronger model with the failure reason and the consult's answer folded into its
prompt. A second failure gets one more consult, and then stops the chain and escalates — fix it by
hand, skip the bead (unclaim it, note why), or abort the run — with the consult's analysis in front
of the human. Nothing retries more than once on its own, and nothing is consulted twice on the same
failure. A worker that is alive but stalled is resumed with a message, not replaced.

## Consulting

The root consults; a worker never does. Every call to another agent is the root's, in the primary
checkout, as every `bd` write is. The procedure is *Consulting* in
`../../../../.codefall/shared/running-agents.md`; this is what implement puts into it.

**The question**, rendered into `../../../../.codefall/shared/consult-prompt.md`: `QUESTION` is why
the worker failed, in its own words from `reason`, and what the bead asked for; `FILES` are the
bead's Design ref section and the files the worker's branch touched, or the design's predicted files
when nothing was pushed; `CONTEXT` is the bead body and acceptance criteria; `OPTIONS` are the
courses the root can see — a different approach the design allows, a missing precondition to name,
a task that is cut wrong and belongs back with `codefall-design`; `PRIOR` is the failure of an
earlier agent in the order, or empty. `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`.

**The order** is `consult.agents`, resolved and walked as `running-agents.md` says, with
`../../../../.codefall/shared/run-agent.sh`. With nothing configured it is one entry on `current`,
and the root runs the consult as a subagent: a fresh context reading the same files, which is why
the default run consults too. The go gate names the resolved order.

**What the answer does.** Before the retry: an answer that names a course the bead allows is folded
into the retry prompt beside the failure reason, as a suggestion the fresh worker weighs and not an
instruction — the worker still reads the design and the code first. `cannotSettle`, or no agent
answering, means the retry runs on the failure reason alone, as before. At the second failure: the
answer, or the fact that none came, goes into the escalation so the human reads the analysis beside
the failure. A consult never changes the graph, never edits a bead, and never overrides a design;
an answer that says the task is cut wrong is reported as exactly that, for `codefall-design`.

**The record.** The `bd comment` that records what ran on the bead names the agent consulted and
its answer in one line; the close-out report lists every consult, by bead, with who answered and
who was skipped or failed.

## Worktrees survive the run

Cleanup is an offer at close-out — never automatic, and never for a worktree whose PR is still open.
