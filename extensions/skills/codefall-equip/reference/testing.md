# Setting up the test harness

The testing track in full: what the declaration says, what to search for, the runner each surface
takes, what setting one up means, what is written where, and how the result is proven. `<root>` is
the testing root `test.dir` declares in `.codefall/settings.json`.

## Contents

- [The declaration](#the-declaration)
- [What to search for](#what-to-search-for)
- [The one question](#the-one-question)
- [Consulting before the one question](#consulting-before-the-one-question)
- [The runner each surface takes](#the-runner-each-surface-takes)
- [Setting up Playwright](#setting-up-playwright)
- [Setting up `go test`](#setting-up-go-test)
- [Declaring the runner](#declaring-the-runner)
- [The Runners line](#the-runners-line)
- [Revising `update`](#revising-update)
- [Proving it](#proving-it)
- [What the user owes](#what-the-user-owes)

## The declaration

Read the `test` block out of `.codefall/settings.json`:

```json
"test": {
  "dir": "testing",
  "runners": ["playwright"]
}
```

- **No block, or no settings file.** Stop. `codefall init` asks for the testing root and makes the
  tree under it; this skill declares neither, and nothing else here can be done without a root.
- **A block whose `dir` names a directory that is not there.** Stop and say `codefall init` makes
  the tree; `codefall doctor` fails `test-dir-exists` on the same state.
- **`runners` already naming a runner.** This run is a revision: read the configuration that runner
  uses, change the smallest thing the project's tools now need, and leave the rest.

## What to search for

Read every hit rather than counting it. The searches answer two questions at once — which runner is
already here, and which surfaces the repository has.

| Look for | Where | What it says |
| --- | --- | --- |
| `playwright.config.*` | repository root, package roots | Playwright is installed; read its test directory, its match, its retries, and its workers |
| an `e2e` or `test:e2e` script | `package.json` scripts | the command the project already runs, and the name to keep |
| `@playwright/test`, or another browser runner | `package.json` dependencies | which runner the project chose |
| `*_e2e_test.go`, a `//go:build e2e` line, a `test-e2e` target | Go packages, `Makefile` | a Go end-to-end suite already exists |
| `e2e/`, `tests/e2e/`, `integration/`, `<root>/test-cases/` | anywhere | where end-to-end tests are kept today |
| a filled Runners section | `<root>/AGENTS.md` | a runner somebody recorded by hand |
| a browser framework, an `electron` dependency, an HTTP server, a `go.mod` | manifests | the surfaces the question has to cover |

A project keeping its end-to-end tests somewhere other than `<root>/test-cases/` is not moved. The
cases go where the declaration says; what the project has stays where it is, and the question names
both.

## The one question

One question, with everything found beside it:

> This repository has a Next.js front end and a Go API, and no end-to-end runner: no
> `playwright.config.*`, no `*_e2e_test.go`, and nothing under `e2e/`. I can set up Playwright for
> the front end and the API and `go test` for the Go packages, or declare a runner you already run.
> Which?

Three answers:

- **Declare what exists.** Read the configuration, check that it collects specs from
  `<root>/test-cases`, and go to [Declaring the runner](#declaring-the-runner). A configuration
  pointed somewhere else is a revision, not a declaration: change its test directory and say so.
- **Set up the default runner for each surface**, per the table below.
- **A surface this version refuses.** Say it here rather than offering it: React Native, Tauri, and
  Flutter take no `spec` runner yet. Their cases run in the `agentic` modality wherever
  `codefall-test` finds a driver, and nothing is declared for them.

**A runner outside the two known names** — Cypress, WebdriverIO, Vitest browser mode — is a plain
report: `test.runners` has no name for it, so `codefall-test` reads the project as unequipped and a
`spec` case has no declared runner. Offer Playwright for the cases under `<root>/test-cases/`,
alongside whatever the project keeps running, and take the answer.

## Consulting before the one question

When the search leaves a surface's runner open — a repository with end-to-end tests under two
runners, or a surface the table below does not name plainly — consult once before the question,
the same way the local track does in *Consulting before the one question* in `signals.md`: the
question is which runner the surface takes, the files are the runner configurations and test
directories found, the options are the runners this file sets up and "declare what exists". The
answer becomes the proposed option in the one question and the user still chooses. A consult never
installs a runner and never lifts the refusal for a surface this version has no spec runner for.

## The runner each surface takes

| Surface | Runner | Spec suffix |
| --- | --- | --- |
| Browser front end | Playwright | `.e2e.ts` |
| Electron shell | Playwright | `.e2e.ts` |
| HTTP API | Playwright, through its request fixture | `.e2e.ts` |
| Go service, Go command line | `go test` | `_e2e_test.go` |
| React Native, Tauri, Flutter | none in this version | — |

A repository with two surfaces gets two runners, and `test.runners` names both. The suffix is what
pairs a spec with its case: a spec sits beside `<slug>.md` and its name begins with the slug.

## Setting up Playwright

**The dependency** is a development dependency, installed with the project's own package manager:
`npm install -D @playwright/test`, or the pnpm, Yarn, or Bun form the lockfile calls for. The
browsers are not installed here — that belongs in `update`.

**The configuration** is `playwright.config.ts` at the root of the package that owns the surface,
with the declared root written out in place of `<root>`:

```ts
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: 'testing/test-cases',
  testMatch: '**/*.e2e.ts',
  retries: 0,
  workers: 1,
  outputDir: 'testing/.artifacts/test-results',
  reporter: [['list'], ['html', { outputFolder: 'testing/.artifacts/playwright-report' }]],
  use: { baseURL: process.env.BASE_URL ?? 'http://localhost:3000' },
});
```

- **The test directory is `<root>/test-cases`** so that a spec is collected where its case file
  already is, and **the match is the suffix** so that the case's own markdown is never collected.
- **Retries are `0` and there is one worker.** A rerun is a new run with its own report, and specs
  run against the services the declared `start` brought up.
- **Everything a run produces goes under `<root>/.artifacts/`**, the entry `codefall init` already
  added to `.gitignore`.
- **The base URL is the project's**, taken from what the declared `start` command brings up.

Commands, as the Runners line will carry them: all specs `npx playwright test`; one case
`npx playwright test testing/test-cases/<area>/<slug>.e2e.ts`; the list `npx playwright test --list`.
A project whose idiom is package scripts gets `"e2e": "playwright test"` and the lines name
`npm run e2e` instead.

## Setting up `go test`

**There is nothing to install.** The specs are a Go package under `<root>/test-cases/<area>/`, so
that a spec sits beside its case, and each file is named `<slug>_e2e_test.go`.

**Every spec file opens with a build tag**, so an ordinary unit run never reaches one:

```go
//go:build e2e

package checkout_test
```

The package name is the area's, suffixed `_test`. The testing root has to be inside the module; when
a nested module owns the surface instead, the package goes under that module and the Runners line
says where it is.

Commands: all specs `go test -tags e2e ./testing/test-cases/...`; one case
`go test -tags e2e -run <TestName> ./testing/test-cases/<area>/`; the list
`go test -tags e2e -list '.*' ./testing/test-cases/...`. There is nothing to configure for retries
or parallelism: `go test` retries nothing, and a spec that never calls `t.Parallel()` runs alone.

## Declaring the runner

Write `runners` into the `test` block, keeping every other key and the file's formatting:

```json
"test": {
  "dir": "testing",
  "runners": ["playwright", "go-test"]
}
```

Only `runners` changes. `dir` stays exactly as it was written, and no other block is touched. The
two names are the only ones the format accepts; a runner outside them is described in
`<root>/AGENTS.md` and named nowhere in settings. What reads this: the check scripts, to decide
which runner-specific rules apply, and `codefall doctor`, for `test-equipped`.

## The Runners line

`<root>/AGENTS.md` has a `## Runners` section that `codefall init` left empty. Fill it with one line
per runner — its name, the command that runs every spec, and the command that runs one case:

```markdown
- **playwright** — all: `npm run e2e`. One case: `npm run e2e -- testing/test-cases/<area>/<slug>.e2e.ts`.
- **go-test** — all: `go test -tags e2e ./testing/test-cases/...`. One case: `go test -tags e2e -run <TestName> ./testing/test-cases/<area>/`.
```

That file is the project's document, so nothing outside the Runners section is touched, and a line
already there for the same runner is replaced rather than repeated. Its other sections — setup and
state-forcing commands, real side effects, environment notes — hold facts only the project knows;
name the empty ones in the report rather than guessing at them.

## Revising `update`

The runner's own install is part of making the environment match the checkout, so it goes into the
declared `update` command by the local track's step 3: the tool's own idempotent form, one line of
comment saying what it brings current, the smallest change to the script the project wrote, and the
diff shown before anything is written.

- **Playwright** needs its browsers, which no lockfile carries:
  `npx playwright install --with-deps chromium`, after the dependency install step.
- **`go test`** needs nothing beyond the `go mod download` an `update` already runs, unless a spec
  builds with a generator or a tool of its own; that follows the same rule.

The contract holds afterwards. `playwright install` against browsers already present exits `0`
quickly, so a second `update` is still cheap — and that is re-proven by running `update` twice, as
the local track does.

## Proving it

In order, and all four:

1. **The runner's list command against the empty tree.** Playwright exits `0` and lists no specs,
   which is what shows the configuration collects from where it says. `go test` has no package to
   compile until the first spec exists and answers `matched no packages`; that is the expected
   answer here, and the command's real proof is the first spec `codefall-implement` writes.
2. **`../../../../.codefall/shared/check-cases.sh`.** Zero cases is a pass; what it proves now is
   that the root it reads from settings is the root the configuration points at.
3. **`../../../../.codefall/shared/check-cases-playwright.sh`**, when Playwright is declared.
4. **`codefall doctor`**, whose **Testing** checks — `test-declared`, `test-equipped`,
   `test-dir-exists` — all pass.

A failure is fixed in the configuration and the proof re-run. A failure that belongs to the
environment rather than the configuration — a package manager that cannot reach the network — is
reported as that, with what to do.

## What the user owes

**Setting a harness up is its own pull request**: the configuration, the dependency and its
lockfile, the `update` revision, `test.runners`, and the Runners line, and nothing else. A task's
pull request never carries harness setup, and no verb sets one up on the way to something else.

The first case comes afterwards, through `/codefall-implement`, from criteria `/codefall-design`
put in the bead.
