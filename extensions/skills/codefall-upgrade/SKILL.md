---
name: codefall-upgrade
description: Bring a project's installed codefall and its documents current. The installed skills, shared files, and hooks come through `codefall upgrade`, offered first when the manifest is behind the binary and run only on a yes. The documents — inherited ADRs, `docs/adrs/_TEMPLATE.md`, `AGENTS.md` skeletons, `.codefall/scaffold.json` — are compared against the current templates, every difference is reported with its provenance, and only the pieces the user takes are applied, one at a time. Works on projects scaffolded before provenance existed, and on repos adopting the stance for the first time. A declined template is not offered again until `adopt` is given.
argument-hint: "[path] [adopt]"
disable-model-invocation: true
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
---

# Upgrade

A project scaffolded at codefall 0.2.1 never receives anything the templates gained since — new
ADRs, revised rules, renamed files all live in the extension, not in the project — and its installed
skills stay at the version that installed them. This verb closes both gaps: it offers
`codefall upgrade` for the install, then reads the project's documents, compares them against what
the extension ships now, reports every difference, and applies exactly the pieces the user takes.

Three rules hold throughout. An existing project is never replaced: this verb adds to it and
supersedes within it. Application is per item, never wholesale. And where a revised template meets
a project that amended its ancestor, the amendment wins.

This skill is **only ever invoked explicitly**: it runs on a project people depend on. Never suggest
it, never fire it from a passing remark, never chain into it from another skill.

Template paths in this document are relative to `../codefall-scaffold/templates/`, resolved from
this skill's directory, because this verb reasons about `codefall-scaffold`'s templates and has none
of its own. Neither path is relative to the user's project.

## Files beside this one

- `lineage.md` — the record of what every current template used to be called. Read whenever a
  project doc carries a historical identifier.
- `reference/pre-provenance.md` — everything this verb does differently for a project scaffolded
  before `scaffold.json` existed. Read only when step 2 finds no provenance.
- `reference/declined.md` — how a declined template is recorded and skipped. Read at step 2 when
  `scaffold.json` holds `declined`, and at step 5 when missing templates are offered.
- `../../../.codefall/shared/landing.md` — the shared procedure for the branch, the commit, and the
  offered pull request. Read at step 6, before the first item is applied.

## Scope — the install, then documents, not code

The install is `codefall upgrade`'s, and this verb only offers it (step 1). Past that it moves
**documents**. It does not restructure code.

| In scope | Out of scope |
| --- | --- |
| Inherited ADRs — new ones; revisions, landed as superseding ADRs; renames | Moving code between architectures or topologies |
| `docs/adrs/_TEMPLATE.md` | Extracting a component into a service |
| `AGENTS.md` skeleton drift, reported | Re-slicing component boundaries |
| `.codefall/scaffold.json` — written or brought current | Touching source, build config, or lint rules |
| A `docs/decision-log.md` line recording the run | The project's own `ADR-NNN` decisions |

Moving a project from ports-and-adapters to package-by-component, or pulling a component out into
a service, are real jobs — and they are code refactors with a different blast radius. They belong
to a future `migrate` verb. If the user asks for one, say that plainly and stop; do not do a small
version of it here.

The project's own decisions — the bare-numbered `ADR-NNN` sequence — have no template behind them
and are never this verb's to update. They enter the picture only when a rename leaves them citing an
old identifier, and even then the verb reports the stale reference rather than silently editing a
decision the project wrote.

## Provenance states

`.codefall/scaffold.json` records, per inherited ADR, an `amended` flag and a `sha256` (see
`codefall-scaffold` step 4). Together they separate four states, and the state decides everything this verb may
do:

| State | Meaning | This verb may |
| --- | --- | --- |
| **untouched** | hash matches the recorded one, `amended: false` | supersede or rename it when the user takes the item |
| **amended** | `amended: true` — changed during the original interview | report the diff, leave it alone |
| **edited** | hash no longer matches — changed since scaffolding | report the diff, leave it alone |
| **unverifiable** | no provenance and no old template reachable | treat as edited |

**Only untouched is ever safe to take mechanically.** Taking a revision never rewrites the
ratified file — it lands as a new, superseding ADR (step 6) — but building that successor from the
current template is only correct when the template fully accounts for what the project has. For an
amended or edited ADR it does not: the project changed the decision on purpose, and a
template-built successor would silently drop that change from what governs. So for those two the
diff *is* the deliverable — authoring the superseding ADR that carries their amendment forward is
the user's work, and after they write it they can run this verb again.

