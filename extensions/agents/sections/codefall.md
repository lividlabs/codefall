<!-- BEGIN CODEFALL PROCESS -->
## Codefall

This project runs codefall's verbs, each a skill that reports, offers, and applies only what the
user takes. `envision`, `specify`, `mock-up`, `scaffold`, `upgrade`, and `equip` are invoked
deliberately by a user; `design`, `implement`, `test`, `review`, and `refresh` may also be run by an
agent, so a session can carry a design through implementation, review, and test without a person
typing each verb. They chain from an idea to open pull requests — `envision` → `specify` →
`mock-up` → `design` → `implement` → `review` → `test` — and each leaves something the next one
reads. Beside the chain, `scaffold` starts a project, `upgrade` brings its install and documents
current, and `equip` and `refresh` keep the local environment level with the checkout. A human performs every merge to `main`. Documents in the repository are
canonical for the why, the what, and the how; the tracker mirrors specs; Beads holds task state.
`.codefall/shared/workflow.md` has the chain, what each verb reads and writes, and who is
authoritative for what.

Instructions from the user or elsewhere in this file take precedence over this section.
<!-- END CODEFALL PROCESS -->
