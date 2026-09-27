# Decision log

The scratchpad for decisions in flight. Settled decisions graduate to an ADR in `docs/adrs/` and are
struck from `Open`; this file never holds the reasoning that belongs in an ADR.

## Locked

Decided at scaffold, 2026-08-16.

- **One surface.** `codefall-cli` is a single Go CLI — no backing service, no UI, no seam. Profile:
  `go`. Inherited: ADR-BASE-01, ADR-BASE-02, ADR-GO-01, ADR-GO-02, ADR-GO-03.
- **Module path** — `github.com/lividlabs/codefall-cli`. No git remote set yet.
- **One app, one package.** Single Go module at the repo root; `cmd/codefall/` is the only binary.
  Codefall has no house opinion on repo layout, and one app doesn't need one.
- **Package-by-component**, chosen over ports-and-adapters even though the capabilities are unnamed —
  see the "For this project" section added to ADR-BASE-02.
- **No components yet, and no example one.** `internal/` is empty by choice: a placeholder component
  was scaffolded, reviewed, and deleted as ceremony that would be replaced wholesale in the next
  release. `main.go` prints the banner directly and wires nothing.
- **No dependencies.** `go.mod` requires nothing. `samber/do` (ADR-GO-01) and `samber/mo`
  (ADR-GO-03) are governing decisions, not yet imports — they arrive with the code that needs them.
- **Boundary enforcement is wired**, ahead of the code it will govern: `internal/` facades
  (compiler) plus `depguard` strict allow-lists in `.golangci.yml` (ADR-GO-02), both proven against
  deliberate violations at scaffold time.
- **Contracts cross facades, not entities.** A facade returns a data shape declared in the
  component's root package, never a `domain` type — recorded as `ADR-001`. Go's `internal/` rule
  does not cover this: it restricts naming a package, not holding a value, so a leaked entity
  compiles for a caller that cannot import it. Neither the compiler nor `depguard` can catch it.
- **`ADR-BASE-03` (extraction readiness) is deliberately not part of this project's set.** Its gate
  is whether a surface could ever be split into a service, and a CLI cannot. Four of its five rules
  have no referent here; the fifth is `ADR-001`. If a backing service ever appears, revisit.
- follows codefall `0.4.0` as of `2026-08-19`
- grafted codefall `0.4.0` on `2026-08-19` — took the `ADR-BASE-02` revision (re-applied in place,
  carrying the `For this project` amendment forward; no supersession, as no code had been written
  against it) and the `AGENTS.md` workflow defaults. Skipped `ADR-BASE-03` as inapplicable.
  `ADR-BASE-01` and `ADR-GO-01/02/03` were already current.
- **ADR-GO-01 amended, 2026-08-19.** Its claim that the composition root "is the only place that
  names concrete implementations" contradicted the layout: `cmd/` cannot import
  `internal/<component>/internal/*`. Concretes are named in each component's registration function;
  `main` calls those functions and resolves facades only. The same fix is owed to the plugin
  templates (go and typescript-react profiles).
- **CLI libraries, 2026-08-27.** Cobra for the command tree, run through Fang; Lip Gloss v2 for
  styled output where it helps; Huh v2 for prompts. Charm's v2 generation only, on the
  `charm.land/…/v2` paths — the two majors do not mix. Recorded as `ADR-002`. Bubble Tea arrives
  transitively with Huh and is the answer if a TUI is ever needed; that is a consequence of ADR-002,
  not a new decision. None of it is imported yet — each library arrives with the code that first
  calls it.
- **First component, 2026-08-27.** `internal/doctor/` is the first capability under `internal/`:
  `codefall doctor` reports whether a directory has `.codefall/settings.json`, Beads, and an
  authenticated `gh` with the scopes codefall needs. It reports and never repairs — it prints the
  remedy for every unmet check and `codefall init` will be the thing that runs them. The composition
  root now exists: `cmd/codefall/main.go` builds the injector, calls `doctor.Register`, mounts
  `doctor.Command`, and hands the tree to `fang.Execute`. The layer rules were re-proven the way
  ADR-GO-02 requires — a facade reach-around from `main` failed `go build`, and a Cobra import from
  `internal/doctor/internal/application/` compiled and failed `golangci-lint run` on the
  `application-layer` rule. Facade shape, settling ADR-002's open question:
  `Register(do.Injector)` plus `Command(do.Injector) *cobra.Command`, with
  `Commands(do.Injector) []*cobra.Command` when a component has several top-level commands. No
  report contract crosses the facade yet, because nothing outside doctor consumes one; add
  `doctor.Report`/`doctor.CheckResult` with the first external consumer, likely `init` (ADR-001).
  Now imported: Cobra, Fang, `samber/do`, `samber/mo`, `charm.land/lipgloss/v2`, and
  `github.com/charmbracelet/colorprofile`. Huh still waits for the first prompt.
  Bubble Tea and Bubbles v2 joined with doctor's spinner, in `presentation/` only — an ADR-002
  consequence rather than a new decision, and not the TUI question a later ADR would settle.
  `schemas/settings.schema.json` is the published definition of `.codefall/settings.json`; the Go
  validator is hand-written in `domain/` so no schema library ships in the binary, and a test holds
  the two equal. One correction to ADR-002: Fang v1.0.0 does import `charm.land/lipgloss/v2`, so its
  note that Fang imports neither Lip Gloss major is out of date. Harmless — it is the v2 major, and
  the v1 generation stays out of the graph.
- **Second component, 2026-08-27.** `internal/initcmd/` is `codefall init`, the thing that creates
  what doctor checks. The package is named `initcmd` rather than `init` because `init` cannot be
  imported unaliased — Go rejects `import ".../internal/init"` because `init` must be a func — and
  `initcmd` keeps the directory named after the command it holds, the maintainer's choice on review
  over the earlier `setup`. The command is `init`, unaffected.
  A run is an ordered list of steps, each reporting Done or Skipped through an `Observer` the
  presentation layer implements twice: a spinner with the running step's title when stdout is a
  terminal, plain lines everywhere else. This change carries one step, `settings`; installing the
  Claude Code plugin and running `bd init` are the next two, appended to the same list. Settings
  that already exist are left alone and the step reports Skipped — `--force` rewrites them — so init
  is safe to run again once it has more steps than this. `--harness` is accepted and validated now,
  with `claude-code` the only value, because the steps after this one are the ones that need it.
  Huh v2 is now imported, the first prompt in the project (ADR-002). The tracker survey lists Jira
  and Linear with "(not yet available)" in the label and refuses them in the field's own validation:
  Huh v2 has no disabled option, and a tracker that is not offered at all reads as a tracker nobody
  thought of. The layer rules were re-proven the way ADR-GO-02 requires — a Cobra import from
  `internal/initcmd/internal/application/` compiled and failed `golangci-lint run` on the
  `application-layer` rule. Still no report contract crosses a facade: init does not consume
  doctor's.
- **Init's plugin step, 2026-08-27.** `codefall init` installs the codefall plugin for Claude Code
  as its second step, by running the harness's own CLI — `claude plugin marketplace add
  lividlabs/codefall-plugin --scope project` and then `claude plugin install codefall@codefall
  --scope project -y`. Both are project scope, so `.claude/settings.json` carries the marketplace
  declaration and the enabled plugin and everyone who clones the repository gets them; a user-scope
  install would work for whoever ran init and for nobody else. What is already done is read out of
  that file rather than asked of `claude plugin list --json`, whose `enabled` field is computed from
  the caller's working directory and stamped onto every row, so it misreports. The CLI merges into
  the file, preserving keys it does not own, which is why init runs the CLI rather than writing the
  file itself. The plugin's identifiers are domain constants because they are facts about codefall;
  the CLI's name and argument shapes are application, because they are how a tool is asked. The step
  switches on the harness with `claude-code` as its only case — presentation has already refused
  every other value, so the default is unreachable and is where the next harness lands. Every run
  now starts with a preflight that checks the tools it will reach for are on PATH, before the first
  step: a tool that turns up missing halfway through leaves the project half set up, which is the
  one state init exists to avoid. It reports every missing tool at once rather than the first, and
  `bd` and git join `claude` there with the Beads step. Every tool init sets up is required on every
  run, including a run where every step would skip, because a run that cannot act is not a run that
  finished. The step's two halves are decided separately: the marketplace is declared whenever the
  project does not already declare it, and the plugin installed whenever it is not already enabled,
  because a plugin enabled with no project-scope marketplace beside it is exactly what a user-scope
  install leaves behind — it works for whoever ran it and for nobody who clones the repository.
- **Init's Beads steps, 2026-08-27.** `codefall init` finishes by initialising Beads and giving
  Claude Code the hook that primes a session with what Beads knows — the third and fourth steps, and
  the last of them. The invocation is `bd init --non-interactive --skip-agents`, which writes
  `.beads/` and appends a Dolt block to `.gitignore` and nothing else. `--skip-agents` is what keeps
  it to that: without it bd appends its own section to `AGENTS.md` and `CLAUDE.md` and writes
  `.claude/settings.json`, `.codex/`, and `.agents/`, and codefall owns the first two. bd has no
  template hook for `CLAUDE.md` at all — `--agents-template` applies only when `AGENTS.md` does not
  exist yet — so there is no invocation that keeps bd's context and codefall's own words in the same
  file. The text bd would have appended is
  [`beads-section-minimal.md`](https://github.com/gastownhall/beads/blob/6c124203e771433a3550c348771a5b5e27fd3c21/internal/templates/agents/defaults/beads-section-minimal.md);
  whoever wants codefall's own `AGENTS.md` to say the same things starts there.
  The session hook is how a Claude Code session gets that context instead: a `SessionStart` entry
  running `bd prime --hook-json`, the hook bd installs for the same purpose, written by init
  rather than by `bd setup claude` — that command cannot be told to leave `CLAUDE.md` alone. `bd
  setup claude --check` accepts what init writes (`✓ Project hooks installed`) and complains only
  about the missing `CLAUDE.md` section, which is the intended difference. The hook step decodes
  `.claude/settings.json` into a plain object so that every key the file has survives being written
  back; key order is the one thing that does not, because `encoding/json` sorts it. It is written
  back through a `json.Encoder` with `SetEscapeHTML(false)`, because `json.Marshal` escapes `<`, `>`
  and `&` and would silently rewrite a permission rule like `Bash(a && b)` in a file codefall does
  not own. It runs after the plugin step because both write that file, and the harness CLI's own
  merge goes first.
  Three guards run in preflight, because bd init decides things for itself that nobody asked.
  It commits what it wrote with `git commit --no-verify` and no pathspec, and before that it stages
  `.gitignore`, `AGENTS.md`, `CLAUDE.md`, `.claude/settings.json`, `.codex`, and `.agents` by name
  when they exist — so both anything already in the index and any uncommitted change to one of those
  paths lands in a commit that says it initialised Beads, under bd's message rather than its
  author's. `--stealth` is the only flag that stops the commit and it also stops tracking `.beads/`,
  which is not the trade. So a run refuses to start while `git diff --cached --quiet` reports a
  staged index, and refuses again when `git status --porcelain` over exactly those six paths reports
  anything — staged, unstaged, or untracked — naming the paths git reported. Both run only when bd
  init is going to run at all, because otherwise what the directory has waiting is nobody's business
  but whoever left it there. And bd init run outside a repository silently runs `git init` first, so
  a run also refuses a directory that `git rev-parse --is-inside-work-tree` does not answer `true`
  in — the word, not the exit code, because inside a bare repository and inside `.git` itself git
  prints `false` and exits 0. The path guard runs in preflight rather than later so that the plugin
  step's own write to `.claude/settings.json` is not what it catches: that write is codefall's to
  make and bd committing it is harmless, and the difference between it and somebody's uncommitted
  edit to the same file is exactly when the question is asked.
  `bd info` is the question that decides whether there is anything to do, the same question doctor
  asks and for the same reason — `BEADS_DIR` relocates `.beads/`, so looking for the directory asks
  something else. Re-running `bd init` where it has already run is an error rather than a no-op, so
  asking first is what keeps init safe to run again.
- **Init's Beads section, 2026-08-27.** codefall ships its own `AGENTS.md` section about Beads and
  `codefall init` writes it, as the fifth step and the last. `--skip-agents` is what makes this
  necessary: bd would append
  [`beads-section-minimal.md`](https://github.com/gastownhall/beads/blob/6c124203e771433a3550c348771a5b5e27fd3c21/internal/templates/agents/defaults/beads-section-minimal.md)
  to `AGENTS.md` and `CLAUDE.md`, and that text describes a setup codefall does not create. There is
  no invocation that keeps bd's context and codefall's own words in the same file — `--agents-template`
  applies only to an `AGENTS.md` that does not exist yet, and bd has no template hook for `CLAUDE.md`
  at all — so codefall writes the section itself. It lives at
  `internal/initcmd/internal/domain/beads_section.md`, embedded with `//go:embed` and exposed as
  `domain.BeadsSection`: the words codefall says are a fact about codefall, and `embed` is the
  standard library, which the domain layer's allow-list already permits. How the section is spliced
  into somebody's file is the application layer's, along with every other encoding.
  The section opens and closes with HTML comment markers, `<!-- BEGIN CODEFALL BEADS -->` and
  `<!-- END CODEFALL BEADS -->`, which is what makes a second run replace it in place rather than
  append a second copy — everything on either side of the pair survives byte for byte, so the
  project's own words are never this step's to rewrite. A file with an opening marker and no closing
  one stops the run: where codefall's words end and the project's resume is not something to guess
  at. A file with no markers keeps what it says and gains the section at the end.
  The step runs after `bd init` rather than before it, because bd stages `AGENTS.md` and `CLAUDE.md`
  when they exist and commits what it staged under its own message. Editing them afterwards leaves
  the change uncommitted, which is where it belongs: the section is the author's to commit.
  `CLAUDE.md` is created only for the `claude-code` harness and only when it is missing, holding one
  line that points at `AGENTS.md`. That is the file convention the codefall plugin's `scaffold` skill
  states — `AGENTS.md` holds the rules, and a harness that reads `CLAUDE.md` is pointed at them —
  and it is what this repository's own `CLAUDE.md` says. A `CLAUDE.md` that already exists is left
  alone whatever it holds, because writing a pointer over somebody's rules throws them away rather
  than pointing at them.

