---
name: codefall-specify
description: Turn a feature idea into a specification another session can implement — interview for what the user will observe, push back on vague answers, draw the mockup while the questions are still open, audit what already exists, have a subagent review the draft as an engineer, then write requirements with EARS acceptance criteria into a spec document in the repository, mirrored to the issue tracker.
argument-hint: "[what you want to build]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - Skill
  - Agent
  - WebSearch
  - WebFetch
---

# Specify

Turn a feature idea into a specification precise enough that another session can implement it
without re-interviewing anyone. The output is a **spec document in the repository** at
`docs/specs/SPEC-NNN-slug.md`, holding one or more requirements, each with a user story and
numbered acceptance criteria. The issue tracker gets a generated mirror. **The document is
canonical**; the issues are regenerated from it. Specifying is not planning: once the criteria are
written and confirmed, stop.

Paths starting `../` or `trackers/` are relative to this skill's directory, not the user's project;
one through `../../../.codefall/` names a file `codefall init` installed in the project's own
`.codefall/`.

## Files beside this one

Read each at the step named; none is loaded up front.

- `reference/interview.md` — asking rather than proposing, vague answers, design concerns to flag,
  the mockup as an interview instrument, the lookup offer, cohesion and splitting. Steps 5 and 7.
- `reference/audit.md` — the surfaces to audit and what to ask about each. Step 6.
- `reference/engineering-review.md` — the subagent that reads the draft as an engineer: what it
  gets, what it checks, the gate on the mechanism check, what each finding becomes. Step 10.
- `engineering-review-prompt.md` — the prompt rendered for that subagent.
- `reference/specification.md` — the user story, the six EARS patterns, observability, prohibited
  vocabulary, numbering, the optional sections. Before step 7.
- `templates/specs/SPEC.md` — the document template. Step 11. Bracketed instructions are stripped
  on emit.
- `templates/specs/AGENTS.md` — the operative rules this skill installs at `docs/specs/AGENTS.md`.
- `trackers/<name>/PROFILE.md` — the tracker profile, `<name>` being `tracker` in
  `.codefall/settings.json`: issue shape, labels, creating, refreshing, archiving. Steps 3 and 13.
- `reference/consulting.md` — a question of fact the user cannot answer, put to the configured
  agents. Steps 5 and 7. Names `../../../.codefall/shared/running-agents.md`,
  `../../../.codefall/shared/run-agent.sh`, `../../../.codefall/shared/consult-prompt.md`,
  `../../../.codefall/shared/consult.schema.json`.
- `../../../.codefall/shared/import-mockup.md` — bringing a user's mockup into the repository.
  Steps 5 and 8.
- `../../../.codefall/shared/landing.md` — the branch, the commit, the push, and the pull request.
  Steps 12 and 14.

## Scope — what, not how

Specify decides **what will be true when this is done**, never **how it gets built**.

| In scope | Out of scope |
| --- | --- |
| Who the consumer is and what they get out of it | Which layer or module the work lands in |
| What they can observe when it works | File paths, libraries, services, frameworks |
| Domain nouns and what they mean | Their fields, types, relations, or storage |
| What must keep working that already works | API shapes, message contracts, schema |
| What is explicitly not included | Work breakdown, ticket sequencing, dependency edges |
| Which questions remain unresolved | Estimates, sizing, build order |

**You may read the codebase, but only to answer four questions**: does this already exist, what
would this change silently break, what do the criteria leave out, and what in them cannot hold?

## What a specification is

One **spec document** per feature, holding one or more **requirements**, each with a user story —
`As a <consumer>, I want <capability>, so that <benefit>`, the "so that" mandatory — and its own
EARS acceptance criteria. A requirement is the unit that becomes a ticket.

Identifiers nest and are written in full: `SPEC-003`, `SPEC-003-REQ-01`, `SPEC-003-REQ-01-AC-01`.
Three digits for the spec, two for the rest, zero-padded, append-only at every level; criterion
numbers restart under each requirement. The rules for stories, criteria, and the optional sections
are in `reference/specification.md`.

## Status and lifecycle

| Status | Meaning | Lives in |
| --- | --- | --- |
| `Draft — <date>` | Being written. The user stopped and is coming back | `docs/specs/` |
| `Ready — <date>` | Written and agreed. The normal end of a session | `docs/specs/` |
| `Archived — <date>` | Superseded or dropped | `docs/specs/archive/` |

