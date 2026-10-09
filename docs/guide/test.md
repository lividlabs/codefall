# /codefall-test

`/codefall-test` runs the tests your project declares and writes down what happened. Use it to run
the whole suite, the tests your changes reach, or one test case, and to test an epic's work before
you merge it.

```
/codefall-test changed
```

## What can it run?

The argument decides what one run covers:

| Argument | What runs |
| --- | --- |
| `suites`, or nothing | Every test suite the project declares |
| `changed` | The suites that the files changed since the default branch reach |
| A suite name | That suite alone |
| A case, such as `trips/export-itinerary` | That one test case |
| An epic, such as `trips-PLAN-007`, or a plan, such as `PLAN-007` | Every test case the epic's beads name, plus the `changed` suites, on the epic's branch |

Before anything runs, the skill shows you what it will run, which commands it will use and where it
found them, and where the report will go, and it waits for you to confirm.

The commands for a suite come from the same three places, in the same order, that
[`/codefall-implement`](implement.md) uses for its checks:

1. a `CUSTOMIZE.md` file for this skill, at `.codefall/skills/codefall-test/CUSTOMIZE.md`;
2. the project's `AGENTS.md`;
3. what the skill can infer from the repository, such as `package.json` scripts or a `Makefile`.

The confirmation says which of the three supplied the commands.

## What is a test case?

A *test case* is one Markdown file that describes how to check a behavior through the running
product. It lives at `<root>/test-cases/<area>/<slug>.md`, where `<root>` is the *testing root*:
the directory `codefall init` asked you for, such as `testing/`. A case for itinerary export might
be `testing/test-cases/trips/export-itinerary.md`.

The file's path is its identifier, `trips/export-itinerary`, so no separate list of cases has to be
kept in step with the directory. The file has two parts:

- **The frontmatter** lists the *variants*, such as paying by card or by saved card, and the case
  runs once for each. It also lists the fixed messages a run is allowed to send.
- **The body** holds the criteria the run is judged against.

Every criterion says where it came from. It either cites a spec criterion in full, such as
`SPEC-003-REQ-01-AC-01`, or it is marked `derived` and names the requirement it elaborates, with one
line saying what it adds. A criterion may never come from the implementation: not from the code, not
from clicking through the app, and not from the pull request that built it. A test written by
reading the code confirms what the code does, bugs included, and then passes without telling anyone
anything.

[`/codefall-plan`](plan.md) decides which tasks need a case, and
[`/codefall-implement`](implement.md) writes the case file before the code.

## How is a case run?

A case runs in one or both of two *modalities*, and each produces a different kind of evidence.

**A `spec` case** is a generated test that sits beside the case file. Your project's own test runner
executes it, and the result is that runner's pass or fail.

**An `agentic` case** is worked through step by step by the agent itself, through a browser tool or
the shell. The agent judges each criterion against what it could observe:

| Criterion verdict | Meaning |
| --- | --- |
| `held` | What the criterion asks for was observed |
| `failed` | The state arose, and the criterion was not met |
| `skipped` | The criterion does not apply to this variant |
| `unreachable` | The state the criterion needs never arose |

Each variant then gets an overall verdict of `PASS`, `FAIL`, `ERROR`, or `PARTIAL`.

The tool the agent drives, called the *driver*, is whatever the session actually has, such as a
browser tool. The run proves the driver with one live call before depending on it. When no driver
works, the run refuses the agentic modality for that case instead of faking a result.

Nothing is mocked, at any layer. When a state cannot be produced through the product's own
interfaces, the criterion that needs it is recorded as `unreachable`.

## What does a run leave behind?

Every run writes a report under `.codefall/tests/`, as a Markdown file to read and a JSON file for
counting across runs. The report records:

- the verdicts, and the evidence for each criterion;
- how many attempts each step took, even when it passed;
- what the run created against real services, and whether it was cleaned up;
- which driver ran;
- an anomaly sweep: anything visibly wrong that no criterion asked about.

Bulky output stays out of the repository's history. Runner output, traces, HTML reports, and
accounts created for one run go under the testing root's `.artifacts/` directory, which is
git-ignored.

## What happens when a test fails?

A failing run produces a finding. It never produces an edit to a criterion. The skill sorts each
finding into one of four classes:

| Class | Meaning |
| --- | --- |
| Real bug | The product is wrong |
| Wrong expectation | The criterion is wrong |
| Flake | The test raced the product |
| Agent variance | A product that plans its own actions took a different route, recorded with how often it happened, such as "2 of 6 runs" |

A finding becomes an issue on your project's tracker, or a bead in a project that keeps no issue
tracker, only when you say so. When the run tested an
epic, it asks one question, "I found N problems. Fix them all?", and each problem you take also
becomes a bead under the epic. The same session then runs `/codefall-implement` on the epic to fix
them.

The skill does not fix what is missing around a test. Setting up the runner belongs to
`/codefall-equip`, and writing a case belongs to `/codefall-implement`. When either is missing, the
skill names the command that fixes it.

## How does a project get a test runner?

`/codefall-equip` sets up the test runner, as its own pull request. It reads the testing root,
searches for an existing runner configuration and for wherever your end-to-end tests live today, and
asks you one question with what it found. You can declare the runner you already have, or have it
set up the default runner for each *surface*, which is a part of the product with its own code, such
as a web front end or a Go service:

| Surface | Default runner |
| --- | --- |
| A browser front end, an Electron shell, or an HTTP API | Playwright |
| Go | `go test` |
| React Native, Tauri, or Flutter | None in this version. The `spec` modality is refused for them, and agentic cases still run |

Equip then writes four things:

- the runner's configuration, pointed at `<root>/test-cases`;
- the runner's name in `test.runners` in `.codefall/settings.json`;
- its run-all and run-one commands in the testing root's `AGENTS.md`;
- whatever the runner's own install needs, such as Playwright's browsers, added to the project's
  `update` script.

To prove the setup works, equip runs the runner against the empty `test-cases/` directory and checks
that it finds nothing. The runner setup never rides along inside a task's pull request.
[The local environment](local-environment.md) covers equip's other jobs.
