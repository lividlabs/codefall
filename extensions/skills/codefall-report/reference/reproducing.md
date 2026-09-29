# Reproducing a bug

How `codefall-report` tries to reproduce a bug while the reporter is still there. Read at step 5.
The attempt checks the report: a step it cannot follow is a step the report is missing.

## Contents

- Where the attempt runs
- The driver
- The confirmation
- Working the steps
- Evidence
- A bug that happens some of the time
- The three outcomes
- What an attempt never does

## Where the attempt runs

**The local environment, at the checkout's commit, started through the project's declared `start`**
— the one `local.start` names in `.codefall/settings.json`. Nothing else is started, and a shared or
production environment is never driven: a bug seen only there is recorded as `Not attempted`, with
that reason, and the reporter's evidence carries the report.

Read the preflight lines from step 1 first.

| Line | What happens |
| --- | --- |
| `behind` above `0`, or `refresh=stale` | Say the environment may not match `main`, offer `/codefall-refresh`, and wait |
| `refresh=undeclared` | Say so and name `/codefall-equip`. A command-line surface the repository runs directly can still be tried; anything that needs services running is `Not attempted` |

**Never run the remedy.** It is the user's.

## The driver

**The surface decides the kind of driver.** A browser tool for a web or desktop-shell surface; the
shell for a command-line or HTTP surface, and the shell is always there. Read this session's own tool
list and name the driver at the confirmation.

**One live call proves it** before the attempt depends on it — a request to the application's health
endpoint or front page. A failure is reported as a failure, never routed around by switching tools
mid-attempt. With no working driver for the surface, the attempt is `Not attempted`, with that reason.

## The confirmation

Nothing is driven before the reporter sees what will be: the steps as you will run them, the driver,
the commit, and any data the steps create — an order placed, an email sent, a record saved. Ask
once, and take no as `Not attempted — the reporter declined`.

## Working the steps

**Follow the reporter's steps as written, in order.** A step you cannot follow without guessing — a
screen you cannot find, an account you do not have, data that is not there — is not guessed at: stop
there, and that step is the question you take back to the interview.

Observe at every step, not just the last one, so a difference is placed at the step where it first
appears.

**Setup the reporter's steps assume is done through commands the project already declares** — its
seed scripts, the commands the testing root's `AGENTS.md` names. A state nothing can force is a
reason for `Not reproduced`, said plainly.

## Evidence

At the step where the product goes wrong, capture what the driver can:

- a screenshot of the surface;
- the console output, and the failing request and its response, for a web surface;
- the command, its exit status, and its output, for a command-line surface;
- the lines the `start` command's own output shows at that moment.

Keep captures in the session's scratch space until step 8 copies them into `docs/bugs/BUG-NNN-slug/`
as `repro-NN-<what-it-shows>.<ext>`. Cut output to what bears on the bug, and strip tokens,
passwords, and personal data from anything kept.

## A bug that happens some of the time

Run the steps up to five times and record the count — "reproduced 2 of 5". One success is
`Reproduced`; zero of five is `Not reproduced`, whatever the reporter's frequency was.

## The three outcomes

The report's `**Reproduced:**` row records one, and its **Reproduction** section says what ran.

| Outcome | Row | The section holds |
| --- | --- | --- |
| Reproduced | `Yes — <date>, at <short commit>, <driver>` plus the count when it was intermittent | The steps as run, where it went wrong, the evidence names |
| Not reproduced | `No — <date>, at <short commit>` | The steps as run, and where the product did what the reporter expected instead |
| Not attempted | `Not attempted — <reason>` | Nothing, when the reason says it all |

`Not reproduced` is not a finding that the bug is absent. The report is written, and
`codefall-design` reads the row before it cuts any work.

## What an attempt never does

- **Never changes code, configuration, a test, or data by hand.** What the steps themselves create is
  the only change, and it is named at the confirmation.
- **Never mocks, fakes, or intercepts** anything.
- **Never looks for the cause.** Reading code to find a control's name is allowed; reading it to
  explain the bug is `codefall-design`'s work.
- **Never replaces the reporter's evidence** with its own. Both are kept, each named for where it came
  from.