- **Shared modules, 2026-08-27.** `internal/shared/` exists, with the two modules the second
  component turned out to have copied from the first. `internal/shared/ui/` owns the palette, the
  styles, the marks, the colour-profile writer, and the spinner runner; a component maps its own
  vocabulary onto a `ui.Tone` and owns nothing else about how a line looks — doctor maps a check's
  status, initcmd maps a step's outcome and keeps its own dash for a skipped step, because nothing
  is wrong there. `ui.RunWithSpinner` is one entry point covering both paths, so neither command
  re-implements the branch between a terminal and a pipe: it takes the writer, draws finished lines
  under the spinner and writes them again once Bubble Tea has cleared its frames, and writes them as
  they arrive when there is nothing to spin on. `internal/shared/process/` owns running an external
  tool and reaching the file system, including the `0o755`/`0o644` modes. Each component's
  `infrastructure/` is now an adapter over it: the gateway interface and the result type stay the
  consumer's to declare (ADR-GO-01), so the two `CommandRunner` interfaces and the two
  `CommandResult` structs remain, and what the adapter adds is the translation.
  What deliberately stays duplicated is `firstLine` and the settings-format constants in both
  `domain/` packages: the layer rules keep `domain/` and `application/` off shared modules
  altogether, and moving those would mean either widening those allow-lists or moving the constants
  out of the layer that owns them. Both copies are pinned by each component's own schema test
  against `schemas/settings.schema.json`, which is what makes the duplication safe rather than
  merely tolerated. (Reversed the same day by **Pure shared modules** below, which widened the
  allow-lists for modules that do not need widening — the option not seen here.)
  The `shared-modules` rule now matches files and was proven, once per deny entry;
  proving it needs a throwaway `internal/shared/proof/` package, because a real shared module
  importing a component is an import cycle and so a compile error rather than the lint failure
  ADR-GO-02 asks for. `domain/` and `application/` were re-proven against an `internal/shared/ui`
  import the same way.

- **Pure shared modules, 2026-08-27.** `domain/` and `application/` may now import a shared module
  that is itself pure — one under `internal/shared/` importing only the standard library and
  `samber/mo`, the same imports `domain/` already has — which is what lets the settings format and
  the small helpers the two components had copied become one definition. The blanket denial that
  produced those copies was aimed at `ui` and `process`, which import Charm and `os/exec`, and it
  could not tell them apart from a module that adds nothing to the inner rings' dependency surface;
  each pure module is now named four times in `.golangci.yml` — the two allow-lists plus the
  `pure-shared-modules` rule's `files` and `allow` entries, the second of which is what also lets it
  import another pure module — so the property the permission rests on is checked rather than claimed. Recorded as [`ADR-003`](adrs/ADR-003-pure-shared-modules.md), a new ADR rather
  than an amendment to ADR-GO-02, which it refines and does not supersede.
  Two modules moved under it. `internal/shared/settings/` is now the one definition of the
  `.codefall/settings.json` format — the version, the schema id, the tracker names, the repository
  pattern, the field tables, `Validate`, and the field-level `ParseTracker`, `ValidateRepo`, and
  `ValidateProject` that a form calls while somebody is still typing. Doctor's `domain/settings.go`
  is gone; initcmd's keeps the `Settings` value object and `NewSettings`, which is where the
  combinations that make sense for a run are decided, and takes every constant and rule from the
  shared module. `Harness` stayed initcmd's, in its own file: nothing else in the project has an
  opinion about which harnesses can be set up. `internal/shared/text/` holds `FirstLine`, which both
  application layers had written identically. The two facade-level schema tests became one
  `schema_test.go` beside the format it pins, carrying every assertion either of them made — the
  reason it could not live there before was that a `domain` test cannot import `os`, and a pure
  shared module is not an inner layer. Names changed with the move, because the package name now
  carries what the prefixes used to say: `SettingsVersion` and `SettingsSchemaID` are
  `settings.Version` and `settings.SchemaID`, `ValidateSettings` is `settings.Validate`, and
  `RequiredSettingsFields` is `settings.RequiredFields`.
  Both halves of the new configuration were proven the way ADR-GO-02 requires, each compiling and
  failing `golangci-lint run`: `charm.land/lipgloss/v2` imported inside `internal/shared/settings/`
  failed the `pure-shared-modules` rule, and `internal/shared/ui` imported from
  `internal/initcmd/internal/domain/` failed the `domain-layer` rule — while that same package's
  import of `internal/shared/settings`, which is what the whole change rests on, passes.
- **Where init gets the repository, 2026-08-28.** `--issues-repo` is a flag nobody should have to
  type in the repository it names. Init asks two sources for it, in order: `gh repo view --json
  nameWithOwner`, and then `git remote get-url origin`. gh goes first because it answers with the
  repository as GitHub knows it today, which is right after a rename and for a fork whose remote
  still names the upstream; the remote goes second because it needs neither authentication nor
  network, and git is already required on every run while gh is required on none. Before the
  fallback, a machine without gh — or with gh installed and signed out — had to be told what the
  `.git/config` in front of it already said. The answer is used the same way either source produced
  it: it pre-fills the survey field when somebody is answering, and stands in for the flag when
  nobody is. A remote is only an answer when its host is `github.com` and what it points at passes
  the same `ValidateRepo` the flag passes — anything else, including an SSH host alias like
  `github.com-work` that a multi-account setup uses, is None and the flag is how the person says so.
  Only `origin` is consulted: a project whose GitHub repository is under some other remote name is
  one its owner knows better than init does.

- **Unified hooks, 2026-09-11.** `extensions/hooks/` is now the one source of truth for what init
  registers with each harness: `hooks/shared/` holds the script every guard runs
  (`codefall-block-merge-to-main.sh`, `--antigravity` selects the stdout-JSON deny contract that
  harness wants), and `hooks/<harness>/` holds the definition — `claude` and `codex` in the
  shared event shape (`PreToolUse` guard plus `bd prime --hook-json` on `SessionStart`),
  `antigravity` keyed by hook name with the guard only, and `opencode` as the plugin file that
  delegates the guard to the shared script and primes `session.created` sessions through
  `session.prompt` with `noReply: true`. Script paths resolve at the repository root two ways:
  Claude and Codex quote `$(git rev-parse --show-toplevel)`, which both harnesses run through a
  shell, and Antigravity uses the workspace-relative `./.agents/…` its own examples use — its
  command execution is not documented to expand `$(…)`, so nothing there depends on it. Either way
  the definitions carry literal commands and nothing rewrites paths.
  The extension step copies everything except those four definition directories — it is handed the
  same list the hook step reads — so `.claude/` gets `hooks/shared/` and `.agents/` gets the same,
  and a definition file lands only where the harness actually reads it. A copied script is made
  executable where it lands: the embedded tree carries no modes to copy, and a hook command names
  the script by path. The step's table
  (`hookSpecs`) holds each harness's source path, destination, and one of two formats: merge or
  copy. The merge is one recursive fold — objects in, arrays appended, scalars the destination
  already says kept — so Claude's `settings.json` keeps its permissions and Codex's `hooks.json`
  keeps its `description`, an Antigravity hook the project disabled stays disabled, and a key
  holding null says nothing the same as an absent key. Arrays
  dedupe per entry on the matcher together with the command string, the command found recursively
  because the formats keep it at different depths: the same command under another matcher still
  leaves the event unguarded, so it joins rather than counting as present. The harness's entry
  joins the project's under the same event key. OpenCode
  copies its plugin instead and skips when the bytes match; a harness with no table entry is
  skipped rather than errored. `domain.BeadsHookEvent`/`BeadsHookCommand` went away: what init
  installs is a property of the definition files now, and the format each file wants lives nowhere
  but the embedded tree. The `ExtensionFetcher` interface is renamed `ExtensionSource`: one
  gateway role for the embedded payload, fetched whole for the extension step, read one file at a
  time for the hook step.
  What an upgrade deliberately does not do is remove files the tree no longer ships. A project
  installed by the previous version keeps `.claude/hooks/hooks.json` and the un-namespaced
  `.claude/hooks/block-merge-to-main.sh`; the flat `hooks.json` was never a location Claude Code
  reads, so the leftover is inert, and the new registration is the only one that runs. Pruning
  manifest-recorded files the tree has since dropped is a separate change with its own rules —
  a manifest that means "mine to delete" is a stronger claim than one that means "mine to have
  written" — and is not made here.

- **Installing below the repository root, 2026-09-11.** `codefall init` run below the root of a
  git work tree asks, as its first question, whether to install in the working directory or at the
  root; `--location here|root` answers without asking, and a run with neither and no terminal
  fails naming the flag. At the root, and outside a work tree, there is no question. The case this
  serves is a large monorepo where one team wants codefall in its own part of the tree without
  setting it up for everyone else. Before this, a subdirectory run passed preflight, which asks only
  whether the directory is inside a work tree, installed everything under the subdirectory, and
  registered Claude and Codex guards naming the root's `.claude/` and `.agents/`, where nothing had
  landed — so every Bash call raised a hook error and the guard never ran.
  The **Unified hooks** entry's "nothing rewrites paths" no longer holds for those two: the hook
  step asks git for the install directory's path below the root (`rev-parse --show-prefix`) and
  writes it after `$(git rev-parse --show-toplevel)/` in every command that names the root. An
  install at the root has an empty prefix, so it registers exactly the command the definition ships
  and a rerun still recognises it. A prefix holding a character that keeps its meaning inside double
  quotes (`"`, `$`, backtick, backslash) is refused by preflight rather than escaped. Antigravity's
  `./.agents/…` is workspace-relative already and is unchanged; the OpenCode plugin now finds the
  guard from its own directory (`import.meta.dir`) rather than from the worktree root. Which
  directory each harness has to be started in to read a subdirectory install is that harness's own
  configuration discovery, and is not verified here for any of them.
  Beads needs no second database: `bd info` searches upward, so a subdirectory of a repository
  whose root has one reports that root database and the beads step skips, as it does anywhere else
  Beads is already initialised. A subdirectory that is the first install in its repository does get
  a database of its own, and there the run adds `--skip-hooks`. `bd init` otherwise points the
  clone's `core.hooksPath` at its own `.beads/hooks` — one setting for the whole repository, which a
  subdirectory install has no business claiming from everyone working in that clone, and which bd
  1.2.2 fills in with a path under the root even when it wrote the database below it, so the
  directory it names need not exist. Verified both ways against bd 1.2.2: without the flag a
  repository's own `core.hooksPath` is replaced by a path that does not exist, and with it the
  setting is left as it was. At the root the setting is bd's to make and the flag is not passed.

