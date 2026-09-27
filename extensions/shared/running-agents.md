# Running another agent

Shared procedure. Every verb that hands a question to another agent — `codefall-review` for its
reviewer, and the verbs that consult when a run cannot settle something on its own — follows it, so
the rules about which agent runs, in what order, and what its answer is allowed to do are stated
once. [ADR-009](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-009-agents.md)
holds the reasoning.

## Contents

- Which harness this is
- Resolving the order
- Walking the order
- `via=` overrides for one run
- An agent proposes
- Consulting

## Which harness this is

Answer, from what your own prompt tells you, exactly one of `claude`, `codex`, `muse`, `opencode`,
`agy`. Check it against the directory this skill was loaded from: a skill under `.claude/skills/` is
`claude`; one under `.agents/skills/` is one of the other four. An answer the directory contradicts,
or no answer, is `unknown`. Nothing is detected from the environment or the process tree.

## Resolving the order

`.codefall/settings.json` defines the agents in a top-level `agents` list — each a `name`, a
`harness` (one of the five above, or `current`, meaning this harness's own subagent), and an
optional `model`. Absent or empty, the list is one entry: `subagent` on `current`.

Take the first of these that is present, in this order:

1. `agentsByHarness.<harness>`, for the harness answered above; skipped when the answer is `unknown`.
2. The use's own `agents` key — `review.agents`, `consult.agents`.
3. The top-level `agents` list, in its own order.

Each is an ordered list of names, and the order is the order to try.

## Walking the order

Run each entry, from the first, with the script beside this file:

```
../../../.codefall/shared/run-agent.sh <name> <prompt-file> <schema-file> <out-file>
```

Read its exit code and act on it:

| Exit | Meaning | Do |
| --- | --- | --- |
| `0` | The agent answered; the out-file holds it | Stop walking; read the answer |
| `70` | The entry is on `current` | Run a subagent of this harness with the same prompt yourself |
| `69`, `64` | Not runnable here: the harness is not on PATH, or the name is unknown | Skip to the next entry |
| `73`, `75`, `76` | Ran and failed: wrote nothing, timed out, or exited non-zero | Advance to the next entry, with the failure folded into its prompt |

An answer that parses but says the agent cannot settle the question advances the walk the same way
a failure does. An answer the verb disagrees with does not: that agent answered.

Each entry is tried once, and the walk ends at the end of the order. No second pass because the
first was inconclusive. When no entry answered, say so and stop, exactly as when the one reviewer
failed before there was an order.

**The report names every agent tried**, in order, with why each was skipped or failed, and which
one answered. A findings file records the one that answered as its reviewer. A configured fallback
that is reported this way is not a silent fallback.

## `via=` overrides for one run

An invocation may name its agent directly: `via=<name>` for an entry in the list, or
`via=<harness>[:<model>]` to run a harness with no entry at all. The override replaces the resolved
order for that run with that one agent, and writes nothing. A configured name wins when the two
forms collide, so `via=codex` runs the project's `codex` entry when it has one.

## An agent proposes

An agent run this way is read-only, and the script starts each harness in the mode that enforces it.
It proposes; this session decides. Its answer never writes a file, never settles a hard-to-reverse
choice, and never stands in for the person the verb's own rules name: a disagreement with the user
is a preference and not a question for another agent, and a missing precondition is an exit and
not a question at all. One round per stuck point, which may put the question to several agents at
once; never a second round because the first was inconclusive.

## Consulting

A consult is one question a run cannot settle on its own, put to the order under `consult.agents`
(resolved as above) with the files that bear on it and the options the run sees. The prompt is the consult prompt
beside this file (consult-prompt.md), rendered once, and the answer follows the consult schema
beside it (consult.schema.json): an `answer`, a `confidence` of `high`, `medium`, or `low`, the `reasoning` with the files
that decided it, whether the answer is `reversible`, and `cannotSettle`. Walk the order as above; an
answer with `cannotSettle` true or `confidence` low advances the walk the way a failure does, and
the first answer that does neither ends it. One pass, one question, first answer wins.

What a verb does with the answer is the verb's rule, and every verb holds to three things. The
session decides: a consult proposes, and a `high` answer on a reversible choice is still confirmed
with the user like anything else written. A consult never settles a hard-to-reverse choice, never
writes an ADR, and is never asked about a preference the user has stated. The report names every
agent consulted, what each said, and what the session did with it; where the answer reaches a
document or a bead, the line that records the decision names the consult that informed it.

The order's default is one entry on `current`, so a project that has configured nothing still
consults its own harness's subagent: a fresh context reading the same files, which is worth having
at no configuration cost. What configuration adds is a second model.

