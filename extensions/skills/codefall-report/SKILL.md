---
name: codefall-report
description: Turn a bug someone ran into into a report another session can fix — interview the person who saw it for the steps, the expected and actual result, screenshots, and the environment, push back on vague answers, try to reproduce it on the spot, then write a bug report with acceptance criteria into the repository, mirrored to the issue tracker.
argument-hint: "[what went wrong, or a test run's report]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
---

# Report

Turn a bug someone ran into into a report precise enough that another session can find the cause
and fix it without asking the reporter anything again.

The output is a **bug report in the repository** at `docs/bugs/BUG-NNN-slug.md`, with its
attachments beside it in `docs/bugs/BUG-NNN-slug/`, and, when the project mirrors to a tracker, one
issue generated from it. **The document is canonical**; the issue is regenerated from it.

Reporting is not fixing. Once the report is written and mirrored, stop.

Paths that start with `reference/`, `templates/`, or `trackers/` are relative to this skill's
directory, not the user's project. A path through `../../../.codefall/` is the one that leaves the
skills directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/reproducing.md` — the attempt to reproduce: the environment, the driver, the evidence
  it captures, the three outcomes, and what an attempt never does. Read at step 5.
- `templates/bugs/BUG.md` — the document template. Read at step 7. Every bracketed instruction in
  it is stripped on emit.
- `templates/bugs/AGENTS.md` — the operative rules this skill installs at `docs/bugs/AGENTS.md`.
- `trackers/<name>/PROFILE.md` — the tracker profile, `<name>` being `tracker` in
  `.codefall/settings.json`: the issue shape, the label, creating, refreshing, and archiving. Read
  at step 3 for the duplicate search and at step 9.
- `../../../.codefall/shared/landing.md` — the branch, the commit, the push, and the pull request.
  Read at step 8.

## Scope — what is wrong, not why

| In scope | Out of scope | Whose |
| --- | --- | --- |
| What the reporter did, expected, and saw | Why the code does it | `codefall-plan` |
| Reproducing it through the running product | Changing code, data, or a test to see what happens | `codefall-plan`, `codefall-implement` |
| Evidence: screenshots, recordings, logs, output | The fix, and its task graph | `codefall-plan` |
| The expected behaviour, as acceptance criteria | A new capability nobody specified | `codefall-specify` |
| Severity as the reporter judges it | Priority and assignment | the team |

**You may read the codebase, but only to answer two questions**: is this already reported, and which
spec criterion states the expected behaviour?

## Identifiers and status

`BUG-NNN`, three digits, zero-padded, the highest existing number plus one, `archive/` included. A
criterion is `BUG-NNN-AC-nn`. **A report is never renumbered and its identifier is never reused.**

| Status | Meaning | Lives in |
| --- | --- | --- |
| `Draft — <date>` | The reporter stopped and is coming back | `docs/bugs/` |
| `Ready — <date>` | Written and agreed. The normal end of a session | `docs/bugs/` |
| `Archived — <date>` | A duplicate, not a bug, or not going to be fixed | `docs/bugs/archive/` |

- **Status describes the document, never the work.** A fixed bug stays `Ready`; its issue closes when
  the fix merges.
- `Archived` gains one line saying why: `**Duplicate of:** BUG-NNN`, `**Not a bug:** <the decision
  that says so>`, or `**Won't fix:** <who decided, and why>`.
- Transitions are this skill's. Report the state and offer; never transition a report on your own
  initiative.

## The bugs directory

Maintain `docs/bugs/AGENTS.md` from `templates/bugs/AGENTS.md`: written when the directory is
created, added on a later run if it is missing. **Never overwrite a file that has drifted** — show
the difference and ask.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report; when it is not `engineer`, follow that persona's section in
`../../../.codefall/shared/personas.md` for this run, and say so.

## Process

### 1. Check preconditions

```bash
"../../../.codefall/shared/preflight.sh" .
```

A report writes no beads, so the Beads lines stop nothing. Keep the checkout, `refresh=`, and
`local` lines for step 5.

### 2. Ask what went wrong

One open question, unless the invocation already answered it:

> "What went wrong? Tell me what you were doing and what you saw — a sentence is enough to start."

**A tracker issue, or a run report or triage note from `codefall-test`, is a starting point**, not a
finished report: what it carries fills what it can, and the interview covers the rest. An issue the
run started from becomes the mirror, per the profile's *Adopting* section.

### 3. Check whether it is already known

Before the interview, search:

- **The reports** — `docs/bugs/`, not `archive/`.
- **The tracker** — the profile supplies the search command.
- **The graph** — `bd search` for an open bead describing it, when Beads is ready.
- **The specs** — `docs/specs/`, for the criterion that states what should have happened. Keep what
  you find for the Spec row.

**Already reported** — link it, ask whether it is the same bug, and on yes offer to add what the
reporter brought as new evidence on that report (the Edit mode) instead of writing a second. Then
stop.

### 4. Interview

**Style.** One or two questions at a time. Use the reporter's words. Skip anything already said.

**Cover what is still unclear**:

1. **Steps** — from a starting point someone else can reach: which account or role, which data,
   which screen or command, then each action in order.
2. **Expected result** — what should have happened, and what says so: a spec, the docs, how it used
   to behave, or the reporter's expectation.
3. **Actual result** — what happened instead, exactly: the message text, the wrong value, what was
   missing.
4. **Evidence** — screenshots, a recording, console or server output, a request ID. Ask for each as a
   file path.
5. **Environment** — version or commit, where it ran (local, staging, production), OS, browser or
   device.
