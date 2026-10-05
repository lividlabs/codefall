# Driver

Runs the eight sessions on the throwaway project, one fresh session each, with a simulated product
manager answering every question; does what the person would do between sessions; then runs the
judge through the project's own `run-agent.sh`. Written against the TypeScript Agent SDK
(`@anthropic-ai/claude-agent-sdk`), run directly by Node 24, which strips the types itself; there is
no build step.

## Why this shape

A skill asks questions mid-session through the `AskUserQuestion` tool, and plain `claude -p` cannot
answer them: with `--permission-prompts none` the call is denied and the verb stops. The Agent SDK
gives the host a `canUseTool` callback that sees every tool call needing a decision. For
`AskUserQuestion` the input carries the `questions` array, and returning `{behavior: "allow",
updatedInput: {...input, answers: {<question text>: <label>}}}` hands the answers to the agent as if
a person had clicked them. That is `session.ts`.

A verb also asks things in prose and ends its turn waiting ("Does that match what you have in
mind?"). Streaming input, a `prompt` that is an `AsyncIterable` of user messages, keeps the session
alive between turns; after each `result` message the simulator reads the turn and either replies,
says the verb is finished, or says it is stuck. The session is a fresh one per verb: a new `query()`
call is the `/clear`.

The simulator (`simulator.ts`) is a model call with the brief, the session's notes from
`protocol.ts`, and the readable transcript so far; it answers in character and returns JSON. It runs
through the same SDK with no tools and no settings, so it uses the Claude Code login already on the
machine and needs no separate API key.

The judge (`judge.ts`) does not use the SDK. It calls `<project>/.codefall/shared/run-agent.sh`,
the script codefall's verbs use to run a reviewer or a consult in another harness, with `muse`
first and `opencode` as the fallback, the prompt and the answer schema as files, and the run
directory as the working directory. The answer is JSON; the driver writes `verdict.md` from it and
records which harness answered.

## What it needs

- Node 24 or newer, `gh` logged in (the same account `setup.sh` used), the `gh-stack` extension,
  `bd`, and the `codefall` binary `setup.sh` built (its path is in `.throwaway.env`).
- Claude Code logged in on this machine (`claude auth status`), or `ANTHROPIC_API_KEY` in the
  environment. The sessions and the simulator go through the SDK, which shells out to the installed
  `claude` binary.
- `muse` or `opencode` on `PATH` for the judge. With neither, the run still finishes; `verdict.md`
  says no agent answered.
- `fixture/setup.sh` run first; it writes `evals/twisted-todo/.throwaway.env`.

```bash
cd evals/twisted-todo/driver
npm install
node run.ts                                  # the whole chain, then the judge
node run.ts --model opus --sim-model sonnet --effort high
node run.ts --from implement --run ../runs/20261005T1030   # resume a run from a session
node run.ts --judge-only --run ../runs/20261005T1030       # judge an existing run again
```

Flags: `--model` (sessions, default `opus`), `--sim-model` (default `sonnet`), `--judge-agents`
(default `muse,opencode`; any `run-agent.sh` agent names, tried in order), `--effort`,
`--max-pm-turns` (default 60 per session), `--max-minutes` (default 90 per session),
`--land-minutes` (how long to wait for the Action per document pull request, default 10),
`--project` and `--repo` (override `.throwaway.env`).

## What the driver does between sessions

| After | The driver, as the person |
| --- | --- |
| `equip-local`, `equip-test` | merges the equip pull request with `gh pr merge --squash`; returns the checkout to `main` |
| `envision`, `specify`, `design` | adds the `auto-merge` label to the document pull request with `gh pr edit <n> --add-label auto-merge --repo <owner>/<repo>`, as the person would after reading the report; waits for the Action to merge it; merges it itself only when the wait runs out, recorded as a deviation; returns the checkout to `main` |
| `design`, when the design is `Draft` with decisions left for an engineer | stops the chain and judges what exists |
| `implement`, `review` | nothing; the code stack stays open |
| `test` | merges the code stack at its top with `gh stack merge --squash --yes`; returns the checkout to `main` |

Every `gh` call runs with the run directory as its working directory and `GH_REPO` set: that is the
"session outside the project directory" the plan describes, and no hook applies to it. A document
pull request that already carries the `auto-merge` label when its session ends, or that is already
merged, is recorded as a deviation, because only the person adds that label. Returning
the checkout to `main` is plain `git fetch`, `git checkout main`, `git pull --ff-only`, recorded in
`deviations.txt` as not a deviation.

## What a run writes

`evals/twisted-todo/runs/<timestamp>/`:

| File | What it is |
| --- | --- |
| `<session>.jsonl` | every message the session emitted, plus `eval_note` lines for the simulator's and the driver's actions |
| `<session>.md` | the readable transcript: `**Verb:**` is the agent, `**PM:**` is what the simulator said, `**PM-sim:**` is why, `> tool` lines are tool calls |
| `driver.log` | the driver's own log |
| `deviations.txt` | what the driver did in the person's place; `(not a deviation)` lines are what the plan expects |
| `evidence/` | pull requests, issues, Action runs, the git log of `main`, the document tree, the settings, the Beads epic and children, the review and test records, the test cases |
| `summary.json` | per-session status, PM turns, and cost; `total_cost_usd` is the SDK's own estimate; `judgedBy` is the harness that answered |
| `rubric.md`, `brief.md` | copies, so the run directory is self-contained for the judge |
| `judge-prompt.rendered.md`, `judge.<agent>.out` | the prompt the judge was given and each agent's raw answer |
| `verdict.md`, `verdict.json` | the judge's verdict, rendered and raw, with the list of agents tried |

## How the driver decides a session is done

After every turn the simulator classifies the agent's last text: a question or a document shown for
approval is a `reply`; a final report that names what happens next or says it is safe to `/clear` is
`done`; a stop whose remedy is a command for the person (install something, run `codefall upgrade`)
is also `done`, and the chain stops there because the next verb would hit the same wall. `stuck` is a
loop or an unanswerable question with no way forward offered. A session also ends on the SDK's own
error result, on `--max-pm-turns`, or on `--max-minutes`.

## Permissions and the guard hook

Sessions run with `permissionMode: "acceptEdits"` and a `canUseTool` that allows everything except
`AskUserQuestion`, which it answers. `settingSources: ["project", "local"]` loads the project's
`.claude/settings.json`, so codefall's `PreToolUse` guard runs on every Bash call and still denies a
merge to `main`; a denial in a transcript is the system working. A `Skill` tool call in the
`review` or `test` transcript is the verb running `implement` itself, and the implement run's own
`AskUserQuestion` calls are answered by the same simulator.

## Known limits

- The simulator decides from text alone. A verb that ends its turn with no text (only a tool call
  that was denied, say) is classified from an empty string; the prompt tells the simulator what to
  do, and the transcript records it either way.
- `canUseTool` also answers permission requests from subagents (implement's workers) because they
  route through the same host. The go gate's permissions condition is therefore met; the transcript
  shows what each worker asked for.
- Cost figures come from the SDK's `total_cost_usd`, an estimate. `modelUsage` in the last `result`
  line of each `.jsonl` has per-model tokens. The judge's cost is the other harness's and is not
  counted.
- `run-agent.sh` reads `harnessConfig` from the project's `.codefall/settings.json`. `codefall init`
  writes none, so `muse` and `opencode` run with their own default model unless `/codefall-equip
  agents` was run on the project first.
