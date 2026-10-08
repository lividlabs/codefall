# Status and lifecycle

Three states, one word plus a date. Read at step 7, when the status is set, and for the Promote,
Revise, and Archive modes.

| Status | Meaning | Lives in |
| --- | --- | --- |
| `Draft — <date>` | Being written. The user stopped and is coming back, or **Decisions needed** is not empty | `docs/plans/` |
| `Ready — <date>` | Written and agreed, beads created. The normal end of a session | `docs/plans/` |
| `Archived — <date>` | Superseded or dropped | `docs/plans/archive/` |

- **Status describes the document, never the work.** Work state is Beads' — `bd list` and
  `bd ready`.
- `Archived` moves the file to `docs/plans/archive/` under the same name; citations still resolve.
- A plan that no longer describes the code is revised or archived, not labelled. Revising
  reconciles the graph, per the revising reference beside this file.
- A `Draft` carrying decisions creates no beads; settling them empties the section and the graph is
  created then.
