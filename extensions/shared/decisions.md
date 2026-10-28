# Decisions

Shared procedure. Every verb follows it. It replaces the rule that a verb applies only what the
user takes: a run decides what it can give a reason for, consults on what it cannot, records every
decision where the next reader finds it, and stops only when it cannot continue. The pull request a
person merges is the gate that was always there, and the record below is what they read before
merging.
[ADR-013.7](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-013.7-deliveries.md)
holds the reasoning.

## Contents

- What a run decides
- What a run does not decide
- Consult, then a human bead, then continue
- The record
- Gates and confirmations

## What a run decides

A recommendation is applied and reported, never asked, when three things hold:

- **The run can state its reason in a sentence.** "Fixing all four: each is on a line the epic
  changed." "Epic branch: twelve beads in three waves." "Moving six P4 cleanups to the checkout
  followup: none is in code this epic touches."
- **The action is reversible.** A bead filed, moved, reopened, or closed with a reason; text in a
  document on a branch; a fix on a branch; a strategy for one run; a default the project's own
  stance gives, which the ADRs, the `AGENTS.md` files, and what the project already uses supply.
- **It stays inside the run's scope.** What the verb was asked for, and the artifacts the verb owns.

So "I found N problems. Fix them all? (I recommend yes.)" is "Fixing all N", with the list and the
reason, and the beads are filed. A triage proposal is applied. The waves, the landing strategy, and
the agent per bead are decided and shown as the plan. A plan revision's three outcomes are taken
with the reason each. A `.ignore` line is appended and said.

## What a run does not decide

Three kinds of question are a person's, and a run never takes silence as an answer to one:

- **A product decision.** What a user gets, which of two behaviours is wanted, what a requirement
  means where the spec does not say. Under the `product-manager` persona these are what `plan`
  parks under *Decisions needed*.
- **A hard-to-reverse choice the project's stance gives no default for.** A new dependency, a
  schema other components will build on, an ADR that would supersede one, a deletion of data. Where
  the stance does give a default, the default is taken and recorded, and `plan` writes the ADR
  from it as it does on "settle now".
- **A preference the person has stated**, in the conversation, in `CUSTOMIZE.md`, or in a
  document. It is followed, never overridden, and never consulted on.

A missing tool, runner, or tracker profile is none of these: it is an exit, as it always was.

## Consult, then a human bead, then continue

For a question that is not the run's to decide:

1. **Consult once**, per *Consulting* in the running-agents reference beside this file: the
   entry's `consult` list walked in order, first answer wins. The list is where a project puts its
   strongest model first, with `current` after it as the fallback.
2. **An answer that settles it is applied** and recorded as below, naming the consult.
3. **An answer that cannot settle it still names the option it would take**, which the consult
   prompt asks for. File a **human bead**: in a delivery a child of the epic, parentless otherwise;
   labelled `human`; its title the question; its body what is asked, the options, the consult's
   recommended default and its reasoning, and the default the run took; `-p 2`, `-s open`. Then
   take that default and continue.
4. **Stop only when the run cannot continue**: the question blocks every piece of work the run has
   left, the whole batch or the one bead. The report names the human bead and the way to resume.

```bash
bd create "Signed payload or session lookup for the stage context?" \
  --id "$(bd config get issue_prefix)-human-stage-context" -l human -p 2 -s open \
  -d "Asked by: round 2 implement, T3. Options: … Consulted codex:astra, cannotSettle: … would take the signed payload because … Taken: the signed payload, in PR #104."
bd update "$(bd config get issue_prefix)-human-stage-context" --parent <epic>
bd dolt push
```

A human bead is never claimed by a round and never in a batch; a bead that depends on its answer is
left out of the batch and the report says so. It closes when the person answers it, with the answer
as the reason, and a different answer from the default taken is new work, filed as triage says.

## The record

Every decision a run makes under this policy is written where the next reader finds it, on the
bead it is about, in that bead's notes, append-only, one line each, naming who decided and when:

```bash
bd update <bead> --append-notes "decided (session) [2026-10-28]: kept the retry loop; the plan's § Error Handling asks for it"
bd update <bead> --append-notes "decided (consult codex:astra) [2026-10-28]: signed payload over session lookup; no store to expire"
```

A decision with no bead of its own, a landing strategy, a plan amendment, which items a round
takes, goes on the epic's notes in the same form, or on the document's pull request when there is
no epic. The verb's report lists every decision it made. Comments keep what they held before: the
pull request link on a task bead, and the consult record on a bead that failed. Always
`--append-notes`; plain `--notes` replaces the whole log.

## Gates and confirmations

A confirmation is a question when the person is present and the artifact is theirs to shape: the
document an interview produces, shown before its first write in `envision`, `specify`, `report`,
`scaffold`, and `equip`, and a plan's draft when a person ran `plan`. Everywhere else it is a
report: it says what the run will do, and does it. That covers every gate after a delivery's one
gate, every verb run by another verb, every offer, and every "fix them all". The one gate of a
delivery is `implement`'s go gate on an epic, and its skill says what it shows and asks.

A gate turned into a report still carries its conditions. The permissions condition, that nobody
is there to answer a prompt, so the session must allow edits and the shell without asking, is
stated and checked, and a run that cannot meet it says so and falls back the way its skill says.
