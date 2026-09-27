# ADR-010: Upgrade

## Status

Accepted — 2026-09-27

## Context

`codefall init` did two jobs under one name. The first run made a directory ready for codefall: it
surveyed for the tracker, the harnesses, and the testing root, wrote `.codefall/settings.json`,
installed the extension, initialised Beads, registered the hooks, wrote the marked sections into
`AGENTS.md`, made the testing tree, and recorded the run in `.codefall/manifest.json`. Every run
after that was an upgrade wearing init's name: it read the harnesses and the testing root back from
the settings, compared the manifest's versions with the binary's, said "already up to date" or asked
whether to move the version, reinstalled what the binary ships, rewrote a harness spelling that had
changed, and wrote the manifest again. `upgrade` existed only as an alias of `init`, and `--force`
existed to make a rerun behave like a first run over settings that were already there.

Three defects followed from the one name, and they are the reason for this record.

**The command could not refuse.** A run had to guess which job it was doing from what it found on
disk, so nothing stopped a person from re-answering the survey over a project a team depended on, and
nothing stopped a script written for one job from doing the other.

**A renamed or retired skill stayed installed for ever.** The extension step copied what the binary
ships and deleted nothing, so a project that upgraded across the rename of `conceptualize` to
`envision` kept both, and a harness loaded both. The manifest recorded every file each run wrote,
which is exactly the record deletion needs, and no command read it for that.

**Nothing warned about a breaking change.** The repository defines one as a change a project has to
answer by hand after upgrading, marks it with `!` and a `BREAKING CHANGE:` footer, and release-please
writes each into a `⚠ BREAKING CHANGES` section of `CHANGELOG.md`. A project moving from `0.16.0` to
`0.20.0` crossed several of them and was told nothing, because the rerun compared two version strings
and asked "upgrade to this version?" with no account of what that meant.

Beside the binary's install sits a second kind of upgrade the binary cannot do: the documents
`codefall-scaffold` wrote into the project, the inherited ADRs, the `AGENTS.md` skeletons, and
`.codefall/scaffold.json`. Those are the project's own once written, so bringing them current is a
skill's job, with the consent machinery a skill has and a binary does not. That skill is
`codefall-graft`, and its name says nothing about upgrading.

Four alternatives were weighed.

**Keep `init` as the rerun.** The smallest change, and the name says the wrong thing: init is what
happens once, and a command that runs once can refuse a second run instead of guessing what the
second run is for. Every defect above is downstream of the guess.

**A hand-maintained table of breaking changes in Go.** One entry per version, printed on upgrade.
It is a second source that drifts from the changelog the moment someone forgets it, and the
changelog already exists, is already generated from the commits that carry the `!`, and is already
the thing a person reads to answer the same question.

**Delete nothing on upgrade.** What the rerun does today. Renamed and retired skills stay installed
in every project until a person notices, which is the defect being fixed, and the manifest already
records what to delete.

**Fold the document upgrade into the binary.** One command for everything. The binary would then
edit a project's ratified ADRs and amended `AGENTS.md` files, which the repository's own rules forbid
without showing the difference and asking, item by item. That is a skill's shape.

## Decision

### `codefall init` runs once

It refuses when `.codefall/manifest.json` exists, in one sentence that names `codefall upgrade`.
`codefall create` still runs init in the directory it makes. `--force` and the `upgrade` alias are
removed: on a project that has never been initialised there is nothing to force, and on one that has,
the work is upgrade's. Changing an answer the settings record is a hand edit until a command exists
for it.

A project set up before the manifest existed has settings and no manifest. Init runs on it one more
time, asks nothing because the answers are on record, and finishes by writing the manifest that sends
every later run to upgrade.

### `codefall upgrade` is the one command that changes an installed project

It reads the settings and the manifest, reinstalls the skills, the shared files, and the hooks for
the harnesses the settings record, replaces the marked sections, rewrites former harness spellings,
and writes the manifest. It refuses when there is no manifest, naming `codefall init`. It reports
"already up to date" where the rerun did, asks before moving the installed version, and `--yes`
answers for a script. `--harness` adds a harness the project did not choose at init; `--location`
says which directory holds the install when run below the repository root.