- **Status describes the document, never the work.** Work state is the tracker's.
- **`Ready` is the normal end of a session.** `Draft` only when the user said they are stopping.
- **A spec is never renumbered and its identifier is never reused.** Archiving moves the file under
  the same name; citations still resolve. `Archived` gains a `**Replaced by:** SPEC-NNN — <date>`
  line when something took its place.
- Transitions are this skill's to make. Report the state and offer; never transition on your own.

## The specs directory

`codefall-specify` maintains `docs/specs/AGENTS.md` from `templates/specs/AGENTS.md`: written when
the directory is created, added on a later run if missing. **Never overwrite a file that has
drifted**: show the difference and ask; replace only on a yes, and on a no say nothing further.

## Mockups are keyed by surface, not by spec

Mockups live at `docs/mockups/<slug>/`, the slug naming the surface — `booking-history`,
`trip-share` — **never** a spec or a vision. Specs reference them by path under **Design notes**.
Do not move a mockup into a spec directory, and do not create one.

## Cohesion and splitting

**A spec holds one cohesive feature**: its requirements share a consumer and a purpose. Push back
once when it looks incohesive, with a specific alternative, then defer; split results are siblings
the vision groups, never a parent and children. Rules in `reference/interview.md`.

## Tracker profiles

The specification is tracker-neutral. Where the mirror lands, and in what shape, is a **tracker
profile**, one directory per tracker: `github` (GitHub Issues, optionally with a Project) and
`beads` (no mirror; the document and Beads carry everything) are supported; `jira` and `linear` are
planned, and a planned profile is an exit — say the mirror is not supported yet and stop.

- The profile is the settings' `tracker`; never ask. Under `github`, confirm the repository.
- Read the profile's capability table before writing, and follow its fallbacks.
- **The spec is written even when the mirror fails.** Give the user the exact command to fix the
  tracker and say the mirror is pending. Never discard the spec.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb; a project's `CUSTOMIZE.md` may
add interview topics, asked at step 5 beside the skill's own. Read the `persona=` line of the
preflight report; when it is not `engineer`, follow that persona's section in
`../../../.codefall/shared/personas.md` for this run, and say so.

## Process

### 1. Check preconditions

Run the shared check.

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances to step 2. Otherwise read `beads_reason`, tell the user what is missing, give
the command that fixes it, say to rerun this verb after it, and **stop**:

| `beads_reason` | What is wrong | Give them |
| --- | --- | --- |
| `not_installed` | `bd` is not on PATH | `brew install beads` |
| `not_initialized` | this repository has no beads database | `bd init` |
| `unreadable` | bd found a database and could not read it | quote `beads_detail` |

Then read the checkout lines. `behind` above `0` or `refresh=stale` means the environment may not
match `main`: say so and run `/codefall-refresh` before continuing; `refresh=undeclared` names
`/codefall-equip` instead. **Never run the remedy.** That is the user's decision.

### 2. Ask what they want to build

One open question:

> "What would you like to build? A sentence or two is enough to start."

**Then look for a vision.** If `docs/visions/` exists, read the live visions there — not
`archive/` — and offer the relevant one as context:

> VISION-002 covers the auditing rework and looks like the frame for this. Want me to work from it?

A vision is **never required**; it carries the *why* for the Context section, not acceptance
criteria: its **Proposed shape** is a rough direction, and step 5 still interviews
for everything. Do not lift criteria out of it or treat its **Open questions** as settled.

Do not change the vision's `Status`; work starting is `codefall-implement`'s to record. A `Draft`
or `Ready` vision the interview shows wrong is amended on this run's branch, shown at step 11 and
committed at step 14; a change bigger than text runs `codefall-envision` here, on this branch, under
the step 11 confirmation. An `Active` vision is frozen; say so.
`../../../.codefall/shared/workflow.md` has the rule.

### 3. Check whether it already exists

Three searches before the interview:

- **The specs** — read `docs/specs/`, not `archive/`.
- **The codebase** — Glob and Grep for what they described: filenames, exported identifiers, route
  segments, domain nouns.
- **The tracker** — the profile supplies the search command.

### 4. Report what you found

- **A spec already covers it.** Link it. Ask whether it is the same thing, and whether to revise
  it rather than write another.
- **It exists in code.** Describe what is there, with file paths. Ask whether that is what they
  meant, and what differs.
- **An issue already covers it.** Link it. Ask whether it is the same thing.
- **Nothing found.** Say so and move on.

If existing work covers their need, **stop the skill**.

### 5. Interview

