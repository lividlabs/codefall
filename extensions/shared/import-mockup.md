# Importing a mockup

Shared procedure. `codefall-specify` follows it when a user brings a mockup mid-interview; `codefall-mock-up` follows it
for its import mode. Defined once so the two never drift.

The job is narrow: get the user's existing mockup into the repository at a stable path, and hand back
that path. Importing is not reviewing, redrawing, or improving.

## Contents

- Accept
- Where it lands
- Rules
- Hand back
- Committing

## Accept

| The user has | Do this |
| --- | --- |
| Exported files — PNG, JPG, SVG, PDF | Copy them in |
| A directory of exports | Copy the directory contents in |
| A design-tool URL — Figma, Sketch, Penpot | Record the URL; do not scrape it |
| An image pasted into the conversation | Save it in |
| Nothing yet | Stop; this is not an import |

**A URL is a weaker artifact than a file** and worth one sentence to the user: a link can move, change
under them, or sit behind a login the implementer does not have. Ask whether they can export instead.
If they cannot, record the URL and move on — do not press twice.

## Where it lands

```
docs/mockups/<slug>/
```

`<slug>` is a short kebab-case identifier for the feature — `booking-history`, `trip-share`. Reuse the
feature's existing slug when one is already in play.

Alongside the assets, write a `README.md`:

```markdown
# <Feature> mockups

Imported <date> from <source — tool name, or "user-supplied export">.

- `<file>` — <one line: which screen or state this shows>
- `<file>` — <one line>

<Only when the source is a URL:>
Source: <url>
```

The one-line descriptions matter more than they look. An implementer opening a directory of
`Frame 12.png` files cannot tell the empty state from the error state, and the person who could tell
them is no longer in the conversation.

### The directory's rules file

`docs/mockups/AGENTS.md` is written when the directory is created, and added on a later run if it is
missing. `codefall-mock-up` writes the same file, so a directory reached from either verb carries it.

**Never overwrite a file that has drifted.** When one exists and its content differs from what is
below, show the user the difference and ask. The difference is a hand edit until they say otherwise.
Replace it only on a yes; on a no, leave it and say nothing further about it.

```markdown
# docs/mockups — operative rules

- A mockup is a drawing of a surface. Never copy its markup, styles, or class names into the
  application — it is a picture of the outcome, not a draft of the implementation.
- A working mockup is still a drawing. It proves an interaction and gets rebuilt properly.
- Mockups are keyed by surface, never by spec or vision. One mockup serves several specs.
- Imported assets are the record of what someone decided. Never edit one; add alongside it.
- The `README.md` says what each file shows. Keep it current when files are added or replaced.
```

## Rules

- **Never edit the assets.** An imported mockup is a record of what the user decided. Redrawing it is
  `codefall-mock-up`'s creation mode, and it is a different act with different consent.
- **Keep the original filenames** where they are legible. Rename only what is meaningless (`image.png`,
  `Untitled-1.svg`), and say in the README what the original was called.
- **Do not interpret the design.** No component names, no annotations about which existing widget to
  reuse, no notes about what should be built differently. That reading is `codefall-plan`'s, and doing it
  here puts solution decisions into a requirements artifact.
- **Ask before overwriting.** If `docs/mockups/<slug>/` already has contents, show the user what is
  there and let them choose: replace, add alongside, or use a different slug.

## Hand back

Return the directory path — `docs/mockups/<slug>/` — to whatever is calling this procedure. The
caller decides what to do with it.

When issues carried `requires-mockup`, a successful import clears it:

```bash
gh issue edit <number> --remove-label "requires-mockup"
```

`codefall-specify` applies that label per requirement, so a feature can have several issues carrying it. Clear
it on the ones this mockup covers and leave the rest, and clear it only when files actually landed. A
recorded URL with no export is a judgement call — ask the user whether the link is enough to design
against, and leave the label on if they are unsure.

## Committing

**This procedure writes files and does not commit them.** The verb that called it lands them with
the rest of its run, per `landing.md` beside this file: `codefall-specify` with the spec,
`codefall-mock-up` with the mockup directory, `codefall-envision` with the vision. Say what was
added and hand back.
