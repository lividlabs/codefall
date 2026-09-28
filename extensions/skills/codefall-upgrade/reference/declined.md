# Declined templates

How `codefall-upgrade` remembers a missing template the user declined, so the next run does not
offer it again. Read at step 2 when `scaffold.json` holds a `declined` list, and at step 5 whenever
the offer holds missing templates.

## The record

`scaffold.json`'s `declined` list holds one `{ "id": "<id>", "date": "<date>" }` per template the
user declined, dated with `date +%F`. It records an answer the user gave, so like the rest of the
file it is written by this verb and never hand-edited.

A decline is by template id. A template the extension adds after the decline is not on the list and
is offered as missing. A declined template that is later revised stays declined.

## Step 2 — a project that was never scaffolded

A `scaffold.json` that holds `declined` and no `adrs` belongs to a project that was never
scaffolded, with an earlier answer recorded. It is not **provenanced**. Take the surfaces from its
`profiles` instead of detecting and confirming them again, unless `adopt` was given.

## Step 4 — the applicable set

A missing template whose id is on the list is left out of the offer. Ask only about a gate whose
template would be offered; a declined template needs no answer. With `adopt`, every declined
template is treated as missing for this run.

## Step 5 — the question

The one question lets the user decline missing templates, all of them or some. Say that a declined
template is recorded and not offered again, and that `/codefall-upgrade adopt` offers it again. An
answer of *not now* records nothing.

When the declined templates are set aside and nothing is left to offer, the documents half of the
report is one line naming how many templates are declined and since when, and that
`/codefall-upgrade adopt` offers them again. Ask no question; go to step 8.

## Step 6 — writing it

Recording a decline is a write like any other: it takes the branch step and a clean tree, and lands
with the rest of the run.

- Add an entry for each template declined in this run. Taking a template whose id is on the list
  removes its entry.
- Where `scaffold.json` doesn't exist and nothing was taken, write it with only `profiles` as
  established, `declined`, and `lastUpgrade`. Leave out `adrs`, `pluginVersion`, `scaffoldedAt`,
  and `decisions`, because the project was never scaffolded.
- The decision-log line names what was declined — *…; declined ADR-GO-03*. A project with no
  `docs/decision-log.md` gets none created by a run that only declined.

## Step 8 — the report

Name what was declined in this run, that it is recorded in `scaffold.json`, and that
`/codefall-upgrade adopt` offers it again.
