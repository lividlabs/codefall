# Consulting during a plan

Read at step 5, when a technical point stays unsettled after research. The procedure every verb
shares is *Consulting* in `../../../../.codefall/shared/running-agents.md`; this is what plan puts
into it and what it does with the answer.

## When

After the concern has been raised once and the user has not settled it, and after the research the
step already calls for has been done: the code read, the library looked up, the ADRs checked. A
consult is for a judgment between approaches that survives that reading, never a substitute for it.
Once per unsettled point, in the same session, before the point goes into the document.

Never consult on a preference the user has stated — that is a decision, and the run follows it. A
hard-to-reverse choice, a new dependency, a schema other components will build on, a rejected
alternative that cost real analysis, is an ADR trigger, and what happens to it follows
`../../../../.codefall/shared/decisions.md`: the project's own stance gives the default when it
gives one, and the ADR is written from it; otherwise it is the person's, and a consult may inform
the analysis they read, and that is all.

## The question

Render `../../../../.codefall/shared/consult-prompt.md`: `QUESTION` is the technical point, in the
words the concern was raised in; `FILES` are the components the plan touches and the ADRs that
bind them; `CONTEXT` is the spec requirement the point serves and the approach so far, in a
paragraph; `OPTIONS` are the approaches weighed, each with what the run thinks it costs; `PRIOR` is
an earlier agent's failure, or empty; `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`.
Run the entry's `consult` list with `../../../../.codefall/shared/run-agent.sh`, as
`running-agents.md` says. With nothing configured the list is one `current` agent, and the
consult is a subagent of this harness reading the same files fresh.

## What the answer does

| The answer | The choice | What happens |
| --- | --- | --- |
| `high`, `reversible` true | Not an ADR trigger | Taken as the run's own decision, naming the consult, per `decisions.md`; it goes into the document like any decision |
| `high` or `medium`, `reversible` false | Any | A choice the project's stance gives a default for takes the default and becomes an ADR by the usual path; otherwise the analysis goes to the person with the concern, or into *Decisions needed* |
| `medium`, `reversible` true | Not an ADR trigger | Taken when the run can state the reason in a sentence; otherwise the analysis goes to the person with the concern |
| `low`, or `cannotSettle` | Any | The walk advanced; when no agent answered, the point goes into the document as a stated risk with the options and costs the run and the consults laid out, and the forced option the last consult named is the default a human bead carries |

When a person ran this skill, step 7 shows every decision before anything is written.

## The record

Where a consult informed a decision that reaches the document, the sentence recording the decision
names it: *chosen after consulting `codex:gpt-5-codex`, which found the session lookup already
serialises on the store.* The report at step 11 lists every consult: the point, who answered, who
was skipped or failed, and which row above it took.
