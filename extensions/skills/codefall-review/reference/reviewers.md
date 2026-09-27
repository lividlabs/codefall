# Who reviews, and how the lenses are run

Read at step 3, before the review starts.

## Contents

- Who reviews
- Lens groups
- Subagents
- Another agent
- Consulting on `notChecked`

## Who reviews

The project's agent order, resolved at step 1 per `../../../../.codefall/shared/running-agents.md`
— `agentsByHarness.<harness>`, else `review.agents`, else the top-level `agents`, else one entry,
`subagent` on `current` — and walked at step 3. The first agent that answers is the reviewer. Two
entries are special:

- **`current`** is a subagent of this harness: one per lens group, in parallel, merged here. With
  nothing configured it is the whole order, so a project that has not chosen still reviews the way it
  always did.
- **This session** is never a config entry. It is a run-time choice, acceptable only when the work
  under review came from somewhere else, and the report says so.

`via=` replaces the order for one run and takes a configured name or a raw harness with an optional
model:

```
via=architect        a name from the project's agents list
via=codex            via=codex:gpt-5-codex
via=claude           via=claude:claude-opus-5
via=muse             via=muse:muse-spark-1.3-contributor
via=opencode         via=opencode:anthropic/claude-sonnet-5
via=agy              via=agy:<model>
via=gemini           via=gemini:gemini-3-pro
```

A configured name wins when the two forms collide. The five harness names are the ones a project may
configure; `gemini` runs by the raw form only. The model strings are examples and will age —
whatever the harness accepts is passed through untouched.

## Lens groups

| Group | Lenses |
| --- | --- |
| 1 | `correctness`, `failures`, `behaviour` |
| 2 | `tests`, `types` |
| 3 | `conventions`, `comments`, `docs`, `simplify`, `local` |
| 4 | `security` |

A document target runs two groups: the mechanical checks — `structure` and `status` — and the
reading, which is everything else for that kind.

| Reviewer | How the lenses are run |
| --- | --- |
| `current` | One subagent per group, in parallel |
| This session | The groups in order, as separate passes |
| Any other agent | **One call carrying every lens in scope** |

A group whose lenses were all dropped at the confirmation does not run.

## Subagents

Each is prompted from `../reviewer-prompt.md`, rendered by substituting `{{TARGET}}`,
`{{REVISION}}`, `{{LENSES}}`, `{{MATERIAL}}` and `{{SCHEMA}}`. **Each returns JSON against
`../findings.schema.json`, and this session merges them.**

- **Unparseable JSON** gets one retry, with the parse error folded into the prompt. A second failure
  drops that group: name it in `notChecked` and carry on with the rest.
- **Overlapping findings.** Same file, overlapping line range, and the same claim: keep the higher
  severity and drop the duplicate. Different claims on the same lines are different findings and
  both stay.

## Another agent

```
../../../../.codefall/shared/run-agent.sh <name>|<harness>[:<model>] <prompt-file> <schema-file> <out-file>
```

Runs the agent's harness in its headless read-only mode in the repository, so the reviewer reads
the files itself. A name is resolved from `.codefall/settings.json`; the raw form bypasses it.
Leaving the model off takes the harness's own default — which is what `via=codex` with no model
means. The prompt file is `../reviewer-prompt.md` rendered with every lens in scope. Codex and
Claude Code also take the schema as a flag — `--output-schema` and `--json-schema` — which makes
their output conform by construction. Muse has such a flag and the script does not pass it: its
validator rejects the schema's `if`/`then` clause, so Muse reads the schema from the prompt like
OpenCode, Gemini, and agy. The exit code decides the walk, per `running-agents.md`: `0` answered;
`70` the entry is `current`, run the subagents; `64` and `69` not runnable here, skip it; `73`,
`75`, and `76` ran and failed, advance with the failure folded into the next prompt.

- The external reviewer runs read-only. It proposes; it never edits.
- **An unauthenticated harness is a failure, not a skip.** No harness reports it before the prompt
  is sent, so it exits `76` with its own error, the order advances, and the report says so with the
  harness's output. When the order ends with no answer, stop, report every agent tried, and offer
  this session as the reviewer. Never fall back silently, and never past the end of the order.

## Consulting on `notChecked`

After the reviewer returns and before the files are written, read its `notChecked`. Some entries
are gaps in the material — a missing hop, a dropped lens, a file it could not reach — and those
stay as they are. An entry that is an unsettled question about the target — "could not tell whether
the retry loop can run twice on one message" — gets one consult, per *Consulting* in
`../../../../.codefall/shared/running-agents.md`.

**The question**, rendered into `../../../../.codefall/shared/consult-prompt.md`: `QUESTION` is the entry in the
reviewer's words; `FILES` are the files it names, or the target's changed files when it names none;
`CONTEXT` is the target and its revision, and which lens raised it; `OPTIONS` are three — it is a
defect, with the severity the run would give it; it is not a defect; the repository does not say.
`PRIOR` is an earlier agent's failure, or empty. `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`. Walk the
`consult` order with `../../../../.codefall/shared/run-agent.sh`.

**What the answer does.** A consult never becomes a finding on its own. An answer of `high`
confidence that names a defect and cites the file and line lets this session promote the entry to a
finding — `open`, the severity the answer argued for, the consult's name and one sentence of its
reasoning in the finding's `consult` field — after this session has read the cited lines and agrees.
Any other answer leaves the entry in `notChecked`, with the consult's view appended: *consulted
`architect` (codex): not a defect, the queue is single-consumer (`queue.go:41`)*. When no agent
answered, the entry stays as the reviewer wrote it.

One consult per entry, one pass over the order, and the report names every consult beside the
reviewer. A promoted finding is triaged like every other one; the consult does not decide its
status.
