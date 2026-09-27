# Who reviews, and how the lenses are run

Read at step 3, before the review starts.

## The three reviewers

`via=` picks the third:

1. **A subagent of this harness.** The default. Separates the reviewer from the fixer.
2. **This session.** The reviewer is then also the fixer, which is acceptable only when the work
   under review came from somewhere else. Say so in the report.
3. **Another harness**, named by `via=`.

```
via=codex            via=codex:gpt-5-codex
via=claude           via=claude:claude-opus-5
via=opencode         via=opencode:anthropic/claude-sonnet-5
via=gemini           via=gemini:gemini-3-pro
via=muse             via=muse:muse-spark-1.3-contributor
```

The five harness names are the supported set. The model strings after the colon are examples of the
form and will age — whatever the named harness accepts is what goes there, and a model this file
has never heard of is passed through untouched.

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
| A subagent | One subagent per group, in parallel |
| This session | The groups in order, as separate passes |
| Another harness | **One call carrying every lens in scope** |

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

## Another harness

```
../../../../.codefall/shared/run-agent.sh <harness>[:<model>] <prompt-file> <schema-file> <out-file>
```

Runs the harness's headless read-only mode in the repository, so the reviewer reads the files
itself. Leaving the model off takes the harness's own default — which is what `via=codex` with no
model means. The prompt file is `../reviewer-prompt.md` rendered with every lens in scope. Codex and
Claude Code also take the schema as a flag — `--output-schema` and `--json-schema` — which makes
their output conform by construction. Muse has such a flag and the script does not pass it: its
validator rejects the schema's `if`/`then` clause, so Muse reads the schema from the prompt like
OpenCode, Gemini, and agy. The script's header holds its exit codes: `0` answered, `69` not on
PATH, `73`, `75`, and `76` ran and failed, `70` the agent is this harness's own subagent.

- The external reviewer runs read-only. It proposes; it never edits.
- A failure — missing CLI, auth error, timeout, non-zero exit — is reported with the harness name
  and its output, with an offer to review with this session instead. Never fall back silently.