There is one narrow exception: a user who has seen the diff may explicitly say *supersede my
version with the template's*. That is their call, and supersession keeps their version on the
record — but their amendment stops governing, so say exactly that, require them to name the file,
and only then land it like any other taken revision. Never offer this as the convenient path.

## Persona

Read `persona` in `.codefall/user.json`; when it is not `engineer`, follow that persona's section in
`../../../.codefall/shared/personas.md` for this run, and say so.

## Process

### 1. Bring the install current — offered, never assumed

The target is the path argument, or the current directory without one; `adopt` offers declined
templates again. Read `.codefall/manifest.json` there. With none, say codefall is not set up here
and `codefall init` comes first, and stop; the user reruns this verb after it.

The installed version is the `shared` entry's `version`, or the highest harness entry's when
`shared` carries none. The binary's is what `codefall --version` prints after `codefall version`.
Compare the leading `MAJOR.MINOR.PATCH`; when either is a development build (`-dev`), any
difference between the two strings counts as behind.

- **No `codefall` on PATH** — say so, name the installed version, and continue to step 2: the
  documents need no binary.
- **Level, or the manifest ahead** — say which in one line and continue.
- **Behind** — name both versions and offer to run `codefall upgrade` first. Never run it unasked.
  On no, continue to step 2. On yes, run it in the directory holding the manifest, never with
  `--yes`, and put its own question to the user, since the session has no terminal to show it:
  1. `printf 'n\n' | ACCESSIBLE=1 codefall upgrade --location here` prints the breaking changes
     between the two versions and asks whether to continue; the `n` declines and nothing changes.
  2. Show the user what it printed, the breaking changes verbatim, and ask its question.
  3. On yes, `printf 'y\n' | ACCESSIBLE=1 codefall upgrade --location here`. On no, continue
     without it.

  A first run that asked nothing had nothing to ask and has already installed. Either way, show
  the report it printed — what it installed, what it removed, what it renamed — before step 2.

The files `codefall upgrade` changed are the install's. They do not count against step 6's clean
tree, and this verb's commit leaves them out; step 8 names them as the user's to commit.

### 2. Read the project — the gate

Name what you found — the project, not just the path — so the user can redirect before anything
else happens. If the target shows no sign of being a project root (no VCS, no manifest, no docs),
stop and ask rather than scanning onward.

Inventory the project: `.codefall/scaffold.json`, `docs/adrs/`, `docs/decision-log.md`, and every
`AGENTS.md`. That yields one of three starting points, and the rest of the process is the same for
all three — they differ only in how much step 3 must reconstruct:

- **Provenanced** — `scaffold.json` exists and lists ADRs. The normal case from codefall 0.4.0 on.
- **Scaffolded, pre-provenance** — no `scaffold.json`, but `docs/adrs/` holds codefall-lineage
  files. Read `reference/pre-provenance.md`; it covers this case from here on.
- **Never scaffolded** — no codefall docs at all. This is **first-time adoption**: every applicable
  template is simply missing, and the same report-then-take flow installs the stance. Detect the
  surfaces from the repo (`package.json`/`tsconfig` suggests `typescript-react`, `go.mod` suggests
  `go`) and confirm rather than assume — the profile catalog and its refusal rule are `codefall-scaffold`'s.
  If a surface has no supported profile, refuse the same way `codefall-scaffold` does: say plainly that the
  stack isn't supported yet, name what is, offer to record the request, and stop. Do not improvise.

Adoption installs documents describing a stance the existing code does not yet follow, and does not
check the code against them — say that in the report rather than letting the docs imply otherwise.
Aligning the code is `migrate`'s job, and the boundary-enforcement obligation lands on the user
exactly as it does after a docs-only scaffold.

Steps 2–5 only read, so a dirty working tree is fine for the report. Note it, and require a clean
tree — or an explicit go-ahead — before step 6 writes anything: every application should be
reviewable as a git diff on its own.

### 3. Establish provenance

**With `scaffold.json`:** verify, don't trust. Recompute each listed file's sha256 and compare to
the recorded one: match with `amended: false` is untouched; `amended: true` is amended; mismatch is
edited. A listed file missing from disk was deleted by the project — report it as that, and leave
the entry alone. Never "correct" the file to make drift disappear; it records history, not
configuration.

