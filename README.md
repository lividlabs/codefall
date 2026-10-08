<p align="center">
  <img src="assets/codefall-logo.svg" alt="Codefall" width="460">
</p>

codefall
--------

**Codefall** is a toolkit for agentic spec driven software development utilizing beads. The toolkit has strong 
opinions — loosely held — regarding software architecture and software development.

The repo holds two components:

- [`cli/`](cli/) — the commaand line tool to help create and manage repos with Codefall.
- [`extensions/`](extensions/) — the installable plugin: skills, hooks, and shared scripts. 

Every skill is a **verb**. The verbs chain: `envision` frames the idea, `scaffold` makes the
project, `specify` states the problem, `report` states what is wrong, `mock-up` shows what it looks
like, `plan` decides the shape, `implement` writes it, `review` checks it.

## Installation

Codefall is a single binary called `codefall` in can be installed in one of the following ways. More coming soon.

### With the install script

```
curl -fsSL https://install.codefall.dev/sh | sh                 # latest release
curl -fsSL https://install.codefall.dev/sh | sh -s -- 0.28.0     # a pinned version
```

The script installs to `~/.local/bin`; set `CODEFALL_INSTALL_DIR` to put the binary elsewhere. If
that directory is not on your `PATH`, the script adds it in your shell's startup file (`.zshrc`,
`.bashrc`, or `config.fish`) and tells you to open a new terminal; set `CODEFALL_NO_MODIFY_PATH=1`
to have it print the line instead. It
verifies the download against the release's `checksums.txt` before installing, and records the
binary's path and version in `~/.local/state/codefall/install.json` (under `XDG_STATE_HOME` when that
is set), so codefall knows the binary came from the script. The source is
[`scripts/install.sh`](scripts/install.sh).

`codefall update` updates a binary the script installed to the latest release, or to a version
given as an argument; `codefall update --check` reports how the running binary was installed and the
latest release without changing anything. For a binary mise or `go install` manages, `update`
changes nothing and prints the command that updates it.

### With [`mise`](https://mise.jdx.dev/)

```
mise use packslip:github.com/lividlabs/codefall     # current project
mise use -g packslip:github.com/lividlabs/codefall  # global installation
```

The repository used to be `lividlabs/codefall-cli`. A `mise` configuration that still names
`packslip:github.com/lividlabs/codefall-cli` keeps installing on mise 2026.9.16 or later, which
follows the rename by repository ID; earlier versions need the new name.

### Download pre-compiled binaries

