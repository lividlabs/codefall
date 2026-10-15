# codefall-specify — where it came from

Why `SKILL.md` looks the way it does. None of this is instruction — the skill is the instruction.
This exists so nobody re-adds something that was removed on purpose.

## EARS

The Easy Approach to Requirements Syntax was published by Alistair Mavin and colleagues at
Rolls-Royce in 2009 and is used unchanged here: zero or many preconditions, zero or one trigger, one
system, one or many responses, which yields the six patterns in `reference/specification.md`. The
two rules the skill insists on — pick the pattern that fits rather than writing everything as
`WHEN … THEN`, and use `IF … THEN` for failure — exist because those are the two errors every
first-time EARS author makes, and the second one hides the failure paths that an implementer most
often finds missing.

## Status describes the document

`codefall-specify` deliberately diverges from `codefall-envision`, which carries an `Active`
state because a vision has no tracker representation to carry work state. A spec has one — its
mirrored issues — so the tracker says whether work is queued, underway, or done, and a status line
would only approximate it. `codefall-plan` takes the same rule for the same reason, with Beads in
the tracker's place.

## Mockups are keyed by surface

A mockup is a view of a surface, not of a specification. One screen is touched by several specs over
its life, sometimes across several visions, and it outlives any one of them. Filing it under the
spec that arrived first makes the second spec either duplicate it or reach into another spec's
directory.

## The recap is substantive on purpose

A recap line that collapses to a fragment or a dash means that topic was not interviewed. The recap
checks the interviewer's work as much as the user's, which is why every line must be at least a
sentence before the skill advances.

## Silent omission is not a deletion

Authoring time is the only cheap moment to enforce it. By implementation time nobody remembers the
specification was written against a stale picture of the surface, and an implementer who takes the
document literally removes what it does not mention.

## The interview took its repertoire from a project's own describe skill

On 2026-10-14 a user compared `codefall-specify` with a skill their project had written for the same
job and found the interview thinner: it asked fewer questions, and not the ones that mattered. The
two topic lists were nearly the same. What that skill had and this one lacked was taken: the design
concerns to flag, which had been cut to one example; UI preferences as a topic in their own right;
and the mockup drawn during the interview and iterated with the person, each pass asking a design
question, which is where most of the missing questions had been. The eval transcript also showed
the interviewer leading with a suggested answer on five questions out of six, which a scripted
product manager accepts and a real one experiences as being asked to agree; the rule that a
preference is asked and never proposed came from that.

Not taken, on purpose: the schema-placement check and the build-boundary splitting rule, which
decide where data lives and how tickets are cut and so belong to `codefall-plan`; a priority
question, which is the tracker's; and the project's directory names, glossary, and board, which a
generic skill cannot carry and which a project adds through `CUSTOMIZE.md` as interview topics.

## The engineering review is a step, and the engineer is a subagent

The schema-placement question, and the others an engineer asks when a product manager runs a draft
by them — do two of these contradict each other, does this assume something we do not have, is this
workflow going to do what you think — were first proposed as plan's to ask, with the answers parked
as open questions or handed back to specify. The user's correction: when an engineer asks, the
product manager answers and the requirement changes, in the same conversation. So the review is a
step of specify, before the document is composed, where rewording a criterion is still free because
nothing cites it yet; the engineer is one subagent of the current harness, read-only, proposing;
and every finding is a flag the person decides. It is not on the `review` or `consult` list in
settings, because it is the skill's own reader and not a configurable second opinion, the same way
implement's workers are the skill's own and not on those lists.

The check for a mechanism dressed as an outcome is gated to a person as consumer under the
`product-manager` persona, because sometimes the workflow is the feature — a travel policy is a
sequence of approvals — and a refactor or new infrastructure specified by an engineer has the
mechanism as its deliverable. It is a flag and never a ban; a mechanism the person keeps is recorded
so plan builds it as written. The step 10 read of the criteria against the code, which landed a week
earlier, became the review's `contradiction` row rather than a second read beside it.

## Assumptions, Out of scope, Open questions are three sections

Keeping them separate is what makes each useful: an assumption is a call made without asking that the
user can overrule; out of scope is a decision made together; an open question is something nobody
knows yet. An assumption is legitimate because the interview cannot ask forty questions, and it is
not a licence to skip push-back.
