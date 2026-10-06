# ADR-015: Skill Prefix

## Status

Accepted — 2026-10-06

## Context

Every skill codefall installs is named `codefall-<verb>`: the directory under `.claude/skills/` or
`.agents/skills/`, the `name:` in its frontmatter, and the slash command a person types,
`/codefall-implement DESIGN-003`. The name is long for something typed many times a day, and a
project has asked for a shorter one. Two shorter forms are in use informally, `cf-` and `cfall-`.

The name is not one string in one place. Counted on 2026-10-06, the installed tree names a skill
about 540 times: inline in prose (`` `codefall-design` ``), as a slash command
(`/codefall-implement`), and as a cross-skill relative path (`../codefall-scaffold/templates/…`,
which `codefall-upgrade` follows to the templates it compares against). The four sections `init`
splices into a project's `AGENTS.md` and the testing skeleton name skills the same way, the files
under `shared/` name them, and the session notice's messages say "run /codefall-refresh". The name
`codefall-` also appears where it is not a skill: the install's own directory `.codefall/`, the
`codefall` binary, the two guard scripts `codefall-block-merge-to-main.sh` and
`codefall-session-notice.sh`, the Antigravity hook key `codefall-merge-guard`, and the landing
Action `codefall-land-documents.yml`. Those are not what a project is asking to shorten.

Three places could do the renaming.

**(a) In the source tree.** Rename the skills in this repository to the short form, or keep three
copies of the tree, one per prefix. The first gives every project the same new name and settles
nothing for a project that wants a different one; the second triples 64 files and every edit to
them.

**(b) At run time.** A skill reads the prefix from `.codefall/settings.json` and composes names as
it goes. The directory names are what a harness loads, and no harness composes them, so the
directories would still have to be renamed by something else. Every one of the 540 references would
become an instruction to compose a name, including the relative paths a skill follows into another
skill, and a slash command in prose cannot be composed at all.

**(c) At install time.** The source keeps one spelling, `codefall-<verb>`, and the installer rewrites
it to the project's prefix while copying: directory names and file contents alike, under one rule.
With the default prefix the copy is the same bytes it is today. The rename is one function in one
place, applied to a tree whose authors never think about it, and the manifest records which prefix
an install used so the next upgrade knows whether to rewrite again.

Option (c) is chosen. What decides it is where the knowledge lives: the installer already knows
every skill directory the tree ships and already records every file it writes, so it can derive the
set of names to rewrite from the tree and can tell a renamed skill from a removed one on the next
run. Nothing else in the system knows both.

The second decision is whether the prefix is free text. It is not. A closed set of three,
`codefall`, `cf`, and `cfall`, is what was asked for, and a closed set is what lets the installer
recognise a skill directory under any prefix a project could have used, which is what makes a change
of prefix a rename in the upgrade report rather than thirteen removals and thirteen additions, and
what lets a project move back. Free text would also admit a prefix that is itself a word in the tree,
and the rewrite would then have to reason about its own output.

## Decision

### The setting

`.codefall/settings.json` gains an optional top-level field, `skillPrefix`, whose value is one of
`codefall`, `cf`, or `cfall`. Absent means `codefall`, which is what every project has today, so
adding the field is not a breaking change. The schema and the validator both close the set.

### The rename happens at install time

The source tree keeps every skill named `codefall-<verb>`. While copying, the installer replaces each
`codefall-<verb>` with `<prefix>-<verb>`, in directory names and in file contents, under the skills
directories, `.codefall/shared/`, and `.codefall/hooks/shared/`, and in the `AGENTS.md` sections and
the testing skeleton it writes from the tree.

The set of verbs is the set of skill directories in the embedded tree, read at run time, never a
list in code: a skill added to the tree is covered by the rewrite without anyone remembering it. A
match is a whole name. `codefall-design` is rewritten; `codefall-designs`, `my-codefall-design`, and
`codefall-lineage` are not, because the character on either side of a match may not be a letter, a
digit, an underscore, or a hyphen. A name that is not a skill is not rewritten whatever it looks
like: `.codefall/`, the `codefall` binary, `codefall-block-merge-to-main.sh`,
`codefall-session-notice.sh` (the file name; the messages inside it are rewritten),
`codefall-land-documents.yml`, and the `codefall-merge-guard` hook key all stay as they are.

With the prefix `codefall` the install is byte for byte what it was before this decision.

### A fresh project chooses at init

`codefall init` takes `--skill-prefix <codefall|cf|cfall>` and, in its survey, asks the question with
`codefall` as the offered answer. A scripted run that passes no flag takes `codefall`: the question
has a default in a way the tracker and the testing root do not, because it is the name every project
had before the field existed. `codefall create` passes the flag through to init. Init writes the value
into the settings explicitly, as it writes the default `agents` entry, so a file that states the
setting shows there is something to change.

