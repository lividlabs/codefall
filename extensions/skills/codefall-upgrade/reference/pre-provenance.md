# Projects scaffolded before provenance

Everything `codefall-upgrade` does differently when a project has no `.codefall/scaffold.json`.
Provenance has been written since codefall 0.4.0, and every project scaffolded before it moves out
of this file the first time the verb writes `scaffold.json` for it. When no such project remains,
delete this file and the places `SKILL.md` points at it.

## Contents

- Recognising one
- Where the old templates come from
- Classifying without provenance

## Recognising one

Step 2's inventory finds no `scaffold.json`, but `docs/adrs/` holds codefall-lineage files — under
current identifiers, or historical ones per `../lineage.md`. The decision log's *scaffolded with
codefall `<version>`* line, when present, names the baseline version.

## Where the old templates come from

With `scaffold.json` present the recorded hashes classify every file without any historical
template. Without it, the old template has to be found. Take the first of these that works:

1. **The local plugin cache** — `~/.claude/plugins/cache/<marketplace>/codefall/<version>/`. Exact
   snapshots, offline, but only of versions this machine actually installed.
2. **Git history** — if the extension root, two levels up from the skill's directory, sits inside a
   clone that can reach the release tag, `git show <tag>:<path>` works. Installed marketplace
   clones are usually **shallow** with few or no tags, so try `git fetch --depth=1 origin tag <tag>`
   before concluding the tag is missing. Tags come in two forms and old paths differ from current
   ones — `../lineage.md` records both.
3. **GitHub** — `gh api repos/lividlabs/codefall-plugin/contents/<historical-path>?ref=<tag>`,
   or the raw URL. Needs network.
4. **Nowhere** — then the file is **unverifiable**. Say so and treat it as edited.

**Never reconstruct an old template from memory of what it probably said.** A plausible-looking
reconstruction marks edited files untouched and untouched files edited. The list above or nothing.

## Classifying without provenance

Identify each codefall-lineage doc via `../lineage.md`, find the template that emitted it as above,
and compare. Normalize the one thing `codefall-scaffold` changes on emission: the date on the
`## Status` line (`Accepted — <date>` became `Accepted — 2026-08-12`). Everything else compares
verbatim. Equal after that means untouched; different means edited — without provenance you cannot
distinguish *amended at the interview* from *edited since*, and edited is the safe reading.

When the decision log does not name the baseline version, there are only a handful of releases a
given filename can come from — `../lineage.md` names them; compare against each. What cannot be
classified is unverifiable, and unverifiable is edited. Do not guess a version, and never invent a
hash.

Step 4's **Provenance** item — offer to write `scaffold.json` — is how the project stops needing
this file next time.
