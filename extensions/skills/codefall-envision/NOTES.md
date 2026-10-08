# codefall-envision — where it came from

Why `SKILL.md` looks the way it does. None of this is instruction — the skill is the instruction.
This exists so nobody re-adds something that was removed on purpose.

## Length is proportional to input

The job is to get the user's words onto paper in an order that reads well, not to fill in what they
did not say. A heading with invented content under it is worse than no heading, because a later
reader cannot tell which parts came from the user. Unknowns are correct output for the same reason:
pretending to have resolved something is a defect, and a vision with four open questions has done
its job.

## The off-ramp is about size, not concreteness

Every tell in the off-ramp table is about the size of the thing, or about the *why* already
existing. None is about how concretely the user can describe the *what*. Someone can name every
screen, state exactly what a user will see, and hand over finished mockups, and still have written
nothing down about the problem any of it solves — and that person needs a vision more than most,
because the reasoning exists only in their head.

## A vision can say how it works

An earlier version put "what a user will observe when it works" out of scope, which pushed visions
to leave out how the product works entirely. People describing an idea usually describe how it
works, and cutting that loses the part of their thinking that explains the rough shape. What
separates a vision from a specification is precision: a team can build a good deal from a spec
alone, and would struggle to build from a vision. So a vision may say how it works in broad
strokes, and stops before exact behavior, acceptance criteria, and data shapes.

## The floor is advice, not a gate

One bug, one screen, one endpoint, one field is usually smaller than a vision is worth. But the
user knows things about the work that the skill does not, and a vision nobody needed costs a file.
So the skill says what it thinks and writes the vision if that is what they want.

## Siblings, not a parent

Each problem that stands on its own is its own vision, and what connects siblings is the source
they came from: each one's `Related` line points at the same saved document or mockup directory.
The `specs` field on `Related` is a list for the same reason — one vision can produce several
specifications, and the vision is what groups them, which is why `codefall-specify` writes sibling
specs rather than a parent.

## `Active` belongs to `codefall-implement`

`Active` records the observable fact that work has started, at the first claim that traces back to
the vision. That is the one transition another verb makes, and the carve-out the repo's rules allow
for recording a fact. A vision still at `Ready` means `codefall-implement` has not run against it.
Two optional header fields, `Revised by` and `Replaced by`, exist because a vision can be under
active work and partly revised at the same time, and a single status line cannot say both.

## `docs/visions/AGENTS.md` stands in for an ignore file

The file says that `archive/` is history and is not read unless asked. That is the closest thing to
an ignore file that works across harnesses: Claude Code has no `.agentignore`.

## Mockups go to `docs/mockups/`

That is where `codefall-plan` and `codefall-implement` look for them, and a surface outlives the
vision that prompted it. The import procedure is shared with `codefall-specify` and
`codefall-mock-up` so the three never drift.

## The source is always saved

The source records what the user actually brought; the vision records what was made of it, and
someone may need to check one against the other. Reformatting a coherent document into the
template's headings destroys work and adds nothing, which is why a strict superset is adopted
unchanged.

## Ask once for material, then drop it

Someone who has a pitch document will say so. Chasing material the user did not mention wastes the
part of the process that produces original thinking, and questions asked mid-dump interrupt the only
part that produces original material.