- **What the manifest asserts, 2026-09-11.** `.codefall/manifest.json` now means one thing: a run
  that finished, for the harness it names, at the version it names. It is written after the last
  step rather than in the extension step, and the gate that reports "already up to date" compares
  the harness as well as the version.
  Both halves were wrong in the same way — the gate answered from a record that claimed more than
  it knew. Written in step two, the manifest stamped the current version with the hook and
  AGENTS.md steps still to come: a run that failed at either left a project half set up and a
  record saying it was current, and the next run reported there was nothing to do. Reproduced
  against a project whose `.claude/settings.json` held `"hooks"` as a string: the run failed at the
  hook step, the rerun printed "already up to date", and the guard was never registered. Comparing
  the version alone did the same to a second harness: a project installed for `claude-code` and
  then run with `--harness antigravity` was skipped with no `.agents/` ever written, though the
  manifest had recorded the harness all along.
  The file list the manifest records is now carried out of the extension step as a local of the
  run, not a field of the use case, which every run of the process shares. A failed run leaves no
  manifest at all, so the next run repeats every step, which they are all built to tolerate — and
  the upgrade question is asked only when the version moves, not when the harness differs.
  What this does not do is record more than one harness. A project set up for two keeps the harness
  of the most recent finished run, so installing for the other one always does its work again.
  Recording a harness per install is a larger change to what the file is for, and the from/to
  comparison an upgrade reports does not need it.

- **What the hook merge calls the same entry, 2026-09-11.** The merge knew an entry by its matcher
  together with the literal command strings it ran, and appended anything else. Four things follow
  from that, and all four are fixed together because they are one question: what makes a
  registration codefall's own.
  A shipped command that changes read as an unrelated registration, so an upgrade appended the new
  one and left the old one running beside it — and no code path could ever remove it, since nothing
  recorded which entries were codefall's. The subdirectory install above makes this concrete: a
  project registered before it carries a guard naming the repository root, and the upgrade would add
  the corrected command without taking the broken one away. **An entry naming a codefall script
  now replaces the destination's entry naming that same script under the same matcher.** The
  script's file name is the identity, which is what the `codefall-` prefix is for; the rule is
  stated in `extensions/AGENTS.md`, beside the tree it governs. The alternative was recording every
  registered entry in `.codefall/manifest.json`, which would have made that file mean "mine to
  delete" — a stronger claim than "mine to have written", and one the **Unified hooks** entry had
  already set aside as its own change. It stays set aside.
  A project that primes Beads under a narrower `SessionStart` matcher had codefall's match-all entry
  appended beside it, which primed on every other session source and twice on the one the project
  chose. **A match-all entry is now covered by any entry running its commands under any matcher.**
  This restores what the pre-`623b53d` `hasBeadsHook` did, in terms that hold for any harness rather
  than for one command. A narrower incoming matcher still joins: the guard registered for `Edit`
  leaves `Bash` unguarded, and that is what the test with both matchers pins.
  An entry whose commands were only partly registered was appended whole, running the command the
  destination already had twice. **It now joins with only its unregistered handlers**, and an entry
  that keeps its command on itself rather than in a handler list joins whole or not at all, because
  there is nothing to split.
  **A destination holding an object or an array where the definition has a scalar is now reported**
  rather than quietly kept, which is what the same disagreement in the other direction already did.

- **Adding the execute bit rather than setting the mode, 2026-09-11.** A copied script was chmodded
  to `0755` on every run, which took a decision away from the project: a script tightened to its own
  user came back world-readable and world-executable at the next upgrade, with no line in the report
  to say so. `MakeExecutable` now adds execute wherever the file is already readable — `0600`
  becomes `0700`, `0644` becomes `0755` — and does nothing at all when the mode already allows it,
  so a rerun with nothing to change cannot fail on a directory that refuses `chmod`. The helper also
  stops passing `dirMode` for a file, which was the right bits under the wrong name.
  Which files get the bit does not change: every `.sh` in the tree, not only `hooks/shared/`. The
  hook commands name the guard by path, and `codefall-implement`, `codefall-specify` and
  `codefall-design` each invoke `shared/preflight.sh` by path — a script nobody can run is a skill
  that fails halfway through.
  Two tests were restored to pinning what they claim. The extension step's test now asserts the
  exclude list it hands `Fetch`, so a fetch that stopped excluding the per-harness definitions
  fails rather than silently shipping every harness's definition into every project. The refused
  merge's error table now also asserts the destination is byte-for-byte untouched, because the merge
  builds its document in place before a later key can fail it, and the error alone never said the
  file survived. Both were proven against the mutation each describes.

- **Harnesses are shared vocabulary, 2026-09-16.** `internal/shared/harness/` is now the one
  definition of which coding harnesses codefall can set up and where each one reads skills, as a
  third pure shared module. This reverses the note in **Shared modules, 2026-08-27** that `Harness`
  stayed initcmd's because nothing else in the project had an opinion about which harnesses can be
  set up: doctor is about to have one. Once a project records the harnesses it chose in
  `.codefall/settings.json`, doctor reads that field and reports on the directories it names, and
  then neither component can be the one that decides where those directories are.
  What moved is the five names, the sorted roster, and the destination directory per harness:
  `Harnesses()` and `ParseHarness` are `harness.All` and `harness.Parse`, and `extensionDestDirs` is
  `harness.SkillsDir`, an Option because a name `Parse` has already refused has no directory to
  report (ADR-GO-03). That one map is also the roster, so a harness codefall can set up is a harness
  with somewhere to install and the two cannot disagree. What stayed in initcmd is `hookSpecs` —
  which file in the embedded tree a definition comes from, and whether it is merged or copied, is how
  init registers a hook rather than what a harness is. Claude Code's hook destination is now the
  literal `.claude/settings.json` its three siblings already used, rather than being composed from
  the skills directory that moved.
  Nothing a user can see changes here: `--harness` still defaults to `claude-code`, and a run still
  installs for exactly one harness. This is the first of four changes that together let a project
  choose several, and it lands on its own so the move is reviewable without the behaviour change on
  top of it.
  Both halves of the new module's configuration were proven the way ADR-GO-02 requires. A
  `charm.land/lipgloss/v2` import in `internal/shared/harness/` compiled and failed
  `golangci-lint run` on the `pure-shared-modules` rule, and the same run left a deliberate
  `harness` import from `internal/initcmd/internal/domain/` unreported, which is what proves the
  `domain-layer` allow entry names the module. The `application-layer` entry needs no throwaway
  proof: the extension and hook steps import the module, and lint passes.

- **A run installs for a set of harnesses, 2026-09-16.** `Request.Harness` is `Request.Harnesses`,
  and the extension and hook steps each do their work once per harness the run is for. This
  supersedes the closing note of **What the manifest asserts, 2026-09-11**, which set recording a
  harness per install aside as a larger change to what the file is for. This is that change.
  `.codefall/manifest.json` now holds an entry per harness — the version that installed it and the
  files it wrote — and a run updates the entries for the harnesses it installed for while leaving
  every other entry as it was. A project set up for two keeps both records, where the file naming one
  harness lost the first record as soon as the second install finished, and the next run for that
  first harness repeated work it had already done. The recorded paths are relative to the directory
  init installed in rather than to the harness's own skills directory: four of the five harnesses
  read `.agents/`, so a bare `skills/…` would not say which directory it landed in.
  Every step reads the set through one function that sorts it and drops repeats, so the order a flag
  or a survey happened to collect them in cannot change what a step does or what it reports. The
  extension step copies once per directory rather than once per harness, because four of them share
  one, and every harness that reads a directory records the files in it. The hook step registers with
  each harness in turn and folds the outcomes: Done when any registration changed something, with
  every harness's own sentence in the detail. A run for one harness reports exactly what it reported
  before, which is what keeps this change invisible. The `CLAUDE.md` pointer is written whenever
  Claude Code is among the harnesses rather than when it is the harness. A run for no harness is
  refused by preflight, where nothing has been touched yet, rather than by six steps in turn
  reporting they had nothing to do.
  The no-op gate compares per harness: a rerun is a no-op when every harness it is for is recorded at
  this binary's version, and the upgrade question is asked only when one of them is recorded at a
  different version — a harness with no record at all is work to do rather than an upgrade.
  Nothing a user can see changes yet, because presentation still passes the one harness `--harness`
  names and that flag still defaults to `claude-code`. Choosing several is the next change in the
  stack. A manifest written before this one decodes to an empty record, so the next run repeats every
  step, which they are all built to tolerate; no project has a manifest yet, so nothing is migrated.

- **The project says which harnesses it uses, 2026-09-16.** `.codefall/settings.json` records a
  required `harnesses` array, `init` asks which harnesses to set up, and `--harness` no longer
  defaults to `claude-code`. The default was the whole of the old decision: a pflag carrying one
  cannot be told apart from one the user typed, so every run that did not pass the flag installed for
  Claude Code silently, and the survey had no harness question at all. Which harnesses a project uses
  is not something codefall can work out — four of the five share one directory, and codefall writes
  those directories itself — so it is asked, and the answer is recorded.
  The field lives in the shared settings module beside the tracker, because both components need it:
  init writes it and doctor is about to report on it (ADR-003). The validator refuses an empty array
  rather than reading it as "none", and refuses a name codefall cannot set up; the schema says the
  same two things with `minItems` and an enum, and the schema test holds the two definitions equal.
  This is also `internal/shared/harness/`'s first consumer outside initcmd: `settings` imports it to
  validate the names, which is the pure-module-importing-a-pure-module case ADR-003 allows and the
  fourth `.golangci.yml` entry permits.
  The survey asks for the harnesses first, as a multi-select with nothing selected and at least one
  answer required. Nothing is preselected on purpose: a preselected option is the same decision made
  on someone's behalf that the flag's default was. A scripted run passes `--harness`, repeated or
  comma-separated, and a run with no terminal and no flag fails naming it, which is the rule every
  other prompted value already follows (ADR-002).
  A rerun does not ask again, because the harnesses are in the settings and init reads them back — so
  the flag is needed the first time, or to add one. Settings that record none, which is what a file
  written before the field existed looks like, are refused with the flag named rather than filled in
  with a guess.
  This breaks twice over: a settings file without `harnesses` fails doctor's completeness check until
  init is run again, and a scripted run that relied on the flag's default now fails naming the flag.
  `codefall create` inherits the second, because it adopts init's flags. No project has a settings
  file yet, so nothing is migrated.

- **Doctor reports on the harnesses the settings name, 2026-09-16.** A fourth category, one check:
  codefall's extension is installed for every harness `.codefall/settings.json` records. It fails
  rather than warns, because a harness the project chose and codefall was never run for has no skills
  and no hooks there — every codefall verb is missing in a harness somebody is using — and the remedy
  is `codefall init`.
  What it stats is `hooks/shared/` under each harness's own skills directory, a path the shared
  harness module names and the initcmd facade test pins against the real embedded tree. Nothing but
  codefall writes that directory, where `.claude/` and `.agents/` exist in plenty of projects that
  have never run codefall — and `.agents/` is read by four of the five harnesses, so its presence
  identifies none of them. The check is deliberately not keyed to a skill's name: the skills are
  still being renamed, and a check that broke on a rename would fail on projects that are set up
  correctly.
  It reads the settings itself rather than taking them from the settings group, and runs only when
  that group found the file complete: settings doctor has already complained about say nothing about
  which harnesses were chosen, and a second complaint about the same file would be noise. A skipped
  check is absent from the report, which is what every other prerequisite failure already does. No
  presentation change was needed — `Sections()` groups by first appearance, so a new category reports
  itself.
  What this entry did not do is warn about a harness codefall finds installed that the settings do
  not name. That needs doctor to read `.codefall/manifest.json`, which means promoting the manifest
  format to a shared module, and codefall's stance is that it never removes what it wrote, so the
  remedy is somebody's to carry out by hand. **Doctor reports an install the settings dropped,
  2026-09-16** below is that change.

