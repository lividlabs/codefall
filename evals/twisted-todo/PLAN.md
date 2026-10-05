# Eval: a todo list with a twist, end to end

An end-to-end test of codefall's chain on a small project, with an agent playing the product
manager and a second agent judging the transcripts. It tests what #171 and #172 changed: a verb
runs the verb upstream of it instead of handing the person a command; specify makes the mockups its
requirements need in the same run; each document pull request is pushed and opened without asking,
the report tells the person to add the `auto-merge` label when they want it merged or to have
someone approve it, and a GitHub Action merges it on that label or on an approving review, so nobody
is asked to merge a document and no verb merges one; design
asks one question about the technical decisions it set aside and, on
"settle now", reaches beads in the same run; review and test ask "Fix them all?" and run implement
themselves; implement's integration asks GitHub rather than rebasing; the product-manager report
ends with one sentence and one command; the chart line and the notes log sit on the epic.

Everything lives under `evals/twisted-todo/`: this plan, the PM brief (`brief.md`), the project
skeleton and its scripts (`fixture/`), the driver (`driver/`), and the judge's rubric, prompt, and
answer schema (`judge/`).

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
one sentence, and says "settle them now" when design asks about the decisions it set aside).

## The fixture

`fixture/` is the project the chain runs against: a web page that says nothing is here, a health
endpoint, a migration runner with one migration, two unit tests, the local scripts, and a Playwright
configuration with no specs. Node 24 with the standard library's `node:sqlite` and `node:http`,
server-rendered HTML, Playwright as the end-to-end runner: Node 24 ships SQLite in the standard
library, so the app has no runtime dependencies and no native build, and Playwright is the runner
`codefall-equip` already declares for a browser front end, so nothing the skills do not know is in
play.

`fixture/setup.sh` creates the throwaway: builds `codefall` from this branch, copies the fixture,
makes a private GitHub repository and pushes it, runs `codefall init --harness claude --tracker
github` (which runs `bd init`), sets the persona to `product-manager`, copies the document-landing
Action from the template codefall ships (`extensions/skills/codefall-equip/templates/
codefall-land-documents.yml`) into `.github/workflows/`, creates the `auto-merge` label the Action
listens for, pushes, and adopts the git origin as the Beads Dolt remote. It declares nothing under `local` or `test.runners`: the chain's first two
sessions are `equip local` and `equip test`, so equip is under test too. `fixture/teardown.sh`
deletes the repository. Neither has been run.

## The run protocol

One session per verb, each a fresh `query()` in the project directory, which is the `/clear`. The
first message is the slash command with its argument; the argument is read from the project tree
(the newest `VISION-`, `SPEC-`, or `DESIGN-` identifier). The driver's `protocol.ts` holds this
table as code, with the "after" column as the step's `after` field.

| Session | First message | The simulator must | The simulator must not | After the session, the driver |
| --- | --- | --- | --- | --- |
| equip-local | `/codefall-equip local` | say yes to engineering work; declare the existing `scripts/local.sh`; say yes to proving it | ask to merge | merges the `equip/local` pull request as the person; returns the checkout to `main` |
| equip-test | `/codefall-equip test` | say yes to engineering work; declare the existing Playwright configuration | ask to merge | merges the `equip/test-harness` pull request as the person; returns to `main` |
| envision | `/codefall-envision <the idea in Mara's words>` | give the idea as spoken, answer why now and non-goals, keep rules at rule level, confirm a Ready document | give behaviour precise enough to build; ask to merge | adds the `auto-merge` label to the `vision/` pull request as the person, waits for the Action to merge it; returns to `main` |
| specify | `/codefall-specify VISION-001` | work from the vision; aim at three requirements; say no mockup exists and ask for them (main page full and empty, Forfeits full and empty); confirm recap and document; Ready | ask to merge; accept more than three requirements without saying so; choose anything technical | adds the `auto-merge` label to the `spec/` pull request as the person, waits for the Action to merge it; returns to `main` |
| design | `/codefall-design SPEC-001` | accept the tier; ask for five tasks or fewer; confirm the criteria cover the five rules; decline technical choices; answer "settle them now" to the one question | decide anything technical | stops the chain if the design is `Draft` with decisions left; otherwise adds the `auto-merge` label to the `design/` pull request as the person, waits for the Action to merge it; returns to `main` |
| implement | `/codefall-implement DESIGN-001` | say yes to engineering work; say go at the gate; accept the serial stack | ask to merge | nothing; the code stack stays open |
| review | `/codefall-review DESIGN-001` | say yes to engineering work; review with every lens; answer "yes, fix them all"; say go at the implement go gate inside the session | fix anything itself | nothing |
| test | `/codefall-test DESIGN-001` | run now; take the problems that break a rule or something she said she would see, decline the rest with a reason; say go at the implement go gate if any were taken | ask for fixes by hand | merges the code stack at its top as the person, from outside the project directory; returns to `main` |

