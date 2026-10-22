---
name: codefall-report
description: Turn a bug someone ran into into a report another session can fix — interview the person who saw it for the steps, the expected and actual result, screenshots, and the environment, push back on vague answers, try to reproduce it on the spot, then write a bug report with acceptance criteria into the repository, mirrored to the issue tracker, and file its bead triaged by type, area, priority, and size. Also triages the backlog — `triage`, with an area to narrow it, every open item whatever its type — and creates a sweep epic from the items the person picks.
argument-hint: "[what went wrong, or a test run's report | triage [area]]"
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
- `../../../.codefall/shared/triage.md` — what triage takes, whose an item is, how bad it is, which
  area, how big the fix is, every field its bead carries, and sweeps. Read at step 6, and for the
  triage mode.

## Scope — what is wrong, not why

| In scope | Out of scope | Whose |
| --- | --- | --- |
| What the reporter did, expected, and saw | Why the code does it | `codefall-plan` |
| Reproducing it through the running product | Changing code, data, or a test to see what happens | `codefall-plan`, `codefall-implement` |
| Evidence: screenshots, recordings, logs, output | The fix, and its task graph | `codefall-plan` |
| The expected behaviour, as acceptance criteria | Specifying a capability nobody built: its requirements beyond what the reporter expected | `codefall-specify` |
| Severity as the reporter judges it, and the triage it sets: priority, area, size | Assignment | the team |

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

The Beads lines stop nothing: a report is written whether or not Beads is ready, and step 10 says
the bead is pending when it is not. Keep the checkout, `refresh=`, and `local` lines for step 5.
The triage mode is the exception: `beads` not `ok` stops it, with the remedy
`codefall-implement`'s preflight table names.

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

**When the expected behaviour was never specified or built**, it reads as a request for something
new. Say so once, in the recap, and write the report anyway: its bead is typed `feature` rather than
`bug`, per *What triage takes* in `../../../.codefall/shared/triage.md`, and the rest of the triage
is the same. Never point the reporter at another verb instead of writing.

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
> **Triage**: bug · P2 · area:checkout · size=fix · beside every epic

**Every line is at least a sentence.** A fragment means that topic was not covered — go back and ask.

**The Triage line** is from `../../../.codefall/shared/triage.md`: the type, `bug` unless the recap
says what else it is, the priority the severity maps to, the area picked from the labels in use or
proposed, the size from `codefall-fix`'s scope test, and whose the item is, which is beside every
epic unless the reporter says it belongs to a delivery in flight. Each comes with its reason in a
line, and the reporter may change any of them here.

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

### 10. File the bead

One bead per report, with every field *What a triaged bead carries* in
`../../../.codefall/shared/triage.md` names: the ID `<prefix>-gh-<n>` when the mirror made an
issue and `<prefix>-BUG-NNN` otherwise, the Triage line's type, priority, area, and size,
`found=report`, `--spec-id` the report's path, `-s open`, and the body naming the report and its
severity. A parent only when the reporter put the item in a delivery in flight, or an open sweep
holds its area. `bd dolt push`. When Beads is not ready, say the bead is pending and give the
command; the report does not wait for it.

### 11. Commit and wrap up

Land it per `../../../.codefall/shared/landing.md`: commit by path — the report, its attachments,
the `AGENTS.md` — push, and open the pull request with `Relates to #<issue>` in its body when the
mirror made one, a draft only when the status is `Draft`. Never add the `auto-merge` label: a person adds it when they want
the report merged.

Report the path, identifier, status, severity, the reproduction outcome, every open question, the
issue with its link, the bead with its triage, the branch, and the pull request, worded as the
landing procedure says: it is open at its URL; add the `auto-merge` label when you want it merged,
or have someone approve it; either one merges it.
**End with one command**: `/codefall-fix BUG-NNN` when the size is `fix`, `/codefall-plan BUG-NNN`
otherwise.

## Other modes

Invoked on an existing report, this skill does one of three things. Ask which if it is not obvious.

- **Promote** `Draft` to `Ready`, or **reopen** `Ready` to `Draft`.
- **Edit** a `Draft` or `Ready` report: new evidence, a new reproduction attempt, an answered open
  question. A new criterion takes the next `AC` number; retired numbers stay retired. Re-mirror
  afterwards.
- **Archive** a report with its reason line, move it and its attachments to `docs/bugs/archive/`, and
  close or relabel the issue per the tracker profile, and close its bead with the reason line.

Each lands per `../../../.codefall/shared/landing.md`.

## Triage mode

`/codefall-report triage`, with an area after it to narrow the list, walks the backlog in Beads and
writes no document. Read `../../../.codefall/shared/triage.md` first; it holds every rule this mode
applies, and *What triage takes* is the first of them: every open item is triaged, whatever its
type and whatever its title calls it. Nothing is asked per item; the whole list is proposed once
and what the person takes is applied.

1. `bd dolt pull`. List the backlog: `bd list -s open,deferred --no-parent --exclude-type epic
   --sort priority --limit 0`, and with an area, `-l area:<name>`. Say how many items there are and
   how many lack a triage field: no `area:` label, no `size` in metadata, or a type that does not
   fit the description, `task` on a title that says something is wrong, `bug` on a request for
   something that never existed.
2. **Fill in what is missing.** For each item lacking a field, propose the field with its reason in
   a line, type included, the whole list in one block, and apply what the person takes with `bd
   update`. A type the person set themselves stays. Two spellings of one area are named and merged
   on yes with `bd label add` and `bd label remove`.
3. **Find duplicates.** Items whose titles or descriptions describe one thing are listed in pairs;
   on yes the later one is closed with `-r "duplicate of <id>"` and its evidence commented on the
   one kept.
4. **Re-triage priority.** An item whose priority does not fit its description under the shared
   scale is named with the priority proposed. Nothing changes unasked.
5. **Offer a sweep.** When an area holds more than a few open items, offer to create a sweep epic from
   the ones the person picks, per *Sweeps* in the shared reference: `bd create "<area> sweep"
   --id <prefix>-SWEEP-<area>-NNN -t epic`, NNN the next number free under that area, then
   `bd update <bead> --parent <sweep>` for each item taken, and `bd dolt push`. Say what was not
   taken and why it stays.
6. **Report.** The counts before and after, every field changed, every duplicate closed, the sweep
   with its children, and `bd dolt push`. **End with one command**: `/codefall-implement <sweep>`
   when one was created, else `/codefall-fix <bead>` naming the highest-priority `size=fix` item.

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
- **Push back once, then defer** — on vagueness and on severity. What kind of item it is goes in
  the bead's type, never in a refusal or a hand-off.
- **Unresolved is recorded** as open questions, never dropped.
- **Never overwrite a file that has drifted.** Show the difference and ask.
- **Every report files one bead**, triaged per the shared reference, and the triage mode changes a
  bead only on the person's word.
- **The triage mode writes no document**, skips no item for its type, and creates a sweep only from
  the items the person picked.