- **Doctor reports an install the settings dropped, 2026-09-16.** A second check in the Harnesses
  category: the manifest records an install for a harness `.codefall/settings.json` no longer names.
  A project set up for two harnesses that later drops one keeps every file codefall wrote for it,
  because codefall only ever writes what it owns and never deletes — so the skills for a dropped
  harness stay where an agent reading that directory will find them, and the hook codefall registered
  for it still runs.
  `internal/shared/manifest/` is now the one definition of the `.codefall/manifest.json` format, a
  fourth pure shared module, for the reason the settings format is one: initcmd writes that file and
  doctor reads it, and a file initcmd wrote that doctor could not read would be the worst bug in
  either (ADR-003). initcmd's private `manifest` and `harnessInstall` structs and its
  `domain.ManifestName` are gone; `Decode`, `Encode`, `Recorded` and `Versions` belong to the format,
  and `Installation` stays initcmd's because what the upgrade gate compares is initcmd's business. The
  format itself is unchanged, so no project's file needs anything done to it.
  The check warns rather than fails, because nothing stops working. Its remedy points at the manifest
  rather than naming a directory to delete: the manifest is what records which files codefall wrote
  for that harness, and harnesses share directories — four of the five read `.agents/` — so the
  directory a dropped harness read may still be another's. Which of those files are safe to remove is
  the reader's judgement, not doctor's.
  A manifest that is missing, unreadable, or no longer a manifest records no install that could be
  left over, so the check is absent from the report rather than complaining a second time: doctor has
  no check for the manifest's own shape, the install check has already reported whatever is missing on
  disk, and init fails on a file it cannot read the next time it writes one.
  The module's four `.golangci.yml` entries were proven the way ADR-GO-02 requires. A
  `charm.land/lipgloss/v2` import in `internal/shared/manifest/` compiled and failed
  `golangci-lint run` on the `pure-shared-modules` rule, and the same run left a deliberate `manifest`
  import from `internal/initcmd/internal/domain/` unreported. That second proof matters here in a way
  it did not for `harness`: no `domain/` package imports the manifest format, so nothing else would
  have exercised the entry.
- **Local environment: equip and refresh, 2026-09-16.** Pulling `main` before starting work has
  consequences the pull itself does not perform — a migration the local database needs, a
  dependency to install, code to regenerate, a container to rebuild — and a teammate who does not
  read diffs has no way to know which. A skill written for one project can close the gap because
  it knows that project's tools; codefall serves many project shapes and needs a version that
  knows none of them.
  The job splits in two. Bringing the **checkout** current is generic and codefall owns it.
  Bringing the **environment** current is project-specific and the project owns it, as two plain
  scripts declared under a `local` block in `.codefall/settings.json`: `start` brings the services
  up, `update` makes the environment match the checkout. Both are idempotent and cheap when nothing
  changed, so the answer to "bring the project up to date?" is always yes, and both are runnable by
  hand, by CI, and by a harness without the extension. Codefall never generates them at run time;
  a script re-derived by a model on every run varies between runs and cannot be run by anyone else.
  **`codefall-equip`** builds and rebuilds them. Every run does the same thing — read the repo, read
  the scripts if they exist, propose the draft or the revision, confirm, write, record the entry
  points — so there is no one-time operation and the verb is not named like one. It is invoked
  explicitly on adoption, and followed as a reference procedure by `codefall-scaffold` at code
  depth and by `codefall-implement` when a task introduces infrastructure, a dependency, a
  migration, or generated code. Its `SKILL.md` is the only copy of the procedure; the other verbs
  read `../codefall-equip/SKILL.md` the way graft reads scaffold's templates, and no skill invokes
  another through the harness, so `disable-model-invocation: true` holds everywhere.
  **`codefall-refresh`** runs them: start what is down, run `update`, record the stamp, and turn a
  failure into a sentence — "the database is not running, start Docker and I'll retry." When
  nothing is declared it names `equip` and stops, the same shape as preflight naming `bd init`. It
  never drafts: a verb that runs daily and also does one-time inference is the overload this design
  spent the most turns removing.
  **The check is frequent and read-only; the action runs at boundaries.** `shared/preflight.sh`
  fetches and reports commits behind the default branch, a dirty tree, and whether the stamp
  matches `HEAD`, and every verb reports what it found. A pull moves the working tree under
  whatever the session is doing, so no hook ever pulls; a `SessionStart` notice is free because the
  slot already exists for `bd prime`. The **stamp** is the commit at which `update` last exited
  clean, written by `refresh` and not by the script, in a machine-local git-ignored file under
  `.codefall/` — per working directory, so each worktree has its own. A stale stamp always means
  run it; a hand run of the script leaves the stamp behind and costs one idempotent re-run.
  **The scripts stay current at the point of introduction**, the same rule as documentation:
  `design` names the change in the task when a bead's predicted file scope touches a compose file,
  a migrations directory, a lockfile, or a codegen config; `implement` counts it toward done in the
  same PR; `review` carries a lens for work that skipped design. `init` writes the generic rule
  into `AGENTS.md` through the same marker mechanism as the Beads section, so an already-equipped
  project gets it at the next `codefall init`. On an existing project, `init`'s report and `doctor`
  both name `equip` as owed work so the adopter drafts the scripts in the adoption PR and the next
  teammate gets a run, not an interview.
  One thing named and left to the project: implement's workers live in worktrees, but the local
  database is usually shared across every checkout on the machine, so a migration applied from a
  feature branch leaves the primary checkout's environment ahead of its code. `update` brings the
  environment level with the checkout it runs in; per-branch databases are the project's call.
  Alternatives seen and not taken. A shared procedure file under `extensions/shared/`, followed by
  three verbs: in this repo that is the same file as a skill minus the slash command, and the
  adopter needs the slash command. A one-time onboarding verb (`codefall-onboard`) that drafts and
  `refresh` that runs: withdrawn once implement became a caller, because first draft and revision
  are one procedure. Repurposing `codefall-graft`: its mechanics are hash comparison against
  templates the extension ships, the scripts have no template behind them, and its "never chained
  into" rule would have to go. `refresh` drafting on its own first run: the one case where a
  non-technical teammate sees the drafting interview. Names rejected: `sync` (git, `bd dolt push`),
  `update-local` and `update` (collide with `init`'s upgrade and with graft, which also bring
  something current), `catch-up`, `provision`. `equip` because the repository is being equipped
  with the tools it needs to run; `refresh` because nothing else in codefall refreshes anything,
  and it describes the routine whether or not anything changed.
  Lands as a stack of eight PRs, this record first; the order is in [`PLAN.md`](PLAN.md). The contract with projects —
  the `local` block, the two script names, idempotency, the stamp — is a breaking-change surface
  once any project declares it, and is recorded as
  [`ADR-005`](adrs/ADR-005-local-environment-scripts.md).

- **The local block and doctor's fifth category, 2026-09-16.** The first PR of the ADR-005 stack:
  `.codefall/settings.json` gains an optional `local` block with two required string fields,
  `start` and `update`, and `codefall doctor` gains a **Local environment** category with two
  checks. The block is optional at the top level and complete when present, the same rule as the
  review block, so every project set up before it existed is still valid settings. The shared
  settings module carries the field table, `RequiredLocalFields`, and `LocalCommands`, which reads
  the two commands out of a document and answers None for any block `Validate` would reject —
  a half-declared block is not a declaration. `Harnesses` joined it for the same reason: doctor's
  harnesses check was decoding the file into a struct of its own, and a second reader of the same
  file should read it through the format module rather than name the field again.
  Fixing the double report along the way: a block the top level had already called "not an object"
  was then looked inside, which said "must be an object" a second time. The review block had this
  latent since it landed, untested; both are now reported once, and the test pins it for both.
  The first check, `local-declared`, **warns** when the block is absent, because nothing else stops
  working without it — what is missing is refresh, and the remedy names `equip`, the verb that
  fills the block in. The second, `local-runnable`, **fails** when the program a declared command
  runs cannot be found: the project declared it and refresh will run it, so a program that is not
  there is a declaration nothing can act on. The program is the command's first word after any
  leading `VAR=value`; a word carrying a slash is looked for in the project, a bare word on `PATH`.
  Doctor never runs either command. Both fields usually name the same script, so a missing program
  is reported once with every field that names it beside it, rather than once per field. Doctor's
  file-system gateway gained `Exists` for the path case; the shared module already had it.
  The category sits between Harnesses and Beads: the first three are about the project and the
  last two are about tools on the machine.

- **Init writes the Local environment section, 2026-09-16.** The second code PR of the ADR-005
  stack. `codefall init`'s agents step now writes two marked sections into `AGENTS.md` rather
  than one: the Beads section it already wrote, and a **Local environment** section that tells an
  agent to run `refresh` before starting new work and rather than pulling by hand, to change the
  `start` and `update` scripts in the same pull request as the change that makes them stale, and to
  run `equip` when no `local` block is declared. The words live in
  `internal/initcmd/internal/domain/local_section.md`, embedded beside the Beads section for the
  same reason. The section has markers of its own, `<!-- BEGIN CODEFALL LOCAL -->` and
  `<!-- END CODEFALL LOCAL -->`, rather than widening the Beads pair: a project set up before this
  section existed has the Beads markers and nothing else, and a widened pair would never be found
  in that file, so init would append a second copy of the Beads words for ever. With its own pair,
  the next `codefall init` on such a project reports "added the Local environment section to
  AGENTS.md" and leaves the Beads section byte for byte — which is how the rule reaches every
  project without a scaffold, and the whole reason the marker mechanism was chosen for it.
  The step reads the file once and writes it once however many sections change, and a section that
  is not there yet goes at the end, after the project's own words, never beside the section it
  belongs with: the step does not move what somebody else wrote. An unclosed marker for either
  section stops the run before anything is written, so a file with one good section and one broken
  one comes back untouched. The step's title is now "Writing codefall's sections to AGENTS.md" and
  its skip reads "codefall's sections in AGENTS.md are current"; the step ID is unchanged.
  Init's closing line still says "Next: codefall doctor", and doctor is what names `equip` when
  nothing is declared. Init could only say so itself by reading the settings it has just written,
  which is a use-case method for one sentence; the section it writes names `equip` instead, and
  doctor is one command away.

- **Preflight reports the checkout and the stamp, 2026-09-16.** The third code PR of the ADR-005
  stack. `shared/preflight.sh` now opens with a checkout report before the Beads precondition:
  the branch, whether the tree is dirty, a fetch, the default branch, commits behind and ahead of
  it, whether settings declare a `local` block, the refresh stamp, and the stamp against `HEAD`.
  Nothing in it changes the exit code — the three verbs that run preflight read the lines, say what
  they found, and offer `/codefall-refresh`; `refresh=undeclared` names `/codefall-equip` instead.
  The fetch runs with `GIT_TERMINAL_PROMPT=0` and ssh in batch mode, because a skill runs this with
  nobody at the keyboard to answer a credential prompt, and a fetch that fails is reported as
  `fetch=failed` rather than stopping anything. The default branch is what `origin/HEAD` names,
  and `main` or `master` when that ref is missing, as it is in a remote added by hand. Whether the
  `local` block is declared is read with jq when jq is there; without it, a `local` key anywhere in
  the file counts, and doctor is the check that reads the shape properly.
  The **stamp** is `.codefall/refresh.stamp`: one line, the commit `update` last exited clean at,
  written by refresh and read here. `refresh=current` means the stamp matches `HEAD`; anything
  else, while a block is declared, is `stale`. It is a per-machine file, so `codefall init`'s
  ignore step now writes two entries rather than one — the `.ignore` line for review findings it
  already wrote, and `.gitignore` gains the stamp — and doctor's Settings category gains a sixth
  check that warns when `.gitignore` does not name it. Committed, the stamp would tell every other
  clone it was current at a commit it never refreshed at, which is the one way the design can lie.
  The skill additions are three sentences each, in the step that already runs preflight, and
  `codefall-specify` sits at 4,780 tokens against the 5,000 guideline afterwards.

- **The equip skill, 2026-09-16.** The fourth code PR of the ADR-005 stack, and the verb that
  builds and rebuilds the local scripts. `extensions/skills/codefall-equip/` holds the `SKILL.md`,
  a `reference/signals.md` with the entry points a project may already have, the signals that say
  which tools its environment needs, what each maps to in `start` and `update`, and the commands
  that fail the contract; a `templates/local.sh` for a project with no task runner idiom of its
  own; and a `NOTES.md` with the lineage. The skill's `SKILL.md` is the one copy of the procedure:
  a **When another verb follows this skill** section says what scaffold does at code depth and
  what implement does on a task that introduces a tool, and neither invokes it through the
  harness, so `disable-model-invocation: true` holds.
  The template is equip's rather than graft's. `skills/AGENTS.md` routes a template that becomes
  the project's document through graft, with provenance; that rule is about documents graft
  compares against a template the extension still ships. The script has no template behind it once
  drafted — it is filled from the project's signals, and the next change to it is a revision by
  this verb — so graft would be a second owner of a file equip already rebuilds. No lineage row,
  because lineage records renames of scaffold's templates.
  Two things the skill states as rules rather than leaving to judgement. **Never destructive**: a
  reset is not idempotent, and "bring the project up to date?" is only always-yes while it never
  costs data, so a candidate that drops or deletes is reported and never declared. **Proving is
  offered**: `start` brings real services up on the user's machine, the one thing here that
  reaches outside the repository, so it is a yes-or-no, and the proof is `update` twice with the
  second exit `0` and quick.

