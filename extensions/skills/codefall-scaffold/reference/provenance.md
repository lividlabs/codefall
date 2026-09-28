# Provenance: `.codefall/scaffold.json`

Written every time, at every depth, and committed. Machine-owned: nobody hand-edits it, and editing
it lies about history rather than changing a setting. Read at step 4.

```json
{
  "pluginVersion": "0.4.0",
  "scaffoldedAt": "2026-08-18",
  "profiles": ["typescript-react"],
  "decisions": {
    "shape": "several-capabilities",
    "architecture": "package-by-component",
    "topology": { "typescript-react": "nextjs" },
    "depth": "docs-only"
  },
  "adrs": [
    { "id": "ADR-BASE-01", "file": "docs/adrs/ADR-BASE-01-clean-architecture.md",
      "amended": false, "sha256": "…" }
  ]
}
```

## Fields

- `pluginVersion` — read from release-please's PR title if there is one, else write `unknown`. Do not
  guess it.
- `shape` — `one-cohesive-domain`, `several-capabilities`, or `unclear`.
- `architecture` — `package-by-component` or `ports-and-adapters`.
- `topology` — one entry per React surface; omitted where a profile has no topologies.
- `depth` — `docs-only`, `project-files`, or `runnable-skeleton`.
- `adrs` — one entry per ADR emitted, with `id`, `file` relative to the project root, `amended`, and
  `sha256`.
- `declined` — written by `codefall-upgrade`, never by this skill: one entry per missing template
  the user declined, with `id` and `date`, so later runs do not offer it again. A file that holds
  `declined` and no `adrs` belongs to a project that was never scaffolded.

## `amended` and `sha256`

`amended` is true only for an ADR the interview changed, and only this step can record it. `sha256`
catches an ADR hand-edited after scaffolding. Together they separate *untouched*, *amended during
the interview*, and *edited since*; only the first is safe for a later tool to update automatically.

**Never invent a hash.** Compute it with `shasum -a 256 <file>` after the file is written.

## The decision log line

`docs/decision-log.md` opens `Locked` with one line — "scaffolded with codefall `<version>` on
`<date>`" — so provenance is visible to someone reading docs. `scaffold.json` stays authoritative;
that line is for humans.
