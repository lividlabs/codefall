---
name: codefall-scaffold
description: Start a new project on the Clean + package-by-component stance — interview for the calls a template can't make (bounded contexts, app topology, per-surface architecture), then emit ratified ADRs, scoped AGENTS.md files, and optionally the project files and boundary-lint wiring.
argument-hint: "[project-name] [path]"
disable-model-invocation: true
allowed-tools:
  - Read
  - Glob
  - Grep
  - AskUserQuestion
  - Write
  - Edit
  - Bash
---

# Scaffold

Start a new project with the architecture decided, written down, and enforceable.

Scaffold's minimum output is **documentation** — the ADRs that fix the architecture and the scoped
`AGENTS.md` files that make them operative. Code is optional and additive on top of that.

Scaffolding is not implementing. Once the decisions are recorded and the project is green and empty,
stop. Features go through `codefall-specify` → `codefall-design` → `codefall-implement`.

Paths that start with `templates/`, `reference/`, or `../` are relative to this skill's directory,
not the user's project.

## Files beside this one

Read each when its step says to; none is loaded up front.

- `reference/catalog.md` — the bundled ADR templates, the three ADR namespaces, the surface
  catalog with each profile's status, and consulting before the stack question. Read at step 1,
  before the stack question. Names `../../../.codefall/shared/running-agents.md`, `../../../.codefall/shared/run-agent.sh`,
  `../../../.codefall/shared/consult-prompt.md`, `../../../.codefall/shared/consult.schema.json`.
- `reference/provenance.md` — the shape and fields of `.codefall/scaffold.json`. Read at step 4.
- `reference/depth.md` — what each depth answer emits, and the boundary-enforcement obligation.
  Read at step 5.
- `templates/adrs/` and `templates/surfaces/<name>/` — the ADRs, profile, and `AGENTS.md.skeleton`
  the skill installs. Read the ADRs before scaffolding; they are the substance of this skill.

## Scope — how, not what

Scaffold decides **how this project will be built**, never **what it does**.

| In scope | Out of scope |
| --- | --- |
| Layering, the dependency rule, where interfaces live | What the features are, or how they work |
| DI, composition roots, naming | The data model, schemas, entity fields |
| Boundary enforcement, lint, CI | API design, endpoints, message shapes |
| Rendering strategy, topology, depth | Business rules, workflows, edge cases |

- **Do not design the application.** No entities, no schema, no questions whose only purpose is to
  understand the product.
- **Unnamed domains are normal.** Step 1 judges whether the thing is one cohesive domain or several
  separable capabilities; missing names decide nothing. Ask once, accept "not yet", move on.
- **The stack is theirs; the stance is not.** Codefall has no opinion about which supported language
  or framework a project uses and will not steer. It does not bend on the architecture: a user who
  wants different layering, package-by-layer, or no boundary enforcement is asking for something
  this skill does not do. Say so plainly.
- **Keep the session short.** Stop asking the moment you have enough to emit the docs; prefer a
  default over a question wherever the answer does not change what gets emitted. A long thread
  about how the thing will work belongs to `codefall-specify` and `codefall-design`; say so and
  finish.

## The stance

**Pure Clean Architecture organized package-by-component**, with boundaries **mechanically
enforced**, and **splittable** — a monolith whose components stay cheap to extract. The core is
stack-agnostic, ADR-BASE-01 through ADR-BASE-03; each supported surface adds a profile with its own
ADRs.

In one paragraph: dependencies point inward only; the interfaces a use case needs live *with the use
case* in `application/`, not in `domain/`; the top level is capabilities, not layers, each behind a
facade with the Clean layers nested inside; a composition root per app binds implementations; the
facades are enforced by tooling, wired on day one; and components stay independently extractable —
each owns its data, no transaction spans two, and contracts cross facades rather than entities. The
surface profile fills in *which* tooling.

Vocabulary is deliberate. Say **gateway**, not "port" or "adapter". Interfaces are the bare noun
(`UserRepository`), implementations are qualified (`PostgresUserRepository`, `DefaultClock`).

The inherited ADRs ship **Accepted** with a real date. Amend one only when the interview requires
it, and say what you changed in the report. Project ADRs are a separate sequence starting at
`ADR-001`.

## Process

### 1. Describe and match — the gate

Read `reference/catalog.md` first.

#### A vision comes first

Read `docs/visions/` in the working directory before asking anything.