All realease binaries can be found on the [Github project release list](https://github.com/lividlabs/codefall/releases).

### Getting Started

There are two ways to get started, with or without an existing repository.

### Currently Supported Harnesses

- Claude Code (`claude`)
- Codex (`codex`)
- OpenCode (`opencode`)
- Antigravity (`agy`)
- Muse (`muse`)

A project can use more than one, and `init` asks which ones to set up rather than choosing for you.
`--harness` answers without asking and takes several: repeat the flag, or separate the names with
commas. Each harness is named for its command-line binary, shown beside it above, and that is the
name `.codefall/settings.json` records. A project set up when Claude Code was `claude-code` and
Antigravity was `antigravity` keeps working: `codefall doctor` warns about the old names, and the
next `codefall upgrade` rewrites them.

### New Projects

Using the `create` command will make your project directory, initialize git with an optional remote, create a barebones first commit,
and then run the `codefall init` worflow.

```
codefall create
```

### Existing Projects

Running the `init` command installs configuration for codefall, and the extension for each harness you
choose. It asks which harnesses the project uses and where the project's test cases live, and records
both in `.codefall/settings.json`. It runs once: a project that already has `.codefall/manifest.json`
is `upgrade`'s, and `init` says so rather than repeating itself.

```
codefall init
```

Running the `upgrade` command brings an installed project level with the binary: it reinstalls the
skills, shared files, and hooks for the harnesses the settings record, replaces the sections codefall
wrote into `AGENTS.md`, rewrites a harness name still spelled the old way, points a `$schema` URL an
earlier release wrote at the current one, and records the run in the manifest. It changes nothing it did not write, and asks before moving the installed version unless
`--yes` answers. It reinstalls the extension on every run, the binary's own version included, so a
skill, shared file, ignore entry, `AGENTS.md` section, hook registration, or testing file that has
gone missing or been edited is put back and reported, and it says "already up to date" only when
nothing changed. `--harness` adds a harness the project did not choose at `init`, installs for it,
and records it in the settings. A project set up before the manifest existed runs `init` one more
time, which writes the manifest, and uses `upgrade` from then on.

Two things happen on an upgrade that a rerun of `init` never did. Before a file changes, `upgrade`
prints the breaking changes recorded in this repository's changelog between the version the manifest
records and the binary's, release by release, and asks to continue; `--yes` continues past them, and a
range with a development build at either end is reported as undeterminable rather than as empty.
After the install, it removes what the previous run recorded writing that this one did not: a skill
the binary no longer ships, or ships under a new name, comes out of every skills directory it
installs for, with the directory it emptied, and the report names each one and says which were
renames. It compares only the harnesses the settings name, only the files the manifest lists, and
nothing outside the install directories; a manifest with no file lists gives it nothing to compare,
and it says so.

```
codefall upgrade
```

### What init writes

The skills go into each chosen harness's own skills directory — `.claude/skills/` for Claude Code,
`.agents/skills/` for Antigravity, Codex, Muse, and OpenCode — because that is where a harness looks
for them without being told. Everything else codefall installs is reached by a path codefall writes,
so it goes into `.codefall/`, once, whatever harnesses you chose:

| Path | Holds |
| --- | --- |
| `.codefall/settings.json` | what the project told `init`, and what the verbs read back |
| `.codefall/manifest.json` | what the last finished run wrote, per harness and for `.codefall/` itself |
| `.codefall/hooks/shared/` | the scripts every harness's hooks run |
| `.codefall/shared/` | the files the skills read, and the scripts a verb runs |

A skill names a shared file `../../../.codefall/shared/<file>`, which is the same file from either
skills directory. Nothing installed is a symlink, and no maintainer document from this repository is
installed into your project. [ADR-006](docs/adrs/ADR-006-install-layout.md) records why.

`settings.json` also carries an `agents` list: who codefall asks when a verb needs another reader,
a reviewer for `review` or a second opinion for a run that cannot settle a question, chosen by the
harness you are running codefall in. Each entry is for one **active agent**, a harness name or
`default`, and holds a `review` list and a `consult` list of agents to try in order, each a harness
named for its binary, or `current` for a subagent of whichever harness you are in, with an optional
model. A session reads the entry for its own harness, else `default`; a list left out means the
default's; and a run skips an agent this machine cannot start and moves to the next, so one
checked-in file serves every machine. `init` writes one `default` entry whose lists each hold
`current`, which is what every verb did before the list existed, and `doctor` reports which harnesses
yours can run and which lists never fall back to `current`.

```json
"agents": [
  { "activeAgent": "default", "review": [{ "harness": "current" }], "consult": [{ "harness": "current" }] },
  { "activeAgent": "muse", "review": [{ "harness": "claude" }, { "harness": "codex", "model": "gpt-5-codex" }] }
]
```

How a harness is started when a list names it — the flag its model goes in, the provider it reaches
the model through, extra arguments, and a command that has to run first — is `harnessConfig`, keyed
by the name a list item uses as its harness. A harness-named key configures that harness; any other
key is a variant whose `harness` field names the binary, so `codex-direct` can call codex a second
way. The model string is whatever the harness accepts for that provider, and the way to get it
right is to run `/codefall-equip agents` from any harness: it finds the model and provider the
harness uses, writes the block, adds the agent to the lists you choose, and proves it by running it
once.

```json
"harnessConfig": {
  "codex": {
    "provider": "amazon-bedrock-runtime",
    "args": ["-c", "model_reasoning_effort=high"],
    "env": "aws configure export-credentials --format env"
  }
}
```

`codefall config` in a terminal opens an editor over it: a menu of Agents, Reviews, and Persona.
Agents lists the active agents, `default` first and then each harness; open one and its Review and
Consult lists are there to reorder with shift and the arrows, add to with `a`, trim with `d`, hand
back to the default with `c`, and save with enter. Reviews holds whether review posts its findings
to the pull request. Esc goes back, `q` quits, and every write is reported once the editor closes,
in the same line a subcommand prints. The subcommands are for scripts:
`codefall config agents muse review claude codex:gpt-5-codex` sets one list, `--clear` in place of
the agents removes it, `codefall config agents` prints them all, and `codefall config review posting
on|off` sets posting, and `codefall config harness codex --provider amazon-bedrock-runtime --env
"aws configure export-credentials --format env"` sets a block (`--clear` removes it). `codefall config
show` prints all of it, and so does the bare command when there is no terminal to draw on.
[ADR-009.3](docs/adrs/ADR-009.3-agents.md) records the shape.

`.codefall/user.json` sits beside `settings.json` and describes the person at the keyboard rather
than the project, so it is yours and is never checked in: `init` adds it to `.gitignore`. Its one
field today is `persona`, `engineer` or `product-manager`, and a missing file or a missing field
means `engineer`. `codefall config persona product-manager` sets it, creating the file and its
`.gitignore` line when either is missing, and `codefall config persona` prints it and where it came
from. Editing the file by hand works too: `{"version": 1, "persona": "product-manager"}`. `doctor` reports the persona and fails a
file it cannot read. The schema is
[`cli/schemas/user.schema.json`](cli/schemas/user.schema.json).

**The persona changes how the verbs talk to you, never what they may do.** `engineer` is the
default and changes nothing. Under `product-manager`, every verb speaks in product vocabulary, the
interviews in `envision` and `specify` get the room, and `plan` sets every technical judgment you
decline or cannot settle aside in a **Decisions needed** section instead of deciding it for you or
reading your silence as a choice. It then asks you one question: settle them with sensible defaults
now so building can start, or leave them for an engineer? On "now" it picks each default, tells
you each in one plain sentence, and goes on to `Ready` and the beads in the same run; on "leave
them" the plan stays `Draft`, creates no beads, and an engineer's `plan` run settles them. The
engineering verbs, `implement`, `equip`, `refresh`, `scaffold`, and `upgrade`, say so in one
sentence and ask before running. Every rule a skill carries holds under any persona, and workers
never see it. What each persona changes is one file, `.codefall/shared/personas.md`;
[ADR-011.2](docs/adrs/ADR-011.2-personas.md) records the decision.

Init also writes into the project's own files. The Beads database it initializes gets
`audit.enabled: false` written into `.beads/config.yaml`, so bd's interaction log stays off until the
project turns it on. `AGENTS.md` gains four marked sections — Codefall, Beads, Local environment,
and Testing — each replaced between its markers on a rerun and never touching a word outside them,
and Claude Code gets a one-line `CLAUDE.md` pointing at it when the project has none. The Codefall
section is the frame the other three sit inside: the chain of verbs, the verbs beside it, and who is
authoritative for what, with the detail in `.codefall/shared/workflow.md`, a copy of
[`docs/workflow.md`](docs/workflow.md) that CI holds to its source.
The testing root it asked about is created with a `test-cases/` directory and skeleton `AGENTS.md` and
`README.md` files, which are yours from the moment they exist: each is written only when it is
missing, and a rerun never rewrites one. `.ignore` gains the two directories codefall commits and
nobody greps, `.gitignore` gains the refresh stamp, `.codefall/user.json`, and the testing root's
`.artifacts/`, and
`.gitattributes` gains a union merge for bd's append-only interaction log, so two branches that both
appended to it merge without a conflict.
[ADR-007](docs/adrs/ADR-007-test-cases.md) records what the tree is for.

Init also registers two hooks with each harness that reads them, merged into the harness's own hook
file beside whatever the project already registered. A `PreToolUse` guard denies any command that
would merge or push to the default branch, `gh stack merge` included, because a stack's trunk is
the default branch. A `SessionStart` notice says what the project needs done
— the checkout behind the default branch, an environment that has not been refreshed since `HEAD`
moved, a testing root or a runner nobody has declared, a Beads precondition that is blocking — and
prints nothing when everything is current. It reports and names the verb that fixes each thing; it
never pulls, never runs the project's `update`, and never fails a session it could not read.
Antigravity has no session event, so it gets the guard alone.

`codefall doctor` reports on the declaration in a **Testing** category: it warns when no testing root
is declared, naming `codefall upgrade`; warns when no test runner is declared, naming `/codefall-equip`;
and fails when the directory the project declared is not there. Like every other category, it names
the remedy and never runs it.

Upgrading from a codefall before this layout: run `codefall upgrade`. It removes the files the old
install recorded that the new one does not write; a manifest from before file lists existed records
none, and then the old copies are deleted by hand — the pull request that landed the change lists
what to delete.

## CLI commands

| Command | What it does |
| --- | --- |
| `codefall create <dir>` | Makes the directory and its git repository, commits a README and a `.gitignore`, runs `init` there, and offers the push |
| `codefall init` | Sets a directory up for codefall, once: settings, the extension for each harness, Beads, hooks, the `AGENTS.md` sections, the testing tree, and the manifest that records the run |
| `codefall upgrade` | Brings an installed project level with the binary, for the harnesses the settings record: warns about the breaking changes in between, reinstalls, removes what it no longer ships, and touches nothing it did not write |
| `codefall config` | Opens an editor over the agents, reviews, and your persona in a terminal; for scripts, shows the effective configuration (`show`), sets or clears who reviews or consults for sessions in one harness (`agents <activeAgent> <review\|consult> <harness[:model]>...`, `--clear`), sets or clears how a harness is called (`harness <key> --provider ... --env ...`), turns posting on or off (`review posting`), and prints or sets the persona (`persona`), and refuses a write the settings would not accept, in the editor and the subcommands alike |
| `codefall doctor` | Reports whether a project has what codefall needs, with a remedy per unmet check, and repairs nothing |

[ADR-010](docs/adrs/ADR-010-upgrade.md) records why `init` runs once and `upgrade` is its own command.

## Skills

TODO: rename the skill names to the actual

| Skill | Does | Status |
| --- | --- | --- |
| [`envision`](extensions/skills/codefall-envision/SKILL.md) | Get an idea onto paper before anyone specifies or scaffolds it: a numbered vision document under `docs/visions/` that carries the problem, the rough shape of an answer, and what nobody has decided yet. | in progress |
| [`scaffold`](extensions/skills/codefall-scaffold/SKILL.md) | Start a new project on the Clean + package-by-component stance: ratified ADRs, scoped `AGENTS.md`, optionally project files and boundary lint. | in progress |
| [`upgrade`](extensions/skills/codefall-upgrade/SKILL.md) | Bring a project's install and docs current: offer `codefall upgrade` when the manifest is behind the binary, then report what changed in the templates since the project's version, with per-file provenance, and apply only what the user takes. Also handles first-time adoption of the stance. | in progress |
| [`specify`](extensions/skills/codefall-specify/SKILL.md) | Turn a feature idea into a specification another session can implement: a spec document under `docs/specs/` holding requirements with EARS acceptance criteria, mirrored to the issue tracker. | in progress |
| [`report`](extensions/skills/codefall-report/SKILL.md) | Turn a bug into a report another session can fix: interview the person who saw it for the steps, the expected and actual result, screenshots, and the environment, try to reproduce it on the spot, and write a bug report under `docs/bugs/` with acceptance criteria, mirrored to the issue tracker. | in progress |
| [`fix`](extensions/skills/codefall-fix/SKILL.md) | Fix something small in one run: `plan` at tier 0 for one bead, then `implement` on it, behind one confirmation instead of two. Takes a bead, a bug report, an issue, or a description; stops and names `plan` when the work needs a plan document, an ADR, or more than one bead. Never merges to `main`. | in progress |
| [`mock-up`](extensions/skills/codefall-mock-up/SKILL.md) | Get the visual surface of a feature into the repository under `docs/mockups/`: import what a design tool exported, or make the mockup here, matching the app's own design system so it looks like it belongs. | in progress |
| [`plan`](extensions/skills/codefall-plan/SKILL.md) | Decide how a feature gets built and put the work into the graph: a plan document under `docs/plans/` scaled to the size of the change, ADRs for the choices that are hard to reverse, and the tasks in Beads with their dependency edges, each carrying the acceptance criteria it is verified against and the test case where one is called for. | in progress |
| [`implement`](extensions/skills/codefall-implement/SKILL.md) | Execute the graph: claim ready beads, build each in an isolated worker worktree with tests as part of done, write the test case a bead's criteria name before the code, verify against acceptance criteria, open PRs, and walk the waves until the frontier is empty. Never merges to `main`, and never sets a test harness up. | in progress |
| [`review`](extensions/skills/codefall-review/SKILL.md) | Review something and fix what the user accepts: uncommitted work, a branch, an open pull request, a commit range, a path, a document, or a description of what to look at. A subagent or another harness reviews, the session triages with you and applies what you take, and every finding is committed under `.codefall/reviews/`. | in progress |
| [`test`](extensions/skills/codefall-test/SKILL.md) | Run what the project declares: every suite, the subset your changed files reach, a named subset, or one test case in its `spec` or `agentic` modality. A spec case runs through the project's own runner; an agentic case is driven step by step through a browser or the shell and judged against the case's criteria. Every run is reported under `.codefall/tests/`. | in progress |
| [`equip`](extensions/skills/codefall-equip/SKILL.md) | Equip a project with what the other verbs need it to have: the local-environment scripts `refresh` runs — `start`, which brings its services up, and `update`, which makes the local environment match the checkout — the test harness `test` runs cases through, a spec runner per surface pointed at the testing root, and the agents `review` and every consult reach for, set up from any harness. Finds what the project already has or drafts it from what the repository or the session shows, then declares it in `.codefall/settings.json`. The fourth track is the document landing: the GitHub Action that merges a document pull request once a person adds the `auto-merge` label to it or approves it, and the label itself. | in progress |
| [`refresh`](extensions/skills/codefall-refresh/SKILL.md) | Bring the checkout, the beads, and the local environment current: fetch, fast-forward `main` when that is safe, sync the Beads database with its Dolt remote, run the declared `start` and `update`, record the commit the environment now matches, and turn a failure into a sentence that says what to do. The routine before starting new work. | in progress |

### Visions

`envision` writes the *why* down first — the problem, who feels it, the rough shape of an answer,
and the open questions — as `docs/visions/VISION-001-slug.md`. It is deliberately informal, and its
length is proportional to what you put in: two paragraphs is a valid vision, and so is a page.
Unknowns stay in the document as unknowns rather than being invented away.

**It takes whatever you arrive with.** A sentence, ten minutes of thinking out loud, a pitch
document, a whiteboard photo, or a folder of design-tool exports. Mockups are routed to
`docs/mockups/` where `plan` and `implement` look for them; everything else is saved verbatim under
`docs/visions/sources/`, and the vision cites both. Arriving with finished screens is not a reason
to be sent elsewhere — you can have every screen drawn and still have written nothing down about the
problem they solve, which is the case this skill is most useful for.

**A big pile usually holds more than one problem.** A pitch document can cover three, and a mockup
set spanning six surfaces usually does. Each problem that stands on its own becomes its own vision,
written as siblings rather than a parent and children, connected by the source they all came from. The
breakdown is proposed and you decide the cut.

**`scaffold` requires a vision**, and offers to run this skill when there isn't one. That requirement
exists because scaffolding without any idea of what is being built is where scaffolds go wrong — the
architecture questions get answered by defaults picked from a one-sentence description. With a vision
in hand most of those questions are already answered, so the scaffold session is shorter *and* the
answers are better. You can decline, and the scaffold records that it ran without one.

`specify` may draw on a vision and never requires one, because a vision carries the *why* and a
specification carries the *what* — they are different documents, and plenty of features need only the
second.

A vision is `Draft` while you are still adding to it, `Ready` once it is written and agreed, and
`Active` once work starts against it. Replacing part of one adds a `Revised by` line; replacing it
whole archives it to `docs/visions/archive/`, where the identifier stays valid and the citations still
resolve.

### Specifications

`specify` writes the *what* as `docs/specs/SPEC-003-slug.md`. A spec holds one cohesive feature, cut
into **requirements** — each with a user story and its own acceptance criteria — because a requirement
is the unit somebody picks up and builds.

Acceptance criteria are written in [EARS](https://alistairmavin.com/ears/), the Easy Approach to
Requirements Syntax, published at Rolls-Royce in 2009 and used here unchanged. It constrains a
requirement to six sentence shapes with the clauses always in the same order:

```
The system SHALL order exported segments by departure time
WHEN a traveler selects export, the system SHALL produce a file containing the itinerary
IF the trip is missing a departure date, THEN the system SHALL name the missing field
WHILE an export is in progress, the system SHALL show progress and allow cancellation
```

The point of the notation is that a criterion reads as an obligation rather than an observation, so
the criteria *are* the requirements and there is no second list to keep in step with them. Failure
behavior gets its own keyword, which is what makes the error paths visible as a group instead of
scattered among the happy ones.

Identifiers nest and share a prefix, so `grep SPEC-003` finds the document, its requirements, and
every test and ticket that cites them:

```
SPEC-003                        the spec
SPEC-003-REQ-01                 a requirement
SPEC-003-REQ-01-AC-01           a criterion
```

Numbering is append-only at every level. A retired number is never reused, so a test citing
`SPEC-003-REQ-01-AC-04` never silently comes to mean something else.

**The document is canonical, and the tracker is a mirror of it.** GitHub gets a parent issue for the
spec and a child issue per requirement, carrying that requirement's story and criteria in full so
nobody has to click through to work the ticket. Re-running `specify` regenerates those bodies. The
spec's own `Status` is `Draft`, `Ready`, or `Archived` and describes the document only — whether the
work is queued, underway, or done is the tracker's to say. The spec lands on its own branch,
committed once the mirror has written the issue number back, pushed, and opened as a pull request
without asking you. The report gives you its address and says: add the `auto-merge` label when you
want it merged, or have someone approve it; either one merges it. A GitHub Action the project
installs with `/codefall-equip landing` merges a labelled or approved pull request that touches only
document paths, so the label or the approval is the one thing anyone does to land a document, and no
verb ever adds the label or approves; a project without the Action gets the same pull request and a
person merges it. That holds for every verb that writes a document, and
[`docs/landing-documents.md`](docs/landing-documents.md) explains it. Document pull
requests do not stack on each other; each branches from `main` and lands on its own.

A feature too large for one cohesive spec becomes sibling specs rather than a parent and children.
The vision above them is what groups them, which is why a vision's `Related` line holds a list.

`codefall-upgrade` and `codefall-equip` are **explicitly invoked** and carry
`disable-model-invocation: true`, because each changes the install or the project's settings. Every
other verb an agent may run too, and a verb runs the verb upstream of it when the work needs it: a
spec that needs a mockup gets one from `mock-up` inside the `specify` run, a plan that finds its
spec still `Draft` runs `specify` to settle it, `review` and `test` run `implement` on the epic to
build the problems you took, and nobody is told to go run a command and come back. Each verb still
reports, offers, and applies only what the user takes; no verb merges to `main`, a person merges
every code pull request, and the project's Action merges document pull requests. The work on one
epic from plan to merge is a **delivery**, taken in
**rounds** of `implement`, `review`, and `test`; every handoff is a bead, a document, a report, or
a pull request, so a `/clear` between verbs loses nothing and each report ends with one next step.

A question of fact you cannot answer in the interview, how the existing system behaves in a case the
audit did not settle, is put to the project's consult agents once and comes back as a proposal you
confirm; a preference is never consulted on.

### Mockups

`mock-up` puts the picture in the repository, at `docs/mockups/<slug>/`. It runs before or after
`specify` — a mockup can be what makes the requirements obvious, or it can be drawn once they are
settled.

Two ways in. When you already work in a design tool, it **imports** what you exported and never
touches the files again: an imported asset is the record of what someone decided, and redrawing it
loses that. When you don't, it **makes** one.

**A mockup, not a wireframe.** Before drawing anything it reads the repository for your design
system, your tokens, your existing screens, and the fonts actually in use, then inlines what it found
so the file stays self-contained. The default is as close to what would ship as the repository lets it
get — someone opening it should see your product with a new screen in it, not a grey diagram of one.
A box drawing is still available when there is nothing to match yet or structure is the only open
question; it just isn't the starting point.

Everything else is a default with the reason attached and a stated case for going the other way:
static HTML, one file per state, realistic content, one viewport. When the open question is how
something *behaves* — a picker, a multi-step flow, a filter that has to feel right — it builds the
thing working, with a pinned library if that is what makes it faithful. Working or not, it stays a
reference: it proves the interaction and the implementer rebuilds it in the app's stack.

**Where the answer is genuinely open, you get options.** Two or three versions of the screen, named
by what differs — `list-table.html` and `list-cards.html` — with a line each on what they are better
at and which one it would pick. The one you take keeps the plain name and the README records the
decision.

The states are where the value is — populated, empty, and the primary failure at minimum, because the
empty and error screens are the ones nobody describes in an interview and where features come back
from review.

It opens by asking whether this is for a spec, for a vision, or a fresh start, and lists what is
there so you can pick one. A vision is a wider frame that usually spans several surfaces, so it says
how big the run would be before starting rather than refusing it.

The slug names the **surface**, not the spec. One screen gets touched by several specs over its life
and outlives all of them, so a mockup filed under whichever spec arrived first makes the second one
either duplicate it or reach into another spec's directory. Specs reference mockups by path under
their **Design notes**.

When `specify` records a requirement whose surface has no picture yet, it runs `mock-up` for that
surface in the same run and on the same branch, so the spec and its mockups arrive together. Only a
requirement whose mockup you chose to skip is labelled `requires-mockup`, and `plan` then runs
`mock-up` for it before it starts rather than refusing. Landing the mockup clears the label.

### Bug reports

`report` writes down *what is wrong* as `docs/bugs/BUG-012-slug.md`, and holds the place a spec holds
for a feature. It interviews the person who saw the bug for the steps from a starting point someone
else can reach, the expected result and what says so, the actual result word for word, screenshots
and logs, the environment, how often it happens, who it stops, and the last version that worked. It
pushes back on "it's broken" and "sometimes" the way `specify` pushes back on "fast".

**Then it tries to reproduce the bug while you are still there**, through the running product at the
checkout's commit, driving your steps as written. A step it cannot follow is a step the report is
missing, and it comes back to you with that step. The attempt changes nothing: no code, no data by
hand, no mocks. It is never a gate, either: a bug seen only in production or only some of the time
is still reported, and the report records whether it was reproduced, where, and what was tried.

Evidence is committed beside the report in `docs/bugs/BUG-012-slug/`, yours and the reproduction's,
each named for where it came from, because `gh` cannot upload images to an issue. The report's
acceptance criteria state the expected behaviour in EARS, citing a spec criterion where one already
says it, and they become the fix's criteria and its regression test. The issue is generated from the
report, or, when you filed one first, adopted rather than duplicated.

`plan` takes a bug report as it takes a spec: it reproduces what the report could not, finds the
cause before it chooses a tier, and most fixes land at tier 0 as one bead. The pull request that
carries the fix closes the issue when you merge it.

### Fixes

`fix` is the short path for small work. Given a bead, a bug report, an issue, or a sentence, it runs
`plan` at tier 0 to establish one bead, with the cause and the acceptance criteria, and then
`implement` on that bead, in one run. What it removes is repetition: one preflight instead of two,
one confirmation that shows the bead and the build plan together, and one report at the end. It is
the path `plan` takes at tier 0 and the path `implement` takes for one bead, written as one
procedure, so its rules are theirs, the merge rule included. When the work turns out to need a
plan document, an ADR, or more than one bead, it stops before writing anything and names `plan`.

### Plans

`plan` decides the *how* and puts the work into the graph. Two outputs, and the second is the one
that always exists: a plan document at `docs/plans/PLAN-007-slug.md`, and the tasks in Beads
with their dependency edges.

**Not every change earns a document.** A fix contained to one component, changing nothing public and
coming to one or two tasks, gets beads and nothing else — a plan document for a null check is the
ceremony this avoids. Anything crossing a component boundary, or fanning out past roughly three
dependent tasks, gets one. The document then has three required sections — Overview, Architecture,
Tasks — and seven more that appear only when their trigger fires. A conditional section with
nothing behind it is deleted, heading and all.

Research findings go inline, next to the decision they bear on. There is no sibling `research.md`,
no `data-model.md`, and no `contracts/` directory: a finding filed away from its decision is a note
nobody reads.

**The task table is staged before it is real.** It starts as a table with local identifiers, so the
dependency edges can be reviewed while they are still cheap to change — a flat list of tasks does not
catch the thing review is for, which is a wrong ordering or a missing prerequisite:

```
| ID | Task                              | Depends on | Plan ref   |
|----|-----------------------------------|------------|------------|
| T1 | Add `StageContext` type + serde   | —          | Components |
| T2 | Wire context load into `/scaffold`| T1         | Architecture |
```

Once you approve it, those rows become beads and the table stays. The callout above it records the
date, the epic — `booking-PLAN-007` — and that each row is `booking-PLAN-007-Tn`: the bead IDs
are the document's own numbering behind the project's Beads prefix, so a person can read and say
them. Beads is authoritative for work state from that moment, and the table holds only what the
plan decided — the tasks, their edges, and the section each came from — never a status column. A
row changes only through a revision, which edits the row and the bead together, so the two stay
level; a removed row's identifier is listed under the table and never reused.

**Revising a plan reconciles the graph rather than rebuilding it.** A bead nobody has touched is
edited, whatever changed. A bead someone has claimed, commented on, or closed is replaced only when
the work already done against the old wording would no longer count — a ticket should not change
under the person holding it. A task that leaves the plan is reported to you, never closed on its
own, because someone may still be working it. A request to revise can also arrive from downstream,
as a `plan-revision` bead, when the verb that found it could not amend it: `implement` and
`review` amend the plan's text themselves when the code disagrees with it, in their own pull
request, and file the bead when the fix would change a task row or a criterion a bead cites, when
the document is frozen, or when you declined the amendment.

**`plan` also decides which tasks need a test case.** A task verified through the wired product —
the real interface, against the real services — has the case named in its bead's acceptance
criteria, with its modalities and every criterion the case will hold: a spec criterion cited in
full, or marked `derived` with the requirement it elaborates and one line saying what it adds.
`agentic` is chosen only where verifying an outcome needs judgement, `spec` otherwise, and neither
where unit tests already verify the task. Approving the task table is where you sign the derived
criteria off, which is why they are shown with it. A gap they expose in the spec is offered as an
appended criterion, written into the spec on the plan's own branch and re-mirrored to its tracker
issue, so the bead cites a real identifier; `derived` is what a criterion stays when you decline.

Choices that are hard to reverse — a new dependency, a schema other components will build on, a
rejected alternative that cost real analysis — become an ADR in the project's own `ADR-NNN` sequence.
Most plans need none. A ratified ADR is never rewritten: a revision lands as a new, superseding
one.

**A technical point the run cannot settle is put to the project's consult agents, once.** After the
concern has been raised and the code and the ADRs read, `plan` renders the question, the files,
and the options it sees into a consult and walks the `consult` order from `settings.json`, a
subagent of the current harness when nothing is configured. A confident answer on a reversible
choice comes back to you as the run's recommendation, naming who was consulted; anything else, and
any hard-to-reverse choice, comes to you as analysis beside the concern, and what stays unsettled
goes into the document as a stated risk. A consult never writes, never settles an ADR, and is never
asked about a preference you have stated.

The plan's `Status` is `Draft`, `Ready`, or `Archived` and describes the document only. Whether the
work is queued, underway, or done is Beads' to say, the same division `specify` makes with its
tracker.

### Implementation

`implement` executes what `plan` put into the graph. Point it at a bead, an epic, or a plan — or
at nothing, and it shows the ready work and asks. An epic means walking the whole graph: each close
unblocks the next tasks, waves of background workers build them in isolated worktrees, and the run
continues until the frontier is empty.

**One approval starts it.** The go gate shows the landing strategy with its reason, a branch diagram,
the waves and the models proposed per task, what will be claimed in Beads, the consult order a
failure will be put to, and — when a vision sits behind the work — that go flips it to `Active`.
After go, only a failure stops the run. A failed worker is consulted on first — the root puts the
failure, the bead, and the courses it can see to the project's `consult` order, a subagent of the
current harness when nothing is configured — and then gets one automatic retry at higher effort
with the answer folded in beside the failure reason. A second failure is consulted on again and
then reaches you, with the analysis in front of you. That absence of mid-run gates is what makes an
overnight run possible, and the consult is what keeps a run from stopping on something a second
reading would have settled.

**Work lands one of two ways.** The default is a serial stack: every bead in topological order,
one branch atop the previous, one pull request each, workers one at a time, linked into a GitHub
stack as they open. The other is an epic branch, for work that must not land on `main` in increments
or when you want parallel waves: workers branch off it in waves, the root merges them at each wave
boundary, and one aggregate pull request reaches `main`. There is no depth cap: each layer of a
stack shows only its own diff, merging the top lands every layer, and GitHub rebases what is above a
layer you merge alone. Nothing is rebased or force-pushed by hand; a fix to a lower layer is
cascaded with `gh stack`. Documents do not stack: each document pull request branches from `main`
and the project's Action merges it on its own.

**A task bead closes at done — acceptance criteria verified, checks green, PR open — not at
merge.** That is Beads' own semantics, and it is what lets a stacked dependent start the moment its
parent's branch is pushed. The merge seam is carried by gates: every PR gates a "landed" bead inside
the epic, so the epic cannot close until you have merged everything, and the next session's
`bd gate check` turns your merges into bead state.

**The delivery is done when the graph says so.** `review` and `test` run against the epic's work
before you merge, and each ends with one question — "I found N problems. Fix them all?" — and what
you take becomes a child of the epic, which that same session then builds by running `implement`
on the epic; you typed one command. Every
verb prints the delivery's one-line chart as it starts and as it ends, and appends one line to the
epic's notes as it finishes, so `bd show <epic>` reads as the delivery's history:

```
booking-PLAN-007 · round 2 · 10/12 ██████████████░░░
```

```
round 1 (implement) [2026-09-30]: 7 tasks built, PRs #101 #102
round 1 (review) [2026-09-30]: 3 fixed, 2 deferred as children
round 1 (test) [2026-09-30]: FAIL 2/7, 1 child filed
```

Nothing open but the landed bead, and the last test run passed: that is done, and the merge is what
remains. [ADR-013.4](docs/adrs/ADR-013.4-deliveries.md) holds the rule.

**A person merges every code pull request.** The run ends at open PRs and the link to the top of
the stack, which lands every layer when merged on GitHub — never a merge command or a list to merge
in order — and the plugin ships a hook that mechanically denies the alternative. Tests are part of
done — the ones the plan named and the ones the work turned out to need — while the round's
end-to-end run, regression, and fresh-context retesting are the `test` verb's.

**A plan the work proves wrong is amended by the work, not worked around.** A worker that can
finish its task despite the plan's text, or the spec's, disagreeing with the code amends that text
in its own branch, names the amendment in its pull request, and the root re-mirrors a spec change to
its tracker issue. A disagreement the worker could not amend — one that would move work, a task row
or a criterion a bead cites; one in a frozen document; one you declined — is filed as a
`plan-revision` bead, and the close-out names those beads separately from code follow-ups and
tells you to run `plan` on that document. A disagreement the task cannot finish
under stops the run instead. [ADR-008](docs/adrs/ADR-008-upstream-amendments.md) holds the rule,
and it applies at any distance up the chain.

**A test case named in a bead's criteria is written before the code.** The worker writes it from
those criteria and from nothing else — not the sibling spec, not the code it is about to write, not
its own pull request text — and then the generated spec where the modality calls for one. The case
counts toward done; running it is `test`'s. If the project has no runner for it yet, those beads do
not start: `implement` says so and names `equip`, because setting a harness up is its own pull
request and never rides along inside a task's. Beads that need only unit tests carry on.

### Tests

`test` runs what your project declares: every suite, the subset your changed files reach, a named
subset, or one test case. Suite commands come from the same three places `implement` takes its
verification commands from — a `CUSTOMIZE.md` for the verb, your `AGENTS.md`, then inference — and
the confirmation says which one answered before anything runs.

**A case is one markdown file**, at `testing/test-cases/<area>/<slug>.md` under the testing root
`init` asked you for. Its path is its identifier, so there is no registry to keep level with the
directory; its frontmatter carries the variants the case runs once each and the fixed messages a run
is allowed to send; and its body carries the criteria. Every criterion says where it came from: a
spec criterion cited in full, or marked `derived` with the requirement it elaborates and one line
saying what it adds. The one source a criterion may never have is the implementation — not the code,
not clicking through the app, not the pull request that built it — because a test written from the
code certifies what the code does, defects included, and it passes loudly.

**Two modalities, and they produce different evidence.** A `spec` case is a generated test beside
the case file, run by your own runner and reported in that runner's pass and fail. An `agentic` case
is worked step by step by the session itself, through a browser tool or the shell, with each
criterion judged against what was observable — `held`, `failed`, `skipped`, or `unreachable` — and
the run as a whole `PASS`, `FAIL`, `ERROR`, or `PARTIAL`. The driver is whatever the session
actually has; one live call proves it before the run depends on it, and a case with no working
driver has its agentic modality refused rather than faked. Nothing is mocked at any layer, and a
state that cannot be forced through the product's own interfaces is recorded as unreachable.

**Every run is written down.** A report lands under `.codefall/tests/` — Markdown to read and JSON
to count across runs — carrying the verdicts, the per-criterion evidence, how many attempts each
step took even when it passed, what the run created against real services and whether it was cleaned
up, which driver ran, and an anomaly sweep of user-visible wrongness no criterion asked about.
Everything bulky — runner output, traces, HTML reports, run-scoped accounts — stays git-ignored
under the testing root's `.artifacts/`.

**A failing run produces a finding, never an edit to the criterion.** Findings are classified as a
real bug, a wrong expectation, a flake, or agent variance with an occurrence count, and they become
issues on your project's tracker only when you say so. Setting the runner up is `equip`'s job,
writing the case is `implement`'s, and `test` names the remedy when either is missing rather than
doing it for you.

**`equip` sets the runner up, as its own pull request.** It reads the testing root, searches for a
runner configuration and for wherever your end-to-end tests live today, and asks one question with
what it found: declare what is already there, or set up the default for each surface — Playwright
for a browser front end, an Electron shell, or an HTTP API; `go test` for a Go surface. React
Native, Tauri, and Flutter have no spec runner in this version and are refused for that modality,
with agentic cases still open to them. What it writes is the runner's configuration pointed at
`<root>/test-cases`, the runner's name in `test.runners`, its run-all and run-one commands in the
testing root's `AGENTS.md`, and whatever the runner's own install needs — Playwright's browsers, for
one — added to `update`. The proof is the runner listing nothing against an empty tree. Nothing
rides along: a task's pull request never sets up a harness.

### Reviews

`review` reads something, says what is wrong with it, and fixes what you accept. Point it at nothing
and it takes your uncommitted work; at a branch, an open pull request, a commit range, a path, or a
document identifier and it takes that; or describe what to look at — "the codepaths on the backend that
handle flight fulfillment" — and it searches, shows you the files it found, and asks before reading
a line of them.

**The context that finds a problem is never the one that fixes it.** The review runs in the first
agent this machine can run from the review list of your project's entry for the harness you are in,
else its `default` entry — a subagent of the current harness when nothing is configured, otherwise
whichever reader the team settled on, each in its own read-only mode — and `via=` names a
`harness[:model]` for a single run. An agent
that is not installed here is skipped, one that fails hands the prompt to the next, and the report
names every one tried. Then you triage, and this session applies what you took. A model that both
finds and fixes grades its own work on the next pass, and the second reading goes through the same
blind spots that made the first one worth doing.

**Eleven questions, asked separately.** Correctness, swallowed failures, behaviour changes, tests,
type design, conventions, comment accuracy, documentation that has fallen behind, simplification,
the local-environment scripts left stale by a change, and security — run as four parallel passes
rather than one reviewer looking for everything at once.
Documents get their own set: a spec is checked against its vision, a plan against its spec, an ADR
against every other accepted ADR. Before anything runs, the skill names what it resolved and which
questions it will ask, and you can drop any of them.

**Only live things are reviewable.** A merged pull request, a merged branch, a superseded ADR, an
archived document — all refused, because the code has moved on and there is nowhere for a fix to
land. Where fixes go is decided by the target, not by where you are standing: reviewing PR #51 from
another branch puts the fixes on #51's branch.

**Findings are committed.** Each review writes a pair of files under `.codefall/reviews/` — JSON for
the record, Markdown to read — carrying what was reviewed, at which revision, which questions ran,
what could not be checked, and what you decided about every finding. What the reviewer could not
settle is consulted on once, and a consult can move an item into the findings only after the session
has verified what it cited. They stay in the repository so
that patterns across reviews are visible, and a `.ignore` entry keeps them out of every search that
goes through ripgrep — `init` writes that entry, `doctor` warns when it has gone missing, and
`review` offers to put it back before writing findings into a directory nothing is hiding. Posting
findings to a pull request is off until a project turns it on.

### Local environment

Pulling `main` has consequences the pull does not perform: a migration the local database needs, a
dependency to install, code to regenerate, a container to rebuild. Each arrives as a diff, and a
teammate who does not read diffs has no way to know which. Two verbs close that gap.

**`equip` builds and rebuilds two scripts the project owns.** `start` brings up what the project
needs running locally; `update` makes the local environment match the checkout. Both are declared
under `local` in `.codefall/settings.json` as plain shell commands, so a Makefile target or a
package script is as good as a script of the project's own, and anyone can run them from a terminal.
On an existing project `equip` searches first, consults the project's agents once on what the search
left ambiguous, and asks one question with what it found; on a new
one `scaffold` writes them at code depth. Both are idempotent and never destructive: "bring the
project up to date?" has to be a question anyone can always answer yes to. The scripts are one of
`equip`'s four tracks — the test harness above, the agents, and the document landing are the
others — and one run equips one of them. The landing track writes the one workflow file,
`.github/workflows/codefall-land-documents.yml`, that merges a document pull request once you add
the `auto-merge` label to it or a colleague approves it, creates that label in the repository, and tells you what your branch rule
has to allow; it changes no branch rule itself
([`docs/landing-documents.md`](docs/landing-documents.md)).

**`refresh` runs them, and is the thing to run instead of pulling by hand.** It fetches,
fast-forwards `main` when the tree is clean and the move is safe, syncs the Beads database with its
Dolt remote so the graph it reads is the team's, runs `start`, runs `update` when the commit has
moved since the last clean run, and records that commit in a per-machine stamp so the next session
on the same commit does nothing. A failure comes back as one sentence saying what
failed, what it means, and what to do. A feature branch is never rebased and a dirty tree is never
stashed; the environment is brought level with the checkout either way.

**The scripts stay current at the point of introduction.** A task that adds infrastructure, a
dependency, a migration, or generated code changes the scripts in the same pull request: `plan`
names it in the task, `implement` counts it toward done, `review` carries a lens for it. `init`
writes the rule into `AGENTS.md`, `doctor` checks the declaration and that the stamp is
git-ignored, and every verb that reads the repository reports when `main` has moved or the
environment is stale. The session-start notice reports the same two things as the session opens,
so the first thing you hear about a moved `main` is not the merge conflict.

### The stance

**Pure Clean Architecture organized package-by-component**, with boundaries **mechanically
enforced** and components that stay **cheap to extract**. Dependencies point inward only; the
interfaces a use case needs live with the use case in `application/`, not in `domain/`; the top level
is capabilities, each behind a facade with the Clean layers nested inside; a composition root per app
binds implementations.

The output is a **monolith on purpose** — one deployable, no network between use cases. What it is
not is a monolith you are stuck with: each component owns its own data, no transaction spans two of
them, and what crosses a facade is a contract rather than an entity. Pulling a component out later is
a deployment change, not a redesign.

That core is stack-agnostic ([ADR-BASE-01 through ADR-BASE-03](extensions/skills/codefall-scaffold/templates/adrs/)) — Clean
Architecture, package-by-component, and keeping the resulting monolith cheap to split. Each surface adds
a profile supplying its own ADRs under its own prefix — `ADR-TS-01` and up for `typescript-react`,
which is Inversify, the TanStack Query / Zustand / `useState` split, and `eslint-plugin-boundaries`;
`ADR-GO-01` and up for `go`. Numbering restarts per profile, so two profiles never collide, and a
project's own ADRs are a separate sequence starting at `ADR-001`.

### Surfaces

Profiles are scoped to a **surface**, not to a kind of product — a project composes as many profiles
as it has surfaces. And a surface is defined by **where domain logic lives**, not by which languages
appear in the repo.

| Surface profile | Covers | Status |
| --- | --- | --- |
| `typescript-react` | TypeScript/Node backends; React frontends, web and Native — including Tauri, Electron, and RN apps whose native side is only wiring | **supported** |
| `go` | Go services, APIs, workers, daemons, and CLIs that hold domain logic | **supported** |
| `rust-native` | Rust services, and Tauri shells that hold domain logic | planned |
| `kotlin-native` · `swift-native` | Android, iOS | planned |
| `dart-flutter` | Flutter, mobile and desktop | planned |
| `python` · `java` | backends and services | planned |

So a **Tauri desktop app and a React Native mobile app are both scaffoldable today**, whole, when
their native side is boilerplate plus a few commands wrapping OS APIs. Those commands are gateway
implementations living in the `infrastructure/` ring — per ADR-BASE-01, a Tauri `invoke`, a bridge call,
an HTTP request, and a platform channel are all just gateways, and the use case never learns which
one it got. Rust or Kotlin in the repo doesn't make it a native surface, any more than a Postgres
driver makes SQL one.

It's two surfaces only when the native side holds real domain logic. Then it needs its own profile
plus a **seam ADR** deciding which side owns the domain.

React is the component model; the renderer is an outer-ring detail, so React DOM and React Native
share one profile. What differs is toolchain — and the profile records the trap: Metro transpiles
with Babel, not `tsc`, so ADR-TS-01's `emitDecoratorMetadata` is inert and Inversify fails at runtime
unless `babel-plugin-transform-typescript-metadata` is added.

`scaffold` reads your vision, or asks you to describe the project when you declined one, decomposes
it into surfaces, and matches each against this table. **If any surface has no profile, it stops** — it won't improvise ADRs for an unsupported
language or scaffold only the half that fits. A profile counts as supported once
`templates/surfaces/<name>/PROFILE.md` is complete.


Run below the root of a git repository, as one team in a monorepo might, and `codefall init` first
asks whether to install in that directory or at the root. `--location here` or `--location root`
answers for a script.

When a description leaves a surface's stack open and the directory already holds files, `scaffold`
consults the project's agents once and folds the answer into the stack question as the proposed
option; you still pick, and a planned profile is never proposed.

## Roadmap

See [extensions/docs/ROADMAP.md](extensions/docs/ROADMAP.md).

## License

[MIT](LICENSE) — Copyright (c) 2026 Livid Labs, LLC, authored by Dave Jensen.

The templates under `extensions/skills/codefall-scaffold/templates/`, and everything `scaffold`
copies from them into your project, are additionally available under [0BSD](LICENSE): no
attribution, no notice, no obligation. Your architecture documents are yours.
