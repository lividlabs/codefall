# Triage

What happens to what a run found. Read it when a run failed, could not reach something, or turned up
an anomaly.

**Search the tracker before classifying anything as new.** A failure already reported is a comment
on that issue, not a second issue.

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

**A finding becomes a tracker issue only on the user's explicit word.** Search for an existing issue
first, and offer to comment on it rather than opening a second. Present the candidates as a list and
wait; nothing is filed because a run found it. A real bug the user wants filed is filed through
`/codefall-report` with the run report's path, so the issue and the bug report are one record.

## What triage never does

- **Never edits a criterion to make a run pass.** Not the words, not the numbers, not the variants
  it names.
- **Never quarantines agent variance.**
- **Never deletes or disables a case** that is failing. A failing case is the finding.
- **Never commits the triage notes.**

**A first run of a new case that finds nothing is worth a sentence in the report.** Criteria written
from the implementation pass on the first run by construction; criteria written from a spec usually
find something. An all-green first run is a reason to check where the criteria came from.