### An existing project changes the setting and runs upgrade

A person sets `skillPrefix` with `codefall config skill-prefix <value>`, in the editor `codefall
config` opens, or by hand, and runs `codefall upgrade`.

The manifest gains a top-level field, `skillPrefix`, recording the prefix the last finished run
installed with. Absent means `codefall`, which is accurate for every manifest written before the
field existed. Upgrade reads both: a settings prefix that differs from the manifest's is work to do,
however current the version, in the way a harness still recorded under an old spelling is work to
do, and the run is not asked about a version that does not move.

The run installs the skills under the new names and the existing cleanup step removes the old ones,
because the manifest lists them and this run did not write them (ADR-010). The step's report names
the change as a rename, `codefall-design/ (renamed to cf-design)`, because the rename lookup
recognises a skill directory under any of the three prefixes.

Project-authored text is never rewritten: the project's own `AGENTS.md` outside codefall's sections,
its documents, its beads, and `.codefall/reviews/`. Where the prefix moved, the cleanup step's report
names the tracked files that still mention a skill under the former prefix, so the person can decide
what to do about each, and names any `.codefall/skills/<former>-<verb>/CUSTOMIZE.md`, which the
skill will no longer find under that name.

### The authoring rule

A skill refers to another skill only as `codefall-<verb>`, written whole: in prose, in a slash
command, and in a relative path. Never split across a line or a variable, never under another
prefix, never abbreviated to the verb where the full name is meant. The install rewrite finds what
is written this way and nothing else. The rule is stated in `extensions/skills/AGENTS.md`, and the
install test under the prefix `cf` fails on any `codefall-<verb>` left in an installed file.

### What this amends in ADR-006

ADR-006 says the skills are copied into each chosen harness's skills directory as they are in the
tree. Under this decision they are copied with their names and contents rewritten to the project's
prefix, and the shared files and hook scripts the same, which is a copy only when the prefix is the
default. The layout — what lives in `.codefall/`, what is copied per skills directory, the one path
a skill uses for a shared file — is unchanged.

## Consequences

- **Not a breaking change.** A project that sets nothing gets the install it has today, byte for
  byte, and a manifest with one more field.
- **A skill's name is now something the installer parses.** The authoring rule is what keeps the
  rewrite complete, and the install test is what checks it: install under `cf`, then fail on any
  `codefall-<verb>` left behind and on any cross-skill relative path that does not resolve. A
  reference written in a form the rule forbids passes the health check in the source tree and fails
  that test.
- **Two records, reconciled by upgrade.** The settings say what the project wants, the manifest says
  what the last run installed, and a difference is work. This is the shape the harness renames
  already have, and `doctor` could read both and warn, as it does for a harness spelling; it does not
  yet.
- **The project's own text keeps the old names.** A design document that says "run
  `/codefall-implement`" still says so after the prefix moves. Upgrade reports the files, once, and
  changes none of them, because every one of them is the project's.
- **A customization file follows the skill's name.** `.codefall/skills/codefall-specify/CUSTOMIZE.md`
  is read by a skill installed as `codefall-specify`; installed as `cf-specify`, the skill reads
  `.codefall/skills/cf-specify/CUSTOMIZE.md`. A project that changes its prefix moves the directory
  by hand, and upgrade names each one it finds under the former prefix.
- **Codefall's own documentation says `codefall-<verb>`.** The README, the ADRs, and the changelog
  name skills under the default prefix, and a project on `cf` reads them with a translation in mind.
  The installed `workflow.md` and the `AGENTS.md` sections are rewritten, so what a session reads is
  consistent with what it can run.
- **The guard scripts and the hook key keep their prefix on purpose.** `extensions/AGENTS.md` already
  says why: the install recognises its own registration in a file it shares with the project's hooks
  by the script a command names. A rewritten script name would leave every project's old registration
  running for ever.
- **Three prefixes, all codefall's.** Moving between them is reversible, because the rename lookup
  knows all three. A fourth is one entry in the closed set, the schema, and this record.

## Related

- ADR-006, *Install Layout* — what is copied where; this record amends how the copy is made.
- ADR-010, *Upgrade* — the manifest's file lists and the cleanup step this decision relies on, and
  the rename reporting it extends.
- `extensions/skills/AGENTS.md` — the authoring rule a skill follows when it names another skill.
- `extensions/AGENTS.md`, *Hooks* — why the guard scripts keep the `codefall-` prefix.
- `cli/schemas/settings.schema.json` — the closed set, published.