Read `reference/interview.md`.

**Style.** One or two questions at a time, never a wall. Start broad and narrow. Use the user's own
vocabulary. **Ask; do not propose.** A question that arrives with a suggested answer gets a
confirmation instead of an answer. A proposal is for a question of fact only.

**Cover what is still unclear** — skip anything the description already settled:

1. Who the consumer is.
2. What they can do that they could not do before.
3. What starts it — an action, an event, a schedule.
4. What they observe when it works.
5. What happens when it does not — empty, unauthorized, upstream failure, bad input.
6. Which boundary conditions matter, and which are being left alone.
7. What this depends on that does not exist yet.
8. What is explicitly out of scope.
9. When there is a visual surface: what it looks like — reference products, constraints, the
   interactions they picture.

Question five feeds the `IF … THEN` criteria, six feeds **Edge cases**, and nine feeds the mockup
and **Design notes**.

**Draw the mockup while the questions are open.** When a visual surface comes up and the user has
no mockup for it, run `codefall-mock-up` for that surface now and iterate on it with the user as
`reference/interview.md` says: every pass asks a design question, and every answer is interview
material. A mockup they already have is imported per `../../../.codefall/shared/import-mockup.md`.
The files sit uncommitted until step 12 branches. Nothing is drawn here.

**Push back on vague answers**, naming the word and asking for a concrete one. **Raise design
concerns as flags, not rulings**: name a problem once, let them decide, two rounds per concern at
most. Both tables are in `reference/interview.md`. What stays unresolved goes under **Open
questions**.

Before advancing, judge the answers against consumer, trigger, observable outcome, and failure
behavior. **If fewer than roughly three of the applicable ones are concrete, do not advance.** Say
which ones are thin and ask. **Insist for at most two rounds.** If the user overrules, write it,
and record every unresolved item under **Open questions** — never quietly dropped.

**A question of fact the user cannot answer is consulted on once**, per `reference/consulting.md`,
and the answer offered as a proposal they confirm. Never a preference. "Like $COMPANY does it" is
looked up once on offer.

### 6. Audit what already exists

A specification describes what the user wants **added**. **Silent omission from a specification is
not a deletion.** Read `reference/audit.md`. For each surface the feature touches — named by the
user or found at step 3 — read the current artifact, enumerate what is there and every path that
reaches the outcome this feature controls, and ask about the difference from what the user
described. Preserved goes under **Existing behavior preserved**; merged is covered by a criterion;
removed is its own criterion; a second path to a gated outcome gets a criterion either way.

### 7. Shape the requirements

Read `reference/specification.md`. Cut what the user described into requirements, each a
capability a consumer can use and a ticket someone can pick up.

