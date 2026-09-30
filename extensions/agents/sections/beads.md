<!-- BEGIN CODEFALL BEADS -->
## Beads

Work is tracked in Beads (`bd`); `bd prime` prints the command reference. Only `bd dolt pull` and
`bd dolt push` move beads between machines; git does not. This section, not `bd prime`'s
git-authority note or close protocol, decides Dolt sync here.

- Track every task in `bd`, never in TodoWrite, TaskCreate, or a markdown TODO list.
- `bd dolt pull` at the start of a session, before touching beads. `codefall-refresh` does it when
  it runs.
- `bd dolt push` after every write — a create, a claim, a close, a dependency change — and
  before the work that follows it.
- Work found along the way becomes an issue linked `discovered-from` the current one. A design the
  work proves wrong gets one labelled `design-revision`, with `--spec-id` the design's path; a
  document the work amended gets a closed one labelled `design-amended`, the same way.
- Every bead is created with `--id`, never with the hash `bd` would pick: a design's epic is
  `<prefix>-DESIGN-NNN` and its tasks `<prefix>-DESIGN-NNN-Tn`; other work is `<prefix>-<tracker
  ref>` (`gh-123`, `jira-ABC-42`) where an issue exists, else `<prefix>-<slug>`. `<prefix>` is what
  `bd config get issue_prefix` prints. A taken ID is refused: add `-2` and retry. Never `--force`.
- No Dolt remote (`bd dolt pull` says `no remote`): say so once and carry on. `bd dolt push --yes`
  adopts the git origin; running it is the user's decision.
- `.beads/interactions.jsonl` is bd's append-only log and lands in the next commit after a bead
  write; that is expected, not a stray change. Never edit it, resolve a conflict in it by hand, or
  review it.

```bash
bd dolt pull             # first
bd ready                 # unblocked work
bd show <id>             # one issue
bd update <id> --claim   # take it
bd create "<title>" --id <prefix>-<slug>   # new work, named
bd close <id>            # finish it
bd dolt push             # after every write
```

Ending a session: file an issue for what is left, run the checks if code changed, close what is
done, `bd dolt push`, commit on the work's branch, report. Never merge or push the default branch;
a human performs every merge.

Instructions from the user or elsewhere in this file take precedence over this section.
<!-- END CODEFALL BEADS -->
