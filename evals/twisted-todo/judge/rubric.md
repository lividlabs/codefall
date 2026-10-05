# Rubric

The judge reads this file and nothing more prescriptive. Blocks 1 to 3 are copied verbatim from the
body of pull request #171 and blocks 4 to 8 from #172; nothing in them has been changed. Blocks 9
to 11 are this eval's own, for the twist's rules, in the same shape.

Each block has an intent, criteria a judge can see in a transcript, one pass and one fail, and what
must not be marked down.

---

## From #171

### 1. `shared/stacks.md`

Intent: a run that lands a pull request knows how GitHub stacks work without the person explaining it, every session.

Criteria:
- The run reads the file before opening a PR on another PR's branch, and its actions match it: `gh stack link`, no hand rebase, merging the top lands the stack.
- Nothing in the run's prose about stacks contradicts the file.

Passes: "I'll open the design PR with its base on the mockups branch and link it into the stack with `gh stack link`."
Fails: "A deep stack costs only a muddy three-dot diff until it drains bottom-up." (true before July 2026, false now)
Not required: any particular command sequence; `gh stack submit` and the website are also correct.

### 2. `docs/recipes/landing-document-stacks.md`

Intent: a repository owner can set up self-landing document stacks from this page alone, and knows the one setting that is theirs to judge.

Criteria:
- An agent given the page's prompt produces the workflow, names the ruleset change and the token, and asks before changing a setting.
- The path allowlist is applied as an allowlist, never as a content inspection.

Passes: an agent that stops and asks before touching the ruleset.
Fails: an agent that merges a stack containing a path outside the list, or edits branch protection unasked.
Not required: the exact YAML; it is a starting point.

### 3. Speak plainly (Codefall section)

Intent: a person reads a report once and knows what happened and what to do, without decoding it.

Criteria:
- Each sentence states one thing in everyday words, in the active voice.
- The concrete thing is named, not the category it belongs to.

Passes: "A verb may only fix text in a document above it, and it is not allowed to run the verb that owns that document, so it stops and asks you to run it."
Fails: "The bounce is the amendment rule's scope plus the flag."
Not required: any particular wording, length, or vocabulary; a synonym is never a failure.

---

## From #172

### 4. The invocation flag comes off six verbs

Intent: an agent can run every verb except `upgrade` and `equip`, so a verb that needs its upstream verb runs it instead of telling the person to.

Criteria:
- A run of specify, mock-up, report, fix, envision, or scaffold started by the agent writes nothing the person has not confirmed in that run.
- No run ends by telling the person to run a verb the agent could have run.

Passes: "The spec needs a mockup for the ranking row; I'll make it now on this branch."
Fails: "Run /codefall-mock-up for SPEC-006, then come back."
Not required: that the agent run a verb nobody needs; a report that has nothing upstream to do is fine.

### 5. A verb runs the verb upstream of it

Intent: a spec and its mockups arrive together from one run; a design finds its spec ready or makes it so, and asks the person only for product decisions.

Criteria:
- Specify makes the mockups its requirements need in the same run, on the same branch, and the report lists them.
- Design on a `Draft` spec or a `requires-mockup` requirement finishes that work in its own run rather than stopping.
- The person is asked a question only when a product decision is open.

Passes: "SPEC-006 is still Draft on one open question, which flights count as non-stop; once you answer I'll promote it and design."
Fails: "A spec that is not ready is a stop." / a report with "run /codefall-specify SPEC-006 to add the amendment".
Not required: any particular order of the upstream work inside the run.

### 6. Documents stack

Intent: one delivery's documents are one GitHub stack, and nobody merges anything until design is done.

Criteria:
- The spec PR targets `main`; the mockup PR targets the spec branch; the design PR targets the mockups' or spec's branch; each is linked with `gh stack link`.
- No document verb asks whether to push or open a PR; it does both and says so.
- Only the design verb's report names a merge, and it says merging that PR lands the stack.

Passes: "Opened #1154 on `spec/SPEC-006-...` as the second layer of the stack; nothing merges until the design is done."
Fails: "SPEC-006 is committed. Push it and open a pull request?" / "Review and merge #1152 (the spec) and #1153 (the mockups)."
Not required: that a stack exist when there is only one document; one PR is not a stack.

### 7. Integration is GitHub's

Intent: implement never rewrites a branch by hand, and reports what GitHub says about each layer.

Criteria:
- At integration the run reads mergeability from GitHub for each open layer and reports the merge order.
- A lower-layer fix is a commit on that branch plus `gh stack rebase` and `gh stack push`; no `git rebase`, no `--force`.

Passes: "All 6 layers mergeable; merge #6 to land the stack, or from #1 up."
Fails: a scratch worktree merge, or "a deep stack costs only a muddy three-dot diff".
Not required: the exact `gh` invocation.

### 8. One next step for the product manager

Intent: the report ends with one plain sentence and at most one command.

Criteria:
- The last line names one action; a second verb to run is a question in a sentence, not a command.

Passes: "Next: run /codefall-design SPEC-006. You also mentioned a Stops spec; want me to write it now?"
Fails: a numbered list of commands and PRs to merge.
Not required: that the action be a command; "say go on" is an action.

---

## This eval's own: the twist's rules

### 9. Tasks come in pairs

Intent: the pair is the unit everywhere, from the spec to the shipped page; nothing lets a task exist alone.

Criteria:
- The spec has a criterion that adding takes two titles and creates two open tasks that are rivals, and a criterion that a form with a blank title adds nothing and names the missing one.
- The design's data model ties each task to its rival (a pair row, or a rival link), and the main screen's mockup and implementation show open tasks as pairs.
- A test case or spec criterion checks that no single task can be added.

Passes: an add form with two fields, a `pairs` row or a `rival_id` in the design, and a Playwright spec that submits one blank title and sees the message naming it.
Fails: an add form with one field; a design with a `tasks` table and nothing linking a task to its rival; pairs shown as a flat list where the rival is not visible.
Not required: any particular layout for a pair, or any particular word for "rival".

### 10. Finishing one forfeits the other

Intent: the forfeit happens at the moment of finishing, needs no confirmation, and is final.

Criteria:
- The spec has a criterion that finishing a task forfeits its rival at the same time, and a criterion that a forfeited task cannot be reopened or finished.
- The implementation changes both tasks in one step (one transaction, or one request that writes both), and the forfeited task leaves the main page with the finished one.
- A test checks that after finishing one side the rival is forfeited and the pair is gone from the main page.

Passes: a criterion "WHEN the person finishes a task, the system SHALL mark its rival forfeited at the same moment", and a test that finishes one task and asserts the pair is absent from the main page and present on the Forfeits page.
Fails: a confirmation dialog before the forfeit; a rival left open after its partner is finished; a forfeited task that can be finished later.
Not required: any particular wording of the button, or a transaction in the strict database sense as long as both tasks change together as the user sees it.

### 11. The Forfeits page is the record

Intent: a second page lists every forfeit with what beat it and when, and the two counts are equal.

Criteria:
- The spec has a criterion for the Forfeits page listing each forfeited task with the task that beat it and the date, newest first, and a criterion for the empty state in a sentence.
- The page shows a finished count and a forfeited count, and they are equal for any data the app can produce.
- Forfeited tasks never appear on the main page.

Passes: a Forfeits mockup with the two counts at the top and rows of "task · beaten by · date", and a test that forfeits one task and reads it on the Forfeits page with the winner named.
Fails: forfeited tasks shown on the main page with a strikethrough; a Forfeits page with no winner column; counts that can differ.
Not required: the name "Forfeits" itself, or any particular date format.
