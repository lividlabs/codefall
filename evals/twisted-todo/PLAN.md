# Eval: a todo list with a twist, end to end

An end-to-end test of codefall's chain on a small project, with an agent playing the product
manager and a second agent judging the transcripts. It tests what #171 and #172 changed: a verb
runs the verb upstream of it instead of handing the person a command; specify makes the mockups its
requirements need in the same run; one delivery's documents are one GitHub stack and nobody is asked
to merge until design; implement's integration asks GitHub rather than rebasing; the
product-manager report ends with one sentence and one command; findings from review and test return
to the epic as children; the chart line and the notes log sit on the epic.

Everything lives under `evals/twisted-todo/`: this plan, the PM brief (`brief.md`), the project
skeleton and its scripts (`fixture/`), the driver (`driver/`), and the judge's rubric and prompt
(`judge/`).

## The twist

The product is **Forfeit**, a todo list in which no task can be added alone. You add tasks two at a
time, and the two are rivals. When you finish one, the other is forfeited at that moment: it leaves
the list, it is never marked done, it cannot be reopened, and it goes to a second page, the Forfeits,
which lists every task you gave up, the task that beat it, and the day it happened, newest first,
under two counts, finished and forfeited, that are always equal. The main page shows only the open
pairs. A title can be corrected while its pair is open; a pair can never be deleted or split, so the
only way out is to finish one side.

## Why it clears the bar

Every familiar todo tool lets you add one task and keeps unfinished tasks until you delete them; none
retires a task because you did a different one, and none would, because it throws away something the
person wrote down on purpose. That is the rule a familiar tool refuses. It changes how the list is
used: adding becomes a decision about what to give up, and the record of what was given up is as much
the product as the record of what was done. It runs through the data model (a task belongs to a pair
and has three states, open, finished, forfeited), the main screen (pairs, not items), the second
view (the Forfeits record with the winner on each row), and the acceptance criteria (finishing one
forfeits the other at the same moment; a forfeit never returns; the two counts are equal). Candidates
set aside: a fixed number of slots (Teuxdeux and others cap a day), pairwise ranking (several
prioritisers do it), coins for completions (Habitica), a strict oldest-first queue (too little data
model to design), and a list you can only see by recalling it (fuzzy matching is a technical choice
the product manager cannot settle, which would stall the design on purpose rather than by accident).

## The PM brief

`brief.md` is the person the simulator plays, Mara, in her own words: what she wants, the five rules
she insists on, how she answers the usual interview questions (one person on one laptop; opening the
page starts it; what she sees on each page; what happens on a blank title, an empty list, and an
empty record; the out-of-scope list), what she does not know (the second page's name, whether finished
tasks are listed on their own, whether to show a pair's age), and how she behaves (short answers,
picks an option when offered, confirms a document that carries the rules, pushes back once, says yes
to engineering work, says go at the gate, never asks for a merge, declines every technical choice in
one sentence).

## The fixture and the stack

`fixture/` is the project the chain runs against: a web page that says nothing is here, a health
endpoint, a migration runner with one migration, two unit tests, the local scripts, and a Playwright
configuration with no specs. Node 24 with the standard library's `node:sqlite` and `node:http`,
server-rendered HTML, Playwright as the end-to-end runner: Node 24 ships SQLite in the standard
library, so the app has no runtime dependencies and no native build, and Playwright is the runner
`codefall-equip` already declares for a browser front end, so nothing the skills do not know is in
play.

`fixture/setup.sh` creates the throwaway: builds `codefall` from this branch, copies the fixture,
makes a private GitHub repository and pushes it, runs `codefall init --harness claude --tracker
github` (which runs `bd init`), declares the local scripts and the Playwright runner in
`.codefall/settings.json` and the Runners line in `testing/AGENTS.md` (what `equip` would write; the
chain under test does not include `equip`, and without the declarations implement refuses a bead
that names a test case), sets the persona to `product-manager`, installs the shared landing
customization, pushes, and adopts the git origin as the Beads Dolt remote. The recipe's Action is in
the fixture. `fixture/teardown.sh` deletes the repository. Neither has been run.

## The run protocol