A hand-backfilled `scaffold.json` — one written after the fact rather than by `codefall-scaffold` — is only
as safe as its `amended` flags. The hashes are re-verified right here, but `amended` is taken as
recorded, and a flag wrongly set to `false` marks a customized file as safe to supersede. Nothing
downstream can catch that lie; a backfill unsure about a file should say `true`, whose worst case
is a diff shown instead of an update offered.

**Without `scaffold.json`:** classify per `reference/pre-provenance.md`. What cannot be classified
is unverifiable, and unverifiable is edited.

### 4. Compare against the current templates

First fix the **applicable set** — which templates this project should have at all. Profiles and
decisions come from `scaffold.json` when present, from which profile's ADRs are on disk otherwise,
and from step 2's detection for adoption. Apply `codefall-scaffold`'s gates, not your own: a surface that
can never be split skips ADR-BASE-03, a backend-only project has no use for ADR-TS-02. If a gate
genuinely can't be answered from the project, ask — once, batched with anything else step 5 needs.

Then classify every difference:

- **Missing** — an applicable template with no counterpart in the project. New since the project's
  baseline, or all of them under adoption.
- **Revised** — the counterpart exists and the current template's content differs from what the
  project has.
- **Renamed** — the counterpart exists under a historical identifier. `lineage.md` is the record:
  a project on 0.2.x has `ADR-003-dependency-injection.md`, and that *is* `ADR-TS-01` — **recognise
  it as a rename, never as a deletion plus an addition.** Renames compose with revisions; report
  both facts about the one file. A rename also implies stale references wherever other docs cite
  the old identifier — find them with Grep, list them with the item.
- **Retired** — a project doc whose template no longer ships. The decision didn't evaporate because
  the extension moved on; report it, leave the file alone.
- **Skeleton drift** — the profile's `AGENTS.md.skeleton` changed. An `AGENTS.md` is filled in per
  project at scaffold time, so it is *always* effectively amended: skeleton changes are reported as
  advisory, never applied wholesale.
- **Provenance** — `scaffold.json` itself is missing or stale. Offer to write it; that is how a
  pre-provenance project stops needing `reference/pre-provenance.md` next time.

### 5. Report — and stop

The report is this verb's default output, and for many runs its only one. For each item: what it is,
its provenance state, what changed and why — one line each, with the diff itself for anything
amended or edited, because there the diff is the deliverable — and whether it can be taken
automatically or is the user's to merge. Say what taking means: a revision lands as a superseding
ADR, never as a rewrite of the one the project ratified.

When the extension's changelog is reachable (a repo checkout, or GitHub — the installed extension subtree
does not carry it), use it to say *why* things changed, grouped by release since the project's
baseline. When it isn't, say the report comes from template comparison alone. Either way the
comparison, not the changelog, is the source of truth for *what* changed.

Then **stop**. Nothing is applied unrequested — a report that quietly rewrote files defeats the
verb. The user takes items by naming them, "everything safe" included; present the choice with one
question, items grouped by whether they can be taken automatically.

### 6. Apply what the user takes — one item at a time

Before the first item, take the branch step of `../../../.codefall/shared/landing.md`: standing
on the default branch, `git switch -c upgrade/<YYYY-MM-DD>`; on another branch, ask once which to use.

Only requested items, and only missing or untouched ones are ever taken. **This verb never rewrites
a ratified ADR.** A taken revision lands as a new ADR that supersedes the old one, which stays on
the record; the only in-place edit this verb ever makes to an existing ADR is flipping its Status
line to `Superseded by <id> — <date>`, which is the edit the discipline prescribes. Git history is
not the record here — the documents are.

- **Missing** — instantiate from the current template, exactly as `codefall-scaffold` step 4 emits it:
  stamp today's real date (`date +%F`, not memory) on the Status line, keep the flat `docs/adrs/`
  layout.
- **Untouched, revised** — write the current template as a **new, superseding ADR** stamped with
  today's date, and flip the old one's Status. When the revision arrived together with a rename,
  the new file simply takes the current template's name — `ADR-TS-01-dependency-injection.md`
  superseding `ADR-003-dependency-injection.md`, which keeps its old name and its place. When the
  name hasn't changed, suffix an edition so both can exist:
  `ADR-BASE-02.2-package-by-component.md`, identifier `ADR-BASE-02.2`, editions counted within
  this project. Cross-link the pair in Related — the successor names what it supersedes.
