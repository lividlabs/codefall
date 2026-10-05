---
name: codefall-specify
description: Turn a feature idea into a specification another session can implement — interview for what the user will observe, push back on vague answers, audit what already exists, then write requirements with EARS acceptance criteria into a spec document in the repository, mirrored to the issue tracker.
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
  - WebSearch
  - WebFetch
---

# Specify

Turn a feature idea into a specification precise enough that another session can implement it
without re-interviewing anyone.

The output is a **spec document in the repository** at `docs/specs/SPEC-NNN-slug.md`, holding one or
more requirements, each with a user story and numbered acceptance criteria. The issue tracker gets a
generated mirror. **The document is canonical**; the issues are regenerated from it.

Specifying is not designing. Once the criteria are written and confirmed, stop.

Paths that start with `../` or `trackers/` are relative to this skill's directory, not the user's
project. A path through `../../../.codefall/` is the one that leaves the skills directory: it names
a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/interview.md` — the vague-answer table, the lookup offer, and the cohesion and
  splitting rules. Read at step 5 and step 7.
- `reference/specification.md` — the user story, the six EARS patterns with a worked example,
  observability, prohibited vocabulary, numbering, and the optional sections. Read before step 7.
- `templates/specs/SPEC.md` — the document template. Read at step 10. Every bracketed instruction
  in it is stripped on emit.
- `templates/specs/AGENTS.md` — the operative rules this skill installs at `docs/specs/AGENTS.md`.
- `trackers/github/PROFILE.md` — the GitHub tracker profile: issue shape, labels, creating,
  refreshing, archiving. Read at step 13, and at step 3 for the duplicate search.
- `reference/consulting.md` — a question of fact the user cannot answer, put to the configured
  agents. Read at steps 5 and 7. Names
  `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`, `../../../.codefall/shared/consult-prompt.md`,
  `../../../.codefall/shared/consult.schema.json`.
- `../../../.codefall/shared/import-mockup.md` — the shared procedure for bringing a user's mockup into the
  repository. Read at step 8 when they have one.
- `../../../.codefall/shared/landing.md` — the branch, the commit, the draft pull request, and
  marking it ready. Read at step 11 and step 14.

## Scope — what, not how

Specify decides **what will be true when this is done**, never **how it gets built**.

| In scope | Out of scope |
| --- | --- |
| Who the consumer is and what they get out of it | Which layer, component, or module the work lands in |
| What they can observe when it works | File paths, libraries, services, frameworks |
| Domain nouns and what they mean | Their fields, types, relations, or storage |
| What must keep working that already works | API shapes, message contracts, schema |
| What is explicitly not included | Work breakdown, ticket sequencing, dependency edges |
| Which questions remain unresolved | Estimates, effort sizing, build order |

**You may read the codebase, but only to answer two questions**: does this already exist, and what
would this change silently break?

## What a specification is

One **spec document** per feature. Inside it, one or more **requirements**, each with a user story —
`As a <consumer>, I want <capability>, so that <benefit>`, the "so that" mandatory — and its own
acceptance criteria in EARS. A requirement is the unit that becomes a ticket.

Identifiers nest and are written in full: `SPEC-003`, `SPEC-003-REQ-01`, `SPEC-003-REQ-01-AC-01`.
Three digits for the spec, two for the rest, zero-padded, append-only at every level. Criterion
numbers restart under each requirement.

The rules for stories, criteria, and the optional sections — Edge cases, Key entities, Assumptions
— are in `reference/specification.md`.

## Status and lifecycle

Three states, one word plus a date.

| Status | Meaning | Lives in |
| --- | --- | --- |
| `Draft — <date>` | Being written. The user stopped and is coming back | `docs/specs/` |
| `Ready — <date>` | Written and agreed. The normal end of a session | `docs/specs/` |
| `Archived — <date>` | Superseded or dropped | `docs/specs/archive/` |

- **Status describes the document, never the work.** Work state is the tracker's.
- **`Ready` is the normal end of a session.** Set `Draft` only when the user said they are stopping
  and will come back.
- **A spec is never renumbered and its identifier is never reused.** Archiving moves the file to
  `docs/specs/archive/` under the same name; citations still resolve.
- `Archived` gains a `**Replaced by:** SPEC-NNN — <date>` line when something took its place.
- Transitions are this skill's to make. Report the state and offer; never transition a spec on your
  own initiative.

## The specs directory

`codefall-specify` maintains `docs/specs/AGENTS.md` from `templates/specs/AGENTS.md`: written when
the directory is created, added on a later run if it is missing.

**Never overwrite a file that has drifted.** When one exists and differs from the template, show the
difference and ask. Replace it only on a yes; on a no, leave it and say nothing further about it.

## Mockups are keyed by surface, not by spec

Mockups live at `docs/mockups/<slug>/`, where the slug names the surface — `booking-history`,
`trip-share`. They are **never** filed under a spec or a vision. Specs reference them by path under
**Design notes**. Do not move existing mockups into a spec directory, and do not create one.

## Cohesion and splitting

**A spec holds one cohesive feature.** Its requirements share a consumer and a purpose. Push back
once when a spec looks incohesive, with a specific alternative, then defer; split results are
siblings the vision groups, never a parent and children. The rules are in `reference/interview.md`.

## Tracker profiles

The specification is tracker-neutral. Where the mirror lands, and in what shape, is a **tracker
profile** — one directory per tracker.

| Tracker profile | Covers | Status |
| --- | --- | --- |
| `github` | GitHub Issues, optionally with a GitHub Project | **supported** |
| `jira` | Jira Cloud and Data Center | planned |
| `linear` | Linear | planned |

- A tracker is **supported** only when `trackers/<name>/PROFILE.md` is complete. A planned profile
  is an exit, not a menu choice: say `codefall-specify` does not mirror to it yet and stop.
- GitHub is the only supported profile, so there is no question to ask: state that the mirror will
  land in GitHub Issues and confirm the repository.
- Read the profile's capability table before writing, and follow its fallbacks rather than
  improvising around a missing feature.
- **The spec is written even when the mirror fails.** The document is the deliverable. Give the user
  the exact command to fix the tracker and say the mirror is pending. Do not discard the spec.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Check preconditions

Run the shared check against the user's project.

```bash
"../../../.codefall/shared/preflight.sh" .
```

`beads=ok` advances to step 2. Otherwise read `beads_reason`, tell the user what is missing, hand
over the command that fixes it, say to rerun this verb after it, and **stop**:

| `beads_reason` | What is wrong | Give them |
| --- | --- | --- |
| `not_installed` | `bd` is not on PATH | `brew install beads` |
| `not_initialized` | this repository has no beads database | `bd init` |
| `unreadable` | bd found a database and could not read it | quote `beads_detail` |

Then read the checkout lines. `behind` above `0` or `refresh=stale` means the environment may not
match `main`: say so and run `/codefall-refresh` before continuing. `refresh=undeclared` names
`/codefall-equip` instead.

**Never run the remedy.** That is the user's decision.

### 2. Ask what they want to build

One open question:

> "What would you like to build? A sentence or two is enough to start."

**Then look for a vision.** If `docs/visions/` exists, read the live visions there — not
`archive/` — and offer the relevant one as context:

> VISION-002 covers the auditing rework and looks like the frame for this. Want me to work from it?

A vision is **never required**. It carries the *why* and improves the Context section. It does not
carry acceptance criteria: its **Proposed shape** is a rough direction, and step 5 still interviews
for everything. Do not lift criteria out of a vision and do not treat its **Open questions** as
settled.

Do not change the vision's `Status`. Work starting is `codefall-implement`'s transition to record.
A `Draft` or `Ready` vision the interview shows wrong or incomplete is amended on this run's branch:
the amendment is shown with the spec at step 10 and committed at step 14. A change bigger than
text is `codefall-envision`'s, and this run runs it, on this branch, under the confirmation at
step 10. An `Active` vision is frozen; say so. `../../../.codefall/shared/workflow.md` has the rule.

### 3. Check whether it already exists

Three searches before spending the user's time on an interview:

- **The specs** — read `docs/specs/`, not `archive/`.
- **The codebase** — Glob and Grep for what they described: filenames, exported identifiers, route
  segments, domain nouns.
- **The tracker** — the profile supplies the search command.

### 4. Report what you found

- **A spec already covers it.** Link it. Ask whether it is the same thing, and whether they want to
  revise that one rather than write another.
- **It exists in code.** Describe what is there, with file paths. Ask whether that is what they
  meant, and if not, what the distinction is.
- **An issue already covers it.** Link it. Ask whether it is the same thing.
- **Nothing found.** Say so and move on.

If the user confirms existing work covers their need, **stop the skill**.

### 5. Interview

**Style.** One or two questions at a time, never a wall. Start broad and narrow. Use the user's own
vocabulary.

**Cover what is still unclear** — skip anything the description already settled:

1. Who the consumer is.
2. What they can do that they could not do before.
3. What starts it — an action, an event, a schedule.
4. What they observe when it works.
5. What happens when it does not — empty, unauthorized, upstream failure, bad input.
6. Which boundary conditions matter, and which are being left alone.
7. What this depends on that does not exist yet.
8. What is explicitly out of scope.

Question five feeds the `IF … THEN` criteria and question six feeds **Edge cases**.

**Push back on vague answers.** Name the vague word and ask for a concrete replacement; the usual
ones and what to ask are in `reference/interview.md`.

Before advancing, judge the answers against consumer, trigger, observable outcome, and failure
behavior. **If fewer than roughly three of the applicable ones are concrete, do not advance.** Say
which ones are thin and ask.

**Insist for at most two rounds.** If the user overrules — "just write it with what we have" — write
it, and record every unresolved item under **Open questions**. Recorded, never quietly dropped.

**A question of fact the user cannot answer is consulted on once**, per `reference/consulting.md`,
and the answer offered as a proposal they confirm. Never a preference.

**Raise design concerns as flags, not rulings.** Name a problem once and let them decide:

> "One thing I want to flag — a destructive action behind a hover has no reachable equivalent on
> touch. How are you thinking about that?"

Cap at two rounds per concern. If it stays unresolved, it goes under **Open questions**. "Like
$COMPANY does it" is looked up once on offer, per `reference/interview.md`.

### 6. Audit what already exists

A specification describes what the user wants **added**. **Silent omission from a specification is
not a deletion.** For each surface the feature touches — a screen, a route, a component, an entity,
a table:

1. **Read the current artifact.** Its contents, not its name. If it does not exist yet, skip.
2. **Enumerate what is there.** Tabs, fields, states, columns, branches.
3. **Intersect with what the user described.**
4. **If there is a real gap, ask.**

   > "The profile screen has three tabs today: Account Info, Travel Preferences, and Saved Passengers.
   > You mentioned the first two. Is Saved Passengers preserved as-is, folded into something new, or
   > intentionally going away?"

   - **Preserved** — record it under **Existing behavior preserved**.
   - **Merged** — record how it composes, and cover it with a criterion.
   - **Removed** — this becomes its own acceptance criterion.

Audit the surface the feature touches, not the whole application.

### 7. Shape the requirements

Read `reference/specification.md`. Cut what the user described into requirements. Each one is a
capability a consumer can use and a ticket someone can pick up.

**Requirements decompose by what a consumer can observe.** `codefall-design` cuts by what can be
built; do not do that cut here. A fact the cut needs and the user cannot supply is consulted on, as
at step 5.

**The sizing question**: could one person hold this requirement in their head well enough to design
it in a single pass? If not, it is more than one requirement.

Then apply the [cohesion check](#cohesion-and-splitting) to the set.

### 8. Mockups

For each requirement with a visual surface, ask whether a mockup exists.

- **They have one** — import it per `../../../.codefall/shared/import-mockup.md`. It lands under
  `docs/mockups/<slug>/`, and the spec references that path under **Design notes**.
- **They do not** — it is made in this run, at step 12, by running `codefall-mock-up` for that
  surface on this branch. Say so; the only question is whether to skip it for now. The spec
  references the path it will land at.
- **Skipped** — the requirement's tracker issue is marked `requires-mockup` at step 13, and
  `codefall-design` makes the mockup before it starts.

Nothing is drawn here; `codefall-mock-up` is the tool.

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

**Every line is at least a sentence.** A fragment or a dash means that topic was not interviewed —
go back and ask.

Advance when every line is substantive, the user has confirmed it, and you could write the criteria
without guessing.

### 10. Write, then confirm

Pick the identifier: read `docs/specs/`, take the highest existing number plus one, zero-padded to
three digits. Read `archive/` for this alone; a retired identifier is never reused.

Compose the full document from `templates/specs/SPEC.md` and **show it to the user before anything
is written**. Omit empty sections, header rows included: a spec with no vision has no
`**Vision:**` line. Say which optional sections you left out and why — "no Key Entities section,
because the nouns here are ordinary English" — so the user can catch an omission that was a gap.

Show any vision amendment beside it.

Then set the status: `Ready`, unless they said they are stopping and coming back, which is `Draft`.

### 11. Branch, then write the spec

Branch first, per `../../../.codefall/shared/landing.md` — `spec/SPEC-NNN-slug`, from the default
branch.

Then write `docs/specs/SPEC-NNN-slug.md`, and `docs/specs/AGENTS.md` if it was missing. The commit
waits for the mirror, which writes the issue number into the document.

### 12. Make the mockups

Run the `codefall-mock-up` skill once per surface step 8 marked, on this branch, with the spec as
its input. Run from here it skips its own branch and landing and hands back the directory and the
per-file lines; the path goes under the requirement's **Design notes**.

### 13. Mirror to the tracker

Follow the creation sequence in `trackers/github/PROFILE.md`. The document is canonical and the
issues are generated from it, so this step never asks the user to re-approve content.
`requires-mockup` goes only on a requirement whose mockup was skipped at step 8.

### 14. Link back, commit, and wrap up

If a vision framed this work, add the spec identifier to its `Related` line, and write the
amendment the user took at step 10; change nothing else in the file.

Then land it per `../../../.codefall/shared/landing.md`: commit by path — the spec, the mockups,
the `AGENTS.md` files, the vision — push, open the pull request as a draft with
`Relates to #<spec-issue>` in its body, and mark it ready for review when the status is `Ready`.
The Action merges it; no person is asked to.