**When there is a vision**, take two things from it: the **surfaces**, and the **shape** judgement
below. Confirm both rather than asking for them — "VISION-001 describes a React web app and a Go
API, and reads as several separable capabilities; correct?" — and skip the describe question when it
does. Its **Environment & constraints** section is written for this moment. Read it for those two
things and nothing else: product detail read here goes in the decision log's **Parking lot**, exactly
as if the user had said it out loud.

**When there is no vision**, say so and offer the vision step:

> Before scaffolding I'd like a vision — a short document saying what this is and why. It takes a few
> minutes, and without one I end up picking architecture defaults from a one-sentence description,
> which is where scaffolds go wrong. Want to run `/envision` first?

On yes, **stop scaffolding** and hand off. On no, continue, and record the override in
`docs/decision-log.md` under `Open` at step 4.

#### Describe

Ask **one question**: what are you building, and what does it run on? A sentence or two.

That answer yields exactly **two** things, and nothing else learned here changes what gets emitted:

1. **The surfaces**, so each can be matched to a profile.
2. **The shape** — *one cohesive domain* or *several separable capabilities*. This decides
   package-by-component versus ports-and-adapters in step 3.

**Judging shape.** "A habit tracker where users log habits and see streaks" is one cohesive domain.
"An internal platform for billing, inventory, and shipping" is several separable capabilities. The
test is whether the thing decomposes into capabilities that could plausibly be owned, deployed, or
extracted separately — **not** whether the user has named them.

If the sentence does not say, ask once, flatly: *does this break into a few separate capabilities,
or is it one cohesive thing?* That is the only follow-up this step is allowed. If it still does not
settle it, **record the shape as unclear and move on.** Do not ask again and do not guess.

**When the user volunteers more than you asked for**, name the surfaces you found and carry on. Do
not ask a follow-up about a feature, propose entities or a schema, or let product detail reach the
ADRs. If something will matter to `codefall-specify` or `codefall-design`, put it in the
decision-log's **Parking lot** and say you did.

**Note any rendering signal without asking for one.** Public pages, sharing, browsing without an
account, or search visibility decide a React surface's topology in step 3, question #1.

#### Decomposition

**A surface is defined by where domain logic lives, not by what languages are present.** A language
that only implements gateways is the outer ring of a surface that already has a profile. Ask of each
part: *does this hold entities and use cases, or does it only reach out on their behalf?*

A Tauri app whose `src-tauri/` is boilerplate plus thin commands wrapping OS APIs is **one surface**,
`typescript-react`; those commands are `infrastructure/` per ADR-BASE-01. It becomes two surfaces
only when the shell holds **real domain logic** — heavy compute, native integrations with their own
rules, security-sensitive work that must not live in the renderer. The same reasoning covers an
Electron main process, a React Native native module, and a thin native wrapper around a web view.
A Flutter app with a Go API is two surfaces. A plain Node service is one.

Ask directly: *is there business logic in the shell, or is it just wiring?* Do not infer it from the
file listing.

#### The stack question

**Name the profile the description already implies, and confirm it.** "A React app" implies
`typescript-react`; "a CLI in Go" implies `go`. Say which you matched and give the user a plain way
to say otherwise.

**Only where the description actually decided.** "A website" decides the frontend and says nothing
about what serves it. Confirm the decided part and ask about the rest; assuming a Node backend
because the frontend is React is the steering this skill promises not to do.

**Do not present every supported profile as a co-equal menu** where one was clearly named. Ask with
the full list **only when the description genuinely leaves it open** — "an API", "a service", "a
background worker" with no language named:

- `typescript-react` — TypeScript/Node backends, React frontends (web and Native), and Tauri or
  Electron apps whose native side is only wiring
- `go` — Go services, APIs, workers, daemons, and CLIs
- **None of these**

**Never list a planned profile as an option**, in this question or any other. When the directory
holds files that could answer it, consult once first, per `reference/catalog.md`, and fold the
answer in as the proposed option; the user still picks.

**"None of these" is the deliberate exit.** When the user picks it, say plainly:

> Sorry — we don't support your stack yet.

Then name what *is* supported, offer to record the request, and stop. Do not steer toward a
supported profile they did not pick.

With more than one surface, ask once per surface, each with the same options.

**If every surface matches a supported profile**, name the profiles you matched and continue.

**If any surface does not, stop** — the project is not scaffoldable, even if the other surfaces are.
Do not substitute the nearest supported profile, hand-author a profile's ADRs for an unsupported
language, or emit a partial scaffold and note the gap in passing. Name which surfaces fit and which
do not, say what is supported, and offer to record the request.

