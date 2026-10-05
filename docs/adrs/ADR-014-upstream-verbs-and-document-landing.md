# ADR-014: Verbs Run Their Upstream Verbs, and Documents Land on Their Own

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
never had it, so the same skill behaved differently per harness.

**The amendment rule's scope.** ADR-008 lets a verb amend an upstream document when the amendment is
text that moves no work. The limit protects beads that someone may already hold. Before `design`
runs there are no beads, so for a spec and its mockups the limit protected nothing and still sent
anything larger than a text fix back to the person.

**One verb, one merge click.** The shared landing procedure had every document verb open its own
pull request and offer the merge at the end. A spec effort with a mockup was two pull requests to
review and two merges to click, each a human step, before `design` could start. The person had
already read and confirmed each document inside the session; the merge click added nothing but a
wait.

**The persona's last line.** The product-manager persona's rule said the last line of every report
"still names the exact next command". That sentence, written when the person had to type the next
verb, produced the list of four.

Two further things came up while settling this. Under the product-manager persona, `design` set
every technical decision aside into **Decisions needed**, ended `Draft`, created no beads, and
handed the document to an engineer; a product manager alone could never reach `implement`. And
`review` and `test` on an epic's work filed children on the epic and ended by naming
`/codefall-implement <epic>` for the person to type, which is the same hand-over as the one the
flag caused.

A first version of this decision stacked a delivery's documents as GitHub stacked pull requests,
with one merge at the design pull request. It was reverted before it merged: a stack put three
documents behind one click and made each document's branch the base of the next, when each document
is signed off on its own and can land on its own.

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
- `review` and `test`, on an epic's work, run `implement` on the epic for the problems the person
  took (below).

After `design` has created beads, ADR-008's limit still applies to amendments, and the
`design-revision` bead remains the path for a change that moves work.

### Design asks one question about the decisions it set aside

When `design` has set technical decisions aside into **Decisions needed** — every one under the
product-manager persona, or one the person said they cannot decide — it asks once, before setting
the status: "I set aside N technical decisions. Settle them with sensible defaults now so building
can start, or leave them for an engineer?" On "settle now", the run picks the simplest default that
fits the project's stance, says each choice in one plain sentence, writes an ADR where the choice
needs one, and continues to `Ready` and the beads in the same run. On "leave them", the design stays
`Draft`, creates no beads, gets a draft pull request, and an engineer's run settles them, as
ADR-011 described. ADR-011's rule that no ADR is written under the product-manager persona gives
way to the person's explicit answer: a default the run chose and told them is a decision they made.
ADR-011.2 is the edition of the personas decision that records this.

### Review and test fix what the person takes

On an epic's work, `review` and `test` end their triage with one question: "I found N problems. Fix
them all? (I recommend yes.)" The person may take all, some, or none. What they take is filed as
`deferred` children of the epic, in the forms ADR-013 defined, and the same session then runs
`implement` on the epic, whose go gate reopens those children and builds them. Document fixes still
land as text on the branch under review. The person's one typed command was the one that started
the verb.

### Document pull requests land on their own

Each document verb — `envision`, `specify` with its mockups, `report`, `design`, and `mock-up` run
alone — branches from the default branch, writes the document, works with the person until they sign
it off, commits, pushes, and opens an ordinary pull request. A GitHub Action in the project merges
that pull request when a person adds the label `auto-merge` to it or approves it, provided its diff
touches only document paths: `docs/visions/`, `docs/specs/`, `docs/bugs/`, `docs/mockups/`,
`docs/designs/`, `docs/adrs/`, `.codefall/reviews/`, `.codefall/tests/`, and
`.beads/interactions.jsonl`. Any other path means the pull request waits for a person. The Action's
trigger is the `pull_request` event of type `labeled` with the label named `auto-merge`, plus
`synchronize` while the label is present, so a late commit is checked again before the merge, plus
the `pull_request_review` event of type `submitted` with the review state `approved`; the path check
and the draft check run the same way for every trigger, and the merge is `gh pr merge --squash`
under the default token. GitHub does not let a pull request's author approve it, and a verb opens
the pull request as the person running it, so an approval always comes from a second person. No verb
adds the label or approves, and no verb marks a pull request ready for review as a signal: the label
is the person's decision, made after the report, after a colleague's review, or not at all.

