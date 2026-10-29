# codefall

Opinionated skills for the software development lifecycle, plus the CLI that installs them. The repo
holds two components; each scopes its own operative rules:

- `cli/` — the Go installer. Rules: [cli/AGENTS.md](cli/AGENTS.md)
- `extensions/` — the skills, hooks, and shared scripts that `codefall init` installs, and the
  sections it writes into a project's `AGENTS.md`. Rules: [extensions/AGENTS.md](extensions/AGENTS.md)

Decisions for the whole repository live in [`docs/adrs/`](docs/adrs/); the in-flight scratchpad is
[`docs/decision-log.md`](docs/decision-log.md). Provenance for the scaffolded documents is
`.codefall/scaffold.json`.

## How the rules are read

Every skill, shared reference, and `AGENTS.md` section here is read by a model that applies it as
written, and a rule written as a gate stops a run on the first case it did not foresee. So a rule is
written to guide a run, not to gate it. These hold for every file in this repository and for every
document codefall installs:

- **A word names the common case, not an entry condition.** A rule written for a bug applies to a
  cleanup, a missing test, or a request that reaches it: what kind of thing the item is becomes a
  field on it, and the item is never turned away for being the wrong kind of thing.
- **An exit is for what a verb cannot do**: a missing tool, an unsupported tracker, a missing surface
  profile. Disagreement about kind, size, or fit is said once, then the run does what the person
  asked. [`extensions/skills/AGENTS.md`](extensions/skills/AGENTS.md) states this for skills.
- **A list lives in the project's data, not in the rule**, where the project has one: the labels in
  use, the types `bd` accepts, the directories under a root. The rule reads from there and takes the
  person's word for a new entry. A fixed list in a rule is for what a tool can do.
- **A case the rule leaves out is decided from the rule's purpose**, and the run says what it decided
  and carries on. It does not stop to ask whether the case is covered, and it does not ask one
  question per item: it proposes the whole list once and applies what the person takes.
- **Exact steps where the operation is fragile; the goal and its reasons where several readings are
  valid.** A mechanical test is applied mechanically, so write one only where no judgment is wanted.

## How prose is written

Every file a person reads here is written in the classic style Steven Pinker describes in *The
Sense of Style* (2014): the writer has seen something and shows it to the reader. Two documents
show the result, and are read before writing a page of either kind:
[mise's getting-started page](https://mise.jdx.dev/getting-started.html) for a guide, and the
opening of
[Better Code and Gardens](https://xddmohl7mf56oiaps45dm6vtoouoxhxkkpocwzuskmwg6mhzggea.ocap.site/)
for a reference. These hold for the README, the guide, the ADRs, the decision log, the text of a
skill, and pull request bodies. The banned words in [`extensions/AGENTS.md`](extensions/AGENTS.md)
apply to all of them.

- **State the fact; do not announce it, hedge it, or comment on it.** No "note that", "it is
  important to", "in order to", "essentially", "arguably", "simply". A sentence that says what the
  next sentence will do is deleted, and the next sentence is kept.
- **Write for a reader who is intelligent and does not know what you know.** A term is defined the
  first time it appears, in the same sentence and set in italics, and an abbreviation is spelled out
  there. One thing has one name throughout a document, because a synonym introduced for variety
  reads as a second thing. Where a rule is abstract, an example follows it.
- **A sentence names who does what.** Concrete nouns and active verbs: "the skill writes a spec",
  not "a spec is produced"; "plan decides", not "plan makes a decision". A noun made from a verb,
  "the review process", "the implementation of", goes back to its verb.
- **One idea per sentence, old before new, light before heavy.** The subject stays near its verb,
  what the reader already knows comes first, and the long or new element comes last. Two ideas
  joined by a semicolon are two sentences.
- **A paragraph has a topic and a point.** It opens with what it is about, keeps that in subject
  position from sentence to sentence, and ends when its point is made. A list holds parallel items,
  each in the same grammatical shape.
- **A guide opens with what the reader will have done by the end**, then gives the steps in order,
  each with its command and what it prints. Background and reasons go after the steps or into a
  linked document. A reference opens with its scope and says where its sources are thin.

## How codefall works

`codefall init` installs the verbs into each harness a project uses, and `.codefall/` beside them
with the settings, shared scripts, and hooks the verbs read. The verbs chain from an idea to open
pull requests, and each leaves something the next one reads: `envision` → `docs/visions/`, `specify`
→ `docs/specs/` mirrored to the tracker, `report` → `docs/bugs/` mirrored the same way, `mock-up` →
`docs/mockups/`, `plan` → `docs/plans/` and beads with dependency edges, `implement` → a
worktree, a test case, and a pull request per task, `review` → `.codefall/reviews/`, `test` →
`.codefall/tests/`. Beside the chain, `fix` runs `plan` at tier 0 and `implement` on one bead in
one run, `scaffold` starts a project, `upgrade` brings its install and documents current, and
`equip` and `refresh` keep the local environment level with the checkout. Bead state travels over
the git remote as `refs/dolt/data`: a verb runs `bd dolt push` after every bead write, and `refresh`
runs `bd sync`.
`upgrade` and `equip` are invoked deliberately by a user; every other verb may also be run by an
agent, and a verb runs the verb upstream of it when the work needs it. Every verb applies only what the user takes; no verb merges to `main`, and a hook denies the
attempt. A person merges code pull requests; a GitHub Action a project installs with `equip landing`
merges a document pull request once a person adds the `auto-merge` label to it or approves it.
[`docs/workflow.md`](docs/workflow.md) holds the full chain, what each verb reads and writes, and
who is authoritative for what.

## Workflow

These hold for both components; each component's own file adds to them.

- Every change lands through a PR, and a PR is **squash-merged**. Merge commits and rebase merges
  are turned off on GitHub, and the branch is deleted on merge.
- A change is **breaking** when a project that ran an earlier codefall has to do something by hand
  after upgrading: a flag, skill, or hook renamed or removed, a settings field or file layout
  changed, a hook command changed. A new one of any of those is not.
- **The PR title becomes the commit message** on a branch of more than one commit; a single-commit
  branch keeps that commit's own title. So the title is a Conventional Commit line naming the
  highest-impact change on the branch, and it carries `!` when any commit on the branch is
  breaking. Adding a commit means reading the title again — a branch that starts as `feat:` and
  gains a breaking rename needs a new title before it merges.
- release-please reads that one header, not the commits underneath it: the squashed body holds
  every commit message, and none of them is parsed. A breaking commit also carries a
  `BREAKING CHANGE:` footer, so it reaches the changelog by a second path if the title fails to.
