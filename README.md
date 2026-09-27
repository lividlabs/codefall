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
project, `specify` states the problem, `mock-up` shows what it looks like, `design` decides the
shape, `implement` writes it, `review` checks it.

## Installation

Codefall is a single binary called `codefall` in can be installed in one of the following ways. More coming soon.

### With [`mise`](https://mise.jdx.dev/)

```
mise use packslip:github.com/lividlabs/codefall-cli     # current project
mise use -g packslip:github.com/lividlabs/codefall-cli  # global installation
```

### Download pre-compiled binaries

All realease binaries can be found on the [Github project release list](https://github.com/lividlabs/codefall-cli/releases).

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
next `codefall init` rewrites them.

### New Projects

Using the `create` command will make your project directory, initialize git with an optional remote, create a barebones first commit,
and then run the `codefall init` worflow.

```
codefall create
```

### Existing Projects

Running the `init` command installs configuration for codefall, and the extension for each harness you
choose. It asks which harnesses the project uses and where the project's test cases live, and records
both in `.codefall/settings.json`; a rerun installs for the harnesses already recorded there and keeps
the testing root already declared, so `--harness` and `--test-dir` are only needed the first time, or
to add a harness.

```
codefall init
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

`settings.json` also carries an `agents` list: the readers a verb reaches for when it needs one other
than the session, such as a reviewer, in the order to try them. Each entry has a name, the harness
that runs it, named for its binary or `current` for whichever harness the session is in, and
optionally a model. A run skips an agent this machine cannot start and moves to the next, so one
checked-in list serves every machine, and `doctor` reports which entries yours can run. `init` writes
one entry, `subagent` on `current`, which is what every verb did before the list existed. `review`
walks it today; consult follows. [ADR-009](docs/adrs/ADR-009-agents.md) records the shape.

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
nobody greps, `.gitignore` gains the refresh stamp and the testing root's `.artifacts/`, and
`.gitattributes` gains a union merge for bd's append-only interaction log, so two branches that both
appended to it merge without a conflict.
[ADR-007](docs/adrs/ADR-007-test-cases.md) records what the tree is for.

Init also registers two hooks with each harness that reads them, merged into the harness's own hook
file beside whatever the project already registered. A `PreToolUse` guard denies any command that
would merge or push to the default branch. A `SessionStart` notice says what the project needs done
— the checkout behind the default branch, an environment that has not been refreshed since `HEAD`
moved, a testing root or a runner nobody has declared, a Beads precondition that is blocking — and
prints nothing when everything is current. It reports and names the verb that fixes each thing; it
never pulls, never runs the project's `update`, and never fails a session it could not read.
Antigravity has no session event, so it gets the guard alone.

`codefall doctor` reports on the declaration in a **Testing** category: it warns when no testing root
is declared, naming `codefall init`; warns when no test runner is declared, naming `/codefall-equip`;
and fails when the directory the project declared is not there. Like every other category, it names
the remedy and never runs it.

Upgrading from a codefall before this layout: rerun `codefall init`, then delete the old copies by
hand. Codefall removes only what it can prove it owns, and those directories hold your own files
beside codefall's — the pull request that landed the change lists what to delete.

## CLI commands

TODO

## Skills

TODO: rename the skill names to the actual

| Skill | Does | Status |
| --- | --- | --- |
| [`envision`](extensions/skills/codefall-envision/SKILL.md) | Get an idea onto paper before anyone specifies or scaffolds it: a numbered vision document under `docs/visions/` that carries the problem, the rough shape of an answer, and what nobody has decided yet. | in progress |
| [`scaffold`](extensions/skills/codefall-scaffold/SKILL.md) | Start a new project on the Clean + package-by-component stance: ratified ADRs, scoped `AGENTS.md`, optionally project files and boundary lint. | in progress |
| [`graft`](extensions/skills/codefall-graft/SKILL.md) | Bring a scaffolded project's docs up to date with the current templates: report what changed since its version, with per-file provenance, and apply only what the user takes. Also handles first-time adoption of the stance. | in progress |
| [`specify`](extensions/skills/codefall-specify/SKILL.md) | Turn a feature idea into a specification another session can implement: a spec document under `docs/specs/` holding requirements with EARS acceptance criteria, mirrored to the issue tracker. | in progress |
| [`mock-up`](extensions/skills/codefall-mock-up/SKILL.md) | Get the visual surface of a feature into the repository under `docs/mockups/`: import what a design tool exported, or make the mockup here, matching the app's own design system so it looks like it belongs. | in progress |
| [`design`](extensions/skills/codefall-design/SKILL.md) | Decide how a feature gets built and put the work into the graph: a design document under `docs/designs/` scaled to the size of the change, ADRs for the choices that are hard to reverse, and the tasks in Beads with their dependency edges, each carrying the acceptance criteria it is verified against and the test case where one is called for. | in progress |
| [`implement`](extensions/skills/codefall-implement/SKILL.md) | Execute the graph: claim ready beads, build each in an isolated worker worktree with tests as part of done, write the test case a bead's criteria name before the code, verify against acceptance criteria, open PRs, and walk the waves until the frontier is empty. Never merges to `main`, and never sets a test harness up. | in progress |
| [`review`](extensions/skills/codefall-review/SKILL.md) | Review something and fix what the user accepts: uncommitted work, a branch, an open pull request, a commit range, a path, a document, or a description of what to look at. A subagent or another harness reviews, the session triages with you and applies what you take, and every finding is committed under `.codefall/reviews/`. | in progress |
| [`test`](extensions/skills/codefall-test/SKILL.md) | Run what the project declares: every suite, the subset your changed files reach, a named subset, or one test case in its `spec` or `agentic` modality. A spec case runs through the project's own runner; an agentic case is driven step by step through a browser or the shell and judged against the case's criteria. Every run is reported under `.codefall/tests/`. | in progress |
| [`equip`](extensions/skills/codefall-equip/SKILL.md) | Equip a project with what the other verbs need it to have: the local-environment scripts `refresh` runs — `start`, which brings its services up, and `update`, which makes the local environment match the checkout — and the test harness `test` runs cases through, a spec runner per surface pointed at the testing root. Finds what the project already has or drafts it from what the repository shows, then declares it in `.codefall/settings.json`. | in progress |
| [`refresh`](extensions/skills/codefall-refresh/SKILL.md) | Bring the checkout, the beads, and the local environment current: fetch, fast-forward `main` when that is safe, sync the Beads database with its Dolt remote, run the declared `start` and `update`, record the commit the environment now matches, and turn a failure into a sentence that says what to do. The routine before starting new work. | in progress |

### Visions

`envision` writes the *why* down first — the problem, who feels it, the rough shape of an answer,
and the open questions — as `docs/visions/VISION-001-slug.md`. It is deliberately informal, and its
length is proportional to what you put in: two paragraphs is a valid vision, and so is a page.
Unknowns stay in the document as unknowns rather than being invented away.

**It takes whatever you arrive with.** A sentence, ten minutes of thinking out loud, a pitch
document, a whiteboard photo, or a folder of design-tool exports. Mockups are routed to
`docs/mockups/` where `design` and `implement` look for them; everything else is saved verbatim under
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
work is queued, underway, or done is the tracker's to say. The spec lands on its own branch, committed
once the mirror has written the issue number back, and the pull request that carries it to `main` is
offered, never merged: that holds for every verb that writes a document, and only the merge is yours.

A feature too large for one cohesive spec becomes sibling specs rather than a parent and children.
The vision above them is what groups them, which is why a vision's `Related` line holds a list.

Skills are **explicitly invoked** — `/scaffold`, `/specify`, and so on. Each carries
`disable-model-invocation: true`, so none of them fire on their own; scaffolding a project or filing
an issue is a deliberate act, not something inferred from a passing remark. `codefall-refresh` is
the one exception: it writes nothing but a git-ignored stamp, so an agent may run it on its own
when the environment is stale.

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

When `specify` records a requirement whose surface has no picture yet, its tracker issue is labelled
`requires-mockup` and `design` refuses to act on it. Landing the mockup clears the label.

### Designs

`design` decides the *how* and puts the work into the graph. Two outputs, and the second is the one
that always exists: a design document at `docs/designs/DESIGN-007-slug.md`, and the tasks in Beads
with their dependency edges.

**Not every change earns a document.** A fix contained to one component, changing nothing public and
coming to one or two tasks, gets beads and nothing else — a design document for a null check is the
ceremony this avoids. Anything crossing a component boundary, or fanning out past roughly three
dependent tasks, gets one. The document then has three required sections — Overview, Architecture,
Task Plan — and seven more that appear only when their trigger fires. A conditional section with
nothing behind it is deleted, heading and all.

Research findings go inline, next to the decision they bear on. There is no sibling `research.md`,
no `data-model.md`, and no `contracts/` directory: a finding filed away from its decision is a note
nobody reads.

**The task plan is staged before it is real.** It starts as a table with local identifiers, so the
dependency edges can be reviewed while they are still cheap to change — a flat list of tasks does not
catch the thing review is for, which is a wrong ordering or a missing prerequisite:

```
| ID | Task                              | Depends on | Design ref |
|----|-----------------------------------|------------|------------|
| T1 | Add `StageContext` type + serde   | —          | Components |
| T2 | Wire context load into `/scaffold`| T1         | Architecture |
```

Once you approve it, those rows become beads and the table stays. The callout above it records the
date, the epic — `booking-DESIGN-007` — and that each row is `booking-DESIGN-007-Tn`: the bead IDs
are the document's own numbering behind the project's Beads prefix, so a person can read and say
them. Beads is authoritative for work state from that moment, and the table holds only what the
design decided — the tasks, their edges, and the section each came from — never a status column. A
row changes only through a revision, which edits the row and the bead together, so the two stay
level; a removed row's identifier is listed under the table and never reused.

**Revising a design reconciles the graph rather than rebuilding it.** A bead nobody has touched is
edited, whatever changed. A bead someone has claimed, commented on, or closed is replaced only when
the work already done against the old wording would no longer count — a ticket should not change
under the person holding it. A task that leaves the design is reported to you, never closed on its
own, because someone may still be working it. A request to revise can also arrive from downstream,
as a `design-revision` bead, when the verb that found it could not amend it: `implement` and
`review` amend the design's text themselves when the code disagrees with it, in their own pull
request, and file the bead when the fix would change a task row or a criterion a bead cites, when
the document is frozen, or when you declined the amendment.

**`design` also decides which tasks need a test case.** A task verified through the wired product —
the real interface, against the real services — has the case named in its bead's acceptance
criteria, with its modalities and every criterion the case will hold: a spec criterion cited in
full, or marked `derived` with the requirement it elaborates and one line saying what it adds.
`agentic` is chosen only where verifying an outcome needs judgement, `spec` otherwise, and neither
where unit tests already verify the task. Approving the task plan is where you sign the derived
criteria off, which is why they are shown with it. A gap they expose in the spec is offered as an
appended criterion, written into the spec on the design's own branch and re-mirrored to its tracker
issue, so the bead cites a real identifier; `derived` is what a criterion stays when you decline.

Choices that are hard to reverse — a new dependency, a schema other components will build on, a
rejected alternative that cost real analysis — become an ADR in the project's own `ADR-NNN` sequence.
Most designs need none. A ratified ADR is never rewritten: a revision lands as a new, superseding
one.

The design's `Status` is `Draft`, `Ready`, or `Archived` and describes the document only. Whether the
work is queued, underway, or done is Beads' to say, the same division `specify` makes with its
tracker.

### Implementation

`implement` executes what `design` put into the graph. Point it at a bead, an epic, or a design — or
at nothing, and it shows the ready work and asks. An epic means walking the whole graph: each close
unblocks the next tasks, waves of background workers build them in isolated worktrees, and the run
continues until the frontier is empty.

**One approval starts it.** The go gate shows the landing strategy with its reason, a branch diagram,
the waves and the models proposed per task, what will be claimed in Beads, and — when a vision sits
behind the work — that go flips it to `Active`. After go, only a failure stops the run, and a failed
worker gets one automatic retry at higher effort before anything reaches you. That absence of
mid-run gates is what makes an overnight run possible.

**How work lands is read from the graph's shape.** Independent chains stack toward `main` in
parallel when their predicted file scopes are disjoint; overlap or fan-in serializes them into one
topological stack; an epic branch appears only when fan-in meets a real need for parallelism, or
when increments must not land on `main`. There is no depth cap — a deep stack costs only a muddy
three-dot diff until it drains bottom-up.

**A bead closes at done — acceptance criteria verified, checks green, PR open — not at merge.** That
is Beads' own semantics, and it is what lets a stacked dependent start the moment its parent's
branch is pushed. The merge seam is carried by gates: every PR gates a "landed" bead inside the
epic, so the epic cannot close until you have merged everything, and the next session's
`bd gate check` turns your merges into bead state.

**A human performs every merge to `main`.** The run ends at open PRs and a reported bottom-up merge
order, and the plugin ships a hook that mechanically denies the alternative. Tests are part of done
— the ones the design planned and the ones the work turned out to need — while regression and
fresh-context retesting stay with the `test` verb.

**A design the work proves wrong is amended by the work, not worked around.** A worker that can
finish its task despite the design's text, or the spec's, disagreeing with the code amends that text
in its own branch, names the amendment in its pull request, and the root re-mirrors a spec change to
its tracker issue. A disagreement the worker could not amend — one that would move work, a task row
or a criterion a bead cites; one in a frozen document; one you declined — is filed as a
`design-revision` bead, and the close-out names those beads separately from code follow-ups and
tells you to run `design` on that document. A disagreement the task cannot finish
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
tracker issues only when you say so. Setting the runner up is `equip`'s job, writing the case is
`implement`'s, and `test` names the remedy when either is missing rather than doing it for you.

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
agent of your project's `agents` order that this machine can run — a subagent of the current
harness when nothing is configured, otherwise whichever reader the team settled on, each in its own
read-only mode — and `via=` names one agent or a raw `harness[:model]` for a single run. An agent
that is not installed here is skipped, one that fails hands the prompt to the next, and the report
names every one tried. Then you triage, and this session applies what you took. A model that both
finds and fixes grades its own work on the next pass, and the second reading goes through the same
blind spots that made the first one worth doing.

**Eleven questions, asked separately.** Correctness, swallowed failures, behaviour changes, tests,
type design, conventions, comment accuracy, documentation that has fallen behind, simplification,
the local-environment scripts left stale by a change, and security — run as four parallel passes
rather than one reviewer looking for everything at once.
Documents get their own set: a spec is checked against its vision, a design against its spec, an ADR
against every other accepted ADR. Before anything runs, the skill names what it resolved and which
questions it will ask, and you can drop any of them.

**Only live things are reviewable.** A merged pull request, a merged branch, a superseded ADR, an
archived document — all refused, because the code has moved on and there is nowhere for a fix to
land. Where fixes go is decided by the target, not by where you are standing: reviewing PR #51 from
another branch puts the fixes on #51's branch.

**Findings are committed.** Each review writes a pair of files under `.codefall/reviews/` — JSON for
the record, Markdown to read — carrying what was reviewed, at which revision, which questions ran,
what could not be checked, and what you decided about every finding. They stay in the repository so
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
On an existing project `equip` searches first and asks one question with what it found; on a new
one `scaffold` writes them at code depth. Both are idempotent and never destructive: "bring the
project up to date?" has to be a question anyone can always answer yes to. The scripts are one of
`equip`'s two tracks — the test harness above is the other — and one run equips one of them.

**`refresh` runs them, and is the thing to run instead of pulling by hand.** It fetches,
fast-forwards `main` when the tree is clean and the move is safe, syncs the Beads database with its
Dolt remote so the graph it reads is the team's, runs `start`, runs `update` when the commit has
moved since the last clean run, and records that commit in a per-machine stamp so the next session
on the same commit does nothing. A failure comes back as one sentence saying what
failed, what it means, and what to do. A feature branch is never rebased and a dirty tree is never
stashed; the environment is brought level with the checkout either way.

**The scripts stay current at the point of introduction.** A task that adds infrastructure, a
dependency, a migration, or generated code changes the scripts in the same pull request: `design`
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

## Roadmap

See [extensions/docs/ROADMAP.md](extensions/docs/ROADMAP.md).

## License

[MIT](LICENSE) — Copyright (c) 2026 Livid Labs, LLC, authored by Dave Jensen.

The templates under `extensions/skills/codefall-scaffold/templates/`, and everything `scaffold`
copies from them into your project, are additionally available under [0BSD](LICENSE): no
attribution, no notice, no obligation. Your architecture documents are yours.
