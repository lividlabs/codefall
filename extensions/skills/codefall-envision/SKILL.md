---
name: codefall-envision
description: Get an idea onto paper before anyone specifies or scaffolds it — take whatever the user arrived with, from a sentence to a folder of mockups, organize it into one or more numbered vision documents under docs/visions/, and record what is still unknown rather than inventing answers.
argument-hint: "[the idea, or a path to a document you already have]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
  - WebSearch
  - WebFetch
---

# Envision

Write down what someone wants to build, and why, before anyone pins down exactly what it does or
how it is built.

The output is a **vision document** at `docs/visions/VISION-NNN-slug.md`. It is deliberately
informal. A vision carries the *why* — the problem, the reason it matters now, the rough shape of an
answer — and it stops well short of the detail a specification needs.

Envisioning is not specifying, and the difference is precision, not subject. A vision may say how
the thing works, as loosely as the user described it. The moment the document is precise enough to
build from, it has become `codefall-specify`'s job.

Paths that start with `reference/`, `templates/`, or `../` are relative to this skill's directory,
not the user's project. A path through `../../../.codefall/` is the one that leaves the skills
directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `templates/visions/VISION.md` — the document template. Read at step 5. Every word of
  instruction in it — the HTML comment, the guidance under each heading, the `(Optional — …)`
  lines — is stripped on emit.
- `templates/visions/AGENTS.md` — the operative rules this skill installs at
  `docs/visions/AGENTS.md`.
- `reference/inputs.md` — where the material a user brings goes, and when a document they brought
  is adopted as the vision. Read at step 2 when they point at anything.
- `../../../.codefall/shared/import-mockup.md` — the shared procedure for bringing a mockup into the
  repository. Read from `reference/inputs.md` when the material is a mockup.
- `../../../.codefall/shared/landing.md` — the shared procedure for the branch, the commit, the
  push, and the pull request. Read at step 6, before the first file is written.

## Scope — why, and a loose what

| In scope | Out of scope |
| --- | --- |
| The problem, and who feels it | Exact behavior: every case, rule, and error |
| Why it matters now rather than later | Acceptance criteria of any kind |
| The rough shape of an answer, including how it works in broad strokes | Field lists, endpoints, entities, schemas |
| What is deliberately not covered | Work breakdown, sequencing, estimates |
| Constraints and systems this touches | Which layer or component the work lands in |
| What nobody has decided yet | Anything a reader could build from |

The tell that this skill has failed is a document someone could implement.

### How big a vision is

There is no ceiling. A whole product, a product line, a capability area, a subsystem, or a rework of
how something already works are all visions.

The floor is about **size, not category**. "Auditing in the accounting system is broadly broken" is
a vision. One bug, one screen, one endpoint, one field is usually smaller than this document is
worth, and `codefall-specify` or the work itself is the better destination. **That is advice, not a
gate.** Say what you think and why — "this looks like one field, which `codefall-specify` handles in
a couple of questions" — and then write the vision if that is what they want.

### When the pile covers more than one problem

**Each problem that stands on its own is a vision.** They are written as siblings —
`VISION-003`, `VISION-004`, `VISION-005` — never as a parent and children. Each one's `Related`
line points at the same saved source or mockup directory.

**Do the breakdown with the user, not for them.** Say what you found and where the seams look like
they are, then let them decide the cut:

> This reads like three problems to me — the audit trail, the export format, and who is allowed to
> see any of it. I would write three visions. Does that match how you think about it?

They may say it is one, and one is then the answer. **Each vision is confirmed before it is
written.** Several documents in a run is several confirmations.

## The specify off-ramp

When what the user describes is really a specification, say so once, say why, and offer to switch.

| Tell | Why it means `codefall-specify` |
| --- | --- |
| One capability, described end to end | A vision frames a problem; this already answers it |
| The *why* is inseparable from the *what* | A vision exists to carry a *why* that outlives any one feature |
| The problem is already written down and agreed | A second document restates it and adds no information |

