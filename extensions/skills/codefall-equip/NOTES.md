# codefall-equip — where it came from

Why `SKILL.md` looks the way it does: what was taken from elsewhere, what was rejected, and what was
tried and dropped. None of this is instruction — the skill is the instruction. This exists so nobody
re-adds something that was removed on purpose. The reasoning behind the contract itself is
ADR-005 in the repository's `docs/adrs/`, and the reasoning behind the testing track is ADR-007.

## Taken

**A per-project sync skill from an earlier toolkit** — the observation that pulling `main` has
consequences the pull does not perform, and that a teammate who does not read diffs needs a single
thing to run. That skill knew one project's package manager, migration tool, and containers. What
this skill keeps is the job; what it drops is knowing the project, which moved into scripts the
project owns.

**`codefall-scaffold`'s stack question** — name what the evidence already implies and confirm it,
rather than asking a question the repository could have answered. Here that is "search first, then
ask with evidence".

**`codefall-upgrade`'s report-then-take** — a candidate is shown with what is wrong with it before
anything is declared, and the user takes it or not.

**`codefall-implement`'s definition of done** — a change that makes the scripts stale changes the
scripts in the same pull request, the same way tests and docs are part of done rather than a
follow-up.

**demo-flights' `dev-test-backfill`, step 1** — a testing cycle does not grow its own tooling. There,
a state no existing harness can produce is recorded as an unforceable criterion rather than
answered by building something new mid-run. Codefall takes the same position and makes it
structural: the harness is built by this verb, in its own pull request, and no run that is about
cases changes what runs them.

## Rejected

**Doing the environment work in the skill.** Re-deriving what to run from the repository on every
invocation varies between runs, spends a model turn on what a lockfile already says, and cannot be
run by a person, by CI, or by a harness without the extension.

**Generating the scripts at run time.** Every cost of the above, plus a file that changes under the
project with no diff anyone reviewed.

**A one-time onboarding verb.** First draft and revision are the same procedure — read the
repository, read the scripts if they exist, propose, confirm, write, declare — so a verb named for
running once would have run every time a tool was introduced. This skill is named for what it does
to the repository rather than for when.

**`codefall-refresh` drafting on its first run.** A verb that runs daily and also does one-time
inference is two verbs. It also put the drafting interview in front of the one person least placed
to answer it: a teammate running refresh because the section in `AGENTS.md` said to.

**A template per stack.** A signals table covers more ground with less to maintain: a project is a
set of signals, not a stack, and a Go API with a Prisma-managed database is two rows, not a profile
nobody wrote.

**Routing the script through `codefall-upgrade`.** Upgrade moves documents against templates the
extension ships, with provenance. The scripts have no template behind them once drafted — they are
the project's — and this skill is already the verb that revises them, so upgrade would be a second
owner.

**A catalog of model strings per harness and provider, shipped by codefall.** Stale within a
release, and five harnesses' worth of it. The agents track reads the strings from where each harness
already keeps them instead.

**Composing a model string from a family name by prefix rules.** Most of the matrix is mechanical,
and the part that is not, Anthropic's ids on Bedrock, would have needed the catalog anyway.

**Setting a harness up only from inside it.** ADR-009.3 required it, on the grounds that the
running session's record holds the exact string. It meant opening every harness to write a few lines
of settings, and it left a Claude Code session unable to set codex up. The running session's record
is now the first place the track looks for that harness, and each other harness's own config and
recent sessions are read from outside. ADR-009.4 dropped the requirement.

**The local and test steps written out in `SKILL.md`.** They were, until the file sat at the
5,000-token guideline and every fix had to cut a sentence to make room. One run follows one track,
so the other three tracks' steps are weight a run never reads. Each track now keeps its steps in its
own reference file — `reference/local.md`, `reference/testing.md`, `reference/agents.md`,
`reference/landing.md` — with a numbered outline in `SKILL.md` that says to read the file before
step 1. The rules that hold for one track alone moved with it; `SKILL.md`'s Rules are the ones that
hold on every track. Moving a track's steps back into `SKILL.md` buys nothing a run needs and costs
the headroom.

## Why the rules are shaped this way

**Never destructive.** A script that resets to reach a known state is not idempotent; it is
starting over. "Bring the project up to date?" has to be a question a teammate can always answer
yes to, and that holds only when the answer never costs them data.

**Proving is offered, not done.** `start` brings real services up on the user's machine. That is
the one thing in this skill that reaches outside the repository, so it is a yes-or-no.

**Two calls of `update` are the proof.** The second call exiting `0` quickly is the contract made
observable: cheap when nothing changed, and safe to run again.

**Two tracks chosen by the argument, rather than one run that does both.** The testing procedure
went in as a second Process section, picked by `local` or `test`, and not as steps 7 onward of the
existing one. The two share the search-first shape and nothing else: different evidence, a different
question, a different block written, a different proof, and in the user's project a different pull
request. A single run would put two confirmation gates and two real-service proofs in one turn, and
would leave a user who came to revise `update` after adding a migration waiting through a runner
interview. The argument also gives `codefall-scaffold` and `codefall-implement` a way to name the
track they follow, which is the local one in both cases.

**The harness is its own pull request.** ADR-005's point-of-introduction rule makes a task's pull
request carry the script change the task caused, and stops there. Setting a runner up is not caused
by a task: it adds a dependency, a configuration file, an `update` step, and a settings key, none of
which the task's reviewer asked to read. `codefall-implement` names this skill and leaves the beads
that need a case unstarted instead.

**Names in settings, commands in the testing root's `AGENTS.md`.** The split is by reader. Programs
read the runner names — the check scripts decide which runner-specific rules apply, doctor decides
whether the project is equipped — and only a person or an agent runs a command, which is also the
only part that varies with the project's own idiom.

**An empty tree is what proves a harness.** The runner's list command against no cases shows the
configuration collects from the declared root, and that is the whole claim this verb makes. Writing
a case to prove the runner would put authoring in the wrong verb and leave a test behind that nobody
designed.