Every remedy in `codefall doctor` and in the skills that said to rerun `codefall init` now says
`codefall upgrade`, and upgrade itself sends a project with no manifest on to init. A remedy for a
project that was never set up still names init.

### One use case, two commands, one component

The rerun shares every step with the first run: settings, extension, Beads, hooks, sections,
testing tree, ignore entries, manifest. What differs is how the request is built, from a survey and
flags or from what the settings and manifest already record, and that is presentation. So the two
commands live in `initcmd`, share its use case and its gateways, and the facade exports `Commands`
for the root to mount both, as the CLI rules already provide for a component with several commands.
A second component would have duplicated the steps or moved them to a shared module, and the steps
are not pure: they read files, run `bd`, and copy the embedded tree.

### Upgrade removes what the previous install wrote and this one did not

Decided here, landed in the next pull request. The manifest lists the files each harness's install
and the shared install wrote. After the new install, a path in the previous list that the new list
does not carry is a file the binary no longer ships under that name: upgrade deletes it and reports
each one. A rename is reported as a rename, not a deletion and an addition, from a table of former
skill names embedded in the binary, the same record `codefall-graft`'s lineage file keeps for
templates. A manifest that is missing, hand-edited so its lists do not decode, or written before file
lists existed gives upgrade nothing to compare, so it deletes nothing and says so.

### Upgrade warns about breaking changes before it applies anything

Decided here, landed in the next pull request. The binary embeds the repository's `CHANGELOG.md`,
which release-please writes from the commits that landed, and upgrade prints the entries of every
`⚠ BREAKING CHANGES` section between the manifest's recorded version and the binary's before it
changes a file. In an interactive run it then asks whether to continue; under `--yes` it continues.
There is no second list to maintain: a breaking change reaches the warning by the footer that
already puts it in the changelog.

### `codefall-graft` becomes `codefall-upgrade`

Decided here, landed in the pull request after that. The skill keeps its procedure for the
documents the binary does not own and gains one opening step: when the manifest's version is behind
the binary's, it offers to run `codefall upgrade` first, so a person has one verb to reach for and
the binary's part happens before the documents' part. The rename is itself a renamed skill, and the
removal above is what keeps `codefall-graft` from staying installed beside it.

## Consequences

- **Every project that rerun `init` has to learn `upgrade`.** Scripts, notes, and habits that said
  "rerun `codefall init`" change, which is why this is a breaking change with the footer, and why the
  refusal names the command to run instead of failing in the abstract.
- **A second run can no longer redo the survey.** `--force` is gone, and a project that wants a
  different tracker or a different testing root edits `.codefall/settings.json` by hand until a
  command owns that.
- **The manifest becomes the record deletion depends on.** Its file lists were written for doctor's
  install check and now decide what upgrade removes. A manifest that is missing or does not decode
  means upgrade deletes nothing and says so, which leaves the stale files where they were and is the
  safe failure.
- **The binary embeds the changelog.** The warning costs the size of that file and nothing else, and
  it is only as good as the commit messages: a breaking change landed without the footer does not
  reach it. The repository rules already require the footer.
- **Two verbs share a word.** `codefall upgrade` is the binary's, `/codefall-upgrade` is the skill's,
  and the skill offers the binary's first. A person who runs the skill gets both; a person who runs
  the command gets the install alone, and doctor's remedies name the command.
- **`doctor`'s remedies say `upgrade` for a project that may have no manifest.** Upgrade then names
  init, so the person is one command further from the fix than a remedy that knew which to name.
  Doctor could read the manifest to choose, and does not yet.

## Related

- ADR-006, *Install Layout* — where the skills, shared files, and hooks land, which is what upgrade
  reinstalls and what the manifest's file lists record.
- ADR-005, *Local Environment Scripts* — `codefall-refresh` brings the environment current with the
  checkout; upgrade brings the install current with the binary. Neither does the other's job.
- `extensions/skills/codefall-upgrade/lineage.md` — the record of former template and skill names that
  the rename reporting follows.
- Repository `AGENTS.md`, *Workflow* — the definition of a breaking change and the footer release-please
  reads, which is where the upgrade warning's text comes from.
- `docs/decision-log.md`, *Upgrade is its own command, 2026-09-27* — the alternatives seen and not
  taken, in brief.