**None of these is about how concretely the user can describe the *what*.** Finished mockups and
named screens with nothing written down about the problem they solve is a vision, not a spec.

Say it plainly and give them the choice:

> This is one capability described end to end, and the reason for it is already written down in
> VISION-002. A vision here would restate what you have. Want to run `/specify` instead?

**If they agree, stop this skill.** If they disagree, write the vision — push back once, then
defer.

## What a vision looks like

Two sections are mandatory: **Problem** and **Proposed shape**. Everything else appears only when
the user actually said something about it. The template is `templates/visions/VISION.md`.

- **The document's length is proportional to what the user put in.** Get their words onto paper in
  an order that reads well; never fill in what they did not say. An optional section with nothing
  to say is deleted, heading included.
- **Unknowns are correct output.** Record what is unresolved under **Open questions**, phrased as
  the question, and move on.
- **The `Related` line carries backward links** when they exist: the saved source under
  `docs/visions/sources/`, any mockup directory under `docs/mockups/`, and any vision this one
  revises or replaces.
- **The `specs` field holds a list** — `SPEC-003, SPEC-004, SPEC-005`. `codefall-specify` appends
  each identifier as it creates one, and changes nothing else in the file.

## Status and lifecycle

`Status` is one word plus a date. Two optional header fields carry the pointers.

| Header | Meaning | Set by | Editable in place | Lives in |
| --- | --- | --- | --- | --- |
| `Status: Draft — <date>` | The user said, or clearly implied, that they are stopping and coming back | this skill, only on that signal | yes | `docs/visions/` |
| `Status: Ready — <date>` | Written and agreed. The normal end of a session | this skill | yes | `docs/visions/` |
| `Status: Active — <date>` | Work has started against it | `codefall-implement` | no, except the Status line | `docs/visions/` |
| `Status: Archived — <date>` | Wholly replaced, or dropped | this skill | no | `docs/visions/archive/` |
| `Revised by: VISION-NNN — <date>` | Part of it was replaced; this vision is still live. Optional, repeatable | this skill | — | — |
| `Replaced by: VISION-NNN — <date>` | Accompanies `Archived` when something took its place. Omitted when the vision was simply dropped | this skill | — | — |

- **`Ready` is the normal end of a session, not `Draft`.** Set `Draft` only when they said they are
  stopping and will come back — "let's leave it there for now", "I want to think about this more".
  They will rarely use the word.
- **`Active` is set by `codefall-implement`**, at the first claim of work that traces back to the
  vision. Do not set it yourself.
- **A vision is never renumbered and its identifier is never reused.** Archiving moves the file to
  `docs/visions/archive/` under the same name; citations still resolve.

## The visions directory

`codefall-envision` maintains `docs/visions/AGENTS.md` from `templates/visions/AGENTS.md`:
written when the directory is created, added on a later run if it is missing.

**Never overwrite a file that has drifted.** When one exists and differs from the template, show the
difference and ask. Replace it only on a yes; on a no, leave it and say nothing further about it.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Set up and look around

No precondition check. Create `docs/visions/` in the working directory if it does not exist, along
with `docs/visions/AGENTS.md`. **Do not ask where the documents go.** The working directory is the
project.

Read what is already there — the vision documents, not the archive — and pick the next identifier
as the highest existing number plus one, zero-padded to three digits. If an existing vision covers
what the user is describing, say so and link it before going further; they may want to revise that
one rather than write another.

### 2. Take the idea

One open question:

> "What's the idea? However it comes out is fine — a sentence, or ten minutes of thinking out loud.
> And if you already have anything — notes, a document, sketches, mockups — point me at it."

**Ask once, at the start, and then drop it.** Do not chase material the user did not mention.

**A stream of consciousness is the best input this skill gets.** Take all of it, ask nothing while
it is arriving, and organize it into a draft before asking anything.

If they pointed at anything, read it and follow `reference/inputs.md`.

### 3. Check the altitude

