# The engineering review

Read at step 10, once the recap is confirmed and the criteria are drafted, before the document is
composed. The only human in the run is the person who wants the feature; the engineer who would
read their spec and ask questions is a subagent. This file is what the subagent gets, what it looks
for, and what the session does with each thing it finds.

## Contents

- When it runs
- The subagent
- What it gets
- What it checks
- The gate on the mechanism check
- What comes back
- What each finding becomes
- The kept mechanism
- One round, then report
- Two worked examples

## When it runs

Once per spec, in the main flow, after step 9 and before step 11. It does not run in the Other
modes, and the edits its findings produce are not reviewed again. It runs whoever the persona is;
the persona changes only how the findings are put to the person.

## The subagent

One subagent of this harness, launched with `Agent`, prompted from `../engineering-review-prompt.md`
by plain substitution of every `{{…}}` placeholder. It is read-only: it reads the draft and the
project, and its final message is one JSON object. It is not the `review` or `consult` list of
`.codefall/settings.json`; those lists belong to
`../../../../.codefall/shared/running-agents.md`, and this review is the skill's own subagent,
launched the way `codefall-implement` launches a worker. The persona is never rendered
into the prompt.

The subagent proposes. It never writes a file, never addresses the person, and never settles
anything: every finding is a question for the person, and the session asks it.

## What it gets

| Placeholder | What the session renders |
| --- | --- |
| `{{SPEC_DRAFT}}` | The draft as it stands: the stories, every criterion with its identifier, Key entities, Existing behavior preserved, Assumptions, Out of scope |
| `{{CONSUMER}}` | Who the consumer is, in a sentence, and whether it is a person or a system |
| `{{VISION_PATH}}` | The vision's path when one frames the work, else `none` |
| `{{ARTIFACTS}}` | The files the audit at step 6 read, one path per line, with the surface each one is |
| `{{STANCE_FILES}}` | `docs/adrs/`, every `AGENTS.md` in the tree, and `docs/decision-log.md` where they exist |
| `{{MECHANISM_CHECK}}` | `on` or `off`, per the gate below |
| `{{REPO_ROOT}}` | The absolute path of the project |

## What it checks

Five things, each a reading of the draft against the code and the project's stance:

| Kind | The question it puts to the code |
| --- | --- |
| `contradiction` | Two criteria that cannot both hold once the system's behavior is known: one removes what another assumes, one gates an outcome a second path reaches, two touch the same thing and say different things |
| `existing-data` | A criterion that adds or collects data the system already holds elsewhere, under another name or in another part of it |
| `unmet-assumption` | A criterion that assumes a fact, a state, or a capability the system does not have, and the spec does not add |
| `broken-mechanism` | A workflow or behavior the criteria describe that cannot deliver its outcome, is far larger than its outcome needs, or goes against an ADR or a scoped `AGENTS.md` |
| `mechanism` | A criterion that describes how something is done rather than what the consumer gets, in product words that pass the vocabulary rule — "emails a code they type back" rather than "confirms it is them" |

The first row is the read the skill used to do itself; it moves here so there is one engineer's
read, not two. The fifth row runs only when the gate allows.

## The gate on the mechanism check

A workflow is sometimes the feature. An expense policy is a sequence of approvals, and the
sequence is what the person wants. A refactor, an architectural change, or new infrastructure
written by an engineer or a technical product manager has a system as its consumer, and the
mechanism is the deliverable. Flagging those as "a mechanism, not an outcome" is noise.

`{{MECHANISM_CHECK}}` is `on` only when **both** hold: the consumer is a person, and the persona is
`product-manager`. Otherwise it is `off`, and the subagent skips the fifth row. The other four run
regardless.

When it is on and a `mechanism` finding comes back, the question to the person is in two parts:
what does this get the user, and is the mechanism the thing you want or how you pictured it. "The
workflow is the point" ends it in one round, and the criterion stays as written.

## What comes back

The subagent's final message is one JSON object:

```json
{"findings": [
  {"kind": "contradiction",
   "criteria": ["SPEC-004-REQ-02-AC-03", "SPEC-004-REQ-03-AC-01"],
   "evidence": "src/accounts/decline.ts:41 deletes the account row; lists reference owner_id",
   "reason": "a pending account can own lists and then be removed, which orphans them",
   "question": "What happens to the lists and tasks of a pending account that is declined?"}
]}
```

`criteria` names the identifiers involved, in full. `evidence` is where in the code or the stance
the finding rests, with file and line. `reason` is one sentence an engineer would say. `question` is
the thing to ask, in neutral words; the session rephrases it for the person. An empty `findings`
list is a result.

## What each finding becomes

Each finding is one flag to the person, put the way step 5 puts a design concern: name it once,
say why, let them decide, two rounds at most. The `evidence` is for the session, not the person: the
flag names the surface and what it does there — "the landing page a shared link opens also creates
an account" — and the file follows that only when the person would need to open it. Under the
`product-manager` persona the evidence is restated as a product fact and no file is named.

| Kind | The question | The answer becomes |
| --- | --- | --- |
| `contradiction` | Which should hold, or what happens when both do | A criterion changed, or one added under the requirement that resolves it |
| `existing-data` | Is this the same thing the system already has, or different | A Key entity naming the existing noun, a criterion reframed around it, or an Out of scope line |
| `unmet-assumption` | Does this feature add what it assumes, or does something else | A criterion that adds it, a dependency under the story, or an Out of scope line |
| `broken-mechanism` | Here is why this cannot do what you want, or costs more than it should; what is the outcome you need | The criterion rewritten as the outcome, or kept with the line below |
| `mechanism` | What does this get the user, and is the how the point | Kept as written when the how is the point; otherwise rewritten as the outcome |

A question the person cannot answer goes under **Open questions** as any other does, with the
finding's reason appended. Nothing is patched into the draft silently: every edit is one the person
took.

## The kept mechanism

When the person keeps a mechanism the review flagged, the spec says so, in one line under
**Design notes** naming the requirement:

> SPEC-004-REQ-02: the approval sequence is a product requirement, kept after review; plan builds
> it as written.

Without that line `codefall-plan` reads the mechanism as an accident and may swap it for something
simpler. Nothing is recorded when the person rewrites the criterion as an outcome; the rewrite is
the record.

## One round, then report

One review per run. The criteria the person edits in answer to a finding are not sent back for a
second read; the person saw the edit and took it. Before step 11, say what the review found in a
sentence or two — how many findings, of which kinds, and what each became — and say "the review
found nothing" when that is the result. The report at step 14 carries the same.

## Two worked examples

**A gate with a second door.** The feature makes account creation invite-only. The criteria cover
the sign-up page. The search at step 3 found a second place accounts are created: the landing page a
shared link opens, which lets a newcomer create an account and signs them in. The audit at step 6
surfaces it as a path to the gated outcome, and the review catches it again as a `contradiction`
between the gate and the path. The question to the person is whether the gate applies to shared-link
sign-ups, and the answer is a criterion either way.

**A removal that orphans something.** One criterion lets a pending account sign in while the gate is
off. Another says that declining a pending account removes it. Read as a pair, they mean a pending
account can own lists and tasks and then be removed. The code shows that lists and tasks
reference their owner, so the consequence is a real table and not a worry. The review returns it as
the `contradiction` in the JSON above; the question to the person is what happens to that activity,
and the answer is a criterion under the decline requirement.
