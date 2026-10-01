# The run report

Two files per run, sharing one stem. Read before the first write.

## Contents

- Naming
- What is committed, and what is not
- The Markdown shape
- An agentic run
- A suite or spec run
- The JSON record

## Naming

```
.codefall/tests/<YYYY-MM-DDTHHMMSSZ>-<run-key>.md
.codefall/tests/<YYYY-MM-DDTHHMMSSZ>-<run-key>.json
```

`<run-key>` is the case id with its separators slugged — `checkout-place-order` for
`checkout/place-order` — plus the variant when a single variant was named
(`checkout-place-order-saved-card`). A suite run takes `suites`, `suites-changed`, or the suite's own
name slugged. An epic run takes the epic's ID — `booking-DESIGN-007` — and carries one section per
case it ran, then the `changed` suites. In full:

```
.codefall/tests/2026-09-17T142201Z-checkout-place-order.md
```

## What is committed, and what is not

**Both files are committed.** They are prose and a small record about a run, and they are what makes
variance countable across runs rather than remembered.

**Nothing a runner or a driver produced is committed.** Runner output, logs, traces, HTML reports,
screenshots, and any account or session the run minted stay under `<root>/.artifacts/`, which is
git-ignored. A report cites a path there; it never copies the content in.

The `.ignore` entry `.codefall/tests/` keeps committed reports out of ripgrep-backed codebase
search.

## The Markdown shape

```markdown
# Run: <run-key>

**Ran:** <timestamp> · **Target:** <case id, suite, or epic> · **Modality:** <spec | agentic | suite>
**Driver:** <the tool that drove it, for an agentic run> · **Commit:** <HEAD, plus dirty>
**Epic:** <epic bead ID> · **Branch:** <the branch checked out> — for an epic run

## <variant name> — PASS | FAIL | ERROR | PARTIAL

| # | Criterion | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | Confirmation names the order id | held | "Order A7K2P9 confirmed" on the confirmation |
| 3 | Declined card leaves the cart intact | failed | cart emptied; no reason shown |
| 4 | No card form with a card on file | skipped | `card` variant |

**Attempts:** step 2 took 3 attempts; every other step took 1.

**Side effects:** order `A7K2P9` — cancelled. Order `B3M8Q1` — not cancelled, the run ended first.

**Anomaly sweep:** the total on the confirmation renders with two currency symbols.

## Triage

<class per finding, the issue and the bead filed when the user asked, and where the notes were written>
```

One section per variant. A run of one variant has one.

## An agentic run

Everything above, with the agentic vocabulary: `PASS`, `FAIL`, `ERROR`, `PARTIAL` per variant, and
`held`, `failed`, `skipped`, `unreachable` per criterion.

- **Attempt counts are recorded even when the run passed.**
- **The driver is recorded**, by the name this session knows it as.
- **Every side effect is listed** with its identifier and whether it was cleaned up.
- **The anomaly sweep is its own heading**, never folded into the verdict.
- **A run that ended part way is still written**, marked `ERROR`, carrying everything observed up to
  that point and every side effect it created.

## A suite or spec run

A suite run and a spec case report the runner's own pass and fail, and **the agentic vocabulary is
not used for them**. The two modalities produce different kinds of evidence, and a shared word would
suggest otherwise.

What such a report carries: the commands that ran and which source they came from, the filter a
changed-files subset resolved to, the runner's own counts, the failing tests by name with the
runner's own message, and the path under `<root>/.artifacts/` where the full output landed.

## The JSON record

`../run.schema.json` is the shape. It holds the same run as the Markdown: the target, the driver,
the commands, and per variant the verdict, every criterion with its verdict and evidence, the
attempt counts, the side effects, and the anomaly sweep.

It exists so that a question about several runs is answered by reading files rather than by
remembering — how often one criterion failed across the last ten runs of a case, which drivers have
run it, whether a side effect was ever left behind. The agent-variance occurrence count `triage.md`
asks for is that question.

Write it beside the Markdown, at the same moment, from the same run.