**Requirements decompose by what a consumer can observe.** `codefall-plan` cuts by what can be
built; do not do that cut here. A fact the cut needs and the user cannot supply is consulted on, as
at step 5. **The sizing question**: could one person hold this requirement in their head well
enough to design it in a single pass? If not, it is more than one. Then apply the
[cohesion check](#cohesion-and-splitting) to the set.

### 8. Settle each requirement's mockup

Each requirement with a visual surface has a mockup from step 5, referenced by path under **Design
notes**; or the surface appeared only now, and the mockup is made or imported here the same way; or
the user skips it, and its **Design notes** say `Mockup: pending`, which step 13 mirrors as a label
and `codefall-plan` makes before it starts. The only question is whether to skip.

### 9. Recap before writing

> Here is what I have. Tell me what is wrong.
>
> **Consumer**: …
> **Requirements I would write**: …
> **What starts each one**: …
> **What they observe**: …
> **Failure behavior**: …
> **Boundary conditions being left alone**: …
> **Already exists and must keep working**: …
> **Assuming without asking**: …
> **Out of scope**: …
> **Unresolved**: …
>
> Does that match what you have in mind?

**Every line is at least a sentence.** A fragment or a dash means that topic was not interviewed;
go back and ask.

Advance when every line is substantive, the user has confirmed it, and the criteria could be
written without guessing.

### 10. Engineering review

Draft the criteria, then read `reference/engineering-review.md` and hand the draft to **one
subagent of this harness**, prompted from `engineering-review-prompt.md`, read-only, to read it as
an engineer with the code open: contradictions between criteria, data the system already holds
elsewhere, assumptions it cannot meet, mechanisms that are broken or against the project's stance,
and — only when the gate in the reference allows — a criterion that is a mechanism rather than an
outcome.

You are the facilitator. Put each finding to the user as a flag, in their vocabulary, one or two at
a time, two rounds at most. An answer edits a requirement, a criterion, Key entities, or Out of
scope; a mechanism the user keeps gets one line under **Design notes** saying it is a product
requirement. One review per run; the edits are not re-reviewed. Say what the review found, even
when it found nothing.

### 11. Write, then confirm

Pick the identifier: the highest number in `docs/specs/` and `archive/` plus one, zero-padded to
three digits. A retired identifier is never reused.

Compose the full document from `templates/specs/SPEC.md` and **show it to the user before anything
is written**, any vision amendment beside it. Omit empty sections, header rows included: a spec
with no vision has no `**Vision:**` line. Say which optional sections you left out and why.

Then set the status: `Ready`, unless they said they are stopping and coming back, which is `Draft`.

### 12. Branch, then write the spec

Branch first, per `../../../.codefall/shared/landing.md` — `spec/SPEC-NNN-slug`, from the default
branch; the mockups come along uncommitted. Then write `docs/specs/SPEC-NNN-slug.md`, and
`docs/specs/AGENTS.md` if it was missing. The commit waits for the mirror, which writes the issue
number into the document.

### 13. Mirror to the tracker

Follow the creation sequence in `trackers/<name>/PROFILE.md`. The issues are generated from the
document, so this step never asks the user to re-approve content. `requires-mockup` goes only on a
requirement whose Design notes say `Mockup: pending`.

### 14. Link back, commit, and wrap up

If a vision framed this work, add the spec identifier to its `Related` line and write the step 11
amendment; change nothing else in it.

Then land it per `../../../.codefall/shared/landing.md`: commit by path — the spec, the mockups,
the `AGENTS.md` files, the vision — push, and open the pull request with
`Relates to #<spec-issue>` in its body when the mirror made one, a draft only when the status is
`Draft`.

Report the spec path, identifier, and status, every open question, every consult and what it
settled, what the engineering review found and what each finding became, the issues created with
links, the mockups made and where, any vision amendment, the branch, and the pull request, worded
as the landing procedure says.
**End with one command**: `/codefall-plan SPEC-NNN`; a sibling spec the user asked for is offered
in a sentence before it.

## Other modes

Invoking this skill on an existing spec does one of four things. Ask which if it is not obvious.
None of them runs the engineering review.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft`.
- **Edit** a `Draft` or `Ready` spec. A new requirement takes the next `REQ` number, a new
  criterion the next `AC` number within its requirement; retired numbers stay retired. Re-mirror
  afterwards.
- **Archive** a spec: set `Status: Archived`, add `Replaced by` if something took its place, move
  the file to `docs/specs/archive/`, and close or relabel its issues per the tracker profile.
- **Split** a spec into siblings, per [cohesion and splitting](#cohesion-and-splitting).

Every one is the user's decision: report the state and offer. Each lands per
`../../../.codefall/shared/landing.md`.

## Rules

- **Nothing is written without the user confirming the document first.**
- **The document is canonical**; tracker issues are regenerated from it.
- **The specification says what, never how.** No file paths, libraries, services, or schema; a
  domain noun may be named, its fields may not.
- **Acceptance criteria are EARS, and nothing else is.**
- **"so that" is mandatory** in every user story.
- **Criteria are observable in a running system**; the instrumentation that makes them so is part
  of the requirement, said out loud.
- **Criteria exist for coverage, not symmetry.** **Numbering is append-only at every level.**
- **Status describes the document, never the work.**
- **Silent omission is never a deletion**; a removal is its own criterion.
- **The code raises questions and never answers them**; what should happen is the user's.
- **A preference is asked, never proposed.** A proposal is for a question of fact, and the user
  confirms it.
- **Mockups are keyed by surface**, not by spec.
- **A mockup a requirement needs is made in this run**, through `codefall-mock-up`, during the
  interview when the surface comes up there; never drawn here, never left for the person to start.
- **The engineering review runs once per spec**, before the document is composed, by a subagent
  that only proposes; every finding is a flag the user decides, and a kept mechanism is recorded.
- **Push back once, then defer** — on vagueness, on design concerns, on cohesion — and **record
  what stays unresolved** as open questions, never dropped.
- **A consult answers a question of fact, never a preference**, and the user confirms it first.
- **A vision found wrong is amended here** when it is `Draft` or `Ready`; an `Active` one is not.
- **Never overwrite a file that has drifted.** Show the difference and ask.
