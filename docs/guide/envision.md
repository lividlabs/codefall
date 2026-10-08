# /codefall-envision

`/codefall-envision` writes down an idea before anyone specifies or builds it. Use it when you know
what problem you want to solve but have not settled exactly what the product should do about it.

```
/codefall-envision Travelers keep losing track of their bookings across airlines and hotels.
```

The result is a *vision*: a short, informal document that says what is wrong, who it affects, the
rough shape of an answer, and what nobody has decided yet. It is the first step in the sequence the
[README](../../README.md#how-it-works) describes, and it is optional.

## What it writes

The skill writes the vision to `docs/visions/VISION-001-slug.md`, where the number is the next one
free and the slug is a short name. Two sections are always there:

- **Problem:** what is broken, missing, or costing something, and who feels it.
- **Proposed shape:** the rough shape of an answer, in broad strokes.

Other sections appear only when you said something that belongs in them: Why now, Non-goals,
Environment and constraints, Open questions, and Alternatives considered.

The document's length follows what you put in. Two paragraphs is a complete vision, and so is a
page. When you don't know something, the vision records it as an open question instead of inventing
an answer. A vision stops short of the detail anyone could build from; that detail belongs in a
[spec](specify.md).

The skill shows you the whole document before it writes anything. It then commits the vision on a
branch named `vision/VISION-001-slug`, pushes it, and opens a pull request. You merge that pull
request yourself, or you add the `auto-merge` label to it or have someone approve it, if the
project has installed the merge action described in
[`docs/landing-documents.md`](../landing-documents.md).

## What you can bring

The skill takes whatever you arrive with: a sentence, ten minutes of thinking out loud, a pitch
document, a whiteboard photo, or a folder of exports from a design tool. It saves everything you
bring and never edits it.

| What you bring | Where it goes |
| --- | --- |
| Pictures of a screen someone will build, such as design-tool exports or a mockup set | `docs/mockups/`, where `/codefall-plan` and `/codefall-implement` look for them |
| Everything else: notes, transcripts, pitch documents, sketches of the problem | `docs/visions/sources/`, saved word for word |

The vision's `Related` line links to both. When a pitch document already has everything a vision
needs, the skill adopts it as the vision without rewriting it.

Arriving with finished screens is no reason to skip this skill. You can have every screen drawn and
still have written nothing down about the problem they solve, and that is the case the skill helps
with most.

## When one idea is several

A large pile of material usually holds more than one problem. A pitch document can cover three, and
a set of mockups that spans six screens usually does. The skill proposes where the problems divide,
and you decide. Each problem that stands on its own becomes its own vision, such as `VISION-003`,
`VISION-004`, and `VISION-005`. They are siblings, not a parent and its children, and each one's
`Related` line points at the source they all came from.

## How visions relate to the other skills

`/codefall-scaffold` asks for a vision before it starts a new project, and offers to run this skill
when there isn't one. Without a vision, scaffold has to answer its architecture questions with
defaults based on a one-sentence description, and that is where new projects go wrong. With a
vision, most of those questions already have answers, so the scaffold session is shorter and the
answers are better. You can decline, and scaffold records that it ran without a vision.

`/codefall-specify` can use a vision as context but never requires one. A vision says *why*, and a
spec says *what*. They are different documents, and plenty of features need only a spec. When what
you describe is already a single, fully described feature, this skill says so once and offers to
switch to `/codefall-specify`.

## Statuses

A vision's `Status` line holds one word and a date.

| Status | Meaning | Set by |
| --- | --- | --- |
| `Draft` | You said you are stopping and will come back to it | this skill |
| `Ready` | Written and agreed; the normal end of a session | this skill |
| `Active` | Work has started against it, and the vision is frozen | `/codefall-implement` |
| `Archived` | Wholly replaced, or dropped | this skill |

To change a vision later, run the skill on it. It can promote a `Draft` to `Ready`, edit a `Draft`
or `Ready` vision in place, or archive one. An `Active` vision cannot be edited, so revising part of
it creates a new vision and adds a `Revised by` line to the old one. Replacing a vision entirely
archives it: the file moves to `docs/visions/archive/` under the same name, and gains a
`Replaced by` line when something took its place. A vision's number is never reused, so citations of
an archived vision still resolve.
