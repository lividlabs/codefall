# The interview's harder moments

Read at step 5, when an answer is vague or names another product, and at step 7, when the set of
requirements looks like two specs.

## Contents

- Vague answers
- "Like $COMPANY does it"
- Cohesion and splitting

## Vague answers

Name the vague word and ask for a concrete replacement:

| They said | Ask |
| --- | --- |
| "fast" | Faster than what? What latency is acceptable, and at which percentile? |
| "good UX" | What does good look like here — a reference product, a specific interaction? |
| "manage" | Which actions: create, edit, delete, reorder, archive? |
| "integrate with X" | Which part of X — their search, their booking flow, their SSO? |
| "like before" | Like which screen, which flow? Walk me through it. |
| "just works" | What is the success path, and what is the failure path? |
| "real-time" | Under a second, or under a minute? |

## "Like $COMPANY does it"

Offer once to look it up. On yes, summarize only the patterns that matter and confirm the summary
with the user before it reaches the document.

## Cohesion and splitting

A spec holds one cohesive feature: its requirements share a consumer and a purpose. Requirements
that share nothing but the session they were written in are two specs.

- **Push back once when a spec looks incohesive**, with a specific alternative — not "this is
  large" but "requirements one through three are about exporting and four and five are about
  sharing permissions; those look like two specs to me."
- **Then defer.** If the user disagrees, write what they asked for.
- **Split results are siblings, not a parent and children.** `SPEC-003`, `SPEC-004`, and `SPEC-005`
  sit alongside each other; the vision above them is what groups them. Do not invent a parent spec.
- When a sibling deserves its own interview, say so at the report and ask whether to write it now,
  rather than writing a thin document in passing.
- Record the concern in the spec **only** when the user did not engage with it, phrased as an
  observation for `codefall-design` to weigh. If they considered it and disagreed, nothing goes in.
