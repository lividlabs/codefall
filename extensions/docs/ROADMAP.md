# Roadmap

The objective of this extension is to create skills to streamline the operations of
a software factory with an opinionated approach. Future versions will allow for
customization but the first version will be a fixed approach.

## Process via Skills

The process resembles the waterfall process of yore but the ability it iterate on functionality end-toend within hours instead of months.

 - **envision** an idea into a short document that says what the problem is and why it matters, before anyone specifies or scaffolds it; visions live in the repo at `docs/visions/VISION-NNN-slug.md`. `codefall-scaffold` requires one — a requirement the user can break, recorded in the decision log when they do — because its worst failure is picking architecture defaults from a one-sentence description. `codefall-specify` may draw on one and never requires it.
 - **scaffold** the architecture and this process
 - **specify** requirements for a feature and their acceptance criteria; the specification is a document in the repo at `docs/specs/SPEC-NNN-slug.md`, holding one or more requirements, each with a user story and criteria written in EARS notation. The document is canonical and is mirrored to an issue tracker — a parent issue for the spec, a child issue per requirement — so people can see what is ready, in progress, and done. The first version mirrors only to GitHub Issues; Jira and Linear are profiles that come later.
 - **report** a bug as a document in the repo at `docs/bugs/BUG-NNN-slug.md`, the place a spec holds for a feature. It interviews the person who saw it for the steps, the expected and actual result, evidence, and the environment, tries to reproduce it through the running product while they are still there, and writes acceptance criteria in EARS that the fix is verified against. It is mirrored to one tracker issue, and `design` takes it as it takes a spec, finding the cause before it chooses a tier.
 - **fix** something small in one run: `design` at tier 0 for one bead, then `implement` on it, with one confirmation instead of two. It takes a bead, a bug report, an issue, or a description, and stops to name `design` when the work needs a design document, an ADR, or more than one bead.
 - **mock-up** the visual surface of a feature; runs before or after specify. It imports mockups exported from a design tool, or helps create one when the user doesn't have or want a design tool. Mockups land in `docs/mockups/<slug>/` and the specification references them; an issue labelled `requires-mockup` blocks design until the mockup exists.
 - **design** a system to implement a feature; the output is a design document in the repo at `docs/designs/DESIGN-NNN-slug.md` and a work breakdown whose tickets live inside of beads. The document is scaled to the change — a contained fix skips it entirely and gets beads only. It determines dependencies on the tickets it creates as well as any potential dependencies on existing tickets, and inserts the work into the graph. Choices that are hard to reverse become ADRs in the project's own sequence.
 - **implement** a feature following the dependency graph and executing work in parallel waves where possible; every test the current work needs — unit, integration, and end-to-end, planned by design or discovered mid-task — is part of the definition of done. It also owns the vision transition to `Active` — a vision is Active once work has started against it, and no earlier skill can observe that moment.
 - **review** something and fix what the user accepts — uncommitted work, a branch, an open pull request, a path, a document, or a description of what to look at. The reviewer is a subagent or another harness, so the context that finds a problem is never the one that fixes it; the session triages the findings with the user and applies the ones they take. Findings are committed under `.codefall/reviews/` and kept out of codebase search by a `.ignore` entry.
 - **test** the completed system after work lands: regression passes, retesting in a fresh context, coverage campaigns, and agentic driven tests. The boundary with `codefall-implement` is time and scope — implement's tests prove the change being built; `test` re-proves the system.

 <insert a diagram?>

 ## Roadmap

 - [x] scaffold skill for Typescript projects: Node.js, React, and/or Next.js
 - [x] envision skill: vision documents in `docs/visions/`, required by scaffold, optional for specify
 - [ ] scaffold skill for Golang projects
 - [x] specify skill: spec documents in `docs/specs/` with EARS acceptance criteria, mirrored to Github Issues
 - [x] mock-up skill: import from a design tool, or author one when there isn't one
 - [x] report skill: bug reports in `docs/bugs/` with evidence beside them, reproduced on the spot where the environment allows, mirrored to GitHub Issues
 - [x] fix skill: design at tier 0 and implement on one bead, behind one confirmation
 - [ ] specify tracker profiles for Jira and Linear
 - [ ] tracker resolution for specify: read the destination from a project-level record instead of asking each run
 - [x] design skill integrated with Beads, this requires a Beads configuration in the application
 - [x] implement skill integrated with Beads, including the vision transition to `Active`
 - [x] review skill: findings under `.codefall/reviews/`, reviewed by a subagent or another harness
 - [x] test skill: case files under the testing root, the `spec` and `agentic` modalities, run reports under `.codefall/tests/`
 - [x] equip sets the test harness up as well as the local scripts: a spec runner per surface, declared in `test.runners` and with its commands recorded in the testing root's `AGENTS.md`
 - [x] design decides which tasks need a test case and drafts its criteria into the bead; implement writes the case file from those criteria before the code, and stops a bead whose harness is not equipped
 - [ ] scaffold skill for Flutter projects
