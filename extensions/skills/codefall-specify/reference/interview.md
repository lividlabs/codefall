# The interview's harder moments

Read at step 5, before the first question, when an answer is vague or names another product, and
when a visual surface comes up; and at step 7, when the set of requirements looks like two specs.

## Contents

- Ask, do not propose
- Vague answers
- Design concerns to flag
- The mockup as an interview instrument
- "Like $COMPANY does it"
- Cohesion and splitting

## Ask, do not propose

A preference is the person's to state: who the consumer is, what they can do, what they see, what
happens on failure, what is out of scope. Ask it open. A question that arrives with a suggested
answer — "I'd suggest newest first; does that match?" — gets a confirmation instead of an answer,
and the spec ends up carrying the interviewer's defaults.

The two exceptions are both facts, not preferences: a question of fact the person cannot answer is
consulted on per `consulting.md` and offered as a proposal they confirm, and a fact the audit found in
the code is stated as a fact before the question it raises. When the person says they do not know a
preference, the answer is an **Assumption**, written so they can overrule it, never a proposal they
are asked to accept.

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
| "real-time" | Under a second, or under a minute? The two are different features. |

## Design concerns to flag

Beyond vague words, push back when what the person pictures has a problem they may not have seen.
Phrase it as a flag — "One thing I want to flag — …; how are you thinking about that?" — never a
ruling. Name it once, let them decide, cap at two rounds; what stays unresolved goes under **Open
questions**. If they considered it and kept their answer, nothing is recorded.

| They picture | Flag |
| --- | --- |
| A destructive action reached only by hovering | Touch has no hover. How does a phone user reach it? |
| A modal for the main flow | A modal takes the user out of what they were doing. Is the interruption the intent, or would it work in place? |
| One form with many fields | Are any of these optional or conditional? Is there a natural split, or a reason it has to be one screen? |
| "Make it a dashboard" | What does the user do on it? A dashboard nobody acts on is wallpaper. |
| An interaction that needs data nobody collects | That needs X to exist. Is X somewhere already, or is collecting it part of this feature? |
| "Real-time" without a number | Under a second and under a minute are different features. Which one is this? |
| A step the user has to repeat every time | Is the repetition the point, or should the system remember? |

## The mockup as an interview instrument

A visual surface is interviewed with the mockup open, not described and drawn later. When one comes
up and the person has no mockup, run `codefall-mock-up` for the surface then, on the first screen
only, and bring it back into the interview. Each pass makes one or two changes and asks one design
question, and each answer is interview material that lands in a criterion, an edge case, or
**Design notes**:

- Where does the primary action live, and what is secondary?
- What does this look like with nothing in it yet?
- What does the user see when it fails, and what can they do next?
- What differs on a phone?
- What happens to the longest realistic content — a name that does not fit, a list of hundreds?
- Which of these states does not apply? Take the answer and move on.

Ask for the empty state and the primary failure; they are the two nobody describes unprompted. Stop
iterating when the person says it is right, and say what the mockup settled when you recap at step 9.
`codefall-mock-up` owns the files, their states, and the `README.md`; the interview owns the
questions. A mockup the person already has is imported per
`../../../../.codefall/shared/import-mockup.md` and read the same way:
walk through it with them and ask the questions above about what it shows.

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
  observation for `codefall-plan` to weigh. If they considered it and disagreed, nothing goes in.
