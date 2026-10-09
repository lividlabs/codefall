# Tracker profile: beads

**Status:** supported.

No mirror. `tracker` is `beads` in `.codefall/settings.json`, which means the project copies its bug
reports nowhere: the document is the whole record, and Beads holds the fix from the moment
`codefall-plan` or `codefall-fix` creates its bead, as it does under every tracker. Nothing in this
profile touches GitHub, and `gh` is used only for the pull request.

Every section below answers the question the `github` profile answers under the same heading, so a
verb that follows that profile's sequence finds its step here and does what it says.

## Preconditions

None.

## Issue shape

There is no issue. Leave the `**Issue:**` row out of the report.

## Labels

None.

## Searching for duplicates

Nothing to search beyond what the step already searched: the reports, the graph, and the specs.

## Creating

Write nothing. The report is committed without an issue number, and the pull request body carries no
`Relates to` line.

## Adopting an issue the reporter filed

There is no tracker to adopt from. A run that started from a `codefall-test` report or triage note
still takes what that carries, per step 2.

## Refreshing

Nothing to refresh: an edited report is the refreshed record.

## Archiving

Nothing beyond the report's own archiving.

## Work state

Beads' to say, from the bead `codefall-plan` or `codefall-fix` creates for the fix. The fix's pull
request carries no `Fixes #` line, and the report stays `Ready` as it does under every tracker.