A document whose status is `Draft` gets a draft pull request. The run that later promotes it to
`Ready` marks the pull request ready for review before pushing, because GitHub does not merge a
draft; the pull request still waits for the label.

The report of each of those verbs says the pull request is open at its URL, that the person adds the
`auto-merge` label when they want it merged or has someone approve it, either of which merges it,
and ends with one next command: `envision` → `/codefall-specify VISION-NNN`, `specify` →
`/codefall-design SPEC-NNN`, `report` → `/codefall-design BUG-NNN`, `design` → `/codefall-implement
DESIGN-NNN`. A project without the Action hears that the pull request waits for a person to merge
it.

Documents do not stack. `equip`, `scaffold`, and `upgrade` push and open their pull request the same
way, without asking, and a person merges those because they change code and settings. Push and pull
request stop being a question for every verb.

A first version of this decision had each document verb open a draft pull request and mark it
ready for review when the document reached `Ready`, with the Action merging on that event. It was
changed before this record merged: marking a pull request ready for review is a routine GitHub
state that people and tools set for other reasons, and a verb setting it would have been the verb
merging the document. A label a person adds is a deliberate act with one meaning.

### The Action ships with codefall, installed by `equip`

`codefall-equip` gains a fourth track, `landing`, which writes
`.github/workflows/codefall-land-documents.yml` from a template the extension ships, creates the
`auto-merge` label in the repository, reads the default branch's protection and rulesets with `gh
api`, tells the owner what the rule has to allow for the workflow's token to merge, and changes no
other repository setting. A repository that requires a review on its default branch has to let the
Action through; that is the owner's decision, made in the open. `codefall init` does not install the
workflow: the installer rewrites its subtrees on every upgrade, and the workflow is a file the owner
may edit.

### Integration is GitHub's

`implement`'s integration step asks GitHub whether each open layer of its code stack is mergeable
rather than merging in a scratch worktree. A lower layer is fixed on its own branch and cascaded
with `gh stack rebase` and `gh stack push`. Nothing is rebased or force-pushed by hand; the extension
does the cascade and GitHub rebases on merge. ADR-013.3 carries this into the deliveries decision.
`shared/stacks.md` records the GitHub feature for `implement`; no document verb reads it.

### One next step

For the product-manager persona, the last line of every report says in one plain sentence what
happens next and names exactly one command when there is a step for the person to start. Never a
list. Everything else the verb could do, it did in the run or asked about in a sentence. The
engineer persona is unchanged.

## Consequences

- A spec effort is one run and one pull request, mockups included, and a person is never handed a
  verb to run between two verbs the agent could have run itself.
- A product manager can take a feature from `envision` to merged code with one typed command per
  verb and one label per document: `design` settles its own decisions when asked to, and `review`
  and `test` build their own fixes.
- "A human performs every merge" becomes "no verb merges": a person merges every code pull request,
  and the project's Action merges a document pull request on a person's label. The guard hook is
  unchanged, and the label is the one thing a verb never adds.
- A project without the Action sees the same pull requests and merges them by hand; the report
  names `/codefall-equip landing`.
- Six more descriptions sit in every session's listing. ADR-004.3 records the new count.
- `Skill` joins `allowed-tools` for `specify`, `design`, `review`, and `test`. Harnesses without
  the flag or the tool list run as before.
- The design verb's "spec not ready" gate changes from a stop to work it finishes, so a `Draft` spec
  or a missing mockup costs a question inside the run, not a round trip.
- An ADR can now be ratified from a default the run chose under the product-manager persona, once
  the person said to settle the decisions now. The ADR's Context says so, and an engineer who
  disagrees supersedes it as with any other.

## Related

- [ADR-004.3](ADR-004.3-skill-length-guidelines.md) — which skills carry the invocation flag.
- [ADR-008](ADR-008-upstream-amendments.md) — text amendments; this ADR adds running the upstream
  verb before beads exist, and leaves ADR-008's limit standing after.
- [ADR-011.2](ADR-011.2-personas.md) — personas; the edition that records the product-manager
  report rule and the design paragraph as this ADR changed them.
- [ADR-013.3](ADR-013.3-deliveries.md) — deliveries; integration moves to `gh stack`, and review
  and test run implement.
- `extensions/shared/landing.md` — the landing procedure every document verb follows.
- `extensions/shared/stacks.md` — how GitHub stacked pull requests work, for `implement`.
- `docs/landing-documents.md` — how codefall lands documents, for a project owner.
