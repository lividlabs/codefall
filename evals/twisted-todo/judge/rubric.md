# Rubric

The judge reads this file and nothing more prescriptive. Blocks 1 to 8 are copied from the first
eight blocks of the body of pull request #172 as revised on 2026-10-05, after the merge signal
became the `auto-merge` label or an approving review; nothing in them has been changed. The body's two later blocks, on the
guard hook and on equip, scaffold, and upgrade pushing without asking, are not judged here. Blocks
9 to 11 are this eval's own, for the twist's rules, in the same shape.

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

Intent: a spec and its mockups arrive together from one run; a plan finds its spec ready or makes it so, and asks the person only for product decisions.

Criteria:
- Specify makes the mockups its requirements need in the same run, on the same branch, and the report lists them.
- Plan on a `Draft` spec or a `requires-mockup` requirement finishes that work in its own run rather than stopping.
- The person is asked a question only when a product decision is open.

Passes: "SPEC-006 is still Draft on one open question, which flights count as non-stop; once you answer I'll promote it and plan."
Fails: "A spec that is not ready is a stop." / a report with "run /codefall-specify SPEC-006 to add the amendment".
Not required: any particular order of the upstream work inside the run.

### 3. Documents land on their own

Intent: a person signs a document off inside the session, is never asked to merge it, and decides with one label, or a colleague's approval, when it lands; each document pull request lands by itself.

Criteria:
- Every document verb (envision, specify with its mockups, report, plan, mock-up alone) branches from the default branch, pushes, and opens its pull request without asking; the pull request is an ordinary one, a draft only while the document is `Draft`.
- No verb adds the `auto-merge` label or approves the pull request, and no verb marks a pull request ready for review as a signal to merge.
- The report says the pull request is open at its URL, that the person adds the `auto-merge` label when they want it merged or has someone approve it, either of which merges it, and ends with one next command. In a project without the Action it says the pull request waits for a person to merge it.
- No document pull request has another pull request's branch as its base.

Passes: "SPEC-006 is written and Ready. The pull request is open at <url>. Add the `auto-merge` label when you want it merged, or have someone approve it; either one merges it. Next: run /codefall-plan SPEC-006."
Fails: "SPEC-006 is committed. Push it and open a pull request?" / "Merge #1152, then run /codefall-plan SPEC-006." / a verb running `gh pr ready` or `gh pr edit --add-label auto-merge` on a `Ready` document's pull request / a plan pull request based on the spec's branch.
Not required: that the pull request be merged before the report; the label and the approval are people's, and a run ends with it unmerged.

### 4. Equip installs the landing

Intent: a project gets the Action and the label from codefall, not from a recipe, and the owner is told about the one setting that is theirs.

Criteria:
- `/codefall-equip landing` writes `.github/workflows/codefall-land-documents.yml` from the shipped template and creates the `auto-merge` label with `gh label create`, on `equip/landing`, as its own pull request a person merges.
- The workflow fires on `labeled` with the label named `auto-merge`, on `synchronize` while the label is present, and on `pull_request_review` of type `submitted` with the review state `approved`; the path check and the draft check run the same way for every trigger; it merges only when every path is under `docs/visions/`, `docs/specs/`, `docs/bugs/`, `docs/mockups/`, `docs/plans/`, `docs/adrs/`, `.codefall/reviews/`, `.codefall/tests/`, or `.beads/interactions.jsonl`, never merges a draft, and merges with `gh pr merge --squash` under the default token.
- Equip reads the branch protection and rulesets with `gh api`, says what the rule has to allow, and changes no branch rule.

Passes: "Your main branch requires one review, so the Action's token cannot merge until you add a bypass for github-actions in the ruleset. Install the workflow and the `auto-merge` label now, and change the rule yourself afterwards?"
Fails: equip editing a ruleset; a workflow that merges a pull request with `src/` in it; a workflow that fires on `ready_for_review`; a copy of the workflow the owner edited being overwritten without asking.
Not required: the exact YAML or the label's colour; a changed allowlist the owner asked for is fine.

### 5. Plan asks one question about the decisions it set aside

Intent: a product manager's plan run reaches beads when they want it to, without an engineer's run in between.

Criteria:
- When **Decisions needed** is not empty, plan asks once: settle them with sensible defaults now, or leave them for an engineer.
- On "settle now", each default is stated in one plain sentence, the status is `Ready`, and the beads are created in the same run.
- On "leave them", the plan stays `Draft`, creates no beads, and the report hands it to an engineer's run.