- **The refresh skill, 2026-09-16.** The fifth code PR of the ADR-005 stack, and the verb a
  teammate runs. `extensions/skills/codefall-refresh/` holds the `SKILL.md`, a
  `reference/failures.md` mapping the failures `start` and `update` produce most often to what
  each means and what to say, and a `NOTES.md` with the lineage.
  The checkout is moved in exactly one case — default branch, clean tree, and `origin/<default>`
  is a fast-forward — with `git pull --ff-only`, which cannot lose anything and cannot conflict.
  Every other state is reported and left alone: a diverged default branch, a dirty tree, a feature
  branch, a detached HEAD, a failed fetch. Rebasing a feature branch was rejected because a
  conflict inside the verb a teammate runs to get going is the opposite of the point, and stashing
  was rejected because a stash is state nobody asked for in a stack other sessions share. The
  environment is brought level with the checkout it finds whatever the row, because that is the
  half a hand pull leaves undone.
  `start` runs always: cheap when everything is up, and `update` assumes it ran. `update` runs
  when the stamp does not match `HEAD` or the user asks; a matching stamp is the skip. The stamp
  is written after a clean `update` and never otherwise, and only by this verb: written before, a
  failed `update` would leave a record claiming an environment that does not exist. Refresh checks
  the stamp is git-ignored with `git check-ignore` and names `codefall init` when it is not; it
  never adds the entry, which would make it a second owner of init's line.
  A failure is three things in one paragraph — what failed, quoted in one line; what it means;
  what to do — and the failures reference sorts each into the environment's (run refresh again
  once fixed), the script's (`codefall-equip`, since refresh never edits a script), or the
  checkout's (a lockfile out of step with its manifest is the branch's problem, not the machine's).
  The README gains a **Local environment** section covering both verbs. The `SessionStart` notice
  the ADR mentions is not in this stack; the slot exists and a notice can join it later.

- **The scripts stay current at the point of introduction, 2026-09-16.** The sixth code PR of the
  ADR-005 stack, and the rule that keeps the contract from rotting. Three verbs carry it, each at
  the moment it can act. `codefall-design` adds a criterion to any task whose predicted files
  include a compose file, a migrations directory, a lockfile, a codegen config, or an
  `.env.example`: the declared `start` and `update` were changed for it. `codefall-implement`
  counts that change toward done, beside tests, and its worker prompt tells the worker to follow
  `codefall-equip`'s *When another verb follows this skill* section — the smallest revision, never
  destructive — in the same branch. `codefall-review` gains an eleventh code lens, `local`, in
  group 3 with the other convention lenses: it looks at the diff's shape rather than its code, and
  its finding is a change of that shape with nothing changed under the declared commands, or a
  change under them that drops or resets. The lens joins the closed enum in `findings.schema.json`,
  since a lens name that does not validate is a lens that never runs.
  Why three places rather than one: design is where a task's file scope is predicted, so it is the
  earliest moment the need is visible; implement is where the work happens and the only place a
  worker reads; review is the backstop for work that came through neither. The same sentence in
  the `AGENTS.md` section init writes covers an ad hoc session that used none of the verbs.

- **Scaffold equips at code depth, 2026-09-16.** The last PR of the ADR-005 stack. Either code
  tier of `codefall-scaffold` now writes the `start` and `update` scripts and proves them with the
  rest of what it emitted, following `codefall-equip`'s *When another verb follows this skill*
  section: nothing exists to search for, so the draft comes from the manifest and lockfile the
  tier wrote and the task runner if the profile added one. `start` exits `0` with nothing to bring
  up, because scaffold decides how a project is built and never what services it uses, and the
  first task that adds a service revises it under the point-of-introduction rule. Each profile's
  depth notes say what `update` does for its language — `go mod download`, the package manager's
  frozen install — and where a `Makefile` exists the scripts are two targets there.
  The declaration goes into `.codefall/settings.json` when the file exists. `codefall create` runs
  init before anything else, so it usually does; when a scaffold runs in a directory init has not
  seen, the scripts are written and the declaration is reported as owed once init has run, rather
  than scaffold writing a settings file that is init's. Docs-only output writes no scripts and
  names `codefall-equip` as owed, beside boundary enforcement.
  The stack is eight PRs counting the record that opened it, not seven; the first entry said
  seven, and is corrected there rather than rewritten around.

- **The install layout, 2026-09-17.** Recorded as ADR-006; the implementation is the next pull
  request of this stack. The problem is that the extension step copies the embedded tree into every
  skills directory a chosen harness reads, and there are two of them — `.claude/` for Claude Code,
  `.agents/` for the other four — so a project on Claude Code plus anything else holds the tree
  twice: 64 files under `skills/` at about 375 KB, plus seven files at about 28 KB that no harness
  looks for by convention.
  Three layouts were seen. **One real copy in `.agents/` with `.claude/` linking into it** is the
  smallest change and bets on one harness following a symlinked skills directory. **One real copy
  in `.codefall/` with every harness directory linking into it** removes every duplicate byte and
  makes that bet against every harness at once, including harnesses codefall does not support yet.
  **`.codefall/` holding what is reached by a path codefall writes, with the skills still copied per
  skills directory** was taken: it depends on no harness following a link, so a harness added later
  cannot break it, and the duplication it leaves is bounded at two copies of the skills because
  every harness but Claude Code shares `.agents/`.
  Found on the way and not deciding: the Claude Code documentation does describe symlinking a skill
  into a skills directory, so option (a) was supported rather than a guess. It was still refused,
  because what the docs say about Claude Code says nothing about Codex, OpenCode, Muse,
  Antigravity, or the next one, and a layout that works only while each of them behaves is a layout
  with a failure mode — no skills found — that reads to a user as codefall not being installed.
  Also found: symlinks were available. `.goreleaser.yml` builds `darwin` and `linux` only, so
  Windows is not a release target and none of the usual reasons to refuse links applied here.
  Two things land with the layout. **The maintainer documents are installed nowhere** — `AGENTS.md`,
  `skills/AGENTS.md`, `README.md`, `docs/ROADMAP.md`, and every `NOTES.md`. They are written for
  someone working on codefall, no skill or script names one, and a project that uses codefall has no
  reader for them. **There is no automatic clean-up.** Init writes the new layout and re-registers
  its hooks; codefall deletes only what it can prove it owns, and the old copy sits in directories
  that hold the project's own files, so the by-hand list goes in the pull request description.
  One thing the implementation has to settle that this record does not: doctor's per-harness install
  check stats `<skills dir>/hooks/shared` today, and under the new layout that directory is not
  written per harness at all. The evidence has to become something still per harness — the files the
  manifest records for that harness, or the skills directory — and the decision-log entry *Doctor
  reports on the harnesses the settings name* is where the constraints on that check were set.

- **Installing the layout, 2026-09-17.** ADR-006 built. The extension step no longer mirrors the
  tree: it copies three named subtrees, `skills/` into each chosen harness's skills directory and
  `hooks/shared/` and `shared/` into `.codefall/`, each file keeping the path it has in the tree.
  Naming what is copied, rather than excluding what is not, is what leaves the per-harness hook
  definitions behind — the exclude list that used to do that job is gone, and a subtree nobody names
  is a subtree nobody installs. The two maintainer documents that sit *inside* `skills/` still need
  excluding: `skills/AGENTS.md` by path, and `NOTES.md` by file name, because there is one beside
  every skill. Both stay in the embedded tree and are left behind at copy time, since the facade test
  reads that tree whole and a tree that stopped carrying them would make the exclusion silently stop
  excluding anything.
  **The once-only files are recorded as their own manifest entry**, `shared`, beside `harnesses` and
  of the same shape: the version that wrote them and the paths, each prefixed with `.codefall/` so it
  still says which directory it landed in. Recording them inside every harness's entry was the
  alternative, and it would have made the record disagree with itself — four harnesses share
  `.agents/` and all five share `.codefall/`, so each entry would claim to have written files the
  others wrote too. A run replaces that entry rather than merging into it, because every run writes
  the directory whole.
  **Doctor's install check now reads the manifest's files for each harness and stats them.**
  `harness.SharedHooksDir` and `SharedHooksPath` are gone: neither described anything once the
  scripts stopped landing per harness, and the module is back to being the names and the two skills
  directories. The recorded files are the per-harness evidence the ADR left to this pull request to
  choose, and reading them is also what keeps the check clear of a skill's name — the names come out
  of the file the last run wrote rather than out of the check, so a rename moves both at once. Two
  consequences were taken deliberately: a half-written install reads as not installed, which is what
  `codefall init` repairs either way, and a manifest that cannot be read reports every chosen harness
  as missing, with the same remedy, because init rewrites the file it could not read. The `shared`
  entry is recorded and nothing reads it yet; what it is for is the by-hand clean-up the layout
  leaves, and a project that wants to know what codefall put in `.codefall/` can now be told.
  The hook definitions all name `.codefall/hooks/shared/codefall-block-merge-to-main.sh`. The forms
  did not change with the destination — Claude Code and Codex still quote
  `$(git rev-parse --show-toplevel)`, Antigravity still uses its workspace-relative `./…`, and the
  subdirectory prefix still goes in after the root expression — and the OpenCode plugin's climb from
  `import.meta.dir` is the same two levels it always was, because `.opencode/plugins/` and the new
  destination are both directly below the install directory. Only the directory name moved.
  The twenty-one `../../shared/…` references became `../../../.codefall/shared/…`, one `../` deeper
  again from a supporting file, and `extensions/scripts/skill-health.sh` resolves anything holding
  `/.codefall/` under `extensions/` instead of relative to the skill. That rewrite is what the
  strict run of that script proves; the facade test proves the other half, walking the real embedded
  tree to check every such reference names a file that is in `shared/` and that no skill still uses
  the old path. What is not proven mechanically is the resolution itself — that
  `../../../.codefall/shared/x` from `.claude/skills/<verb>/` is the file init wrote — because no
  test installs into a directory and reads back out of it. The path is pinned in three places that
  would have to disagree for it to be wrong.
  No clean-up, as ADR-006 says: the pull request description lists what an earlier install wrote
  under `.claude/` and `.agents/` that this one does not.

- **The testing decisions, 2026-09-17.** Recorded as ADR-007; the implementation follows in the next
  five pull requests of this stack — the `test` block and init's tree, the `codefall-test` verb,
  equip's widened scope, the rules in design and implement, and the session-start notice. The gap is
  that codefall says where an expectation is written and never what verifies the wired product, and
  the thing worth writing down is not a procedure but a rule about where an expectation may come
  from, since a test written from the code certifies the code's bugs as readily as its behaviour.
  The design is taken from `demo-flights`, ADR 0011 and its `testing/CLAUDE.md`, written for that
  project's backfill of a shipped flight-booking application. Adopted as it stands: the case file as
  the one home for a case's inputs, prompts, and expectations; the file path as the identifier with
  no registry; sibling pairing checked both ways; two modalities and no `manual` one; the two
  agentic verdict vocabularies and the per-step attempt counts; the anomaly sweep; no mocking at any
  layer; an unforceable state dropped as a criterion and recorded; and the triage classes, including
  agent variance as its own class rather than flake, because the nondeterminism is in the product.
  Left behind, all of it project knowledge rather than contract: the Sabre CERT notes about which
  routes carry which inventory, the stranded-booking ledger, the npm aliases, and the TypeScript
  loader the case files are read through. The shape those take here is a scoped `AGENTS.md` under
  the testing root that the project fills in — S1, **what a program reads is in settings, what an
  agent reads is in `<root>/AGENTS.md`** — and a runner the project declares.
  Three things changed on the way across. **GitHub issues stop being the source of criteria**: the
  model used them because an issue was the only record that predated the code, and codefall has
  specs with numbered criteria, so a project needs no issue tracker to write a case. **The spec's
  criteria became the floor rather than the whole list**, which was the QA-expansion concern:
  pinning a case to the spec alone makes the spec the ceiling too, and boundaries, negative paths,
  and error handling are the work, so a criterion beyond the spec is marked `derived`, cites the
  requirement it elaborates, and says in one line what it adds. A gap that turns up goes back to
  `codefall-specify` rather than being settled in the case, which append-only numbering makes safe.
  **Agentic run reports are committed** under `.codefall/tests/` beside the review findings, which
  was Plan B against a Plan A that ignored everything a run produced; split by kind is what it
  bought — counted variance across runs in prose that is cheap to keep, with the runner output,
  traces, HTML reports, and run-scoped accounts still ignored under `<root>/.artifacts/`.
  Alternatives seen and not taken. **Runner names in `<root>/AGENTS.md` with everything else** —
  amended to settings as S2, because the check scripts and doctor are programs and would be parsing
  prose; the commands stayed prose, so the split by reader survives with one exception it explains.
  **A new verb for test setup** rather than widening `codefall-equip`: the narrowing to the `local`
  block was only ever in equip's own `SKILL.md`, ADR-005 does not restrict it, and a project equips
  once. **Setup inside the task's pull request** that first needs a case: refused, a task's PR that
  also installs a test runner is two changes in one review, and the cost taken instead is that a
  bead can stop with `codefall-equip` named. **`codefall-implement` deciding modality**, which is
  where it sat first: moved to `codefall-design`, because the derived criteria need a sign-off and
  the task plan is where the user already approves one, and the bead's criteria are then the signal
  implement reads to know a case is owed. **A static registry of drivers**: refused, the agent reads
  its own tool list and names what it found, and one live call proves it before the run depends on
  it. And the hook: "backup" was a misreading of what the session-start notice is for — it is an
  early notice and never a conditional gate, so it and every verb's own preflight run both always
  run.
  Two things the implementation settles that the record does not. Whether a committed run report is
  Markdown only, as the model's were, or Markdown beside a JSON record with a schema the way
  `codefall-review` writes its findings — the JSON would make variance counts mechanical. And how
  the `CODEFALL TESTING` section names a configurable directory when the two sections init already
  writes are static embedded text: either the body is filled from settings, or it refers to the
  `test` block without naming the path.

