# Tracker profile: beads

**Status:** supported.

No mirror. `tracker` is `beads` in `.codefall/settings.json`, which means the project copies its
specs and bug reports nowhere: the document is the whole record, and Beads holds the tasks from the
moment `codefall-plan` creates them, as it does under every tracker. Nothing in this profile
touches GitHub, and `gh` is used only for the pull request.

Every section below answers the question the `github` profile answers under the same heading, so a
verb that follows that profile's sequence finds its step here and does what it says.

## Fits when

- `tracker` is `beads`.

## Capabilities

None. There is no issue, no label, no board, and no work-state transition.

## Preconditions

None.

## Issue shape

There is no issue. Leave the `**Issue:**` row out of the document.

## Labels

None. `draft` is the document's own `Status`, and a skipped mockup is the `Mockup: pending` line
under the requirement's **Design notes**, which `codefall-plan` and `codefall-mock-up` read from the
document.

## Searching for duplicates

Nothing to search beyond what the step already searched: the documents and the code. Beads holds
tasks, not specs, and `bd search` finds only work already planned.

## Creating

Write nothing. The document is committed without an issue number, and the pull request body carries
no `Relates to` line.

## Refreshing

Nothing to refresh: an edited document is the refreshed record.

## Archiving

Nothing beyond the document's own archiving.

## Work-state transitions

None. Whether the work is queued, underway, or done is Beads' to say from the moment the plan's task
table becomes beads; before that moment nothing tracks it, and the document's `Status` describes the
document only.
