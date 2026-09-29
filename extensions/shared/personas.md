# Personas

Shared procedure. Every verb follows it. A persona describes the person at the keyboard, not the
project: it is read from `.codefall/user.json`, a file that is never checked in, so two people
running the same verb on the same project can be answered differently. The rule about what a persona
may and may not change is stated once, here.
[ADR-011](https://github.com/lividlabs/codefall-cli/blob/main/docs/adrs/ADR-011-personas.md) holds
the reasoning.

## Contents

- Reading the persona
- What a persona may change
- `product-manager`

## Reading the persona

The shared preflight emits `persona=engineer` or `persona=product-manager`. A verb that runs no
preflight reads the `persona` field of `.codefall/user.json` itself; a missing file, a missing
field, or a value this file has no section for is `engineer`.

`engineer` is the default and changes nothing. For any other value, read that persona's section
below before step 1, say which persona this run follows, and follow it.

## What a persona may change

A persona changes the register a verb speaks in, what it leads with, and where it sends a decision
the person cannot make. It extends the skill and never relaxes it: the skill's **Rules** hold
regardless, nothing is written without confirmation, and a human still performs every merge. A
persona that would suspend a rule is asking for a different skill; say so and stop.

Workers and subagents never see the persona. It describes the person in the session, not the task,
so a prompt rendered for another agent carries nothing about it.

## `product-manager`

**Every verb.** Reports and questions are in product vocabulary: what a user gets, what changes for
them, what is still open. No file paths, component names, or command output in prose unless the
person asks. The one exception is the last line of every report, which still names the exact next
command, because that line is how the person continues.

**`envision`, `specify`.** The interview is the main event and gets the room. The code audit still
runs, and what it finds is stated as a product fact ("the profile screen has three tabs today")
without naming files. Open questions and what is out of scope are written out in full rather than
compressed.

**`report`.** The interview is the main event. The reproduction attempt is described by what the
person would see on the screen, not by the driver or the commands it ran.

**`mock-up`.** Unchanged.

**`design`.** The person confirms what they can judge: the tier, the task cut against the spec,
and the case criteria, all in product terms. Every technical judgment the person declines or cannot
settle is parked in the document's **Decisions needed** section, per `codefall-design`'s
`reference/document.md`: the run never decides it, and never takes the person's silence or a shrug
as consent. Research before parking, since the person cannot fill the gap; park only what stays a
judgment call afterwards. No ADR is written under this persona: a choice that would trigger one is
parked with a note that it will need an ADR. A design with parked decisions is `Draft` and creates
no beads. The report names how many decisions were parked and ends by handing the document to an
engineer's run.

**`implement`, `fix`, `equip`, `refresh`, `scaffold`, `upgrade`.** Say up front, in one sentence, that the
verb is engineering work, and ask whether to continue. On yes, run unchanged: the work is the same
whoever asks. The report still follows the register rule above.

**`review`.** On a document target, lead with the reading lenses and give `structure` and `status`
one line. On a code target, say up front that it is engineering work and ask, as above.

**`test`.** Unchanged, except the register rule.