- **The test block and init's testing tree, 2026-09-17.** The first code pull request of the ADR-007
  stack: `.codefall/settings.json` gains an optional `test` block, `codefall init` gains a step that
  declares the testing root and makes the tree under it, and `codefall doctor` gains a **Testing**
  category with three checks. The block is optional at the top level and complete when present, the
  same rule the review and local blocks follow, so every project set up before it existed is still
  valid settings. `dir` is required once the block is there and is held to a pattern the schema and
  the validator share: path segments of ordinary file-name characters, each beginning with one that
  is not a dot, which makes every root relative and leaves no way to write `..` as a segment, so a
  declared root cannot climb out of the project that declared it. `runners` is optional and may be
  empty, because a project declares where its cases go before anything is installed to run them;
  it is validated against the enum `playwright`, `go-test`, with no duplicates. `TestDeclaration`
  reads the block and answers None for anything `Validate` would reject, as `LocalCommands` does.
  **Init asks for the root and never moves one.** The survey asks last, with `testing` as the
  starting value rather than a silent fallback; `--test-dir` answers for a script, and a first run
  with no terminal and no flag fails naming the flag, as `--harness` and `--tracker` already do. A
  rerun reads the root back out of the settings the way the harnesses are read back, and a settled
  project that declares none is not surveyed at all — the use case takes the format's own default,
  which is the only place a root is ever inferred. The no-op gate gained one condition for the same
  reason: an install current in every other way is still work while the root is undeclared, because
  doctor's remedy for that is this command and the command has to do something.
  **The declaration is spliced into the settings text rather than encoded with the document.** Most
  projects will get their block on a rerun, over a file the settings step skips, and decoding that
  file and encoding it again would reorder every key and drop whatever the project or
  `codefall-equip` had added to it. The splice puts the block after the last member of the top-level
  object and leaves every other byte where it was. A block that is already there is left exactly as
  it is, runners included. The alternative — writing it in the settings step alongside the tracker —
  was rejected for leaving a settled project no way to declare a root short of `--force`, which
  rewrites the whole file.
  **The step is sixth, between the agents step and the ignore step.** It sits beside the agents step
  because the two write the same pair of documents under the same rule, an `AGENTS.md` and the
  `CLAUDE.md` that points at it, and it has to precede the ignore step, which keeps the root's run
  output out of the repository and cannot name a root before one is declared. The tree is `<root>/`,
  `<root>/test-cases/`, and skeleton `AGENTS.md` and `README.md` files embedded in the domain beside
  the section markdown; the `CLAUDE.md` pointer is the same line the project root gets and only for
  Claude Code. Each file is written only when it is missing and never rewritten, so the report names
  the files it wrote and the directories are made whatever it finds — `MkdirAll` on a directory that
  is there does nothing, and initcmd's file-system gateway has no cheaper way to ask.
  **The files the step creates are recorded in no manifest.** The `shared` entry is what one run
  wrote into `.codefall/` and may replace; these are the project's documents from the moment they
  exist, and recording them would invite a later clean-up to delete somebody's own `AGENTS.md`. What
  records the tree is the `test` block, and `test-dir-exists` is the check that reads it.
  **The `CODEFALL TESTING` section fills a placeholder from the declared root**, which is the
  question ADR-007 left open. The two sections beside it are static text; this one names a path the
  project chose, so the embedded markdown carries `{{TESTING_ROOT}}` and the domain substitutes it.
  The alternative was a section naming the `test` block instead of the path, and that leaves every
  reader of `AGENTS.md` a lookup to perform before it can act — and the reader is an agent reading
  one file for its rules. Otherwise the section is handled exactly as the other two: its own marker
  pair, replaced between the markers, appended at the end of a file that has no markers, and an
  unclosed marker stops the run before anything is written.
  **The ignore step writes four entries rather than two**, `.codefall/tests/` in `.ignore` beside the
  review findings and `<root>/.artifacts/` in `.gitignore` beside the refresh stamp. It now reads and
  writes each file once however many entries it is missing, which is what keeps a project set up
  before the second entry existed from being told twice about one file. Doctor's two ignore checks
  became a table of three, the third being `tests-ignored`: the two were already the same function
  written twice, and each entry keeps its own check ID, title, and remedy rather than widening
  `reviews-ignored` to mean something its name does not say.
  **Doctor's Testing category sits between Local environment and Beads**, where the split is between
  what the project declares and what is on the machine. `test-declared` warns when there is no block,
  with `codefall init`; `test-equipped` warns when no runner is declared, with `/codefall-equip`;
  `test-dir-exists` fails when the declared directory is not there, because the project declared it
  and every verb that writes or runs a case looks for it. A project with no declaration skips the two
  checks below it, which have nothing to ask about until there is a root.

- **Equip sets the test harness up, 2026-09-17.** The third code pull request of the ADR-007 stack.
  `codefall-equip` now equips two things: the local scripts it already owned, and the spec runner
  `codefall-test` runs cases through. ADR-005 never restricted what equip may own, so the widening is
  a change to its own `SKILL.md` and nothing else.
  **The testing procedure went in as a second track chosen by the argument**, `local` or `test`,
  rather than as more steps on the existing one. The two share the search-first shape and nothing
  else — different evidence, a different question, a different block written, a different proof — and
  in the user's project they are two pull requests, so a single run would have put two confirmation
  gates and two real-service proofs in one turn and made a user who came to revise `update` sit
  through a runner interview. The argument is also what lets `codefall-scaffold` and
  `codefall-implement` name the track they follow, which is the local one in both cases. The
  local-scripts procedure and its contract are unchanged.
  **The detail lives in `reference/testing.md`**, because `SKILL.md` had about 1,800 tokens of
  headroom and the procedure needs a configuration file, two runners' commands, and a table of what
  to search for. The skill carries the eight steps and the gates; the reference carries what each
  step writes.
  **The runner follows the surface**, per ADR-007: Playwright for a browser front end, an Electron
  shell, or an HTTP API; `go test` for a Go surface; React Native, Tauri, and Flutter refused for the
  `spec` modality in this version, with agentic cases still open to them. A runner outside the two
  names the settings enum accepts — Cypress, WebdriverIO — is reported as exactly that rather than
  declared under a name it does not have, because `test.runners` is what the check scripts and doctor
  read. Setting a Playwright configuration up means `testDir` on `<root>/test-cases` so a spec is
  collected where its case already sits, `testMatch` on the `.e2e.ts` suffix so the case's own
  markdown is never collected, `retries: 0` and one worker, and every artefact under the git-ignored
  `<root>/.artifacts/`. `go test` means a package under `<root>/test-cases/<area>/` with the
  `_e2e_test.go` suffix and a `//go:build e2e` tag, so an ordinary unit run never reaches one.
  **The proof is an empty tree.** The runner's list command against no cases shows the configuration
  collects from the declared root, and `check-cases.sh` passes with zero cases while proving the root
  it reads is the root the configuration points at. Writing a case to prove the runner would put
  authoring in the wrong verb. `go test` has nothing to compile until the first spec exists and
  answers `matched no packages`; that is recorded as the expected answer rather than worked around.
  **The runner's own install is an `update` step**, revised by the local track's step 3 — Playwright's
  browsers are in no lockfile — which keeps the contract, since installing browsers already present
  exits `0` quickly.
  **Preflight gained the `test=` line** beside `local=`: `undeclared` when there is no block or one
  `Validate` would reject, `unequipped` when no runner is named, `equipped` when one is. Without jq it
  answers `unknown` where a `test` key is present, since the runners cannot be read, and `undeclared`
  where the key is absent — the one place it does not copy `local`, which answers `unknown` for an
  absent key. Exit codes are untouched; the line was proven against four settings files and against a
  `PATH` with no jq on it.
- **Design decides the case, implement writes it, 2026-09-17.** The fourth code pull request of the
  ADR-007 stack, and the one that puts a case in front of the code that it tests. `codefall-design`
  gains the rule that a task verified through the wired product names its case — `<area>/<slug>` —
  its modalities, and every criterion the case will hold in the bead's acceptance criteria;
  `codefall-implement` gains the detection at preconditions, the rule that the case file is written
  first, and the correction of a customization path it had been spelling wrong.
  **The case rule went in beside the point-of-introduction rule**, at step 6 where the tasks are
  staged, because both answer the same question about a task: what this task delivers beyond its
  code. The form the criteria take — the citation for a spec criterion, the `derived` marking and
  its one line, the example — is in `reference/beads.md`, where what a bead carries already lives.
  `SKILL.md` had 123 tokens of headroom and the rule needed about 150, so three duplications came
  out to pay for it: the research-goes-inline sentence, which the Rules and `reference/document.md`
  both already carried, the `Ready`-is-the-normal-end bullet, which the status table and step 7 both
  state, and a transition bullet repeated verbatim in *Other modes*.
  **The sign-off is the task plan's approval**, per ADR-007, so design shows the criteria of every
  task carrying a case at step 7 and nothing later asks again. A gap the criteria expose goes into
  the run's report for `codefall-specify` rather than into the case, which is what keeps the spec
  the floor rather than a record of what testing decided on its own.
  **Implement's detection reads the `test=` line and holds it** until step 3 has read the beads,
  because preconditions run before the scope is fixed and a bead's criteria are the only signal that
  a case is wanted. `unknown` — preflight with no jq and a `test` key present — sends the skill to
  `.codefall/settings.json` to read the block itself rather than treating an unreadable answer as a
  stop. `codefall-test`'s preflight table gained the same row, which PR 5 wrote before the line
  existed.
  **The worker reads the format by an absolute path.** `{{CASE_FILE_FORMAT}}` and
  `{{TESTING_ROOT}}` join the prompt's placeholders: a worker runs in the project, not in the skills
  directory, so a skill-relative path would not resolve for it, and the root is the only thing that
  knows which skills directory this run came from. The worker's `## Verify` gains
  `check-cases.sh` and the runner's list command — evidence the case is well formed and the spec is
  collected, not a run of the case, which stays `codefall-test`'s.
  **The customization file was spelled `.codefall/skills/implement/CUSTOMIZE.md`** in two places in
  `codefall-implement/SKILL.md` and once in `codefall-specify/trackers/github/PROFILE.md`, against
  the `<verb>` rule in `extensions/shared/customizations.md`, which makes it
  `codefall-implement/`. Corrected in all three; nothing read the old path, so no project is
  affected.
  **Token counts after the change**: `codefall-design` ~4,983 of 5,000, `codefall-implement` ~4,943,
  `codefall-test` ~2,990. The two large skills have under 60 tokens of headroom each, so the next
  addition to either one moves text to a reference file before it adds a line.