**How the driver answers.** `AskUserQuestion` calls reach the driver's `canUseTool` callback with
the questions; the simulator returns a label per question and the callback returns them as
`updatedInput.answers`. Every other tool is allowed, so the run never waits on a person; codefall's
guard hook still runs on every Bash call. When a turn ends in prose, the simulator classifies the
last text: a question or a document shown for approval is a reply; a final report, or a stop whose
remedy is a command for the person, is done; a loop is stuck. Caps: 60 PM turns and 90 minutes per
session. Review and test each contain a run of implement that the verb started itself, so those
two sessions are longer than the others.

**Adding the label and waiting for the Action.** After envision, specify, and design, the driver
finds the newest pull request on the verb's branch prefix and adds the `auto-merge` label to it,
acting as the person from outside the project directory (`gh pr edit <n> --add-label auto-merge
--repo <owner>/<repo>`), which is what fires the Action. The Action also merges on an approving
review, but the driver is the only person in this run and GitHub does not let a pull request's
author approve it, so the label is the one way the driver has. Then it waits up to ten minutes
(`--land-minutes`) for the pull request to be merged. If the pull request is still open when the
wait ends, the driver merges it itself and records a deviation. A pull request that already carries
the label when the session ends, or that is already merged, is recorded as a deviation too, because
no verb adds the label and no verb merges. A design left `Draft` with decisions set aside stops the
chain, because the person said to leave them for an engineer and no engineer is in this run; the
brief says to settle them now, so that path is a simulator failure, recorded as such.

**Returning to `main`.** A verb leaves the checkout on the branch it wrote. After each landing the
driver runs `git fetch`, `git checkout main`, `git pull --ff-only` in the project directory, as the
person would, and records it as not a deviation. It does this with plain git rather than a
`/codefall-refresh` session so the run spends no tokens on it.

## Merging, in plain words

The person is away, and no human merges. Documents land through the Action described in
`docs/landing-documents.md`: each document verb pushes and opens an ordinary pull request and tells
the person to add the `auto-merge` label when they want it merged or to have someone approve it; the
Action runs on that label or on an approving review, checks that every path is a document path, and merges with `gh pr merge --squash` under the default token.
The driver adds the label in the person's place, from a session outside the project directory. A
throwaway needs no branch protection, so the default token can merge. Equip's two pull requests
and the code stack are a person's to merge, and the driver merges them acting as the person from
the same outside session, where codefall's guard hook does not apply: `gh pr merge --squash` for an
equip pull request, `gh stack merge <top> --squash --yes` for the code stack, each with the run
directory as its working directory and `GH_REPO` naming the repository. Inside a session the guard
also denies `gh stack merge`, so a verb cannot land the stack itself.

## The judge

The judge runs in another harness through the project's own `.codefall/shared/run-agent.sh`, the
way codefall's verbs run a reviewer or a consult: read-only, headless, with the prompt as a file and
the answer as a file. The driver tries `muse` first and `opencode` as the fallback
(`--judge-agents` changes the list), reads the exit code the way `running-agents.md` says (0
answered; 69 and 64 skip; 73, 75, 76 advance), and records which harness answered in `summary.json`
and `verdict.json`. The judge never goes through the Claude SDK.

`judge/judge-prompt.md` is the judge's instructions, `judge/rubric.md` the rubric, and
`judge/verdict.schema.json` the shape of the answer. The driver renders the prompt as
`judge-prompt.rendered.md` in the run directory with the schema and the run directory's path
appended, and writes `verdict.md` from the JSON the judge returns; the judge writes no file itself.

