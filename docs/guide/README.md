# Codefall guide

These pages hold the detail the [README](../../README.md) leaves out. Start with the README if you
are new to Codefall. [`docs/workflow.md`](../workflow.md) is the map of what each skill reads and
writes; the pages here explain each skill for the people using it.

## Installing and setting up

- [Installing codefall](install.md): the install script's options, mise, and release downloads.
- [Setting up a project](setup.md): what `init` and `upgrade` write, the hooks, and every CLI
  command in full.
- [Configuration](configuration.md): review and consult agents, how each coding agent is started,
  the `config` command, and personas.

## The skills

| Skill | Page |
| --- | --- |
| `/codefall-envision` | [Visions](envision.md) |
| `/codefall-specify` | [Specifications](specify.md) |
| `/codefall-mock-up` | [Mockups](mock-up.md) |
| `/codefall-report` | [Bug reports](report.md) |
| `/codefall-fix` | [Small changes](fix.md) |
| `/codefall-plan` | [Plans and the task graph](plan.md) |
| `/codefall-implement` | [Building the plan](implement.md) |
| `/codefall-test` | [Running tests](test.md) |
| `/codefall-review` | [Reviews](review.md) |
| `/codefall-equip` and `/codefall-refresh` | [The local environment](local-environment.md) |
| `/codefall-scaffold` | [The architecture it sets up](architecture.md) |
| `/codefall-upgrade` | [Upgrading a project's documents](upgrade.md) |

## How do the skills hand work to each other?

### Who starts a skill?

You start `/codefall-upgrade` and `/codefall-equip` yourself. Each one changes the project's install
or its settings, so each carries `disable-model-invocation: true`, which stops an agent from
running it on its own. An agent may run any other skill when the work calls for it.

### Does a skill send you off to run another one?

A skill does not send you off to run another one. When it needs the output of an earlier step, it
runs that step itself:

- `/codefall-specify` runs `/codefall-mock-up` when a requirement needs a picture of a screen.
- `/codefall-plan` runs `/codefall-specify` when the spec it plans from is still a draft, and
  `/codefall-mock-up` for a requirement whose mockup was skipped.
- `/codefall-review` and `/codefall-test` run `/codefall-implement` to build the fixes you chose.

Each skill still reports what it found, offers changes, and applies only the ones you take.

### Who merges?

No skill merges to `main`. A person merges every code pull request. A document pull request, such
as a new spec, merges when a person adds the `auto-merge` label or a colleague approves it, once the
project has installed the GitHub Action for it with `/codefall-equip landing`.

### What is a delivery?

A *delivery* is all the work on one epic, from the plan to the merge. An *epic* is the bead that
groups every task from one plan, and a *bead* is one task in
[Beads](https://github.com/gastownhall/beads), the task tracker Codefall keeps in your git
repository. A delivery proceeds in *rounds*. Each round is one pass of `/codefall-implement`,
`/codefall-review`, and `/codefall-test`, and the problems a round finds become the next round's
work.

### Can you clear the agent's context between skills?

Yes. Every handoff between skills is a bead, a document, a report, or a pull request, so running
`/clear` between skills loses nothing. Each skill's report ends with the one next step.

### What happens when a question has no answer?

Sometimes a skill reaches a question of fact that you cannot answer in its interview, such as how
the existing system behaves in a case nobody has checked. The skill then *consults*: it puts the
question once to the consult agents listed in the project's settings. The answer comes back as a
proposal for you to confirm. A skill never consults on a matter of preference.