6. **Frequency** — every time, some of the time (how many of how many tries), or once.
7. **Impact** — who it affects, what it stops them doing, and any workaround.
8. **When it started** — the last version or date it worked, when there is one.

**An image seen only in the conversation cannot be saved.** Ask for the file's path. If there is
none, describe what the image shows under Evidence, in words, and say that is what happened.

**Push back on vague answers.** Name the vague word and ask for the concrete one:

| They said | Ask |
| --- | --- |
| "it's broken" | What did you see — an error, a blank screen, a wrong value, nothing at all? |
| "it doesn't work" | What did you do last, and what did you expect to see next? |
| "sometimes" | How many times out of how many tries? Anything different about the times it failed? |
| "the usual way" | Walk me through it from the first screen. |
| "an error" | What does it say, word for word? A screenshot? |
| "wrong" | What value did you see, and what should it have been? |
| "recently" | Which version or date is the last one you know worked? |

**Insist for at most two rounds.** When the reporter overrules, write what there is and record every
gap under **Open questions** — recorded, never dropped.

**When the expected behaviour was never specified or built**, it reads as a feature request. Say so
once and offer `/codefall-specify`. If the reporter still wants a bug report, write it.

**Severity** is the reporter's call. Offer the scale and push back once when the impact they
described does not fit:

| Severity | When |
| --- | --- |
| `critical` | Data lost or exposed, or the product unusable for most users, with no workaround |
| `major` | A core flow broken for some users, or a workaround most would not find |
| `minor` | Wrong, with a workaround a user would find |
| `trivial` | Cosmetic; nothing is stopped |

### 5. Try to reproduce it

Read `reference/reproducing.md`, and follow it. The attempt is always offered and never required:
the reporter may decline it, and an attempt that fails does not stop the report.

**An attempt that goes differently from the reporter's account sends you back to step 4** with the
exact step where it went differently. At most two returns; after that, record the difference and
move on.

### 6. Recap before writing

> Here is what I have. Tell me what is wrong.
>
> **Steps**: …
> **Expected**: …
> **Actual**: …
> **Evidence**: …
> **Environment**: …
> **Frequency**: …
> **Impact and severity**: …
> **Reproduced**: …
> **Unresolved**: …

**Every line is at least a sentence.** A fragment means that topic was not covered — go back and ask.

### 7. Write, then confirm

Pick the identifier. Compose the document from `templates/bugs/BUG.md`, and **show it before
anything is written**. Omit empty sections and header rows, and say which you left out and why.

**The acceptance criteria state the expected behaviour**, in EARS, as `codefall-specify` writes them:
`WHEN <the reporter's trigger>, the system SHALL <the expected result>`. A criterion a spec already
states cites it — `(SPEC-003-REQ-01-AC-02)` — rather than being rewritten. These are what the fix is
verified against and what its test case holds.

Then set the status: `Ready`, unless the reporter said they are stopping and coming back.

### 8. Branch, then write

Branch first, per `../../../.codefall/shared/landing.md` — `bug/BUG-NNN-slug`.

Write the document, copy each attachment into `docs/bugs/BUG-NNN-slug/` under a name that says what
it shows, and write `docs/bugs/AGENTS.md` if it was missing. The commit waits for the mirror, which
writes the issue number into the document.

### 9. Mirror to the tracker

Follow the creation sequence in `trackers/<name>/PROFILE.md`, on the project's own tracker per *Who
is authoritative for what* in `../../../.codefall/shared/workflow.md`. **The report is written even
when the mirror fails**: give the user the exact command that fixes the tracker and say the mirror
is pending.

### 10. Commit and wrap up

Land it per `../../../.codefall/shared/landing.md`: commit by path — the report, its attachments,
the `AGENTS.md` — push, and open the pull request with `Relates to #<issue>` in its body when the
mirror made one, a draft only when the status is `Draft`. Never add the `auto-merge` label: a person adds it when they want
the report merged.

Report the path, identifier, status, severity, the reproduction outcome, every open question, the
issue with its link, the branch, and the pull request, worded as the landing procedure says: it is
open at its URL; add the `auto-merge` label when you want it merged, or have someone approve it;
either one merges it.
**End with one command**: `/codefall-plan BUG-NNN`.

## Other modes

Invoked on an existing report, this skill does one of three things. Ask which if it is not obvious.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft`.
- **Edit** a `Draft` or `Ready` report: new evidence, a new reproduction attempt, an answered open
  question. A new criterion takes the next `AC` number; retired numbers stay retired. Re-mirror
  afterwards.
- **Archive** a report with its reason line, move it and its attachments to `docs/bugs/archive/`, and
  close or relabel the issue per the tracker profile.

Each lands per `../../../.codefall/shared/landing.md`.

## Rules

- **Nothing is written without the user confirming the document first.**
- **The document is canonical**; the tracker issue is regenerated from it.
- **A report says what is wrong, never why.** A suspected cause the reporter offers goes under
  Open questions for `codefall-plan`, marked as theirs.
- **Reproducing never changes anything** — no code, no data outside what the steps themselves do,
  no mocks, no test files.
- **A report that was not reproduced is still written**, with what was tried.
- **Acceptance criteria are EARS**, and cite a spec criterion rather than restate it.
- **Numbering is append-only**; retired numbers are never reused.
- **Status describes the document, never the work.**
- **Push back once, then defer** — on vagueness, on severity, on whether it is a bug at all.
- **Unresolved is recorded** as open questions, never dropped.
- **Never overwrite a file that has drifted.** Show the difference and ask.
