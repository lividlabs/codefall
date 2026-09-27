# Consulting during a design

Read at step 5, when a technical point stays unsettled after research. The procedure every verb
shares is *Consulting* in `../../../../.codefall/shared/running-agents.md`; this is what design puts
into it and what it does with the answer.

## When

After the concern has been raised once and the user has not settled it, and after the research the
step already calls for has been done: the code read, the library looked up, the ADRs checked. A
consult is for a judgment between approaches that survives that reading, never a substitute for it.
Once per unsettled point, in the same session, before the point goes into the document.

Never consult on a preference the user has stated — that is a decision, and the run follows it. Never
consult on a hard-to-reverse choice to make it: a new dependency, a schema other components will
build on, a rejected alternative that cost real analysis. Those are the ADR triggers, and an ADR is
the user's to ratify; a consult may inform the analysis the user reads, and that is all.

## The question

Render `../../../../.codefall/shared/consult-prompt.md`: `QUESTION` is the technical point, in the
words the concern was raised in; `FILES` are the components the design touches and the ADRs that
bind them; `CONTEXT` is the spec requirement the point serves and the approach so far, in a
paragraph; `OPTIONS` are the approaches weighed, each with what the run thinks it costs; `PRIOR` is
an earlier agent's failure, or empty; `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`.
Run the order under `consult.agents` with `../../../../.codefall/shared/run-agent.sh`, as
`running-agents.md` says. With nothing configured the order is one entry on `current`, and the
consult is a subagent of this harness reading the same files fresh.

## What the answer does

| The answer | The choice | What happens |
| --- | --- | --- |
| `high`, `reversible` true | Not an ADR trigger | Propose it to the user as the run's own recommendation, naming the consult; on yes it goes into the document like any decision |
| `high` or `medium`, `reversible` false | Any | The analysis goes to the user with the concern; the user decides, and a hard-to-reverse choice still becomes an ADR by the usual path |
| `medium`, `reversible` true | Not an ADR trigger | The same as above; a medium answer is worth reading, not worth taking unasked |
| `low`, or `cannotSettle` | Any | The walk advanced; when no agent answered, the point goes into the document as a stated risk with the options and costs the run and the consults laid out |

Whatever the row, nothing is written without the user confirming it, as step 7 already requires.

## The record

Where a consult informed a decision that reaches the document, the sentence recording the decision
names it: *chosen after consulting `architect` (codex), which found the session lookup already
serialises on the store.* The report at step 11 lists every consult: the point, who answered, who
was skipped or failed, and which row above it took.
