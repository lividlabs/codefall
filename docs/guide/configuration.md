# Configuration

This page explains the settings you can change after `codefall init`: which agents review your work
and answer questions, how Codefall starts each coding agent, and how the skills talk to you.
[Setting up a project](setup.md) covers everything else `init` writes.

Codefall keeps these settings in two files:

| File | Describes | Checked in |
| --- | --- | --- |
| `.codefall/settings.json` | the project: what `init` recorded, and what the skills read back | yes |
| `.codefall/user.json` | you, the person at the keyboard | no |

You can edit either file by hand, but the `codefall config` command does it for you and refuses a
value the settings would not accept.

## An example

Here is the part of a `settings.json` this page covers:

```json
"agents": [
  { "activeAgent": "default", "review": [{ "harness": "current" }], "consult": [{ "harness": "current" }] },
  { "activeAgent": "muse", "review": [{ "harness": "claude" }, { "harness": "codex", "model": "gpt-5-codex" }] }
],
"harnessConfig": {
  "codex": {
    "provider": "amazon-bedrock-runtime",
    "args": ["-c", "model_reasoning_effort=high"],
    "env": "aws configure export-credentials --format env"
  }
}
```

It says three things:

- In most coding agents, reviews and questions go to a subagent of the agent you are using.
- When you run Codefall in Muse, Claude Code reviews your work first. If Claude Code is not
  installed on this machine, Codex reviews it with the `gpt-5-codex` model.
- Codex runs through Amazon Bedrock, with high reasoning effort, using AWS credentials exported
  just before it starts.

The rest of this page explains each part.

## Choose who reviews and who answers questions

Some skills need a second reader. `/codefall-review` hands the code or document to a *reviewer*, a
different agent from the one that wrote it. Several skills also *consult*: when a run reaches a
question of fact it cannot settle, such as how the existing system behaves in a case nobody
described, it asks another agent once and brings the answer to you as a proposal. The `agents` list
in `settings.json` decides who those readers are.

Codefall calls each coding agent a *harness* and names it after its command: `claude`, `codex`,
`opencode`, `agy`, or `muse`. The harness you are running Codefall in is the *active agent*, and each
entry in `agents` applies to one active agent:

- `activeAgent` is a harness name, or `default` for every harness without an entry of its own.
- `review` is the list of agents that review, tried in order.
- `consult` is the list of agents that answer questions, tried in order.

Each agent in a list names a harness and, optionally, a model. The harness `current` means a
subagent of whichever harness you are running in.

A run picks its list this way:

1. It reads the entry for the harness it is running in. If there is none, it reads `default`.
2. If that entry leaves out a list, it uses the `default` entry's list.
3. It tries each agent in order and skips any agent this machine cannot start.

Because a run skips agents it cannot start, one checked-in file works on every machine on the team.

`init` writes a single `default` entry whose `review` and `consult` lists each hold `current`. That
matches what every skill did before the list existed. `codefall doctor` reports which of the listed
harnesses this machine can run, and which lists never fall back to `current`.
[ADR-009.3](../adrs/ADR-009.3-agents.md) records the shape of the list.

## Configure how a harness starts

When a list names a harness, Codefall starts it from the command line. `harnessConfig` says how,
keyed by the name the list uses. A block can hold these fields:

| Field | What it sets |
| --- | --- |
| `modelFlag` | the flag the model is passed with, when the harness's usual flag is not the right one |
| `provider` | the harness's own name for the provider it reaches the model through |
| `args` | extra command-line arguments, appended as written |
| `env` | a shell command whose output runs before the harness starts, so its `export` lines take effect |
| `harness` | for a variant only: which harness binary the variant runs |

A key that is a harness name, such as `codex`, configures that harness. Any other key is a
*variant*: a second way to call a harness, whose `harness` field names the binary. With a variant
named `codex-direct`, for example, a project can call Codex through a provider for reviews and
directly for consults. A harness with no block runs with no extra settings.

The model string is whatever the harness accepts for its provider, which is easy to get wrong by
hand. The reliable way to set it is to run `/codefall-equip agents` in any harness. The skill finds
the model and provider the harness uses, writes the block, adds the agent to the lists you choose,
and runs the agent once to prove it works.

