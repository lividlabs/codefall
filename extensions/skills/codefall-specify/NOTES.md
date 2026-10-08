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

## Assumptions, Out of scope, Open questions are three sections

Keeping them separate is what makes each useful: an assumption is a call made without asking that the
user can overrule; out of scope is a decision made together; an open question is something nobody
knows yet. An assumption is legitimate because the interview cannot ask forty questions, and it is
not a licence to skip push-back.
