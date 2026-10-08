# codefall-scaffold — where it came from

Why `SKILL.md` looks the way it does. None of this is instruction — the skill is the instruction.
This exists so nobody re-adds something that was removed on purpose.

## A vision comes first

Guessing a default in the absence of information is this skill's worst failure mode, and it is why
step 1 will not run without offering the vision step. With a vision, most of the questions are
answered before the session starts, so there are fewer of them and the answers are better. The user
can still decline; the override is recorded in the decision log because it is a decision made
without enough information, which is exactly what `Open` is for.

## Shape decides the architecture, not names

Not having named the domains is the normal starting state of a greenfield project, and it is not
evidence of a single cohesive domain. Reading "we haven't named them" as one of ADR-BASE-02's
conditions for ports-and-adapters would hand p&a to every project this skill exists to scaffold.
The test is whether the thing decomposes into capabilities that could plausibly be owned, deployed,
or extracted separately — the same question ADR-BASE-03 keeps answerable later. Asking it once at
the start is cheaper than discovering the answer during an extraction. Names are colour: they seed
folder names when offered and change nothing else.

## A surface is where domain logic lives

A language that only implements gateways is not a surface; it is the outer ring of a surface that
already has a profile. A Tauri `src-tauri/` of stock boilerplate plus thin commands wrapping OS APIs
is the `infrastructure/` ring of the TypeScript surface, per ADR-BASE-01, and the app is plain
`typescript-react`. Rust being present does not make it a Rust surface, any more than a Postgres
driver makes SQL one. The same reasoning covers an Electron main process, a React Native native
module, and a thin native wrapper around a web view. Two processes are not two domains, and one
process is never two.

## Planned profiles are exits, not options

An option annotated "this is unsupported, I would have to stop" is not a choice — it is a trap with
extra steps. Offering a planned profile and then refusing spends the user's choice on nothing. The
"None of these" option is the deliberate exit: the user picks it knowing what it means, rather than
discovering it after choosing something that looked available. And an improvised profile is worse
than none, because it produces ADRs marked Accepted that nobody actually decided.

## Recommend only what an ADR supports

A "(Recommended)" label with no ADR behind it invents an opinion this project does not hold, and the
user cannot tell the difference between a considered default and one made up on the spot. Codefall
has no opinion on repo layout because no ADR covers it.

## `amended` is the field that earns the provenance file

Only the scaffold step can record it accurately, because the scaffold step is doing the amending.
Reconstructing it later would mean diffing against the exact template version that emitted the
file, which means keeping every historical version reachable. `sha256` covers the other case, an
ADR hand-edited months after scaffolding. Together they separate *untouched*, *amended during the
interview*, and *edited since*, and only the first is ever safe for a later tool to update
automatically. A plausible-looking wrong hash is worse than none: it marks an untouched file as
edited, or an edited one as pristine.

## Boundary enforcement is per-profile

How much tooling the five boundary rules take depends on the language's visibility model.
TypeScript needs a linter because it has no `package-private`. Kotlin `internal`, Rust
`pub(crate)`, and Go's `internal/` packages do part of the job in the compiler — part, not all: Go
rejects import cycles but not outward ones, so the facade rules come free and the layer rule still
needs a linter. That is why the boundary-enforcement ADR lives in the profile, and why a language
with real visibility is not assumed to need nothing.

## The doc workflow this seeds

Discuss → decision-log → ADR → scoped `AGENTS.md` → code. The decision-log holds detail while a
decision is moving; the ADR holds the settled decision and its why; `AGENTS.md` holds the terse
operative rules and links back. `codefall-plan` picks up from there.