Report the spec path, its identifier, its status, every open question it carries, every consult and
what it settled, the issues that were created with links, the mockups made and where, any vision
amendment, the branch, and the pull request with its state worded as the landing procedure says:
merged, ready and being merged by the Action, or ready and waiting for a person because the project
has no Action. **End with one command**: `/codefall-design SPEC-NNN`; where the user asked for a
sibling spec, say so in a sentence and ask whether to write it now.

## Other modes

Invoking this skill on an existing spec does one of four things. Ask which if it is not obvious.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft`.
- **Edit** a `Draft` or `Ready` spec. Adding a requirement appends the next `REQ` number; adding a
  criterion appends the next `AC` number within its requirement. Retired numbers stay retired.
  Re-mirror to the tracker afterwards.
- **Archive** a spec: set `Status: Archived`, add `Replaced by` if something took its place, move the
  file to `docs/specs/archive/`, and close or relabel its issues per the tracker profile.
- **Split** a spec into siblings, per [cohesion and splitting](#cohesion-and-splitting).

Every one of these is a user's decision. Report the state and offer; never transition a spec on your
own initiative. Each lands per `../../../.codefall/shared/landing.md`.

## Rules

- **Nothing is written without the user confirming the document first.**
- **The document is canonical**; tracker issues are regenerated from it.
- **The specification says what, never how.** No file paths, libraries, services, or schema; a
  domain noun may be named, its fields may not.
- **Acceptance criteria are EARS, and nothing else is.**
- **"so that" is mandatory** in every user story.
- **Criteria are observable in a running system**; the instrumentation that makes them so is part
  of the requirement, said out loud.
- **Criteria exist for coverage, not symmetry.**
- **Numbering is append-only at every level**; retired numbers are never reused.
- **Status describes the document, never the work.**
- **Silent omission is never a deletion**; a removal is its own criterion.
- **Mockups are keyed by surface**, not by spec.
- **A mockup a requirement needs is made in this run**, through `codefall-mock-up`, never drawn
  here and never left as a step for the person to start.
- **Push back once, then defer** — on vagueness, on design concerns, on cohesion.
- **Unresolved is recorded** as open questions, never dropped.
- **A consult answers a question of fact, never a preference**, and the user confirms it first.
- **A vision found wrong is amended here** when it is `Draft` or `Ready`; an `Active` one is not.
- **Never overwrite a file that has drifted.** Show the difference and ask.
