# ADR-011: Personas

## Status

Accepted — 2026-09-27

## Context

Every verb assumes the person at the keyboard can answer what it asks. `design`'s rule for a
disagreement is to raise it once and then defer to the user, with a cap of two rounds, after which
the point goes into the document as a stated risk. `specify` records what the user did not settle as
an open question. `implement` escalates a worker's second failure to the human with three choices.
Each of those is right when the person is an engineer, and each has the wrong target when the person
is not: a technical choice deferred to someone who cannot evaluate it produces a confirmed design
that nobody competent confirmed, and a risk line that reads as accepted when it was only not
understood.

Codefall is meant to be run by product managers as well as engineers. A product manager can carry a
vision and a spec through their interviews, can confirm a design's tier and its task cut against the
spec, and cannot settle a choice between a signed payload and a session lookup. The verbs did not
know which person they were talking to, so they could not change where they sent that choice.

The per-user file, `.codefall/user.json`, exists to hold that fact
(`docs/decision-log.md`, *A per-user `.codefall/user.json`, carrying the persona, 2026-09-27*). It is
git-ignored because it describes a person and not the project, and the shared preflight reads it into
one line, `persona=engineer` or `persona=product-manager`, that every verb already reads. This record
decides what the skills do with that line.

## Decision

### One file says what a persona changes

`.codefall/shared/personas.md`, installed beside the other shared procedures, holds every persona:
what it is, how it is read, and one section per non-default persona with a short block per verb. A
skill reads the `persona=` line at step 1, or the file's `persona` field when the verb runs no
preflight, and when the value is not `engineer` it reads that persona's section, says which persona
the run follows, and follows it. No skill restates persona content. `engineer` is the default and
changes nothing, so the default run costs no tokens and reads no extra file.

### A persona extends a skill and never relaxes it

A persona changes the register a verb speaks in, what it leads with, and where it sends a decision
the person cannot make. The skill's Rules hold regardless: nothing is written without confirmation,
a human performs every merge, the reviewer never fixes, a test criterion is never written from the
implementation. A persona that would suspend a rule is asking for a different skill, and the verb says
so and stops. This is the same shape `customizations.md` gives a project's `CUSTOMIZE.md`.

### Workers and subagents never see the persona

The persona describes the person in the session, not the task. A worker prompt, a reviewer prompt,
and a consult prompt carry nothing about it, so the work a run produces is the same whichever person
asked for it, and only the conversation around it differs.

### Under `product-manager`, deferral is redirected

The rule "raise it once, then defer" keeps its first half and changes its second. A technical
judgment the person declines or cannot settle is parked, never decided by the run and never taken
from the person's silence or a shrug. Research comes before parking, because the person cannot fill
a gap the run could have closed by reading; only what stays a judgment call afterwards is parked.

In `design`, parking has a home: a conditional section, **Decisions needed**, carrying per decision
the question, the options the run saw with what each costs, the spec requirement it affects, whether
it will need an ADR, and any consult's analysis. A design with a non-empty **Decisions needed**
section is `Draft`, never `Ready`, and creates no beads. The report ends by handing the document to
an engineer's run of `design`, whose Promote mode settles each entry with that person, moves the
outcome into the section it belongs to, removes the entry, and only then creates the graph. The
section is available under any persona when the person says they cannot decide something; under
`product-manager` it is where every unsettled technical choice goes.

No ADR is written under the `product-manager` persona. A choice that would trigger one is parked
with a note saying it will need an ADR, and the engineer's run writes it, because a ratified ADR is
protected from edits by the ADR-immutability rule and should therefore be ratified by someone who
weighed it.

### The engineering verbs ask before running

`implement`, `equip`, `refresh`, `scaffold`, and `upgrade` produce nothing a product manager can
partially confirm, so under the persona each says in one sentence that it is engineering work and
asks whether to continue. On yes it runs unchanged. `review` does the same on a code target and, on
a document target, leads with the reading lenses.

### Register

Under `product-manager`, every verb's reports and questions are in product vocabulary, with no file
paths, component names, or command output in prose unless asked. The last line of every report is
the exception: it still names the exact next command, because that line is how the person continues,
and a report that ends with what to do next is a rule every skill already carries.

## Consequences

- **A design can sit in `Draft` waiting for an engineer.** That is the intended outcome of a product
  manager running `design`: a document holding everything they could confirm and an explicit list
  of what they could not, instead of a `Ready` design nobody competent confirmed. The cost is a
  second run, by a second person, before beads exist.
- **Two people's runs on one project behave differently.** The same verb on the same commit gives a
  product manager and an engineer different reports and, in `design`, different outcomes. Every
  report under a non-default persona says which persona ran, so a reader of the record knows why.
- **One more shared file is read by every non-default run.** The default run reads nothing extra,
  which is why `engineer` is the default and changes nothing.
- **Persona content is a product decision, revised in one place.** What `product-manager` changes
  per verb was written from a first list and will move as product managers use it. Because it is
  one file that no skill restates, revising it touches no `SKILL.md` and no token budget.
- **The engineering verbs gain one question under the persona.** A product manager who runs
  `implement` on purpose answers yes once; one who ran it by mistake is told what it is before it
  claims anything.
- **Parked decisions carry analysis, not blanks.** Research and any consult happen before parking,
  so the engineer who picks a `Draft` up finds the options and their costs laid out. The cost is
  that a product manager's `design` run spends tokens on questions it will not answer.

## Related

- ADR-009, *Agents* — the consult whose analysis a parked decision carries, and the rule that a
  consult proposes and never decides.
- ADR-010, *Upgrade* — the verbs `upgrade` and `refresh` that ask before running under the persona.
- `docs/decision-log.md`, *A per-user `.codefall/user.json`, carrying the persona, 2026-09-27* — the
  file this record reads, and why it is separate from `settings.json`.
- `extensions/shared/personas.md` — what each persona changes.
- `extensions/shared/customizations.md` — the shape a persona's "extends, never relaxes" rule
  follows.
- `extensions/skills/codefall-design/reference/document.md`, *Decisions needed* — the section's
  shape.