- **The session-start notice, 2026-09-17.** The last pull request of the ADR-007 stack, and the slot
  the refresh skill left open. `hooks/shared/codefall-session-notice.sh` reads the shared preflight
  and turns the lines that need attention into one short notice: the checkout behind the default
  branch, an environment that has not been refreshed since `HEAD` moved or has no scripts declared,
  a testing root or a runner nobody declared, a blocked Beads precondition quoted with its remedy.
  Every line names the verb that fixes it. The notice prints nothing when everything is current,
  always exits `0`, never pulls, never runs `update`, and writes no file — ADR-005's rule about what
  a hook may do, and the reason a session that cannot be read still opens.
  **The bound on the fetch is `read -t` rather than `timeout`.** The notice runs preflight and reads
  its report line by line; every line arrives at once except the one the fetch precedes, so a wait of
  five seconds for the next line is a wait on the network and nothing else. A report that never
  reaches its `beads` line is a run the bound cut off, and the notice runs preflight again with
  `--no-fetch` — the flag preflight gains here, which skips the fetch, emits `fetch=skipped`, and
  answers `behind` and `ahead` from the refs the checkout already has. `timeout` and `gtimeout` were
  the obvious bound and were rejected: neither is on a stock macOS, which is where this runs, and
  taking a coreutils dependency for one hook costs more than the five lines of shell. Preflight's
  default is untouched, and no skill passes an argument other than the project directory.
  **The output contract is selected by a flag**, the way the merge guard's `--antigravity` selects
  its deny shape. `--hook-json` emits `{"hookSpecificOutput": {"hookEventName": "SessionStart",
  "additionalContext": …}}`, which is what `bd prime --hook-json` already emits into the same event
  and what Claude Code's and Codex's definitions register the notice beside; plain text is the
  default, which is what the OpenCode plugin puts into `session.prompt`. A machine with no `python3`
  to build the JSON with prints the text instead, because Claude Code adds a `SessionStart` hook's
  plain stdout to the session as well. Antigravity has no session event and registers nothing.
  **The notice is a second `SessionStart` entry, not a second command inside the prime's entry.**
  Folded in beside `bd prime`, an upgrade of a project that already runs the prime would append the
  notice as its own entry — the merge appends only what is new — and the next upgrade would replace
  that entry with the whole definition, registering the prime a second time. As its own entry it is
  codefall's by the script its command names, so an upgrade replaces it and leaves the prime and the
  project's own entries alone. That is the identity rule from **What the hook merge calls the same
  entry**, exercised for the first time on an event the project also registers things on.
  The notice does not gate anything: a verb's own preflight run stays unconditional, and both always
  run.

- **Beads sync, the interaction log, and its default, 2026-09-22.** Three things bd's own
  `AGENTS.md` section says that codefall's did not. First, where the beads live: a local Dolt
  database, with the team's copy under `refs/dolt/data` on the git remote, moved by `bd dolt pull`
  and `bd dolt push` and by nothing else — a git pull leaves it where it was, and
  `.beads/issues.jsonl` is an export, not the sync. bd's section is conservative by design: its
  session-close protocol syncs only when the repository's instructions say to, and codefall's
  section said not to. It now says to: pull before reading the graph, push after every write and
  before the work that follows it, so a claim is visible before the result. `codefall-design`
  pulls before reading the graph and pushes once the graph verifies; `codefall-implement` already
  pushed after every claim and close, and now after discovered work too; `codefall-refresh` runs
  `bd sync` — pull, positive conflict check, repair of the blocked flags merged edges change, push —
  rather than a pull alone, because a pull leaves a crashed session's unpushed closes on one machine
  and `bd ready` stale on edges merged from elsewhere, and its exit codes are explicit enough to
  turn into a sentence each. It is the one push from a verb whose scope is otherwise run-only, and
  it publishes only Dolt commits already made; nothing in git moves. A project with no Dolt remote
  is told so once, and `bd dolt push --yes`, which adopts the git origin, is named as the user's to
  run. Second, `.beads/interactions.jsonl`: bd's optional append-only interaction log, committed
  when it is on, and the file agents kept mistaking for a stray change in a pull request. Init now
  appends `.beads/interactions.jsonl merge=union` to `.gitattributes`, the same append-only-if-
  missing treatment the ignore files get, and doctor warns when the line is absent; the section
  says what the file is and that a conflict in it is never resolved by hand. Third, its default:
  bd 1.3 leaves `audit.enabled` commented out and off, and init now writes `audit: enabled: false`
  into the database's `config.yaml` through `bd config set`, right after `bd init` and only then —
  a database that was already there may have turned the log on, and that is its decision to keep.
  `bd config set` rather than an edit because bd knows where the database is and codefall does not.
  bd warns that the key is unrecognised and writes it anyway; `bd audit --help` names that exact
  command as the way to set it.
- **Init writes the Codefall section, 2026-09-22.** PR #98 described how codefall works in this
  repository's `AGENTS.md` and in `docs/workflow.md`, and neither reaches a project: nothing `init`
  installed said which verbs exist, what order they run in, what each leaves for the next, or who
  is authoritative for documents, the tracker, and Beads. An agent in a project learned the chain
  only from a skill that names the next one. `codefall init` now writes a fourth marked section,
  **Codefall**, between `<!-- BEGIN CODEFALL PROCESS -->` and `<!-- END CODEFALL PROCESS -->`,
  from `internal/initcmd/internal/domain/codefall_section.md`, embedded beside the other three for
  the same reason. It is first in `sectionsFor`, because it is the frame the other three sit
  inside, so a file that has none opens with it; a file that already has the three gains it at the
  end, after them, because the step never moves what somebody else wrote, including sections it
  wrote itself on an earlier run. The section is one paragraph — the chain, the verbs beside it,
  that every verb is invoked deliberately and applies only what the user takes, that a human
  performs every merge, and one line on who is authoritative — and it points at
  `.codefall/shared/workflow.md` for the rest rather than carrying it. The detail file ships from
  `extensions/shared/`, which the extension step already copies into `.codefall/shared/` per
  ADR-006; no conciseness pass on the sections has landed to choose another home. It holds the
  chain table, the four verbs beside the chain, and who is authoritative for what, and not what
  init puts in place, which is the CLI's business. **`docs/workflow.md` is the source and the copy
  is checked, not hand-edited.** `extensions/scripts/workflow-sync.sh` compares the three shared
  sections byte for byte, `--strict` fails CI when they differ, and `--write` copies them in,
  keeping the copy's own opening. A header in each file saying the other is the copy was the
  alternative, and nothing enforces a header; a generator was the other, and it would need a
  header it did not overwrite, which is the same script with `--write` and no check. The copy
  stays byte for byte, so the two ADR links in *Keeping the project current* became absolute
  GitHub URLs, which resolve from a project's `.codefall/shared/` as `adrs/…` does not. The
  `codefall init` rerun bullet stays in the copied section: upgrading is the project's concern
  even though what init installs is not. Doctor gains no check; the section is one more marked
  span the agents step already replaces. Not a breaking change: a new section and a new shared
  file, nothing renamed or removed.
- **The AGENTS.md sections stay self-contained, 2026-09-22.** The Beads section doubled in
  PR #100, and every harness reads the four sections at session start (chars/4 on 2026-09-22:
  Codefall ~234 tokens, Beads ~664, Local ~228, Testing ~209). Seen and not taken: pointers to
  files under `docs/codefall/`, and pointers to files under `.codefall/shared/`. No file is
  added; the sections get shorter in place. Most of each section is a rule the agent has to have
  in context to follow. The rest — the Dolt model, the no-remote case, what the interaction log
  is, what `refresh` runs — is reference that `.codefall/shared/workflow.md` already carries and
  the Codefall section already points at, and `bd prime` prints the command reference where a
  harness has a session event. Antigravity and Muse have none, so the quick-reference block
  stays. `docs/codefall/` would be a second home for shipped files, which ADR-006 rules out, and
  the manifest and doctor know only `.codefall/`; a file under `.codefall/shared/` would hold
  what workflow.md holds. The rewrite removes about 450 tokens from every session start, and a
  linked file would save nothing further. Not a breaking change. **The rewrite, same day.** Beads
  664 → 349 tokens, Local 228 → 149, Testing 209 → 144 (chars/4, markers included). Every rule
  stayed; what left is the Dolt explanation, the hook that runs `bd prime`, the `bd create`
  command line, what `refresh` records, and the numbered close protocol, now one sentence. The
  three sections share one voice: each opens with the fact the rules rest on, tells the agent to
  run a verb as `/codefall-<verb>` and names one as `codefall-<verb>` — the Beads section had
  `/codefall-refresh` where it was naming, not telling — and closes with the same precedence line.
  No Go code, test, or shipped file changed. **Beads then went back over its target, to ~399,**
  after a session in which an agent created a bead, neither pulled nor pushed, and was confused by
  the interaction log. Four things went back in: the enumeration of what counts as a write, since
  a create had not read as one; the pull rule keyed to the start of a session rather than to
  reading the graph, which a create skips; the clause that the log's appearance is expected and
  not a stray change; and one sentence saying this section, not `bd prime`'s git-authority note,
  decides Dolt sync. That last one is the likely cause: in the three harnesses with a session
  event, `bd prime` prints "do not push, pull, or run remote sync" under bd's default
  conservative profile, into the same session as the section, and bd's own text says repository
  instructions override it without the section claiming that. Setting `agent.profile
  team-maintainer` at init would fix the wording and also authorize git pushes at session close,
  which the section forbids, so the sentence is the cheaper fix. Clarity over the target, there.
- **The AGENTS.md sections move to `extensions/agents/`, 2026-09-22.** The four sections and the
  two testing skeletons lived at `cli/internal/initcmd/internal/domain/*.md`, each embedded by a
  sibling Go file, because `//go:embed` cannot reach outside its package and the words were held
  to be a fact about codefall (**Init writes the Beads section**, above). That placement was
  consistent with the layer rules and impossible to find: `extensions/` is described everywhere
  as what `init` installs into a project, and the one kind of installed prose was not under it.
  The files now live at `extensions/agents/sections/{codefall,beads,local,testing}.md` and
  `extensions/agents/testing/{AGENTS,README}.md`, in the tree `extensions/embed.go` already
  embeds, and the agents and testing steps read them through `ExtensionSource.Read` the way the
  hook step reads a harness's definition: content the binary ships and `init` splices or writes
  rather than copies. The extension step copies nothing from under `agents/`, since it names the
  subtrees it copies and `agents/` is not one. The markers stay in the domain, as
  `domain/sections.go`, because the splice depends on them and the words do not; the placeholder
  the Testing section carries stays beside them, and the step fills it in every section rather
  than one so a section that comes to name the root later needs no code change. What the domain
  tests pinned — one pair of markers, first line and last, newline at the end — is now checked
  where the section is read, as an error naming the file, and pinned against the real tree by the
  facade test beside the hook pins. Seen and not taken: a pure shared module holding the embeds
  (still under `cli/internal/`, and a shared module with one consumer); a `templates/`
  subdirectory of the domain package (solves nothing); a third top-level directory (the repo has
  two components and no reason for a third). Not a breaking change: the words, the markers, and
  what lands in a project are byte for byte what they were. ADR-006 is not touched; it already
  records that the hook definitions are read from the binary and never copied, and this is one
  more file of that kind.

- **`codefall-refresh` is model-invocable, 2026-09-22.** The one skill that carries no
  `disable-model-invocation: true`. The rule held everywhere because no skill invoked another
  through the harness (**Local environment: equip and refresh, 2026-09-16**), and refresh is the
  one verb whose run has no stakes: it moves the checkout only on a fast-forward of a clean default
  branch, runs scripts the project already declared, never drafts or settles anything, and writes
  nothing but a git-ignored stamp. So an agent may run it when the session notice or preflight
  reports the environment stale, and `implement`, `design`, and `specify` now run it at that point
  instead of offering it. `test` still offers and never runs a remedy: a test run that moved the
  checkout underneath itself would invalidate what it was about to test. Every other verb stays
  deliberate — the interview verbs and `scaffold` write documents and beads, `implement` claims
  and opens pull requests, `review` and `graft` have no meaning without a user taking findings,
  `equip` drafts and confirms. `codefall-test` was the one arguable second: its `suites` and
  `changed` modes are mechanical and its description already carries trigger phrases, but findings
  are triaged with the user and the agentic modality drives the product, so it stays as it is.
  With the flag off, refresh's description sits in the startup listing every session, so it was
  cut to the first sentence and the "safe to run at any time" line. `skill-health.sh` holds the
  exception as `MODEL_INVOCABLE`, and checks a listed skill for the absence of the line as it
  checks every other for its presence. ADR-004's "every codefall skill sets" line is history and
  stays as written.