If the user, having been told, explicitly asks for what *is* covered — the stack-agnostic core, or
the supported surfaces alone — emit it, and state plainly which surfaces were left undecided and that
their boundary-enforcement obligation is unmet.

### 2. Seams — only when a client owns domain logic

**Default: the domain lives on the server.** A browser, a thin native shell, or any client that
renders and calls an API holds no entities and no use cases of its own, so there is no seam — a
`fetch`, an `invoke`, and a platform channel are gateway implementations under ADR-BASE-01.

**Skip this step unless a client holds domain logic of its own**, in one of three cases:

- **Offline-capable mobile or desktop apps** that must decide, validate, and reconcile without
  asking the server.
- **Games**, where in-game simulation lives on the client while accounts, inventory, and progression
  live on the server. Two domains.
- **No server at all** — a standalone SPA, a CLI, a local-only tool. Then the client *is* the whole
  app: one surface, and still no seam.

Ask directly: *does this client decide anything on its own, or does it always ask?*

When a client does hold domain logic, settle where the two meet:

- **Which side owns the domain.** Entities can live on either side, or — badly — on both. Pick one.
- **What the gateway across the seam looks like.** An `invoke` command, an HTTP call, and a platform
  channel are the same shape to a use case. The inner rings must not name the transport.

Record the answers as `ADR-001`, the first in this project's own sequence, from
`templates/adrs/_TEMPLATE.md`. This is the one place this skill writes a new decision rather than
instantiating a template — draft it, then have the user confirm it before writing.

### 3. Interview

The templates deliberately do not decide four things. **Batch all of it into one round of
questions**, not a conversation. Every item has a workable default. #3 is not asked: it is read
from step 1's shape judgement.

**Recommend only what an ADR supports.** Where a decision traces to an ADR, say so and name the
recommendation. Where Codefall has no stance, present the options flat, say there is no house
opinion, and let the user choose. Never label an option "Recommended" without an ADR behind it.

1. **App topology** — how many apps? Mostly answered by step 1's surfaces. Each app gets its own
   composition root and its own `AGENTS.md`. Codefall has **no opinion on repo layout** — monorepo
   or separate repos, workspaces or a single package. One app: default to a single package without
   asking. More than one: ask where the user wants them, say there is no house preference, and follow
   the answer.
   **Also settle each React surface's topology** — SPA plus a separate API, or a Next.js SSR shell.
   The choice turns on one question: does anything need to be publicly reachable and worth indexing?
   If step 1's description answered it, take it. If it was silent, ask exactly that question once —
   not "do you want Next.js". Default to SPA plus API when the answer is no.
2. **Bounded contexts** — do the capabilities already have obvious names? Ask **once**, in one
   sentence, with "not yet" as a first-class answer; for a new project it is the expected one. Names
   seed the top-level component folders and change nothing else — they do **not** decide the
   architecture. If the user names some, keep them **coarse**, per ADR-BASE-02: two or three is a
   fine start. If the answer is "not yet", vague, or hesitant, take it and move on. Do not push, do
   not suggest candidates.
3. **Per-surface architecture** — package-by-component (the default) or ports-and-adapters? Decide
   from **step 1's shape judgement**, never from whether #2 produced names. ADR-BASE-02 names two
   conditions favoring p&a: one cohesive domain, or boundaries genuinely unknown. Several separable
   capabilities means package-by-component **even when nobody has named them yet**, and **even when
   the project will never be deployed as separate services**.
   If p&a is chosen because the shape is *cohesive*, amend ADR-BASE-02 for that surface. If it is
   chosen because the shape is *unclear*, leave ADR-BASE-02 alone and record it under `Open` in the
   decision log at step 4.
4. **Depth** — docs only, docs + project files, or a runnable skeleton? Default to **docs only**
   unless the user wants more. See `reference/depth.md`.

**Ask where it goes**, in the same batch. The current working directory is not a default. Only then
check the chosen target: never scaffold into a
non-empty directory without saying so first, and never overwrite an existing path. A directory
holding only `docs/visions/` is the expected state, not a non-empty directory.

### 4. Emit the docs — always

Branch first, per `../../../.codefall/shared/landing.md` — `scaffold/<project-or-surface>`.