- **Renamed, unrevised** — when only the identity moved and the decision didn't, a move is
  addressing, not substance: `git mv` where the tree is git-managed, update the identifiers the
  file itself carries, and say so in the report. In practice renames almost always arrive with a
  revision, and then the supersession above already produces the new name.
- Either way a rename strands references. Fix ones in *untouched* inherited docs as part of the
  item. Stale references in amended or edited docs, in `AGENTS.md`, or in the project's own ADRs
  are theirs: list each one, offer the targeted edit, and make it only on a yes.
- **Amended / edited** — never superseded by taking an update, even one that renames the file
  around them. The narrow, user-named exception is in [Provenance states](#provenance-states).
- **`AGENTS.md`** — targeted section edits on request only, never a wholesale replacement.
- **`.codefall/scaffold.json`** — after every applied item, update its entry: `id`, `file`, and a
  `sha256` computed with `shasum -a 256` on the file as written — never invented. A supersession
  touches two files: add an entry for the successor, which is what future runs compare against,
  and recompute the superseded file's hash after its Status flip. Add
  `"lastUpgrade": { "pluginVersion": "<version or unknown>", "date": "<date>" }` at the top level.
  A file carrying the older key `lastGraft` is read, never refused; the write replaces that key with
  `lastUpgrade`, and the report says so. Take the version from `.codefall/manifest.json` — do not
  guess it. Where the file didn't exist, write it fresh: this extension version, today's date, the
  profiles and decisions as established, and an `amended` per ADR — `false` only for files that now
  hash-match a current template, `true` for anything kept that differs.
- **`docs/decision-log.md`** — one line under `Locked`: *brought documents current with codefall
  `<version>` on `<date>` — took ADR-BASE-03, ADR-TS-01; left ADR-TS-02 (amended)*. Humans read
  this; `scaffold.json` stays authoritative.

A partial run is a normal outcome, not a failure state. Per-file hashes carry the truth about
what is current, so the next run needs no memory of this one.

### 7. Verify

Whatever was written must actually hold together. Every link in every touched doc resolves.
`scaffold.json` parses, and every hash in it matches the file on disk — recompute, don't assume.
No doc still cites a renamed identifier except the ones the user chose to leave — Grep for the old
IDs and check the hits against that list. Every Status line that says Superseded names an ADR that
exists, and every successor's Related names what it superseded. Then show the git diff summary:
the documents' changes should be reviewable as one coherent change.

### 8. Report what took

- What `codefall upgrade` did, when it ran, and the files it changed, left uncommitted.
- What was **taken**, file by file.
- What was **left**, and why — amended, edited, retired, or declined.
- Stale references the user chose to keep, so they aren't rediscovered as a surprise.
- That `scaffold.json`'s `lastGraft` key was renamed to `lastUpgrade`, when this run renamed it.
- The landing, per `../../../.codefall/shared/landing.md`: what took, committed by path on the
  branch, and the push and pull request offered. The merge is the user's. A run where nothing took
  has nothing to land.
- What the user still owes the project: the hand-merges they said they'd do, and — after an
  adoption — the boundary-enforcement obligation, named exactly as `codefall-scaffold` names it after a
  docs-only run.
- **Last, what the user does next**: commit the install's changes when `codefall upgrade` ran,
  merge the pull request, then what is owed above. Nothing else.

## Rules

- **Only ever invoked explicitly.** Never suggested, never fired from a passing remark, never
  chained into from another skill.
- **`codefall upgrade` runs only on a yes**, never with `--yes`, and its breaking changes reach the
  user before they answer its question.
- **Documents, not code.** Moving code between architectures or extracting a service is `migrate`'s
  work; say so and stop.
- **Report, then stop.** Nothing is applied unrequested, and what is applied is one item at a time.
- **Only untouched is taken mechanically.** Amended and edited get the diff; the superseding ADR
  that carries their amendment forward is the user's to write.
- **A ratified ADR is never rewritten.** A revision is a new, superseding ADR; the only in-place
  edit is the Status line.
- **A rename is recognised as a rename**, never as a deletion plus an addition. `lineage.md` is the
  record.
- **Never reconstruct an old template from memory, and never invent a hash.**
- **`scaffold.json` records history.** It is verified, never corrected to make drift disappear.
- **A clean tree before anything is written.** Every application is reviewable as a git diff on
  its own.
- **`AGENTS.md` gets targeted edits on request only**, never a wholesale replacement.
- **This verb writes nothing outside the target project.** The extension's own files — templates,
  `lineage.md`, this skill — change through pull requests, not through a run of this verb.
