# Project customizations

Shared procedure. Every verb follows it except `codefall-scaffold`, which runs in a directory that
has no `.codefall/skills/` yet, and `codefall-upgrade`, which compares a project's documents against
the extension's templates, where a project procedure would work against the comparison. Factored out
of the individual skills once a third one needed it, so that the rule about what a customization may
and may not do is stated once.

If `.codefall/skills/<verb>/CUSTOMIZE.md` exists in the user's project, read it before step 1, say
you loaded it, and follow it for this run. `<verb>` is the name of the skill that is running —
`codefall-specify` reads `.codefall/skills/codefall-specify/CUSTOMIZE.md` and nothing else.

It carries procedure this extension cannot know — a system to consult, a question this project always
asks, a section every document carries, a step that runs after writing.

One customization is shared rather than per verb: `.codefall/skills/shared/CUSTOMIZE.md`, read by
the landing procedure beside this file, for how this repository lands the documents every verb
writes the same way.

It extends the skill and never relaxes it: the skill's **Rules** hold regardless. A customization
that would suspend one is asking for a different skill — say so and stop.