One session per verb, each a fresh `query()` in the project directory, which is the `/clear`. The
first message is the slash command with its argument; the argument is read from the project tree
(the newest `VISION-`, `SPEC-`, or `DESIGN-` identifier). The driver's `protocol.ts` holds this
table as code.

| Verb | First message | The simulator must | The simulator must not | Done when |
| --- | --- | --- | --- | --- |
| envision | `/codefall-envision <the idea in Mara's words>` | give the idea as spoken, answer why now and non-goals, keep rules at rule level, confirm a Ready document | give behaviour precise enough to build; argue when told to merge | the report ends with what happens next |
| specify | `/codefall-specify VISION-001` | work from the vision; aim at three requirements; say no mockup exists and ask for them (main page full and empty, Forfeits full and empty); confirm recap and document; Ready | ask to merge; accept more than three requirements without saying so; choose anything technical | the report names the design command |
| design | `/codefall-design SPEC-001` | accept the tier; ask for five tasks or fewer; confirm the criteria cover the five rules; decline technical choices and let them be parked | decide anything technical | the report ends; then the driver waits for the Action |
| implement | `/codefall-implement DESIGN-001` | say yes to engineering work; say go at the gate; accept the serial stack | ask to merge | the report gives the merge order |
| review | `/codefall-review DESIGN-001` | say yes to engineering work; review with every lens; fix blockers and importants, defer minors, say yes to filing deferred ones as children | fix anything itself | the report ends |
| test | `/codefall-test DESIGN-001` | run now; file a real bug (a broken rule or a missing thing she said she would see) as an issue and a child of the epic; decline the rest with a reason | ask for fixes | the report ends |

**How the driver answers.** `AskUserQuestion` calls reach the driver's `canUseTool` callback with
the questions; the simulator returns a label per question and the callback returns them as
`updatedInput.answers`. Every other tool is allowed, so the run never waits on a person; codefall's
guard hook still runs on every Bash call. When a turn ends in prose, the simulator classifies the
last text: a question or a document shown for approval is a reply; a final report, or a stop whose
remedy is a command for the person, is done; a loop is stuck. Caps: 60 PM turns and 90 minutes per
session.

**Between design and implement.** If the design ended `Draft` with a Decisions needed section (the
product-manager persona parks every technical choice), the driver by default runs one extra design
session, `design-settle`, with the persona flipped to `engineer`, records it as a deviation, and
flips the persona back. It then waits up to fifteen minutes for the Action to merge the design pull
request, and falls back to merging the stack itself, recorded as a deviation.

**After test.** The driver merges the open code stack at its top, as the person would, so the
delivery reaches `main`; it records that it did.

## Merging, in plain words

The person is away, and no human merges. Documents land through the recipe in
`docs/recipes/landing-document-stacks.md`: a GitHub Action in the throwaway merges the document
stack when the design pull request is open and not a draft. A throwaway needs no branch protection,
so the default `GITHUB_TOKEN` can merge, and the recipe's starting workflow is adapted in two ways:
it also fires on `opened` for a non-draft design pull request, because a Ready design is opened
non-draft and `ready_for_review` alone would never fire for it, and it uses `github.token` instead
of a stored secret. The path allowlist is the recipe's, applied as an allowlist. Code pull requests
are merged by the driver acting as the human from a session outside the project directory, where
codefall's guard hook does not apply: the driver runs `gh stack merge <top> --squash --yes` with the
run directory as its working directory and `GH_REPO` naming the repository.

## The judge

