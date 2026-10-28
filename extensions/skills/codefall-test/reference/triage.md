# Triage

What happens to what a run found. Read it when a run failed, could not reach something, or turned up
an anomaly.

## Contents

- The four classes
- Priority
- Where the notes go
- Issues
- What triage never does

**Search the tracker and the graph before classifying anything as new**, per *Duplicates* in
`../../../../.codefall/shared/triage.md`. A failure already reported is a comment on that issue or
bead, not a second one.

## The four classes

| Class | What it means | What happens |
| --- | --- | --- |
| **Real bug** | The product is wrong | The criterion stays as it is, stating the correct behaviour. Record the failure with the evidence the run has. |
| **Wrong expectation** | The criterion is wrong | Change it only against the decision that supersedes it, cited in the case file beside the criterion. Never change it to make a run pass. |
| **Flake** | The test raced the product | Record what raced. Never weaken a criterion to settle it. |
| **Agent variance** | A product that plans its own actions took a different route | Record it **with an occurrence count** — "receipt line absent 2 of 6 runs". Never quarantined. |

**Agent variance is product behaviour.** When the product under test is itself an agent, the route
it takes is what a user meets, so a pile of variance entries is a finding about the product and not
noise in the harness. A failure that is consistent across attempts and across runs is not variance:
classify it as a real bug in that agent's own instructions.

**An anomaly from the sweep is triaged like anything else**, and it stays out of the verdict. It was
not a criterion; it does not fail one.

## Priority

Every finding carries a Beads priority beside its class, and a bead filed for it is created with
`-p <n>` set to that priority. The class word stays; the priority is what `bd ready` sorts by.

**A real bug takes its priority from the shared scale** in `../../../../.codefall/shared/triage.md`,
by how bad it is for a user: critical P1, major P2, minor P3, trivial P4. Failing a criterion says
the product is wrong, not how wrong. The other classes take these:

| Finding | Priority |
| --- | --- |
| Wrong expectation | P2 |
| Agent variance | P3 |
| Anomaly from the sweep, when it is not a real bug | P3 |
| Flake | P4 |

P0 is a person's to set by hand on a bead they judge urgent; no verb assigns it. The run record
carries the priority on each triage entry as `priority`, one of `P0` to `P4`, and the Markdown
report's Triage section shows it on the finding's line.

## Where the notes go

```
<root>/.artifacts/triage/<area>.md
```

Working notes for the cycle, under the git-ignored artifacts directory, grouped by the product area
the case belongs to. They are never committed: the committed record of a run is its report under
`.codefall/tests/`, and a triage note is a draft of something that has not been decided yet.

One entry per finding: the case and variant, the criterion or the anomaly, the class, the evidence
in a line, and the occurrence count where the class carries one.

## Issues

**A real bug becomes a tracker issue**, filed by this run per
`../../../../.codefall/shared/decisions.md`, and only on the project's own tracker, per *Who is
authoritative for what* in `../../../../.codefall/shared/workflow.md`: a bug in codefall itself,
or in any other repository, is handed to the person as issue text and never filed. Search for an
existing issue first, and comment on it rather than opening a second. The report lists every issue
filed and every comment added.

```bash
gh issue list --search "<keywords>" --state all --json number,title,state,url,labels --limit 10
```

**File it in this session.** A new issue carries the `bug` label and a body written from the
run: the case and variant, the criterion that failed with its verdict, the evidence as paths under
`<root>/.artifacts/`, and the run report's path. A comment on an existing issue carries the same.
Record the number in the run record's `issue` field and in the Markdown report's Triage section.

**Every real bug is also a bead**, filed per `../../../../.codefall/shared/triage.md`: whose it
is, where it goes, how bad it is, which area, how big the fix is, and every field the bead carries.
When the run's target was an epic's work, the list is a report — "Found N problems; filing all
N" — and each bug's line shows the triage the reference proposes, which a person present may
change. Each files an issue and a bead beside it, in the `code` form under *Discovered work* in
`../../codefall-implement/reference/beads.md` with the triage fields. A bug that is the epic's is
a `deferred` child of it, taken at `codefall-implement`'s next round boundary, which triages the
epic's children and builds the batch; this run builds nothing. One the round still running should
build is an `open` child. A bug beside the epic is filed `open` with no parent, or under the open
followup for its area, and the report names where it went:

```bash
bd create "<the issue's title>" --id "$(bd config get issue_prefix)-gh-<n>" --external-ref gh-<n> \
  -t bug -s deferred -p <n> -l area:<name> --deps caused-by:<bead> \
  --metadata '{"size":"fix","found":"test"}'
bd update "$(bd config get issue_prefix)-gh-<n>" --parent <epic>
bd dolt push
```

`-p <n>` is the bug's priority from the shared scale. The edge is `caused-by` when the epic's
change made the bug and `discovered-from` when the run found one already there; `<bead>` is the
child whose criteria name the failed case: the one citing the failed criterion's identifier where
two children name the same case, every such child (one edge each) where none does, and the epic
itself for a suite failure that names no case. Record the bead's ID in the run record's `bead`
field beside `issue`.

**A wrong expectation whose criterion a bead cites** is filed as a `plan-revision` child in the
`plan` form of the same reference, with `--spec-id` taken from that bead's `spec_id` and `-p 2`,
and this run settles it before its report: run the `codefall-plan` skill on the document through
the `Skill` tool, in this session, for its Revise mode, which decides whether the criterion or the
plan moves, per `../../../../.codefall/shared/decisions.md`.

**A bug document is the user's to ask for.** `/codefall-report <run report path>` writes
`docs/bugs/BUG-NNN-slug.md` with acceptance criteria and adopts the issue this run filed rather than
opening a second; it interviews the person running it, so it is never run from here.

## What triage never does

- **Never edits a criterion to make a run pass.** Not the words, not the numbers, not the variants
  it names.
- **Never quarantines agent variance.**
- **Never deletes or disables a case** that is failing. A failing case is the finding.
- **Never commits the triage notes.**

**A first run of a new case that finds nothing is worth a sentence in the report.** Criteria written
from the implementation pass on the first run by construction; criteria written from a spec usually
find something. An all-green first run is a reason to check where the criteria came from.
