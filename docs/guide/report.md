# /codefall-report

`/codefall-report` turns a bug someone ran into into a report that another session can fix without
asking the reporter anything again. Use it when something that should work does not. It holds the
place for a bug that a [spec](specify.md) holds for a feature.

```
/codefall-report Exporting a trip with no departure date shows a blank page.
```

## What it asks

The skill interviews the person who saw the bug for:

- the steps, starting from a point someone else can reach: which account, which data, which screen;
- the expected result, and what says so, such as a spec, the docs, or how it used to behave;
- the actual result, word for word;
- screenshots, recordings, and logs, as file paths;
- the environment: version or commit, where it ran, operating system, browser or device;
- how often it happens, who it stops, and any workaround;
- the last version that worked, when there is one.

It pushes back on vague answers such as "it's broken" or "sometimes" the way `/codefall-specify`
pushes back on "fast", by asking for the concrete detail: what you saw, or how many times out of how
many tries. You also choose a severity, `critical`, `major`, `minor`, or `trivial`, and the skill
pushes back once if it does not fit the impact you described.

Before the interview, the skill checks whether the bug is already reported, in `docs/bugs/`, on
GitHub, or in Beads. If it is, the skill offers to add what you brought to the existing report
instead of writing a second one.

## How it tries to reproduce the bug

**The skill tries to reproduce the bug while you are still there.** It runs the product locally at
the checkout's commit and follows your steps as written, up to five times. A step it cannot follow
is a step the report is missing, so it comes back to you with that step.

The attempt changes nothing: no code, no data edited by hand, and no mocks. It is never required,
either. You can decline it, and it never drives a production environment. A bug seen only in
production, or only some of the time, is still reported. The report records one of three outcomes:

| Outcome | What the report records |
| --- | --- |
| Reproduced | When, at which commit, how, and how many tries out of how many |
| Not reproduced | The steps as run, and where the product did what the reporter expected |
| Not attempted | Why, such as a bug seen only in production |

## What it writes

The skill writes the report to `docs/bugs/BUG-012-slug.md`, where the number is the next one free.
Evidence goes beside it in `docs/bugs/BUG-012-slug/`, both yours and the reproduction's, each file
named for what it shows. The evidence is committed because the GitHub CLI cannot upload images to an
issue, so the issue links to the files by their path in the repository.

The report ends with acceptance criteria: testable statements of the expected behavior, written in
[EARS](specify.md#how-acceptance-criteria-are-written), such as
`WHEN a traveler exports a trip with no departure date, the system SHALL name the missing field`.
A criterion a spec already states cites it, as `(SPEC-003-REQ-01-AC-02)`, instead of restating it.
These criteria become the fix's criteria and its regression test.

The skill shows you the whole report before writing it. It then writes the report on a branch named
`bug/BUG-012-slug`, generates a GitHub issue from it when the project uses GitHub Issues, commits,
pushes, and opens a pull request. If you filed an issue first, the skill adopts that issue instead
of creating a duplicate. You merge the
pull request the way [`docs/landing-documents.md`](../landing-documents.md) describes.

## What happens next

The report ends by naming `/codefall-plan BUG-012`. [`/codefall-plan`](plan.md) takes a bug report
the way it takes a spec: it reproduces what the report could not and finds the cause before it
decides how much planning the fix needs. Most fixes land at *tier 0*, the size that needs no plan
document, as a single *bead*, which is a task in Beads. For a fix that small,
[`/codefall-fix`](fix.md) plans and builds it in one run. The pull request that carries the fix
closes the issue when you merge it.

## Statuses

A report's `Status` describes the document, never the work. A fixed bug stays `Ready`; its issue
closes when the fix merges.

| Status | Meaning |
| --- | --- |
| `Draft` | The reporter stopped and is coming back |
| `Ready` | Written and agreed; the normal end of a session |
| `Archived` | A duplicate, not a bug, or not going to be fixed; the report and its evidence move to `docs/bugs/archive/` with a line saying why |
