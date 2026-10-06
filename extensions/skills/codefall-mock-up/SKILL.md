---
name: codefall-mock-up
description: Get the visual surface of a feature into the repository under docs/mockups/ — import what a design tool exported, or make the mockup here, matching the application's own design system so it looks like it belongs, static or working depending on what the open question is.
argument-hint: "[the screen or surface, or a path to a mockup you already have]"
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
---

# Mock-up

Show what a feature looks like, so that `codefall-design` and `codefall-implement` are not guessing at it.

Two ways in. Either the user has a mockup and this **imports** it, or they do not and this **makes**
one. Making one is the interesting half, and the one place in this extension where inventing something
is the job rather than the failure.

The output is `docs/mockups/<slug>/` — the files, and a `README.md` saying what each one shows.

`codefall-specify` runs this skill for each requirement with a visual surface, and `codefall-design`
runs it for a requirement still waiting on one; a person can also run it alone, before or after a
spec. A mockup can be what makes the requirements obvious, or it can be drawn once they are settled.
**Run from another verb**, the spec and the surfaces are given, the files go on that verb's branch,
and that verb lands them: steps 1, 6, and 10 say what is skipped.

Plugin paths in this document — the ones that start with `../` — are relative to this skill's
directory, the one holding this `SKILL.md`. Resolve them from where the file lives; they are not
relative to the user's project. A path through `../../../.codefall/` is the one that leaves the
skills directory: it names a file `codefall init` installed in the project's own `.codefall/`.

## What it decides, and what it leaves alone

| This skill | `codefall-design` |
| --- | --- |
| What is on the screen, and its hierarchy | Which component or module renders it |
| The states the surface has | Where the data comes from |
| Realistic content at realistic lengths | Field names, types, schemas, endpoints |
| How an interaction behaves, when that is the open question | How that interaction is built in the app |

**Even a working mockup is a drawing.** One that runs proves the interaction; it does not become the
component. It was written to answer a question, not to the application's conventions, and it gets
rebuilt properly. That is the first rule in the directory's `AGENTS.md` and it holds for every file
this skill produces.

## The two modes

| The user has | Mode |
| --- | --- |
| Exported files, a directory of them, a design-tool URL, or an image pasted into the conversation | **Import** — follow `../../../.codefall/shared/import-mockup.md` |
| Nothing, or only part of what they need | **Make it** |
| A mockup already in `docs/mockups/<slug>/` that needs another state | **Amend** — add a file, never redraw the set |

A user who would rather draw it properly in their own tool should hear that running this again with
the export takes a minute. That is a remark, not a refusal — if they want something to look at in the
meantime, make it.

### Import

