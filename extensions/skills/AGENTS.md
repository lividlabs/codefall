# codefall skills — operative rules

Terse on purpose: the why lives in the ADRs and the linked docs. This file holds the rules for
writing and changing a skill; the extension-wide rules, ADR immutability, prose, and workflow, are in
[`../AGENTS.md`](../AGENTS.md).

## Shape

- Named as **verbs** (`codefall-scaffold`, `codefall-upgrade`), one directory each:
  `<verb>/SKILL.md`.
- `codefall-upgrade` and `codefall-equip` carry `disable-model-invocation: true`; a user invokes
  each deliberately, because each changes the install or the settings. Every other skill carries no
  such line, so an agent may run it. Each still reports, offers, and applies only what the user
  takes; no skill merges to the default branch.
- **A verb runs the verb upstream of it** when the work needs that verb's judgment, in the same run
  and on the same branch, under the confirmation it already holds: `specify` runs `mock-up` for a
  requirement that needs one, `plan` runs `specify` on a `Draft` spec and `mock-up` on a
  requirement still waiting, `specify` runs `envision` when a vision needs more than a text
  amendment, `review` and `test` run `implement` on the epic to build the problems the person took.
  It asks the person only for a product decision. It never tells the person to run a verb and come
  back. `plan` with technical decisions it set aside asks once whether to settle them with
  defaults now or leave them for an engineer, and on "now" reaches `Ready` and beads in the same
  run. `implement`, `plan`, and `specify` run refresh when preflight reports the environment
  stale. Running a skill needs `Skill` in `allowed-tools`.
- A delivery — one epic from plan to merge — runs as rounds of implement, review, and test, and
  every handoff is an artifact, so a verb never relies on the conversation that ran the one before
  it; its report ends with one next step and says it is safe to `/clear`.
- **`codefall-fix` restates parts of `codefall-plan` and `codefall-implement`** for one bead at
  tier 0; its `NOTES.md` has the table of which parts and where. A change to any of those parts
  checks `codefall-fix` in the same pull request, and a change to `codefall-fix` checks that it
  still agrees with them. What fix links rather than restates reaches it without an edit.
- A skill reports and offers; it applies only what the user takes. Nothing lands unrequested.
  Recording an observable fact is the exception: a skill that owns a status transition sets it when
  the fact occurs and reports that it did — `codefall-implement` flipping a vision to `Active` at first
  claim is this shape. Judgment transitions — promote, archive, revise — stay offer-only.
- A skill that writes to the repository lands its files per `../shared/landing.md`: the branch,
  the commit, the push, and the pull request are part of the write the user already confirmed, none
  of them is a question, and the merge is never the skill's. A document verb — `codefall-envision`,
  `codefall-specify`, `codefall-report`, `codefall-plan`, `codefall-mock-up` on its own — opens
  an ordinary pull request, a draft only while the document is `Draft`, and its report says to add
  the `auto-merge` label when the person wants it merged or to have someone approve it; the GitHub
  Action `codefall-equip`'s landing track installs merges a labelled or approved pull request, and
  no skill adds the label, approves, or marks a pull request ready for review as a signal. `codefall-equip`, `codefall-scaffold`, and `codefall-upgrade` open a
  pull request a person merges. Document pull requests never stack; `codefall-implement`'s code pull
  requests do. No skill says "do not commit"; the one prohibition is the default branch, and the
  guard hook holds it.
- **A verb files on the project's own tracker and nowhere else.** A bug it finds in codefall, or in
  any other repository, is handed to the person as issue text, never filed and never offered. The
  rule is stated once, under *Who is authoritative for what* in [`docs/workflow.md`](../../docs/workflow.md);
  a skill that touches the tracker points at the installed copy and never restates it.
- **Refuse only what you cannot do.** A missing surface profile, a missing tool, an unsupported
  tracker — those are exits. Disagreeing about size, altitude, or fit is not: say what you think and
  why, then do what the user asks. `codefall-envision`'s floor and `codefall-specify`'s cohesion check are both
  this shape.
- **A report ends with what the user does next** — merge the pull request, run the next verb,
  answer a question — as its last line, and a stop that hands over a remedy says to rerun the verb
  after it. A user should never have to work out the next step from what was produced. One step,
  never a sequence: for `codefall-equip`, `codefall-scaffold`, and `codefall-upgrade` that step is
  the merge, and what follows it is stated as what happens once the pull request is merged.
