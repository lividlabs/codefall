# Running another agent

Shared procedure. Every verb that hands work to another agent — `codefall-review` for its
reviewer, the verbs that consult when a run cannot settle something on its own, and
`codefall-implement` for a worker in another harness — follows it, so the rules about which agent
runs, in what order, and what it is allowed to do are stated once.
[ADR-009.7](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-009.7-agents.md) holds the
reasoning.

## Contents

- Which harness this is
- Resolving the list
- How a harness is called
- Walking the list
- `via=` overrides for one run
- An agent proposes
- Consulting
- Building in another harness

## Which harness this is

Answer, from what your own prompt tells you, exactly one of `claude`, `codex`, `muse`, `opencode`,
`agy`. Check it against the directory this skill was loaded from: a skill under `.claude/skills/` is
`claude`; one under `.agents/skills/` is one of the other four. An answer the directory contradicts,
or no answer, is `unknown`. Nothing is detected from the environment or the process tree.

## Resolving the list

`.codefall/settings.json` holds a top-level `agents` array of entries, one per harness a session
may run in:

```json
"agents": [
  {
    "activeAgent": "default",
    "review":  [ { "harness": "current" } ],
    "consult": [ { "harness": "current" } ]
  },
  {
    "activeAgent": "muse",
    "review":  [ { "harness": "claude" }, { "harness": "codex", "model": "gpt-5-codex" } ]
  }
]
```

`activeAgent` is the harness the session is running in, one of the five above, or `default`. Each
entry may have a `review` list, a `consult` list, and a `work` list, and each item names a
`harness`, one of the five or `current`, meaning a subagent of this harness, with an optional
`model`.

Take the entry whose `activeAgent` is the harness answered above, else the `default` entry; an
`unknown` answer takes the `default` entry. Within it, take the feature's list: `review` for a
review, `consult` for a consult, `work` for a worker that builds a bead. A list the entry does not have is the `default` entry's list, and
with no `default` entry it is one `current` agent. The list's order is the order to try.

## How a harness is called

A list item says which harness and which model. How that harness is started — the flag its model
goes in, the provider it reaches the model through, extra arguments, and a command that has to run
first — is stated once per harness, in a top-level `harnessConfig` object keyed by name:

```json
"harnessConfig": {
  "codex": {
    "modelFlag": "--model",
    "provider": "amazon-bedrock-runtime",
    "args": ["-c", "model_reasoning_effort=high"],
    "env": "aws configure export-credentials --format env"
  },
  "codex-direct": {
    "harness": "codex",
    "args": ["-c", "model_reasoning_effort=high"]
  }
}
```

A list item's `harness` is the lookup key. A key that is one of the five harness names configures
that harness. Any other key is a variant whose `harness` field names the binary, so
`{ "harness": "codex-direct", "model": "gpt-6-astra" }` runs codex the second way, and
`via=codex-direct:gpt-6-astra` does the same for one run. Every field is optional: `modelFlag`
defaults to `--model`; `provider` becomes the harness's own switch (the script header says which
harnesses have one); `args` are appended as given; `env` is a shell command whose output is
evaluated before the harness starts, which is how a run gets credentials an interactive shell would
have exported. A harness name with no block runs bare.

**OpenCode must be version 2 or later.** Both scripts call `opencode run --standalone`, which OpenCode
2 has and runs on a private server so that the PATH, the working directory, and the environment
reach the tools. OpenCode 1.x is not supported: an `opencode` agent on it is not expected to run,
and the person upgrades OpenCode by hand. OpenCode 2 refuses `--pure`, so neither
script passes it, and OpenCode has no switch for its hooks alone: the project's codefall plugin
loads in the run. Its merge guard runs on the agent's shell commands. Its session notice does not,
because both scripts set `CODEFALL_HEADLESS=1` in the environment of every harness they start, and
the notice reads it. `codefall-equip agents` reads `opencode --version` when it sets OpenCode up
and says to upgrade when it finds 1.x.

The model string is whatever the harness accepts for the provider in use, and the easiest way to
get it right is to run `/codefall-equip agents` from any harness: it finds the model and provider
the harness uses, writes the block, and adds the agent to the lists you choose.

## Walking the list

Run each agent, from the first, with the script beside this file:

```
../../.codefall/shared/run-agent.sh <harness>[:<model>] <prompt-file> <schema-file> <out-file>
```

Read its exit code and act on it:

| Exit | Meaning | Do |
| --- | --- | --- |
| `0` | The agent answered; the out-file holds it | Stop walking; read the answer |
| `70` | The agent is on `current` | Run a subagent of this harness with the same prompt yourself |
| `69`, `64` | Not runnable here: the harness is not on PATH, or is unknown | Skip to the next agent |
| `73`, `75`, `76` | Ran and failed: wrote nothing, timed out, or exited non-zero | Advance to the next agent, with the failure folded into its prompt |

