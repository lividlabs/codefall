# Legacy documents

A verb that was renamed may have left documents under its earlier directory: `codefall-design`
wrote `docs/designs/DESIGN-NNN-slug.md` before it became `codefall-plan` and its documents moved to
`docs/plans/PLAN-NNN-slug.md` (`lineage.md` has the record). Those documents, their identifiers,
and their beads stay as they are: a bead identifier cannot be renamed, so a moved file would split a
plan's identity between the document and its graph. What this verb offers instead is a marked
section in a verb's `.codefall/skills/<verb>/CUSTOMIZE.md` that tells the renamed verb where to
look. Read at step 4 when `docs/designs/` exists.

## The item

`templates/customize/legacy-designs.md` holds one section per verb that needs the bridge, each
between its own `<!-- codefall:legacy-designs -->` markers and headed by the file it goes in. The
condition is the same for every one of them:

- `docs/designs/` exists in the project, and
- that verb's `CUSTOMIZE.md` carries no `<!-- codefall:legacy-designs -->` marker.

One item per verb whose condition holds, reported at step 5 as *legacy documents* with the verb
named, and takeable automatically: nothing it writes touches a document the project ratified. A
file that already carries the markers is present, whatever sits between them; the person may have
edited the section, and it is theirs.

## Applying it

- The file exists: append the section after its last line, markers and all, exactly as the template
  gives it for that verb. Nothing outside the markers is touched.
- The file does not exist: write it with the section as its whole content. `customizations.md` in
  `../../../../.codefall/shared/` says what else a `CUSTOMIZE.md` may carry; this verb adds nothing
  beyond the section.
- Never rewrite a section already present, and never remove one: the project removes it when
  `docs/designs/` holds only archived plans whose beads are closed, as the section itself says.

The section's contents are the template's, verbatim. A project whose old documents need something
the template does not say gets it from the person, in the same file, outside the markers.