- `docs/adrs/` — `ADR-BASE-01` through `ADR-BASE-03` plus every ADR the matched profiles supply, plus
  `_TEMPLATE.md`. One flat directory. Dated and Accepted, amended per the interview. Drop ADRs that
  do not apply: a backend-only project has no use for `ADR-TS-02`, and a surface that could never be
  split into services at all — a CLI, a desktop or mobile app, a library — has no use for
  `ADR-BASE-03`. That gate is about the surface, not its layout; having no *current* plan to split an
  extractable backend is not a reason to drop it.
- `.codefall/scaffold.json` — **provenance**, per `reference/provenance.md`. Written every time, at
  every depth. Compute every hash with `shasum -a 256 <file>` after the file is written; never
  invent one.
- `docs/decision-log.md` — `Locked` / `Open` / `Parking lot`. Open `Locked` with the scaffold line.
  Seed `Open` with anything the interview surfaced but did not settle — a provisionally chosen
  ports-and-adapters; a scaffold run without a vision: "scaffolded without a vision on `<date>`;
  shape judged from a one-sentence description." Seed `Parking lot` with the product detail step 1
  heard but did not act on.
- A scoped `AGENTS.md` per app/package, from the profile's skeleton: fill the name and the one-line
  description, fix the ADR links to the right relative path, delete the `<frontend only>` blocks on
  backend surfaces, delete the `<never-splittable>` block and its ADR-BASE-03 link on a surface that
  can never be split, and fill in Gotchas. Keep it terse — it links the ADRs and never restates the
  why.

If the harness in use reads `CLAUDE.md` rather than `AGENTS.md`, add `CLAUDE.md` as a one-line
pointer to `AGENTS.md`. Do not maintain two copies of the rules.

### 5. Emit code — per the depth answer

Read `reference/depth.md`. The concrete file list is the profile's — follow its **Depth notes**
rather than what you would reflexively pick for the language. Whenever you write the boundary
config, it enforces all five rules.

Docs-only output leaves the boundary-enforcement obligation unmet. Say so plainly in the report and
name it as the first task the user owes the project.

**Either code tier also equips the project**, per the `codefall-equip` skill's section *When
another verb follows this skill* and the profile's depth notes: draft from what this run emitted,
`start` exits `0` with nothing to bring up yet, and declare both under `local` in
`.codefall/settings.json` when it exists — otherwise report the declaration as owed once
`codefall init` has run. Docs-only output writes no scripts and names `codefall-equip` as owed.

### 6. Verify

Whatever you emitted must actually work. If there are project files, run install, lint, and test —
and confirm the boundary rules **fail on a deliberate violation**, because a lint config that catches
nothing is the common failure here. Do not assume; run it. Prove the local scripts too: `update`
twice, the second exiting `0` and quickly. If there are only docs, check that every ADR cross-link
and every `AGENTS.md` link resolves.

### 7. Report

- The surfaces identified and the profile matched to each.
- The interview answers, as the decisions now recorded.
- Any ADR amended, and what changed — this is what `.codefall/scaffold.json` records as `amended`,
  so the report and the file must agree.
- Files created, landed per `../../../.codefall/shared/landing.md`.
- What the user still owes the project — always including boundary enforcement if it is not wired,
  and `codefall-equip` after a docs-only run.
- **Last, what the user does next**: merge the pull request, what is owed above, then
  `/codefall-specify VISION-NNN`.

## Rules

- **Nothing is designed here.** No entities, no schema, no feature behavior. Product detail goes to
  the decision log's Parking lot.
- **Shape decides the architecture; names decide folder names.** Several separable capabilities is
  package-by-component whether or not anyone has named them.
- **Never show or improvise a profile that is not supported.** Planned profiles are exits, and
  ADRs for an unsupported language are never hand-authored.
- **A consult proposes a profile; the user picks it.** Only when the description left it open.
- **Recommend only what an ADR supports.** Where Codefall has no stance, present the options flat.
- **The inherited ADRs ship Accepted.** Amend only when the interview requires it, and report every
  amendment.
- **Provenance is written every time and never hand-edited.** Hashes are computed, never invented.
- **Never scaffold into a non-empty directory unannounced, and never overwrite an existing path.**
- **Boundary enforcement is owed on day one.**
- **A code tier equips the project**, per `codefall-equip`; docs-only output names it as owed.
- **Verify what you emitted**, including that the boundary rules fail on a deliberate violation.
- **The doc workflow this seeds: discuss → decision-log → ADR → scoped `AGENTS.md` → code.**
  `codefall-design` picks up from here.