`judge/rubric.md` holds the eight rubric blocks from the revised body of #172 word for word, and
three blocks of this eval's own for the twist's rules in the same shape: tasks come in pairs;
finishing one forfeits the other; the Forfeits page is the record. The judge names the transcript
each block applies to, quotes the deciding lines, gives `pass`, `fail`, `not exercised`, or
`unclear`, and checks the block's "Not required" line before any fail. The prompt also states the
facts about this run that bear on reading the rubric: every document pull request has `main` as its
base and was opened as an ordinary pull request, with the `auto-merge` label added by the driver as
the person and never by a verb; specify commits the mockups with the spec, so there is no mockup pull
request; the Action and the label were installed by the setup script, so block 4 (equip's landing
track) has no transcript and is marked not exercised; design should reach `Ready` in its own
session because the simulator says "settle them now"; review and test each contain an implement
run.

The judge reads the transcripts and an `evidence/` directory the driver gathers: pull requests with
bases, draft flags, labels, and states, issues, Action runs, the git log of `main`, the document tree on
`main`, the settings, the Beads epic and its children and notes, the review and test records, and
the test cases.

**Pass for the whole run:** every block from 1 to 8 that was exercised passed; every document pull
request was reported as open with the `auto-merge` label or an approval left to people, and was
merged by the Action
once the driver added the label; the code stack was
reported mergeable at implement's integration; and the twist blocks (9 to 11) passed on the shipped
work. "Pass with findings" is the same except one block failed on a single sentence or a single
session. Anything else is a fail, named by the first block that failed and the line that failed it.

## Token budget

Rough, before a first run; the SDK's `total_cost_usd` and `modelUsage` on each session's last result
line are the measurement for the sessions and the simulator. The judge's cost is in the other
harness's own accounting, not here.

| Session | Estimate | What drives it |
| --- | --- | --- |
| equip local, equip test | 0.3M | two short searches, two declarations, two proofs |
| envision | 0.2M | the interview and the template |
| specify, with two mockup surfaces in four files | 0.6M | the interview, the audit, the mockup files, the tracker mirror |
| design | 0.5M | reading the spec, the code, the references; the plan file; the one question and the defaults |
| implement, five beads as a serial stack | 2.5M | one worker session per bead, each reading the design, the spec, the case format, and running checks; the root's bookkeeping |
| review, with its implement run | 1.0M | the reviewer subagents on the stack's diff, then a worker per problem taken |
| test, with its implement run | 0.8M | the Playwright specs and the report, then a worker per problem taken |
| simulator, about fifty calls | 0.7M | the transcript tail on every call |
| total | about 6.6M | mostly cache reads |

**The bead count is the token lever.** Each bead is one worker session that reads the same
documents and runs the same checks, so five beads is roughly five times one; four beads saves a
fifth of implement. The other levers, in order: the number of mockup states (four files, no loading
or error drawings); the worker model (`sonnet` for workers where the design proposes it); the effort
level; dropping review lenses the simulator does not need; one variant per test case.

## Open questions for the maintainer

1. **The Action with the default token.** The workflow has not been run. If `gh pr merge` under
   `GITHUB_TOKEN` is refused in the throwaway, the driver's fallback merges the pull request itself
   and records the deviation. Should the fallback count against the run?
2. **Merging the code stack after test.** The driver merges it so the delivery reaches `main` and
   the evidence shows the shipped app. The alternative is to stop at open pull requests, which is
   where implement's report leaves a person. Keep the merge?
3. **Models and effort.** Sessions default to `opus`, the simulator to `sonnet`. Workers take what
   the design proposes. The judge's model is whatever `muse` or `opencode` is configured with in the
   project's `harnessConfig`, which `codefall init` leaves empty, so each runs with its own default.
   Change any of these?
4. **The Dolt remote.** `setup.sh` runs `bd dolt push --yes`, which pushes `refs/dolt/data` to the
   throwaway so no verb asks about adopting a remote. Fine for a throwaway?
5. **Review's implement run under the product-manager persona.** Implement says it is engineering
   work and asks whether to continue, then shows a go gate; inside a review session that is two more
   questions the simulator answers. The protocol notes tell it to say yes and go. Is that the
   behaviour you want to see, or should review's implement run skip the engineering-work question
   because review already asked it?