- **Every writing verb lands on a branch, 2026-09-22.** A `specify` run left its spec sitting
  uncommitted on `main`: the skill said "Do not commit" at the write step, created no branch, and
  offered nothing. `conceptualize`, `mock-up`, and `design` said the same, `equip` and `graft`
  listed the commit as something the user still owed, and the Beads section every session reads
  closed with "Do not commit or push git unless told to" — a sentence that had gone in to counter
  `bd prime`'s Dolt wording (**Beads then went back over its target**, above) and read as a git
  policy. None of that matched the one rule the guard hook holds, which is that a human performs
  every merge to the default branch. The procedure now lives once, in
  `extensions/shared/landing.md`, and every verb that writes to the repository points at it:
  branch before the first file (`spec/SPEC-003-slug`, `design/DESIGN-002-slug`, and so on; on
  another branch, ask once which to use; detached, stop), commit by path once the last file is
  written — after the mirror for `specify`, since the mirror writes the issue number into the
  document, and after the Task Plan collapses for `design` — and then one offer for the push and
  the pull request, whose body carries `Relates to #<issue>` where a tracker issue exists. The
  branch and the commit are not offered: nothing is written without the user confirming the
  document, and the commit is the record of that confirmation. The push and the pull request are
  one offer because they leave the machine. `implement` keeps its per-task landing and `review`
  its fixes-on-the-target's-branch; `test` commits run reports on whatever branch it ran on and was
  left alone. The Beads section's closing line now says commit on the work's branch and never
  merge or push the default branch, which is what the hook enforces. The tracker profile's
  `blob/main` link resolves once the pull request merges, and says so.

- **Beads are named, never hashed, 2026-09-23.** `bd` picks `<prefix>-<hash>` for a new issue —
  `codefall-k3f9` — which is unique across branches and unreadable to a person: no order, no
  meaning, nothing to say aloud. Checked against `bd` 1.3.0: `bd create --id` takes any ID under
  the database's prefix, uppercase and dots included; a taken ID is refused at creation; an ID under
  another prefix is refused unless `--force`; `bd rename` moves an ID and carries the parent, the
  edges, and every reference with it; `--id` and `--parent` do not combine on one create; the graph
  plan file has no ID field; `bd config get issue_prefix` prints the bare prefix. So every bead is
  now created with `--id`. A design's beads take the document's own numbering, which was already
  unique and append-only: `<prefix>-DESIGN-NNN` for the epic and `<prefix>-DESIGN-NNN-Tn` for a
  task, set by a rename pass right after `bd create --graph`, beside the pass that sets `spec_id`.
  Under Revise the old bead is renamed `<id>-superseded` and the replacement takes the plain ID, so
  the current bead always holds the name the mapping line records; the bead the merge gates block is
  `<epic>-MERGED`, named for the condition that closes it. Other work — tier 0, discovered — is `<prefix>-<tracker ref>`
  with `--external-ref` set to the same value where an issue exists, else `<prefix>-<slug>`, and a
  refused duplicate gets `-2`. Seen and not taken: a slug on the design form
  (`<prefix>-DESIGN-007-T2-stage-context`), which the title already carries; a hash suffix on the
  slug form (`<prefix>-<slug>-<hash>`), which puts the noise back to defend against a case
  creation already refuses; a shorter prefix, which is the project's to choose and never a skill's
  to write — the prefix is read from `bd config get issue_prefix` and no skill or template names
  one. The mapping line stays and lists the IDs in full, so `revising.md` and `/implement
  DESIGN-NNN` work as before. Existing hash IDs keep working: nothing must be renamed, and the
  Beads section of `AGENTS.md` grows by one bullet (chars/4 on 2026-09-23: 413 → 525 tokens).
- **Conceptualize became envision, 2026-09-23.** The skill is `codefall-envision`, the document is
  `docs/visions/VISION-NNN-slug.md`, and the header row and `Related` key are `Vision`. "Concept" sat
  too close to "spec": both name a thing to be built, so readers heard two grades of one document
  and the why-only scope got overlooked when explained. "Vision" names the act of seeing a problem
  and why it matters, which is the skill's whole scope. The worry that vision sounds too large was
  already answered in the skill: size is not a category. Seen and not taken: renaming the skill
  alone, which would reintroduce the mismatch; keeping `conceptualize` as an alias, unneeded because
  no project has run the verb. Material the user brings is now called a "pitch document" so the
  word vision names only the output.
- **The Task Plan stays after creation, 2026-09-23.** A design's task table used to collapse to a
  line mapping each local ID to its bead once the graph existed, on the reasoning that a task list
  in a git-tracked document diverges from the graph and nobody updates it. Since beads took the
  design's own numbering the line carried nothing a reader could not derive, and it discarded the
  one human-readable account of the breakdown: the titles, the edges, and the design refs. What can
  diverge is work state, which the table never held; titles, edges, and refs change only through
  Revise, which edits the document first and the graph after. So the table stays. The callout above
  it flips from `Staged. Not yet in Beads.` to the creation date, the epic's full ID, the per-row
  form `<prefix>-DESIGN-NNN-Tn`, and the sentence that Beads is authoritative and a difference is
  reconciled by revising the design; a revision adds its date; a removed row's ID goes on a
  `Retired:` line under the table so numbering stays append-only. The designs `AGENTS.md` now says
  a design bead's title, edges, and ref change only through Revise, never with `bd` directly, which
  is what keeps the snapshot level with the graph. Seen and not taken: a bead ID column, which
  repeats the per-row rule; a "may be out of date" disclaimer, which teaches readers to distrust the
  table and says nothing about how a difference is resolved; a task block rendered from `bd list`
  into the document, which puts work state into git. A design carrying the old mapping line still
  resolves, its epic being `<prefix>-DESIGN-NNN` regardless: `/implement DESIGN-NNN` checks the epic
  with `bd show` rather than the line, and the design's first revision rebuilds the table from the
  beads.
- **Every report ends with what the user does next, 2026-09-24.** The skills reported what they
  produced and ended with a prohibition — "do not start a specification", "do not merge" — and
  the user was left to work out the next step from the output. Now the last line of every verb's
  report is the user's next action: merge the pull request and run the next verb, in the chain's
  order (`envision` → `specify` → `mock-up` where a requirement carries `requires-mockup` →
  `design` → `implement` → review and merge the pull requests → `test`), or the remedy that is
  owed, or nothing. A preflight stop that hands over a remedy says to rerun the verb after it. The
  line lives in each `SKILL.md`, because that is the only file a project session loads; the rule
  in `extensions/skills/AGENTS.md` binds authors here and reaches no project. Three skills sat at
  the 5,000-token guideline and were trimmed of restated reasoning to make room. Seen and not
  taken: a shared `next.md` under `.codefall/shared/`, because the next action depends on where in
  the run the turn stopped, which a table per verb cannot say. Left for later: reviewing several
  pull requests in one run — a stack is one commit range, pull requests against the default branch
  are one pass each — which the review skill does not yet describe.
- **Review takes a commit range, and says how several pull requests are reviewed, 2026-09-24.**
  `codefall-implement` leaves one pull request per task, and the review skill said nothing about
  reviewing them together. A stack, where each pull request is based on the one below it, is one
  target: its top branch already diffed against the default branch, and now `<from>..<to>` names
  the same work by its ends, resolved by `git diff <from> <to>` with both commits reachable from a
  live branch, fixes landing on the branch whose tip is `<to>`, `range` in the schema's target
  kinds, and `range-<from7>-<to7>` as the findings-file key. Pull requests against the default
  branch share nothing and are one invocation each, in the order the implement report listed them;
  implement's next-action line says so. A single commit stays refused: a range is live work between
  two points, a commit is history. Seen and not taken: an epic identifier as a review target that
  fans out to one review per pull request, which would change one invocation being one review for a
  case the user can run as several.

- **Upstream documents are amended in the run that finds them wrong, 2026-09-24.** Every step
  reads the documents before it, and every finding that one was wrong went to a later run of the
  document's own verb: a spec gap in `design`'s report for `specify`, a design disagreement in a
  `design-revision` bead for `design`'s Revise mode, a vision with no path at all. Between the
  finding and that run the document stayed wrong with nothing on the page saying so. Now a verb
  amends any upstream document, at any distance, in its own run and pull request, when the
  document is `Draft` or `Ready`, the amendment is text that moves no task row or cited criterion,
  and the user takes it at the confirmation the verb already holds; every document between the
  change and the step is amended together, or none is. `design` appends the criterion a case
  exposes and re-mirrors it, so the bead cites a real identifier and `derived` narrows to what the
  spec declined; a worker amends the design's or spec's text in its branch and reports it in an
  `amended` list the root reads at the wave boundary; `review` fixes a design finding like any
  document finding; `specify` amends a `Draft` or `Ready` vision. ADR-008 records it and names
  the one consequence of ADR-007 it revises. Seen and not taken: superseding ADR-007 whole for one
  consequence, which would retire a record that is otherwise current; and letting a worker edit a
  Task Plan row or a bead's criteria, which is the graph and stays `design`'s.
- **A harness is named for its binary, 2026-09-26.** Two of the five harness names were product
  names: `claude-code` for Claude Code, whose command is `claude`, and `antigravity` for
  Antigravity, whose command is `agy`. The review script already runs a harness by
  `command -v <name>`, and `via=claude` named the binary while `.codefall/settings.json` said
  `claude-code`, so every script that starts a harness a project chose would have needed a table
  from one name to the other. Now every harness is named for its binary — `agy`, `claude`, `codex`,
  `muse`, `opencode` — and the rule is stated once, in `cli/internal/shared/harness/`. The hook
  definitions follow it: `extensions/hooks/antigravity/` is now `extensions/hooks/agy/`. The old
  spellings are still read: `harness.Parse` accepts them and returns the new name, validation
  passes a file that carries one, `codefall doctor` warns with `codefall init` as the remedy, and
  `codefall init` rewrites them in the settings and the manifest on its next run, changing nothing
  else in either file. The schema's enum lists only the new names. The merge guard's
  `--antigravity` flag keeps its name, because projects' hook files already register it. Seen and
  not taken: listing the old spellings in the schema's enum, which would document them as current;
  and a table from harness name to binary in each script, which is the mapping this removes.

- **Agents are an ordered list in project settings, 2026-09-26.** `review` picked its reviewer per
  invocation, by whoever remembered `via=`, and the consult the skills are about to gain needs the
  same choice made once. `.codefall/settings.json` gains a top-level `agents` array — each entry a
  name, a harness named for its binary or `current` for the harness running the session, and an
  optional model — whose order is the default every use walks; `review.agents` and later
  `consult.agents` carry an ordered subset when a use wants a different order, and `agentsByHarness`
  carries one per harness a session may be running in, because a second opinion from the model
  already in use is worth less. A run skips an entry this machine cannot start, advances on a
  mechanical failure or an explicit "cannot settle", tries each entry once, and names every one it
  tried. `via=` becomes a one-run override. The agent states its own harness, checked against the
  skills directory it was loaded from. Absent means `subagent` on `current`, which `init` writes
  explicitly; `doctor` reports which agents this machine can start and warns about an order with
  no `current`. ADR-009 records it. Seen and not taken: a map keyed by name, which cannot say what
  to try next; a per-user substitution file for a missing harness, when a run can look on PATH
  itself; detecting the harness by environment variable or process walk, in preflight or anywhere,
  when every harness tells its agent what it is; and `harnessAgents` or a nesting under
  `harnesses` for the override, where `harnesses` already means what the project is set up for.

## Open

- **UI composition.** Half settled by **Shared modules, 2026-08-27** above: the theme, the styles,
  the colour-profile writer, and the spinner runner now live in `internal/shared/ui/`, and rule 4
  holds there — proven, not assumed. What is still open is the other half: reusable Bubble Tea
  models (list, table, status bar) generic over the data they show, with each component's
  `presentation/` binding its own data, and a shell owning arrangement and navigation (`main` for
  the command tree; `main` or an `internal/tui/` component for a TUI — unsettled). A facade
  re-exports a view type by alias when the shell must name it. Alternatives seen and not taken: one
  shared module holding all presentation (breaks rule 4); a shell rendering generic widgets from
  contracts alone (widens every facade). Graduates to a later ADR with the first component that has a
  view, or a committed TUI, whichever comes first.

## Parking lot

Product detail heard during scaffolding but deliberately not acted on — it belongs to `specify` and
`architect`, not here.

- (empty — the project was scaffolded before its capabilities were described)
