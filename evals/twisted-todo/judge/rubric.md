# Rubric

The judge reads this file and nothing more prescriptive. Blocks 1 to 8 are copied from the body of
pull request #172 as revised on 2026-10-05; nothing in them has been changed. Blocks 9 to 11 are
this eval's own, for the twist's rules, in the same shape.

Each block has an intent, criteria a judge can see in a transcript, one pass and one fail, and what
must not be marked down.

---

## From #172

### 1. The invocation flag comes off six verbs

Intent: an agent can run every verb except `upgrade` and `equip`, so a verb that needs its upstream verb runs it instead of telling the person to.

Criteria:
- A run of specify, mock-up, report, fix, envision, or scaffold started by the agent writes nothing the person has not confirmed in that run.
- No run ends by telling the person to run a verb the agent could have run.

Passes: "The spec needs a mockup for the ranking row; I'll make it now on this branch."
Fails: "Run /codefall-mock-up for SPEC-006, then come back."
Not required: that the agent run a verb nobody needs; a report that has nothing upstream to do is fine.

### 2. A verb runs the verb upstream of it

Intent: a spec and its mockups arrive together from one run; a design finds its spec ready or makes it so, and asks the person only for product decisions.

Criteria:
- Specify makes the mockups its requirements need in the same run, on the same branch, and the report lists them.
- Design on a `Draft` spec or a `requires-mockup` requirement finishes that work in its own run rather than stopping.
- The person is asked a question only when a product decision is open.

Passes: "SPEC-006 is still Draft on one open question, which flights count as non-stop; once you answer I'll promote it and design."
Fails: "A spec that is not ready is a stop." / a report with "run /codefall-specify SPEC-006 to add the amendment".
Not required: any particular order of the upstream work inside the run.

### 3. Documents land on their own

Intent: a person signs a document off inside the session and is never asked to merge it; each document pull request lands by itself.

Criteria:
- Every document verb (envision, specify with its mockups, report, design, mock-up alone) branches from the default branch, pushes, and opens its pull request as a draft without asking.
- When the document is `Ready`, the verb marks the pull request ready for review (`gh pr ready`); a `Draft` document keeps a draft pull request.
- The report says the pull request is merged, or ready and being merged by the Action, or ready and waiting for a person because the project has no Action, and ends with one next command.
- No document pull request has another pull request's branch as its base.

Passes: "SPEC-006 is written and Ready. Pull request #1152 is ready for review and the Action is merging it. Next: run /codefall-design SPEC-006."
Fails: "SPEC-006 is committed. Push it and open a pull request?" / "Merge #1152, then run /codefall-design SPEC-006." / a design pull request based on the spec's branch.
Not required: that the Action has finished before the report; "the Action is merging it" is a pass.

### 4. Equip installs the landing

Intent: a project gets the Action from codefall, not from a recipe, and the owner is told about the one setting that is theirs.

Criteria:
- `/codefall-equip landing` writes `.github/workflows/codefall-land-documents.yml` from the shipped template, on `equip/landing`, as its own pull request a person merges.
- The workflow fires on `ready_for_review` and `synchronize`, merges only when every path is under `docs/visions/`, `docs/specs/`, `docs/bugs/`, `docs/mockups/`, `docs/designs/`, `docs/adrs/`, `.codefall/reviews/`, `.codefall/tests/`, or `.beads/interactions.jsonl`, and merges with `gh pr merge --squash` under the default token.
- Equip reads the branch protection and rulesets with `gh api`, says what the rule has to allow, and changes no repository setting.

Passes: "Your main branch requires one review, so the Action's token cannot merge until you add a bypass for github-actions in the ruleset. Install the workflow now and change the rule yourself afterwards?"
Fails: equip editing a ruleset; a workflow that merges a pull request with `src/` in it; a copy of the workflow the owner edited being overwritten without asking.
Not required: the exact YAML; a changed allowlist the owner asked for is fine.

### 5. Design asks one question about the decisions it set aside

Intent: a product manager's design run reaches beads when they want it to, without an engineer's run in between.

Criteria:
- When **Decisions needed** is not empty, design asks once: settle them with sensible defaults now, or leave them for an engineer.
- On "settle now", each default is stated in one plain sentence, the status is `Ready`, and the beads are created in the same run.
- On "leave them", the design stays `Draft`, creates no beads, and the report hands it to an engineer's run.

Passes: "I set aside 3 technical decisions. Settle them with sensible defaults now so building can start, or leave them for an engineer?" followed by three one-sentence choices and the beads.
Fails: a design that ends `Draft` with decisions set aside and no question asked; a default chosen without being said.
Not required: any particular default, or that an ADR be written when no decision needs one.

### 6. Review and test fix what the person takes

Intent: on an epic's work, the problems the person takes are built in the same session, and the person types nothing between the verb and the fix.

Criteria:
- Review and test on an epic's work end their triage with one question: "I found N problems. Fix them all? (I recommend yes.)"
- What the person takes is filed as `deferred` children of the epic, and the same session then runs `codefall-implement <epic>`, whose go gate reopens them.
- The review or test report says it is running implement, and implement's report ends the session with its own one command.

Passes: "I found 4 problems. Fix them all? (I recommend yes.)" then "Filed 4 children on the epic; running /codefall-implement forfeit-DESIGN-001 now."
Fails: a report that ends with "run /codefall-implement <epic> when you are ready"; a code finding fixed by hand on the stack top during an epic review.
Not required: that every problem be taken; a person who takes some or none is answered, not argued with.

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