Before interviewing, judge whether this calls for a vision at all. Apply
[the specify off-ramp](#the-specify-off-ramp) and [the floor](#how-big-a-vision-is). Doing this
now costs one exchange; doing it after the interview wastes the whole session.

### 4. Interview

Only for what is genuinely missing. A user who arrived with a clear problem and a rough answer has
given you everything a vision needs — write it.

Ask about, in this order, and skip anything already answered:

1. What is broken, missing, or costing something — and who feels it.
2. The rough shape of an answer, if they have one, including how it would work as far as they have
   thought it through. "I don't know yet" is a complete answer and goes under Open questions.
3. Why now rather than later.
4. What this deliberately does not cover.
5. What it touches that already exists, and what it must not break.

**Two rounds is a cap on your insistence, not on the conversation.** Stop pressing an unanswered
point after the second try and put it under Open questions. If the user keeps going, keep going
with them.

**Do not bikeshed.** Naming, wording, and which of two rough shapes is better are not this
document's problems.

**"Like $COMPANY does it."** When the user references another product, offer once to look it up. On
yes, research it and summarize only what matters to the vision; confirm the summary with the user
before it reaches the document. On no, move on without searching.

### 5. Draft and confirm

Compose the full document from `templates/visions/VISION.md` and show it before anything is
saved. Say which sections you left out and why — "no Non-goals section, because nothing you said
was at risk of being misread as in scope" — so the user can catch an omission that was a gap.

Then set the status: `Ready`, unless they said they are stopping and coming back, which is `Draft`.

### 6. Branch, write, land, and report

Read `../../../.codefall/shared/landing.md` and take its branch step first: standing on the default
branch, `git switch -c vision/VISION-NNN-slug`; on another branch, ask once which to use.

Write the vision, the source document if there was one, and `docs/visions/AGENTS.md` if it was
missing. Then commit those files by path, push, and open the pull request, all per the same
procedure; it is a draft only when the status is `Draft`. Never add the `auto-merge` label: a person
adds it when they want the document merged.

Do not create issues. Do not start a specification.

Report the path, the identifier, the status, every open question the document carries, the branch,
and the pull request, worded as the landing procedure says: it is open at its URL; add the
`auto-merge` label when you want it merged, or have someone approve it; either one merges it.
**End with one command**: `/codefall-scaffold` for a project not yet scaffolded, otherwise
`/codefall-specify VISION-NNN`.

## Other modes

Invoking this skill on an existing vision does one of four things. Ask which if it is not obvious.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft` when nothing cites it yet.
- **Edit** a `Draft` or `Ready` vision in place.
- **Revise** an `Active` vision — which is frozen, because work is underway. The revision is a new
  vision, and the old one gains a `Revised by` line.
- **Archive** a vision: set `Status: Archived`, add `Replaced by` if something took its place, and
  move the file to `docs/visions/archive/`.

Each of these changes files, and lands them as a new vision does: on a branch, per
`../../../.codefall/shared/landing.md`.

Every one of these is a user's decision. Report the state and offer; never transition a vision on
your own initiative.

## Rules

- **Nothing is written without the user confirming the full document first.**
- **The vision document says why, and how it works only loosely.** Nothing in it is precise enough to
  build from. That is about the document. Mockups and other material can arrive with it and be cited
  from it; they are inputs, not the vision.
- **Length is proportional to input.** Never fill a heading. An optional section with nothing behind it
  is deleted, heading and all.
- **Unknowns are recorded, not resolved.** Open questions are correct output.
- **Push back once, then defer** — on altitude, on scope, on size, on a shape you think is wrong. Say
  what you think and why, then do what they ask. The user knows the domain and you may be wrong.
- **Everything the user brought is saved and never edited**, whether it lands in `sources/` or in
  `docs/mockups/`.
- **Identifiers are append-only.** A retired number is never reused, and archiving moves a file without
  renumbering it.
- **`Active` is not yours to set.** `codefall-implement` owns that transition.
- **Do not draw, specify, design, or estimate.** Every one of those is another verb's work.
