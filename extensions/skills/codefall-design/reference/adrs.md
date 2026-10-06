# ADRs

Read at step 4, when deciding whether this design writes one, and at step 7, when it is shown.

A separate artifact, `docs/adrs/ADR-NNN-title.md`, in the Nygard shape — Status, Context, Decision,
Consequences, Related — written from `../../codefall-scaffold/templates/adrs/_TEMPLATE.md`.

- **The number continues the project's own sequence:** the highest bare `ADR-NNN` in `docs/adrs/`
  plus one, starting at `ADR-001`. The prefixed sequences — `ADR-BASE-NN`, `ADR-<PREFIX>-NN` — are
  inherited stance and are never continued here.
- **The trigger** is the design's Alternatives Considered holding a choice that is hard to reverse
  or that other components will build on: a new dependency; a schema or protocol decision other
  components will be written against; a rejected alternative that cost real analysis. Most designs
  need none.
- **One home for the rationale.** The design names the choice and points at the ADR from the `adr`
  label on `Related`; the ADR carries the reasoning.
- **A ratified ADR is never rewritten.** A revision is a new, superseding ADR; the only in-place
  edit is the Status line, to `Superseded by <id> — <date>`.
- Draft the ADR and show it before writing. It ships `Accepted` with a real date once the user
  confirms it.
- **A decision the person set aside and then asked the run to settle with a default** gets its ADR
  written from that default, with the Context saying the person could not judge it and the run
  chose the simplest option that fits the project's stance. The report names the ADR in a sentence
  so an engineer knows to read it.