`../../../.codefall/shared/import-mockup.md` is the whole procedure, and it is shared with `codefall-specify` so the two
never drift. Follow it as written: what to accept, where the files land, the `README.md`, and the
rules about never editing or interpreting what the user brought. Then go to
[step 8](#8-clear-the-requires-mockup-label).

## Where mockups live

```
docs/mockups/<slug>/
```

`<slug>` names the **surface** — `booking-history`, `trip-share` — never the spec or the vision that
prompted it. One screen gets touched by several specs over its life and outlives all of them, so
filing it under whichever spec arrived first makes the second one either duplicate it or reach into
another spec's directory. `codefall-specify`'s SKILL.md carries the same rule for the same reason.

A run that covers several surfaces writes several directories. Reuse a surface's existing slug when
it has one; adding a state to a surface that has a directory is an amendment to that directory.

### The directory's rules file

`docs/mockups/AGENTS.md` is written when the directory is created, and added on a later run if it is
missing. `../../../.codefall/shared/import-mockup.md` writes the same file, so a directory reached from either
verb carries it.

**Never overwrite a file that has drifted.** When one exists and its content differs from what this
skill ships, show the user the difference and ask. The difference is a hand edit until they say
otherwise, and a repair that silently discards someone's rule is worse than a stale file.

```markdown
# docs/mockups — operative rules

- A mockup is a drawing of a surface. Never copy its markup, styles, or class names into the
  application — it is a picture of the outcome, not a draft of the implementation.
- A working mockup is still a drawing. It proves an interaction and gets rebuilt properly.
- Mockups are keyed by surface, never by spec or vision. One mockup serves several specs.
- Imported assets are the record of what someone decided. Never edit one; add alongside it.
- The `README.md` says what each file shows. Keep it current when files are added or replaced.
```

## Making one

### It should look like it belongs in the app

**This is a mockup, not a wireframe.** The default is as close to what would ship as the repository
lets you get — the application's own colors, type, spacing, controls, and vocabulary, so that someone
opening it sees their product with a new screen in it rather than a grey diagram of one.

A grey box drawing is a wireframe, and it is one thing this skill can make. It is the right answer
when there is nothing to match yet, or when structure is genuinely the only open question. It is not
the starting point.

So look before drawing. In the repository:

- A design system or component library — the real controls, and what they are called.
- Design tokens: CSS custom properties, a Tailwind config, a theme or brand file, a type scale.
- Existing screens or pages of the same kind, read for layout conventions and density.
- The global stylesheet and the fonts actually in use.

**Inline what you find.** Copying the token values into the file's `<style>` block usually gets most
of the way there and keeps the file self-contained. Reach for the app's real stack — a pinned Tailwind
or component-library build from a CDN — when inlining will not carry the look, and say once that the
file then needs a network to open.

Say what you found and what you are matching before you draw. When there is nothing to match, say
that too, then either take a direction from the user or pick one and name the choice.

### Defaults, not rules

Where to begin, not what is permitted. Each gives way to a user who wants something else, and each
carries the reason, which is worth saying once when it looks like it might matter.

| Default | Because | Go the other way when |
| --- | --- | --- |
| Match the app's design system | A mockup that looks like the product gets judged on the decision it is asking about | There is nothing to match, or the user wants it deliberately rough |
| Self-contained — inline tokens, styles, and script | The file still opens years after the tool that made it is gone | A real library is what makes it faithful; pin the version |
| Static HTML, one file per state | A state behind a click is a state nobody reviews | The interaction itself is the open question |
| Realistic content, never lorem ipsum | Even-length placeholder text hides the wrapping that breaks layouts | Never, in practice |
| One stated viewport, drawn as a frame | It keeps breakpoint decisions from being made by accident | The reflow is the thing being decided |
| Screens as HTML | It opens anywhere and diffs in a pull request | The surface is not a screen — a terminal transcript, an email, a printed page |

### When there is more than one good answer

Ambiguity is normal, and it is the best reason to make more than one thing. When the layout, the
emphasis, or the interaction has two or three reasonable answers and the choice actually matters,
**build them and let the user look** rather than picking quietly and presenting it as settled.

Two or three, not six. **Name them by what differs** — `list-table.html` and `list-cards.html`, not
`-a` and `-b` — so the distinction survives outside the README. Say in one line each what the
difference is and what each one is better at, and say which you would pick.

When the user chooses, the chosen one takes the plain name, the others come out unless they want them
kept, and the README's `Changes` line records what was decided. Do not manufacture options where
there is an obvious answer.

### Working mockups

When the question is how something behaves rather than what it looks like — a picker, a multi-step
flow, a drag interaction, a filter that has to feel right — build it working. That is a good use of
this skill and the reason the static default is only a default.

Two things still hold. Every state a reviewer needs to see is reachable without reading the code,
which usually means the working file plus a still of each state that is hard to reach. And it is
still a reference: it proves the interaction, and the implementer rebuilds it in the app's stack.

### States worth having

A screen made only in its happy state is the most common way a feature comes back from review. Ask
which apply:

| State | Why it earns a file |
| --- | --- |
| Populated | The state everyone pictures already |
| Empty | First-run looks nothing like the populated screen, and nobody describes it |
| Loading | When it is slow enough to matter |
| Failed | What the user is told, and what they can do next |
| Denied | When the surface is permission-gated |
| Overflowing | Long content, many rows, a name that does not fit |

Populated, empty, and the primary failure are the ones to push for. If the user says a state does not
apply, take that and move on.

### The README

```markdown
# <Surface> mockups

Made <date> by `/mock-up`. <One line: what kind of thing these are, and what they were matched
against — the app's design system, a named theme, nothing yet.> <Viewport, if it matters.>

- `<file>` — <one line: which screen or state this shows>

## Changes

- <date> — <what moved, or which option was taken, and why>
```

Omit `Changes` until there is one. Keep the per-file lines current: an implementer opening a
directory of files cannot tell the empty state from the error state, and the person who could tell
them is no longer in the conversation.

**A directory can hold both imported and made files.** A user who exported two screens and wants the
failure state drawn gets exactly that, and the README says which is which. **Never redraw an imported
asset** — make the missing state alongside it.

## Project customizations and persona

Follow `../../../.codefall/shared/customizations.md` for this verb. Read the `persona=` line of the
preflight report, or `persona` in `.codefall/user.json` when this verb runs no preflight; when it is
not `engineer`, follow that persona's section in `../../../.codefall/shared/personas.md` for this
run, and say so.

## Process

### 1. Ask what this is for

**No precondition check.** This skill needs a place to write and nothing else.

**Run from `codefall-specify` or `codefall-design`**, the spec and the surfaces are given: read the
spec and go to step 2. Otherwise ask one question, and offer the three answers:

> Is this mockup for a spec, for a vision, or are we starting fresh?

**For a spec.** If they named one, take it. Otherwise read `docs/specs/` and list what is there —
identifier, title, and status — and let them pick:

> SPEC-002 — Trip export (Ready)
> SPEC-004 — Booking history (Ready, two requirements marked `requires-mockup`)
> SPEC-005 — Share links (Draft)

Then read it. A spec answers most of what an interview would: the surface, who the consumer is, and
what they observe when it works. Requirements marked `requires-mockup` are the ones waiting on this
run.

**For a vision.** Same shape — take the one they named, or list the live visions in
`docs/visions/`, not `archive/`. Then read it.

A vision is a wider frame than a spec and usually names several surfaces, so a run from one can be
large. **Say the size out loud, then do what they ask:**

> That vision touches four surfaces. Making all of them is a dozen files or so. All four, or start
> with the one you most want to see?

**Starting fresh.** Plenty of surfaces get drawn before anyone writes anything down, and that is one
of the reasons this skill exists. Ask what they want to see and go.

If neither directory exists, do not ask the question — say so and start fresh.

Read `docs/mockups/` either way, for the slugs already in play.

### 2. Name the surface

Settle the slug before anything is written — the surface, not the spec or the feature request. Reuse
an existing one where the surface already has a directory.

**Do not ask where the files go.** `docs/mockups/<slug>/` is the answer.

### 3. Pick the mode

Ask what they have and route with [the modes table](#the-two-modes).

### 4. Survey what it should look like

Read the repository for the application's visual language, following
[It should look like it belongs](#it-should-look-like-it-belongs-in-the-app). Report what you found
and what you intend to match, in a sentence or two, before any drawing starts. This is where the
mockup stops being generic, and it costs one pass over the codebase.

### 5. Fill the gaps

Only what a spec, a vision, the survey, or the user's own description did not already answer:

1. Which screen, and who is looking at it.
2. What is on it, and the one thing someone comes here to do.
3. Which states apply — walk [the states table](#states-worth-having).
4. Whether anything needs to actually work.
5. Anything the survey should have found and did not.

**Two rounds is a cap on your insistence, not on the conversation.** Stop pressing an unanswered
point after the second try, make the sensible choice, and say in the recap that you assumed it.

**Do not bikeshed.** Which of two labels reads better is not worth a question — make one and let the
user move it.

### 6. Branch, make one thing, then stop

Before the first file, read `../../../.codefall/shared/landing.md` and take its branch step:
`mockup/<slug>`, from the default branch. **Run from another verb, stay on its branch**; the files
are part of its commit.

Build the first screen and show it: the path, and what to open it with.

When that first screen is genuinely ambiguous, this is where the options go — build the two or three
that are worth looking at, following
[When there is more than one good answer](#when-there-is-more-than-one-good-answer).

This is the step that matters most. Twelve files in a direction the user dislikes is twelve files of
rework, and the corrections that arrive on the first one — density, ordering, vocabulary, how
finished it should look — apply to all of them.

Advance when they have actually looked at it.

### 7. Make the rest, and write the directory

Apply every correction from the first one. Then write `README.md`, and `docs/mockups/AGENTS.md` if it
was missing. Report each file with what it shows.

Nothing is committed yet: the label and the back-link come first.

### 8. Clear the `requires-mockup` label

Follow the **Hand back** section of `../../../.codefall/shared/import-mockup.md` — it works the same
for a mockup made here as for an imported one. `codefall-specify` applies the label per requirement,
so clear the issues this mockup covers, leave the rest, and clear none of them unless files actually
landed.

### 9. Link it back

When a spec prompted this run, offer to add the directory path to its **Design notes**. When a
vision did, offer to add it to the vision's `Related` line. Add the path and change nothing else in
either document — they belong to `codefall-specify` and `codefall-envision`.

### 10. Land and wrap up

**Run from another verb**: land nothing. Hand back the directories and the per-file lines, and stop;
the calling verb commits them with its document.

Otherwise land the run per `../../../.codefall/shared/landing.md`: commit the mockup directory, the
`AGENTS.md` if it was written, and the spec or vision if one was edited — by path, nothing else —
push, and open the pull request once the person has seen the last file. Never add the
`auto-merge` label: a person adds it when they want the mockup merged.

Do not create issues. Do not start a design.

Report the directories, every file with what it shows, what the mockup was matched against, options
offered and which was taken, states deliberately not made and why, anything assumed because the user
did not answer, any labels cleared, the branch, and the pull request, worded as the landing
procedure says: it is open at its URL; add the `auto-merge` label when you want it merged, or have
someone approve it; either one merges it. **End with one command**: `/codefall-design SPEC-NNN`
where a spec was waiting on this mockup; otherwise say in a sentence that the mockup is in place
and nothing is pending.

## Rules

- **Look before drawing.** A mockup that ignores the design system sitting in the repository is worth
  less than the hour it took.
- **Show the first one before making the rest.**
- **Offer options where the answer is genuinely open**, and say which you would pick.
- **Never edit an imported asset.** It is the record of what someone decided. Add alongside it.
- **A mockup is a reference, never source.** Working ones included.
- **Mockups are keyed by surface**, never filed under a spec or a vision.
- **Say what it looks like, not how it is built.** Naming a component, a route, or a data source is
  `codefall-design`'s work happening in the wrong document.
- **Clear `requires-mockup` only when files landed**, and only on the issues this mockup covers.
- **Nothing lands outside `docs/mockups/`.** No application code, no styles, no components.
