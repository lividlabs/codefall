# ADR-012: UI Composition

## Status

Accepted — 2026-09-27

## Context

The decision log carried "UI composition" as Open from the day the shared `ui` module was made. Half
of the question was settled then: the palette, the marks, the colour-profile writer, and the spinner
runner live in `internal/shared/ui/`, a component maps its own vocabulary onto a `ui.Tone`, and the
module imports no component. The other half waited for the first component with a view: where
reusable terminal models live, who binds data to them, who owns arrangement and navigation, and
whether a facade may ever export a view. The entry said it would graduate to an ADR with the first
committed TUI.

`codefall config` is that component. It arrived as a set of scripted subcommands, one per settings
key, and the shape of `settings.json` leaked into the command line: `config review agents`,
`config consult agents`, `config agents for <harness>`. A person who wanted to change one order had
to know which key held it. The user's verdict on the tree was that it was confusing, and that what
they wanted was a set of menus to navigate to the settings.

Three ways to build that were weighed.

**Huh forms in a loop.** Huh is a form library: it runs a group of fields to completion and returns.
A menu the person enters, edits under, and returns from, with an agents list they reorder in place,
needs navigation state that survives between forms and a list widget that moves entries. Looping Huh
forms from a `for` loop would work and would read as a questionnaire: every section a fresh screen,
no way back but finishing, no reordering. ADR-002 chose Huh for prompts, and this is not a prompt.

**One shared module holding all presentation.** Every model, every view, and every binding in
`internal/shared/ui/`. It fails rule 4 of ADR-GO-02 the moment a shared model needs a component's
data type, which a list of agents does on its first line, and it makes the shared module the one
place every component's screen is written.

**A shell rendering generic widgets from contracts alone.** `main`, or an `internal/tui/` component,
draws every component's screen from what its facade returns. It widens every facade with a view type
or a view-shaped contract for the shell to render, and it puts the knowledge of how a component's
data reads on a screen in a package that does not own the data.

Bubble Tea with Bubbles and Lip Gloss is the tool the repository has been keeping for this. ADR-002
named Bubble Tea as the answer if a TUI were ever needed. Bubble Tea and Bubbles v2 were already
imported for the spinner, Huh v2's fields are Bubble Tea models, and all four are the v2 generation
that ADR-002 requires.

## Decision

### The stack

Bubble Tea v2 is the program, Bubbles v2 supplies the list and the key bindings, Lip Gloss v2 draws
through the shared palette, and Huh v2 fields are embedded as Bubble Tea models wherever a value is
typed or chosen, so `init`'s survey and the editor share one input library. Nothing from the v1
generation is added.

### Reusable models live in `internal/shared/ui/`, generic over their data

A model goes into the shared module when it knows nothing about what it shows: `ui.OrderList` holds
entries with an ID, a label, and a switch, moves the cursor and the entries, and hands the IDs back.
It never imports a component (rule 4), and a component's `presentation/` binds its own data to it,
turning agents into labels on the way in and reading names back on the way out. A model that needs a
component's type stays in that component's `presentation/`.

### The shell is the component's own `presentation/`

The program that owns arrangement and navigation — which screen has the keyboard, what Esc returns
to, what the heading and the help line say — is written in the component's `presentation/` layer,
beside its Cobra commands. `main` mounts commands and nothing else, as before. A component with a
screen owns its screen the way it owns its command tree, and no shell outside it draws for it.

### A facade never exports a view

A component's root package exports `Register` and `Command` (or `Commands`), and contracts of
primitives (ADR-001). No view type, no model type, no alias to one, crosses the facade. The open
item's provision for re-exporting a view type by alias is not taken: nothing outside a component
needs its screen, because the component's own command opens it.

### The editor and the scripted subcommands share one use case

Every write the editor makes goes through the same use case a subcommand calls, and each earns the
same one-line report, printed once the program has left the screen. The editor decides nothing a
script cannot: a write the settings module would refuse is refused inline with the same words, and
the file is left as it was. This is what keeps the two ways in from drifting.

### Bubble Tea stays out of `application/` and `domain/`

The rule of ADR-GO-02 rule 5 covers Bubble Tea as it covers Cobra: it appears in `presentation/`
and in `internal/shared/ui/`, and nowhere inward. `depguard` denies `charm.land/` in both inner
layers, and the rule was re-proven against a deliberate Bubble Tea import from config's
`application/`: it compiled and failed lint.

### The scripted tree collapses

`codefall config` with no arguments opens the editor in a terminal, and prints what `show` prints
with a one-line hint anywhere else. The subcommands are for scripts and for whoever knows exactly
what to change: `show`, `persona [value]`, `agents list|add|remove|order`, and one
`order <review|consult|harness> <name>...` with `--clear` in place of the three forms that had
followed the settings keys. Those three never shipped, so their removal breaks nothing.

## Consequences

- **The editor is the way in for a person.** Nobody has to learn which settings key holds an order
  to change it; the menu names the sections in words and shows each one's current value.
- **One more shared model to hold generic.** `ui.OrderList` is the first reusable Bubble Tea model
  and sets the pattern: entries carry an ID and a label, and the module never learns what they are.
  A second component with a screen reuses it or adds a sibling, and neither may import a component.
- **Two ways in, one behaviour.** Because the editor and the subcommands share one use case, every
  refusal, every layout-preserving write, and every report line is written once. The cost is that
  the editor can do nothing a script cannot, which is the point.
- **A program that cannot be driven from a pipe.** The editor needs a terminal, and a test drives
  its model with key messages rather than the program. The bare command's non-terminal path prints
  `show` and a hint, so a script that ran `codefall config` gets facts and not a hang.
- **The persona and the review block loosen together.** `postToPullRequest` became optional with
  `false` as its meaning when absent, so the editor's review section, and `config order review`,
  can create a `review` block that holds only an order. Posting to a pull request stays a decision
  the project makes explicitly.
- **Bubble Tea is no longer confined to the spinner.** The rule in `cli/AGENTS.md` that it appears in
  `internal/shared/ui/` only is replaced by this record's: `presentation/` and the shared module,
  never inward.

## Related

- ADR-002, *CLI Libraries* — Cobra, Fang, and the Charm v2 generation; Bubble Tea named as the TUI
  answer.
- ADR-GO-02, *Boundary enforcement* — rule 4, shared modules never import components, and rule 5,
  no CLI framework in the inner layers.
- ADR-001, *Facade contracts* — why a facade exports contracts and never a view.
- ADR-009, *Agents* — the list and the orders the editor edits.
- `docs/decision-log.md`, *Shared modules, 2026-08-27* — the half of this question settled then,
  and *UI composition, 2026-09-27* — the entry that struck the open item.
- `cli/internal/shared/ui/orderlist.go` — the first reusable model.
- `cli/internal/config/internal/presentation/editor.go` — the first shell.