An answer that parses but says the agent cannot settle the question advances the walk the same way
a failure does. An answer the verb disagrees with does not: that agent answered.

Each agent is tried once, and the walk ends at the end of the list. No second pass because the
first was inconclusive. When no agent answered, say so and stop, exactly as when the one reviewer
failed before there was an order.

**The report names every agent tried**, in order, with why each was skipped or failed, and which
one answered. A findings file records the one that answered as its reviewer. A configured fallback
that is reported this way is not a silent fallback.

## `via=` overrides for one run

An invocation may name its agent directly: `via=<harness>[:<model>]`, one of the five harnesses,
`current`, or a `harnessConfig` key, with an optional model. The override replaces the resolved
list for that run with that one agent, and writes nothing.

## An agent proposes

An agent run through `run-agent.sh` is read-only, and the script starts each harness in the mode
that enforces it.
It proposes; this session decides. Its answer never writes a file, and what the session does with
it follows the decisions reference beside this file: applied and recorded when the question is the
run's to decide, a human bead carrying the agent's forced option when it is not. A disagreement with the
user is a preference and not a question for another agent, and a missing precondition is an exit
and not a question at all. One round per stuck point, which may put the question to several agents at
once; never a second round because the first was inconclusive.

## Consulting

A consult is one question a run cannot settle on its own, put to the entry's `consult` list
(resolved as above) with the files that bear on it and the options the run sees. The prompt is the consult prompt
beside this file (consult-prompt.md), rendered once, and the answer follows the consult schema
beside it (consult.schema.json): an `answer`, a `confidence` of `high`, `medium`, or `low`, the `reasoning` with the files
that decided it, whether the answer is `reversible`, and `cannotSettle`. Walk the list as above; an
answer with `cannotSettle` true or `confidence` low advances the walk the way a failure does, and
the first answer that does neither ends it. One pass, one question, first answer wins.

What a verb does with the answer is the verb's rule, and every verb holds to three things. The
session decides: a consult proposes, and the session applies the answer or files a human bead per
the decisions reference beside this file. A consult never writes an ADR, and is never asked about a
preference the user has
stated. When the walk ends with no answer that settles the question, the last `cannotSettle`
answer's forced option is the default the human bead carries. The report names every agent
consulted, what each said, and what the session did with it; where the answer reaches a document
or a bead, the line that records the decision names the consult that informed it.

The list's default is one `current` agent, so a project that has configured nothing still
consults its own harness's subagent: a fresh context reading the same files, which is worth having
at no configuration cost. What configuration adds is a second model.

## Building in another harness

A worker builds one bead in a worktree, so it writes, and it runs through the script built for
that, beside this file (run-worker.sh):

```
run-worker.sh <harness>[:<model>] <worktree> <prompt-file> <out-file>
```

It takes the same agents and the same `harnessConfig` blocks as `run-agent.sh`, and its exit codes
mean the same: `0` finished, `70` run a subagent yourself, `69` and `64` not runnable here, `73`,
`75`, and `76` ran and failed. It adds `77`: the run changed the repository's git config or a hooks
directory outside the worktree, which is a failure too, and the files it names are inspected and
restored before any other git command in the primary checkout. It starts codex, claude, and
opencode in their build modes with scoped permissions, never a blanket approval: edits allowed, the
shell allowed except `git push`, `gh`, and `bd`, the network tools denied. Only Codex's sandbox
confines the shell, to the worktree and the repository's git directory; a Claude Code or OpenCode
worker's shell can write anywhere the user can. Shims first on the worker's PATH
refuse the same three commands, except `bd hooks run`, which the beads git hooks call on every
commit. Muse and agy have no write mode yet, and the script exits `64` for them. A run is killed
after `CODEFALL_WORK_TIMEOUT` seconds, an hour by default. The final message lands in the out-file
and the transcript in `<out-file>.transcript`; after an OpenCode run the script prints the model and
variant OpenCode recorded for the session.

A worker never runs `bd`, never pushes, and never opens a pull request. The session that ran it
creates the worktree and installs the dependencies first, and afterwards restores anything the
git-directory report names, reviews the diff against what the worker was asked to do, re-runs the
checks, breaks the code each new test covers to see the test fail, and then pushes itself. Two
verbs run workers from the same `work` list. `codefall-implement` builds a bead, choosing a worker
per bead at its gate; its `reference/workers.md` holds the procedure, and it opens the pull request
itself. `codefall-review` applies the fixes the person took on a branch or pull request, when the
person picks a worker at triage; its `reference/worker-fixes.md` holds the procedure. Nothing walks
the list on its own.