`judge/rubric.md` holds the rubric blocks from #171 and #172 verbatim, nothing added to them, and
three blocks of this eval's own for the twist's rules in the same shape: tasks come in pairs;
finishing one forfeits the other; the Forfeits page is the record. `judge/judge-prompt.md` is the
judge's instructions: name the transcript each block applies to, quote the deciding lines, give
`pass`, `fail`, `not exercised`, or `unclear`, check the block's "Not required" line before any fail,
and write `verdict.md` plus a JSON block. It also states the facts about this run that bear on
reading the rubric: the vision is the bottom layer, so the spec targets the vision's branch; specify
commits the mockups with the spec, so there may be no mockup pull request; the Action lands the stack;
block 2 (the recipe's own rubric) has no transcript here and is marked not exercised unless a
transcript shows otherwise. The judge reads the transcripts and an `evidence/` directory the driver
gathers: pull requests with bases and states, issues, Action runs, the git log of `main`, the Beads
epic and its children and notes, the review and test records, and the test cases.

**Pass for the whole run:** every block from #172 (4 to 8) that was exercised passed; the document
stack landed through the Action or the design verb's report said merging its pull request lands the
stack; the code stack was mergeable at implement's integration; and the twist blocks (9 to 11)
passed on the shipped work. "Pass with findings" is the same except one block failed on a single
sentence or a single session. Anything else is a fail, named by the first block that failed and the
line that failed it.

## Token budget

Rough, before a first run; the SDK's `total_cost_usd` and `modelUsage` on each session's last result
line are the measurement.

| Session | Estimate | What drives it |
| --- | --- | --- |
| envision | 0.2M | the interview and the template |
| specify, with two mockup surfaces in four files | 0.6M | the interview, the audit, the mockup files, the tracker mirror |
| design | 0.5M | reading the spec, the code, the references; the plan file |
| implement, five beads as a serial stack | 2.5M | one worker session per bead, each reading the design, the spec, the case format, and running checks; the root's bookkeeping |
| review | 0.5M | the reviewer subagents on the stack's diff |
| test | 0.5M | the Playwright specs and the report |
| simulator, about forty calls | 0.6M | the transcript tail on every call |
| judge | 0.3M | reading every transcript once |
| total | about 5.5M | mostly cache reads |

**The bead count is the token lever.** Each bead is one worker session that reads the same
documents and runs the same checks, so five beads is roughly five times one; four beads saves a
fifth of implement. The other levers, in order: the number of mockup states (four files, no loading
or error drawings); the worker model (`sonnet` for workers where the design proposes it); the effort
level; dropping review lenses the simulator does not need; one variant per test case.

## Open questions for the maintainer

1. **Design under the product-manager persona.** The persona parks every technical choice the
   person declines, and a design with parked decisions is `Draft` and creates no beads. Mara declines
   every technical choice. If the design parks anything, the chain stops at design unless the
   fallback `design-settle` session (engineer persona) runs. The driver runs it by default and records
   it as a deviation. Is that the right default, or should the eval stop and treat a parked design as
   its result?
2. **`equip` before the chain.** `setup.sh` writes the `local` and `test.runners` declarations
   itself. The alternative is a seventh session running `/codefall-equip local` and
   `/codefall-equip test` before envision, which would test `equip` too at the cost of two more
   sessions and two more pull requests to merge. Which do you want?
3. **The Action and the stack-merge API with `GITHUB_TOKEN`.** The recipe's workflow has not been
   run, and the stacked-pull-requests feature is in public preview. If `gh stack merge` under the
   default token is refused, the driver's fallback merges the stack itself and the run records the
   deviation. Should the fallback count against the run?
4. **Merging the code stack after test.** The driver merges it so the delivery reaches `main` and
   the evidence shows the shipped app. The alternative is to stop at open pull requests, which is
   where implement's report leaves a person. Keep the merge?
5. **A second round.** If review or test files children, the delivery needs another
   `implement` round, which T4 does not include. The driver stops after test and the judge sees the
   children as evidence. Should a second round run when children exist?
6. **Models and effort.** Sessions default to `opus`, the simulator and the judge to `sonnet` and
   `opus`. Workers take what the design proposes. Change any of these?
7. **The Dolt remote.** `setup.sh` runs `bd dolt push --yes`, which pushes `refs/dolt/data` to the
   throwaway so no verb asks about adopting a remote. Fine for a throwaway?
8. **A finding before any run.** `codefall-envision`'s SKILL.md step 6 still ends with "merge the
   pull request, then `/codefall-specify VISION-NNN`", and `codefall-mock-up`'s and the design
   verb's reports are consistent with the stack while envision's is not. Under the product-manager
   persona the one-command rule may hide it, but block 6's third criterion ("only the design verb's
   report names a merge") would fail on envision's report if it follows its own text. Is that a known
   gap in #172, or something to fix before the run?
