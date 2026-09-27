# Consulting during the interview

Read at steps 5 and 7, when a question the interview needs answered is one the user cannot answer.
The procedure every verb shares is *Consulting* in `../../../../.codefall/shared/running-agents.md`; this is what specify
puts into it and what it does with the answer.

## When

Only for a question of fact the user does not have: how the existing system behaves in a case the
audit did not settle, what a term of the domain means in this codebase, whether a dependency the
feature needs exists yet. The user has said "I don't know", or the question is about the code and
they are not the person who wrote it. Once per such question, in the same session, before the
question goes under **Open questions**.

Never consult on a preference — who the consumer is, what they should be able to do, what is out of
scope, how a failure should read. Those are the user's answers, and a vague one is pushed back on,
not looked up. Never consult on a design concern raised as a flag: the flag is the user's to weigh,
and `codefall-design` decides the how.

## The question

Render `../../../../.codefall/shared/consult-prompt.md`: `QUESTION` is the interview question in the words it was asked;
`FILES` are the artifacts the audit at step 6 found for the surface, and the vision when one frames
the work; `CONTEXT` is the consumer and the capability so far, in a paragraph; `OPTIONS` are the
answers the run can see, or the one line "none seen" when it cannot; `PRIOR` is an earlier agent's
failure, or empty; `SCHEMA` is `../../../../.codefall/shared/consult.schema.json`. Run the order under `consult.agents`
with `../../../../.codefall/shared/run-agent.sh`, as `running-agents.md` says. With nothing configured the order is one
entry on `current`, a subagent of this harness reading the same files fresh.

## What the answer does

The answer is offered in the interview as a proposal, naming the consult, and the user confirms or
corrects it before it reaches the document:

> Consulting `architect` (codex): the profile screen already blocks a delete while an export is
> running, in `profile/actions.ts:88`. So the failure path here is "the delete waits", not "the
> delete fails". Does that match?

| The answer | What happens |
| --- | --- |
| `high` or `medium`, and the user confirms | It is the user's answer now; the criterion or the section is written from it, and the sentence recording it names the consult |
| `high` or `medium`, and the user corrects it | The user's answer stands; the consult is not mentioned in the document |
| `low`, or `cannotSettle`, or no agent answered | The question goes under **Open questions** as it would have, with the consults' view appended when there is one |

Nothing a consult said is written unconfirmed, and nothing in **Open questions** is written as
settled because an agent guessed. The report at step 13 lists every consult: the question, who
answered, who was skipped or failed, and which row above it took.
