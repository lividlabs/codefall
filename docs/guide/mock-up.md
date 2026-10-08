# /codefall-mock-up

`/codefall-mock-up` puts a picture of a screen into the repository, so that the people and agents
who plan and build a feature are not guessing what it looks like. Use it before or after writing a
[spec](specify.md): a mockup can be what makes the requirements obvious, or it can be drawn once they
are settled.

```
/codefall-mock-up the itinerary export dialog
```

## What it writes

Mockups go in `docs/mockups/<slug>/`, one directory per screen, with a `README.md` that says what
each file shows. The slug names the screen, or *surface*, such as `booking-history` or
`trip-share`, and never the spec that prompted it. One screen gets changed by several specs over its
life and outlives all of them. If its mockup were filed under whichever spec came first, the second
spec would have to duplicate it or reach into another spec's directory. Specs link to mockups by
path under their **Design notes**.

The skill writes its files only under `docs/mockups/`, never application code. With your agreement,
it also adds the mockup's path to the spec's **Design notes** or the vision's `Related` line,
whichever prompted the run. It commits on a branch named `mockup/<slug>` and opens a pull request
once you have seen the last file. When another skill runs it, the files go on that skill's branch
instead. You merge it like any other document, as
[`docs/landing-documents.md`](../landing-documents.md) describes.

## Importing a mockup you already have

When you already work in a design tool, the skill imports what you exported, such as a file, a
directory, or an image, and never changes it afterwards. An imported file is the record of what
someone decided, and redrawing it would lose that. When a state is missing, the skill draws it
alongside the imported files, and the README says which files were imported and which were drawn.

## Making a new one

When you have nothing to import, the skill makes the mockup. It opens by asking whether the mockup
is for a spec, for a vision, or a fresh start, and lists the specs or visions that exist so you can
pick one. A [vision](envision.md) usually spans several screens, so the skill tells you how large the
run would be before starting.

**It draws a mockup, not a wireframe.** Before drawing anything, it reads the repository for your
design system, your design tokens (the named colors, sizes, and spacing values your styles use),
your existing screens, and the fonts in use. It then copies what it found into the file, so the file
stays self-contained. The goal is as close to what would ship as the repository allows: someone
opening it should see your product with a new screen in it, not a grey diagram. A grey box drawing
is still available when there is nothing to match yet, or when layout is the only open question.

**It shows you the first screen before making the rest.** Corrections to the first one, such as
density, ordering, wording, or how finished it should look, apply to all of them.

**It has defaults, each with a reason and a case for going the other way:**

| Default | Reason | Go the other way when |
| --- | --- | --- |
| Static HTML, one file per state | A state hidden behind a click is a state nobody reviews | How it behaves is the open question |
| Self-contained file | It still opens years after the tool that made it is gone | A real library is what makes it faithful; the version is pinned |
| Realistic content | Placeholder text hides the wrapping that breaks layouts | Never, in practice |
| One stated viewport | Layout decisions for other screen sizes are not made by accident | The reflow is what is being decided |

**It builds working mockups when behavior is the question.** For a date picker, a multi-step flow,
or a filter that has to feel right, the skill builds the interaction working. A working mockup is
still a reference: it proves the interaction, and the implementer rebuilds it in the app's own code.

**It offers options when the answer is open.** When a layout has two or three reasonable answers and
the choice matters, the skill builds each one and names the files by what differs, such as
`list-table.html` and `list-cards.html`. It says in a line what each is better at and which it would
pick. The one you choose takes the plain name, and the README records the decision.

## Which states to draw

The empty and error screens are the ones nobody describes in an interview, and they are where
features come back from review. The skill pushes for at least three states: populated, empty, and the
main failure. It also asks about loading, permission-denied, and overflowing content where they
apply.

## How mockups connect to specs

When [`/codefall-specify`](specify.md) records a requirement whose screen has no picture yet, it runs
this skill for that screen in the same run and on the same branch, so the spec and its mockups arrive
together. A requirement whose mockup you chose to skip gets a `requires-mockup` label on its GitHub
issue, and [`/codefall-plan`](plan.md) runs this skill for it before planning, instead of refusing.
When the mockup's files land, the skill removes the label from the issues it covers.