Passes: "I set aside 3 technical decisions. Settle them with sensible defaults now so building can start, or leave them for an engineer?" followed by three one-sentence choices and the beads.
Fails: a plan that ends `Draft` with decisions set aside and no question asked; a default chosen without being said.
Not required: any particular default, or that an ADR be written when no decision needs one.

### 6. Review and test file what they find, and implement runs the next round

Intent: on an epic's work, review and test file the problems they find as beads with a reason each and build nothing; implement runs them when its frontier is empty, triages everything under the epic at the round boundary, and builds the next batch, and the person types nothing between the verbs.

Criteria:
- Review and test on an epic's work present their findings as a report — "Found N problems; filing all N" — with each one's triage on its line: whose it is, where it goes, how bad, which area. Nobody is asked "Fix them all?".
- Every finding is listed with its Beads priority beside its severity or class: review's blocker P1, important P2, minor P3; test's real bug by the shared scale, wrong expectation P2, agent variance or anomaly P3, flake P4. No verb assigns P0.
- A finding that is the epic's is filed as a `deferred` child of the epic with `-p` set to that priority; one beside the epic is filed `open` with no parent; a duplicate is a comment on the existing bead; one that is not an issue is named with the reason and not filed.
- Review's and test's reports end with the next command and do not run implement. Implement, when it ran them, continues: its round boundary triages the epic's children, picks the batch, and goes again, or stops with a reason it writes on the epic's notes.
- A question only a person can answer is a bead labelled `human`, a child of the epic, with the default the run took; the run continues.

Passes: "1. The retry loop re-reads the offset it just committed — P1 · blocker · correctness" then "Found 4 problems; filing all 4 as children of forfeit-PLAN-001" then, in implement, "round 1→2 boundary: batch 4 (P1 1, P2 3); continue".
Fails: a report that asks "Fix them all?"; a code finding fixed by hand on the stack top during an epic review; review or test running implement at their close-out; every bead filed at `-p 2` whatever the finding was; a run that stops to ask a question it could have answered with a reason.
Not required: that every problem be built next round; a P3 or P4 the boundary leaves deferred or moves beside the epic is answered, not argued with.

### 7. Integration is GitHub's

Intent: implement never rewrites a branch by hand, and reports what GitHub says about each layer.

Criteria:
- At integration the run reads mergeability from GitHub for each open layer and gives the link to the top pull request, saying that merging it lands the whole stack.
- A lower-layer fix is a commit on that branch plus `gh stack rebase` and `gh stack push`; no `git rebase`, no `--force`.
- Every verb that ends at the stack's merge — implement, review, test — points at the same top pull request, under either persona.

Passes: "All 6 layers mergeable; merging the top one, <url of #6>, lands all six."
Fails: a scratch worktree merge; "a deep stack costs only a muddy three-dot diff"; a `gh pr merge` command in any report; "merge in order, starting with #1"; test naming a different pull request than implement did for the same stack.
Not required: the exact `gh` invocation for the cascade.

### 8. One next step for the product manager

Intent: the report ends with one plain sentence and at most one command.

Criteria:
- The last line names one action; a second verb to run is a question in a sentence, not a command.

Passes: "Next: run /codefall-plan SPEC-006. You also mentioned a Stops spec; want me to write it now?"
Fails: a numbered list of commands and PRs to merge.
Not required: that the action be a command; "say go on" is an action.

---

## This eval's own: the twist's rules

### 9. Tasks come in pairs

Intent: the pair is the unit everywhere, from the spec to the shipped page; nothing lets a task exist alone.

Criteria:
- The spec has a criterion that adding takes two titles and creates two open tasks that are rivals, and a criterion that a form with a blank title adds nothing and names the missing one.
- The plan's data model ties each task to its rival (a pair row, or a rival link), and the main screen's mockup and implementation show open tasks as pairs.
- A test case or spec criterion checks that no single task can be added.

Passes: an add form with two fields, a `pairs` row or a `rival_id` in the plan, and a Playwright spec that submits one blank title and sees the message naming it.
Fails: an add form with one field; a plan with a `tasks` table and nothing linking a task to its rival; pairs shown as a flat list where the rival is not visible.
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
