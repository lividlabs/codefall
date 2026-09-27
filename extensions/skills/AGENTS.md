# codefall skills — operative rules

Terse on purpose: the why lives in the ADRs and the linked docs. This file holds the rules for
writing and changing a skill; the extension-wide rules, ADR immutability, prose, and workflow, are in
[`../AGENTS.md`](../AGENTS.md).

## Shape

- Named as **verbs** (`codefall-scaffold`, `codefall-graft`), one directory each:
  `<verb>/SKILL.md`.
- `codefall-envision`, `codefall-specify`, `codefall-mock-up`, `codefall-scaffold`,
  `codefall-graft`, and `codefall-equip` carry `disable-model-invocation: true`; a user invokes
  each deliberately. `codefall-design`, `codefall-implement`, `codefall-test`, `codefall-review`,
  and `codefall-refresh` carry no such line, so an agent may also run them: a session can carry a
  design through implementation, review, and test without a person typing each verb. Each still
  reports, offers, and applies only what the user takes; a human still performs every merge.
  `implement`, `design`, and `specify` run refresh when preflight reports the environment stale.
  No other skill invokes another through the harness.
- A skill reports and offers; it applies only what the user takes. Nothing lands unrequested.
  Recording an observable fact is the exception: a skill that owns a status transition sets it when
  the fact occurs and reports that it did — `codefall-implement` flipping a vision to `Active` at first
  claim is this shape. Judgment transitions — promote, archive, revise — stay offer-only.
- A skill that writes to the repository lands its files per `../shared/landing.md`: the branch and
  the commit are part of the write the user already confirmed, the push and the pull request are
  one offer, and the merge is never the skill's. No skill says "do not commit"; the one prohibition
  is the default branch, and the guard hook holds it.
- **Refuse only what you cannot do.** A missing surface profile, a missing tool, an unsupported
  tracker — those are exits. Disagreeing about size, altitude, or fit is not: say what you think and
  why, then do what the user asks. `codefall-envision`'s floor and `codefall-specify`'s cohesion check are both
  this shape.
- **A report ends with what the user does next** — merge the pull request, run the next verb,
  answer a question — as its last line, and a stop that hands over a remedy says to rerun the verb
  after it. A user should never have to work out the next step from what was produced.
- Never present an option that would be refused — unsupported stacks and planned profiles are
  exits, not menu choices. See the stack question in `codefall-scaffold`'s SKILL.md.
- Every verb reads `.codefall/skills/<verb>/CUSTOMIZE.md` from the user's project when it exists —
  project procedure the extension cannot know. The procedure lives in
  `extensions/shared/customizations.md`; a skill points at it and never restates it.

## Length

Guidelines, not hard limits — [ADR-004](../../docs/adrs/ADR-004-skill-length-guidelines.md) has the
reasons and the sources.

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
  `extensions/shared/`, which is where `../scripts/skill-health.sh` resolves the `.codefall/` prefix.

## Templates and ownership

- **Whose document is it** decides who repairs it. A file the extension ships that nobody amends — the
  operative rules a verb installs alongside a directory it owns, like `docs/visions/AGENTS.md` — is
  repaired by the verb that owns it, on run. A template that becomes the project's own document, one
  that gets stamped, amended, and cited, belongs to `codefall-graft`, and ships a row in its scope table and in
  `lineage.md`. Do not route a fixture through `codefall-graft`: it buys consent machinery for a decision with
  no stakes, and costs a registration step whose failure is silent. Either way, **never overwrite a
  file that has drifted** — show the difference and ask.
- Renaming, moving, or retiring a template ships a row in
  `codefall-graft/lineage.md`, in the same PR. `codefall-graft` can only tell a rename from
  a deletion plus an addition because that record exists.

## Prose

- Declarative, refusals stated plainly, reasons linked rather than attached. Read
  `codefall-scaffold`'s SKILL.md end to end before writing one.
- The banned words in [`../AGENTS.md`](../AGENTS.md) apply.
