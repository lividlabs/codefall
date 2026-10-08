# Personas

Shared procedure. Every verb follows it. A persona describes the person at the keyboard, not the
project: it is read from `.codefall/user.json`, a file that is never checked in, so two people
running the same verb on the same project can be answered differently. The rule about what a persona
may and may not change is stated once, here.
[ADR-011.2](https://github.com/lividlabs/codefall/blob/main/docs/adrs/ADR-011.2-personas.md) holds
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
below before step 1 and follow it. Tell the person once, in a few words at the start, which persona
the run is using, so they can correct it if it is wrong. Do not refer to the persona again after
that.

## What a persona may change

A persona changes how a verb talks to the person, what it leads with, and where it sends a decision
the person cannot make. It extends the skill and never relaxes it: the skill's **Rules** hold
regardless, nothing is written without confirmation, and no verb merges to the default branch. A
persona that would suspend a rule is asking for a different skill; say so and stop.

Workers and subagents never see the persona. It describes the person in the session, not the task,
so a prompt rendered for another agent carries nothing about it.

## `product-manager`

**Every verb — the communications rule.** Reports and questions are in product vocabulary: what a
user gets, what changes for them, what is still open. No file paths, component names, or command
output in prose unless the person asks. The last line of every report says in one plain sentence
what happens next and names **exactly one command** when there is a step for the person to start —
the button they press. Never a list of commands: anything else the verb could do, it did in this
run or asks about in a sentence.

**`envision`, `specify`.** The interview is the main event and gets the room. The code audit still
runs, and what it finds is stated as a product fact ("the profile screen has three tabs today")
without naming files. Open questions and what is out of scope are written out in full rather than
compressed.

**`report`.** The interview is the main event. The reproduction attempt is described by what the
person would see on the screen, not by the driver or the commands it ran.

**`mock-up`.** Unchanged.

**`plan`.** The person confirms what they can judge: the tier, the task cut against the spec,
and the case criteria, all in product terms. Every technical judgment the person declines or cannot
settle is set aside in the document's **Decisions needed** section, per `codefall-plan`'s
`reference/document.md`: the run never decides it on its own, and never takes the person's silence
or a shrug as consent. Research before setting a decision aside, since the person cannot fill the
gap; set aside only what stays a judgment call afterwards. Then, once, at the end of the draft, ask
one question:

> I set aside N technical decisions. Settle them with sensible defaults now so building can start,
> or leave them for an engineer?

On **settle now**, the run picks the default for each one — the simplest choice that fits the
project's stance and what it already uses — says each choice in one plain sentence, moves it into
the section of the document it belongs to, and carries on to `Ready` and the beads in the same
run. A decision that needs an ADR gets one, written from the default; the report says so in a
sentence, so an engineer knows where to look. On **leave them**, the plan stays `Draft`, creates
no beads, its pull request is a draft, and the report says an engineer's run of
`/codefall-plan PLAN-NNN` settles them. The report names how many decisions were set aside and
which answer the person gave.

**`implement`, `fix`, `equip`, `scaffold`, `upgrade`.** Before anything else, say in one plain
sentence what this run will write or change, in the person's terms, and ask whether to continue.
Example: "This will write code and open a pull request. Do you want to continue?" No file names,
no commands, no mention of the persona. On yes, run unchanged: the work is the same whoever asks,
and the verb's own approval still comes. `equip` and `scaffold` ask about scripts and project
layout during the run; those questions follow the communications rule like everything else.

**`refresh`.** Unchanged, except the communications rule.

**`review`.** On a document target, lead with the reading lenses and give `structure` and `status`
one line. On a code target, say what the run will do and ask, as above.

**`test`.** Unchanged, except the communications rule.
