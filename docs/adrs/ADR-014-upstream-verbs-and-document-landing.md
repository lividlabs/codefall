# ADR-014: Verbs Run Their Upstream Verbs, and Documents Stack

## Status

Accepted — 2026-10-05

## Context

A product manager ran `specify` on a small feature. The run wrote the spec and the mockups as two
pull requests, and when feedback on the mockups changed the spec, the report ended with four
instructions: merge two pull requests, run `specify` again for the amendment, run `specify` for a
second spec, run `design`. The person had asked for a feature; they were handed a procedure.

Four things produced that report, and each was a rule written on purpose.

**The invocation flag.** `codefall-envision`, `codefall-specify`, `codefall-report`,
`codefall-fix`, `codefall-mock-up`, `codefall-scaffold`, `codefall-upgrade`, and `codefall-equip`
carried `disable-model-invocation: true`, so an agent could not run them. It was put on to keep
those verbs deliberate and, under ADR-004.2, to keep their descriptions out of the session listing.
Its effect in practice was that a verb which found its upstream document wanting could only tell
the person to go run the upstream verb. The flag exists only in Claude Code; the other harnesses
never had it, so the same skill behaved differently per harness. A coworker who wanted a mockup
while writing a spec hit it directly.

**The amendment rule's scope.** ADR-008 lets a verb amend an upstream document when the amendment is
text that moves no work. The limit protects beads that someone may already hold. Before `design`
runs there are no beads, so for a spec and its mockups the limit protected nothing and still sent
anything larger than a text fix back to the person.

**One verb, one pull request.** The shared landing procedure had every document verb branch from the
default branch and open its own pull request, and offer the merge at the end. A spec effort with a
mockup was two pull requests to review and two merges to click, each a human step, before `design`
could start.

**The persona's last line.** The product-manager persona's rule said the last line of every report
"still names the exact next command". That sentence, written when the person had to type the next
verb, produced the list of four.

GitHub shipped stacked pull requests in public preview on 2026-07-30, after the training data of the
models that run the skills. A stack is a chain of pull requests each based on the one below; each
layer shows only its own diff, merging a layer merges everything below it, merging the top lands the
stack, and GitHub rebases the layers above a merged one. The shared `stacks.md` reference records
the feature from GitHub's own documentation so no run works from a model that has never seen it.

## Decision

### The invocation flag stays on two verbs

`codefall-upgrade` and `codefall-equip` keep `disable-model-invocation: true`, because each changes
the install or the project's settings. Every other verb drops it. The listing cost is six more
descriptions, each under 400 characters. A skill that runs another skill lists `Skill` in its
`allowed-tools`.

### A verb runs the verb upstream of it

When the work needs an upstream verb's judgment, the running verb runs that verb, in the same
session and on the same branch, under the confirmation it already holds, and asks the person only
for a product decision. It never tells the person to run a verb and come back.

- `specify` runs `mock-up` for each requirement with a visual surface and no mockup, and runs
  `envision` when the vision needs more than a text amendment.
- `design` runs `specify` to settle and promote a `Draft` spec, and runs `mock-up` for a requirement
  still labelled `requires-mockup`, instead of refusing to start.
- `mock-up`, run from another verb, takes the spec and the surfaces as given, writes on the calling
  verb's branch, lands nothing itself, and hands the paths back.

After `design` has created beads, ADR-008's limit still applies to amendments, and the
`design-revision` bead remains the path for a change that moves work.

### Whether a requirement gets a mockup is `specify`'s conversation

`specify` decides, per requirement, whether there is a visual surface and whether a mockup exists.
`mock-up` is the tool it picks up to make one. Only a requirement whose mockup the person chose to
skip is labelled `requires-mockup`, and `design` clears that by running `mock-up`, not by stopping.

### One delivery's documents are one stack

Each document verb branches from the layer below when that layer has an open pull request: `specify`
from the vision's, `mock-up` from the spec's, `design` from the mockups' or the spec's. It opens its
pull request with that branch as base and links it into the stack with `gh stack link`. Nobody is
asked to merge until `design` is done; merging the design pull request lands the whole stack. A
repository that wants the stack to land itself when the design pull request is marked ready says so
in a shared landing customization, `.codefall/skills/shared/CUSTOMIZE.md`, and sets up the Action
the recipe under `docs/recipes/` describes. `implement`'s task pull requests stack the same way,
from `main`, after the documents land.

Push and pull request stop being a question. The document was confirmed before it was written, and
the pull request is how it reaches the next verb.

### Integration is GitHub's

`implement`'s integration step asks GitHub whether each open layer is mergeable rather than merging
in a scratch worktree. A lower layer is fixed on its own branch and cascaded with `gh stack rebase`
and `gh stack push`. Nothing is rebased or force-pushed by hand; the extension does the cascade and
GitHub rebases on merge. ADR-013.3 carries this into the deliveries decision.

### One next step

For the product-manager persona, the last line of every report says in one plain sentence what
happens next and names exactly one command when there is a step for the person to start. Never a
list. Everything else the verb could do, it did in the run or asked about in a sentence. The
engineer persona is unchanged.

## Consequences

- A spec effort is one run and one pull request, mockups included, and a person is never handed a
  verb to run between two verbs the agent could have run itself.
- The document stack means two merge clicks per delivery at most, one for the documents at the
  design pull request and one for the code, and none for the documents in a repository that lands
  the stack itself.
- Six more descriptions sit in every session's listing. ADR-004.3 records the new count.
- `Skill` joins `allowed-tools` for `specify` and `design`. Harnesses without the flag or the tool
  list run as before.
- The flag's deliberateness is replaced by the confirmation every verb already holds. A verb that
  an agent starts still writes nothing the person has not seen.
- The design verb's "spec not ready" gate changes from a stop to work it finishes, so a `Draft` spec
  or a missing mockup costs a question inside the run, not a round trip.
- The recipe for self-landing stacks is documentation, not shipped behaviour; `equip` could automate
  it later, and the recipe says so.

## Related

- [ADR-004.3](ADR-004.3-skill-length-guidelines.md) — which skills carry the invocation flag.
- [ADR-008](ADR-008-upstream-amendments.md) — text amendments; this ADR adds running the upstream
  verb before beads exist, and leaves ADR-008's limit standing after.
- [ADR-011](ADR-011-personas.md) — personas; the product-manager report rule changes here.
- [ADR-013.3](ADR-013.3-deliveries.md) — deliveries; integration moves to `gh stack`.
- `extensions/shared/stacks.md` — how GitHub stacked pull requests work.
- `docs/recipes/landing-document-stacks.md` — landing a document stack without a merge click.
