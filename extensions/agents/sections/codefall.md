<!-- BEGIN CODEFALL PROCESS -->
## Codefall

This project runs codefall's verbs, each a skill that reports, offers, and applies only what the
user takes. `upgrade` and `equip` are invoked deliberately by a user; every other verb may also be
run by an agent, and a verb runs the verb upstream of it when the work needs it, rather than telling
the person to. They chain from an idea to open pull requests — `envision` → `specify` →
`mock-up` → `design` → `implement` → `review` → `test` — and each leaves something the next one
reads, so a delivery runs as rounds of `implement`, `review`, and `test` with a `/clear` between
verbs, and ends when the epic has no open children and the last test run passed. A bug a person saw
enters at `report`, and `design` takes its report as it takes a spec; a bug `test` found enters as
the issue it files on the user's word, and as a child of the epic inside a delivery; `fix` runs
`design` at tier 0 and `implement` on one bead in a single run. Beside the chain, `scaffold` starts
a project, `upgrade` brings its install and documents current, and `equip` and `refresh` keep the
local environment level with the checkout. A human performs every merge to `main`. Documents in the
repository are canonical for the why, the what, and the how; the tracker mirrors specs and bug
reports; Beads holds task state. `.codefall/shared/workflow.md` has the chain, what each verb reads
and writes, and who is authoritative for what.

Speak plainly. Use everyday words, full sentences, and the active voice. Omit needless words. Say
the concrete thing, not the category it belongs to.

Instructions from the user or elsewhere in this file take precedence over this section.
<!-- END CODEFALL PROCESS -->