## Edit the settings with codefall config

Run `codefall config` in a terminal to open an editor with four entries:

- **Agents** lists the active agents, `default` first and then each harness. Open one to see its
  Review and Consult lists.
- **Reviews** sets whether `/codefall-review` posts its findings to the pull request.
- **Persona** sets your persona, described below.
- **Quit** leaves the editor.

In a Review or Consult list, these keys apply:

| Key | Action |
| --- | --- |
| shift and the arrow keys | move an agent up or down |
| `a` | add an agent |
| `d` | remove an agent |
| `c` | drop this list, so the `default` entry's list applies |
| enter | save |
| esc | go back |
| `q` | quit |

When the editor closes, it prints one line for each change it saved, the same line the matching
subcommand prints.

## Set the configuration from a script

Each setting also has a subcommand, for scripts and for setting one value quickly:

| Command | What it does |
| --- | --- |
| `codefall config show` | Prints the whole configuration: agents, harness blocks, review posting, and persona. |
| `codefall config agents` | Prints the agents lists. |
| `codefall config agents muse review claude codex:gpt-5-codex` | Sets one list: here, the review list for sessions in Muse. Each agent is a harness with an optional model after a colon. |
| `codefall config agents muse review --clear` | Removes that list, so the `default` entry's list applies. |
| `codefall config harness` | Prints the `harnessConfig` blocks. |
| `codefall config harness codex --provider amazon-bedrock-runtime --env "aws configure export-credentials --format env"` | Replaces one block with exactly the flags given. The flags are `--harness`, `--model-flag`, `--provider`, `--arg` (repeat it for several), and `--env`. |
| `codefall config harness codex --clear` | Removes the block, so the harness runs with no extra settings. |
| `codefall config review posting on` | Lets review post its findings to the pull request. `off` stops it. |
| `codefall config persona` | Prints your persona and where it came from. |
| `codefall config persona product-manager` | Sets your persona. |

Run with no subcommand and no terminal to draw on, `codefall config` prints the same output as
`codefall config show`. Every subcommand refuses a value the settings would not accept, as the
editor does.

## Set your persona

Your *persona* tells the skills how to talk to you. There are two:

- `engineer`, the default, which changes nothing.
- `product-manager`, which has the skills speak in product terms and hold technical decisions for an
  engineer.

The persona lives in `.codefall/user.json`, beside `settings.json`. That file describes you rather
than the project, so it is never checked in: `init` adds it to `.gitignore`. Its one field today is
`persona`, and a missing file or a missing field means `engineer`.

To set it, run:

```sh
codefall config persona product-manager
```

The command creates the file and its `.gitignore` line if either is missing. You can also write the
file by hand:

```json
{"version": 1, "persona": "product-manager"}
```

`codefall doctor` reports your persona and fails if it cannot read the file. The schema is
[`cli/schemas/user.schema.json`](../../cli/schemas/user.schema.json).

### What the product-manager persona changes

A persona changes how the skills talk to you. It never changes what they are allowed to do: every
rule a skill carries still holds, and the agents a skill starts for its work never see the persona.
Under `product-manager`:

- Every skill reports in product terms: what a user gets, what changes for them, and what is still
  open.
- `/codefall-envision` and `/codefall-specify` give their interviews more room.
- `/codefall-plan` does not decide technical questions you decline or cannot answer, and it does not
  take your silence as a choice. It sets each one aside in a **Decisions needed** section of the
  plan, then asks you one question: settle them with sensible defaults now so building can start,
  or leave them for an engineer? If you say now, it picks each default, tells you each one in a plain
  sentence, and carries on to `Ready` and the beads in the same run. If you say leave them, the plan
  stays `Draft`, no beads are created, and an engineer's run of `/codefall-plan` settles them later.
- The skills that write code or change the project's setup (`/codefall-implement`,
  `/codefall-fix`, `/codefall-equip`, `/codefall-scaffold`, and `/codefall-upgrade`) first say in
  one sentence what the run will do, and ask whether to continue.

Everything each persona changes is in one file, `.codefall/shared/personas.md`.
[ADR-011.2](../adrs/ADR-011.2-personas.md) records the decision.
