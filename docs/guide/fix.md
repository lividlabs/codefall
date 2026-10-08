# /codefall-fix

`/codefall-fix` takes one small change from a description to an open pull request in a single run.
Use it for a bug fix or a small change that needs no plan document.

```
/codefall-fix BUG-012
```

You can give it a bead (a task in Beads), a bug report, a GitHub issue, or a sentence describing the
change.

## What counts as small

A change is small enough for this skill when all four of these hold:

- it stays inside one component;
- it changes no public interface;
- it is one task: one bead, one branch, one pull request;
- it needs no ADR (architecture decision record), because it adds no dependency and makes no choice
  that is hard to reverse.

[`/codefall-plan`](plan.md) calls this size *tier 0*: work that gets beads and no plan document. When
the work turns out to need a plan document, an ADR, or more than one bead, this skill stops before
writing anything and names `/codefall-plan`.

## What it does

The skill runs `/codefall-plan` at tier 0 to establish one bead, with the cause and the acceptance
criteria, and then runs [`/codefall-implement`](implement.md) on that bead, all in one run. For a
bug, it reproduces the bug if the report could not and finds the cause, citing the file and line.

Running the two skills separately would repeat work. This skill does each step once:

- one check of the project's prerequisites instead of two;
- one confirmation that shows the bead and the build plan together, including the files it will
  touch, the branch, and the commands that verify the work;
- one report at the end.

The skill follows the same rules as `/codefall-plan` at tier 0 and `/codefall-implement` for one
bead, including the merge rule: it opens the pull request and stops, and a person merges it. When the
bead traces to a bug report with an issue, the pull request closes that issue when you merge it.
