# codefall-test — where it came from

Why `SKILL.md` and its reference files look the way they do: what was taken from elsewhere, what was
rejected, and what was decided on the way. None of this is instruction — the skill is the
instruction. This exists so nobody re-adds something that was removed on purpose.

The decisions themselves are [ADR-007](../../../docs/adrs/ADR-007-test-cases.md), *Test Cases*, and
the alternatives are in the decision log entry it names.

## Taken

**`demo-flights`, a flight-booking application, and its ADR 0011**, *E2E test cases and the backfill
discipline*, with the working conventions in its `testing/CLAUDE.md` and the run procedure in its
`.claude/skills/dev-test-agentic/SKILL.md`. Taken as it stands: one markdown case file as the single
home for a case's inputs, prompts, and expectations; the file path as the identifier with no
registry beside it; two modalities with two verdict vocabularies; fixed messages with `{named}`
tokens for run-minted values; setup as scripts and steps through a driver; setup-only retry with
attempts always recorded; assertions that never retry; judging by what is observable and never by
the application's code; the transient-state technique and the second method for a negative
observation; the anomaly sweep; no mocking at any layer; an unforceable state dropped as a criterion
rather than relabelled; and the four triage classes with agent variance counted rather than
quarantined.

**`codefall-review`** — the shape of the skill: what one invocation is, the files beside it, a scope
table, targets by argument, a confirmation before anything runs, the `.ignore` offer, and a pair of
committed files per invocation with a JSON record beside the Markdown.

## Left out of the model

Everything that knows one project, which is the split ADR-005 drew for the local scripts:

- the Sabre CERT notes — which routes return which inventory, which carrier is least flaky, which
  property sells out at sixteen nights;
- the stranded-PNR ledger and its `recordStrandedPnr` helper, generalised here to the side-effect
  rule and `<root>/AGENTS.md`'s cleanup line;
- the npm aliases (`npm run e2e`, `e2e:seed`, `test:cases`), replaced by the Runners section of
  `<root>/AGENTS.md` and the three-source resolution for suites;
- the TypeScript loader — `support/caseFile.ts` and `caseFileForSpec(__filename)` — which is one
  runner's way of reading a case file. The format is documented instead, and a project's spec reads
  it however that project reads files.
- **GitHub issues as the only source of criteria.** The model had no specs, so an issue was the only
  record that predated the implementation. Codefall produces specs, so the spec's numbered criteria
  are the source, cited in full, and a tracker is not required to write a case.

## Rejected

**A `manual` modality.** The model dropped it and the reason holds: a case nobody runs is a coverage
claim with nothing behind it, and the label was doing the work of hiding a state that could not be
forced. An unforceable criterion is dropped and recorded.

**A registry of drivers.** A table of which tool drives which surface would be stale the week after
it was written, and wrong in any session configured differently. The agent reads its own tool list,
names what it found, and proves it with one live call.

**Working around a driver failure.** Falling back to a second tool, or to reading the application's
logs, produces a run that reports on something other than what the case names. A failure is
reported, and with no working driver the modality is refused for that case.

**Runner-specific rules in the shared check script.** The two Playwright rules earn their place —
one of them catches a check that reports a failure as an absence — but they do not generalize. A
second runner brings its own checks or none.

## Decided here

**The report is Markdown plus a JSON record.** The model git-ignored everything a run produced. The
Markdown is what a person reads; the JSON is what makes a question about several runs answerable by
reading files rather than by remembering — how often one criterion failed, which drivers have run a
case, whether a run ever failed to clean up a side effect. Triage's agent-variance class asks for an
occurrence count, and a count nobody can compute is a count nobody keeps. `run.schema.json` follows
`codefall-review`'s `findings.schema.json`: a closed enum wherever a free-text value would produce
two spellings of one thing.

**`check-cases.sh` does not call `check-cases-playwright.sh`.** They are two scripts, and a project's
CI calls whichever it wants. The Playwright rules need `npx playwright test --list`, which starts a
Node toolchain and fails for reasons no case file causes; keeping them apart means one script's exit
code always means one thing. `check-cases.sh` prints a line naming the other when `test.runners`
declares that runner, so nobody has to remember it exists.

**The frontmatter is parsed conservatively, and says what it does not read.** The check script is
bash with coreutils and optional jq — no YAML parser — so it reads top-level keys, lists in either
form, and `variants` as a block sequence of mappings. A shape it cannot read is reported as a
problem rather than passed over, which is the behaviour that keeps a silent pass from meaning
nothing.

**The sibling rule matches a prefix, not a suffix.** `check-cases.sh` knows no runner's file
extension and does not want to: a file beside `<slug>.md` whose name begins with the slug is that
case's spec, whether the project writes `.spec.ts`, `_test.go`, or something else. The cost is that
a case whose slug is a prefix of a neighbouring slug can be satisfied by the neighbour's spec, which
is a mis-pairing no check will catch.

**`allowed-tools` mirrors `codefall-review`'s, plus nothing.** Review already carries `Agent`. The
driver's own tools are deliberately absent: which tool drives a surface is discovered per session,
and a list here would be the stale registry this skill refuses to keep.

**A multi-variant agentic case runs one variant per subagent.** The model ran one chat session per
variant for the same reason: what one variant saw — a state, a page, a phrasing — should not shape
how the next one is judged. This session keeps the report and the triage, which is where the run's
whole picture belongs.

**The skill reads preflight's `test=` line.** That line lands with `codefall-equip`'s own change, one
pull request after this one. Until then no preflight emits it, so the skill reads the `test` block
from `.codefall/settings.json` directly when the line is absent and treats the two the same way.