- Never present an option that would be refused — unsupported stacks and planned profiles are
  exits, not menu choices. See the stack question in `codefall-scaffold`'s SKILL.md.
- Every verb except `codefall-scaffold` and `codefall-upgrade` reads
  `.codefall/skills/<verb>/CUSTOMIZE.md` from the user's project when it exists — project procedure
  the extension cannot know. The procedure lives in
  `extensions/shared/customizations.md`; a skill points at it and never restates it.

## Length

Guidelines, not hard limits — [ADR-004.2](../../docs/adrs/ADR-004.2-skill-length-guidelines.md) has
the reasons and the sources.

- A `SKILL.md` body stays **under 500 lines** and **under 5,000 tokens**. Past 5,000 tokens the tail
  of the skill is dropped after a compaction, and the tail is where Process and Rules sit.
- Tokens are estimated: characters ÷ 4 and words × 1.33, the larger one read as the number.
- The `description` stays under 1,024 characters.
- `../scripts/skill-health.sh` reports every skill against these and the supporting-file rules
  below. CI runs it; `--strict` makes it fail.

## What goes where

- `SKILL.md` holds **instruction**: what to do, in what order, with what gate. The level of detail
  follows the operation — exact steps where the operation is fragile, the goal alone where several
  approaches are valid. The how stays; the why does not.
- **Reasons and rejected alternatives** go to an ADR when the repo decided it, or to the relevant
  `AGENTS.md` when it is a convention. A skill states a rule; it does not argue for it.
- **Lineage** — what a skill took from other tools and what it dropped on purpose — goes to
  `<verb>/NOTES.md`, following `codefall-review`. Never loaded by the skill.
- **Material read on demand** — lifecycle tables, strategy detail, lens definitions — goes to a
  supporting file that `SKILL.md` links directly, with one sentence saying what it holds and when
  to read it.

## Supporting files

- Linked **directly from `SKILL.md`**, one level deep. A supporting file does not link on to a file
  `SKILL.md` does not also link.
- Over **100 lines**: open with a table of contents.
- Directories are named for what they hold: `templates/` for what a skill installs, `trackers/` for
  tracker profiles, `scripts/` for what a skill runs, `reference/` for what a skill reads on demand.
  Add a directory when the content is a kind these do not describe.
- Extension paths inside a skill are relative to the skill's own directory —
  `../codefall-scaffold/templates/…` — never `${CLAUDE_PLUGIN_ROOT}`.
  Harnesses that mirror the tree under `.agents/skills/` do not define that variable, and the
  relative form resolves under them and under Claude Code alike. Hooks are the exception, and the
  rules for them live in `../hooks/`.
- **A shared file is named `../../../.codefall/shared/<file>`**, from a `SKILL.md`, and one `../`
  deeper from a supporting file. `codefall init` installs `shared/` once, into the project's
  `.codefall/`, whatever harnesses it was run for (ADR-006), and that path reaches it identically
  from `.claude/skills/<verb>/` and `.agents/skills/<verb>/`. In this repository the same file is at
  `extensions/shared/`, which is where `../scripts/skill-health.sh` resolves the `.codefall/` prefix,
  after checking that the `../` in front of it reaches the project root from the file that names it.

## Templates and ownership

- **Whose document is it** decides who repairs it. A file the extension ships that nobody amends — the
  operative rules a verb installs alongside a directory it owns, like `docs/visions/AGENTS.md` — is
  repaired by the verb that owns it, on run. A template that becomes the project's own document, one
  that gets stamped, amended, and cited, belongs to `codefall-upgrade`, and ships a row in its scope table and in
  `lineage.md`. Do not route a fixture through `codefall-upgrade`: it buys consent machinery for a decision with
  no stakes, and costs a registration step whose failure is silent. Either way, **never overwrite a
  file that has drifted** — show the difference and ask.
- Renaming, moving, or retiring a template ships a row in
  `codefall-upgrade/lineage.md`, in the same PR. `codefall-upgrade` can only tell a rename from
  a deletion plus an addition because that record exists.

## Prose

- Declarative, refusals stated plainly, reasons linked rather than attached. Read
  `codefall-scaffold`'s SKILL.md end to end before writing one.
- The banned words in [`../AGENTS.md`](../AGENTS.md) apply.
