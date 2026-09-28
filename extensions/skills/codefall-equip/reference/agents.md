# Setting up an agent from inside its harness

The agents track in full: what to show, the one question, and what each answer leads to. The track
runs inside the harness being set up, because that harness is the one that knows the exact model
string its provider accepts, and nobody should have to remember that `gpt-6-astra` is
`us.openai.gpt-6-astra` on Bedrock. The shape being written is *How a harness is called* in
`../../../../.codefall/shared/running-agents.md`.

**Every reply on this track fits on one screen.** Reasons, paths, environment variables, and
alternatives stay in this file. The user sees the configuration, one question, and then the one
change they chose.

## Contents

- [Which harness this is](#which-harness-this-is)
- [Showing the configuration](#showing-the-configuration)
- [The one question](#the-one-question)
- [Choice 1 — set up this harness](#choice-1--set-up-this-harness)
- [Choice 2 — change the lists](#choice-2--change-the-lists)
- [Choice 3 — nothing](#choice-3--nothing)
- [Reading this session's record](#reading-this-sessions-record)
- [Writing](#writing)
- [Proving it](#proving-it)
- [The report](#the-report)

## Which harness this is

Answer it per *Which harness this is* in `../../../../.codefall/shared/running-agents.md`: one of
`claude`, `codex`, `muse`, `opencode`, `agy`, checked against the directory this skill was loaded
from. This track cannot run as `unknown`: the whole point is to read the session's own record, and
a session that cannot say what it is has no record to read. Say so and stop.

## Showing the configuration

Read `harnesses`, `harnessConfig`, and `agents` out of `.codefall/settings.json`. No
`.codefall/settings.json` at all: stop and say `codefall init` comes first.

When `codefall doctor` is on `PATH` and its settings checks say the file does not match the schema,
show nothing of it. Say one line and go straight to choice 1:

> The agents config is out of date and can't be shown. I'll rebuild it.

Otherwise render it in exactly this shape, with the running harness in place of `claude`:

```markdown
# Agents in this project

## Harnesses

- **claude**: bedrock, credentials from aws
- **codex**: not set up

## Who claude asks

- **Review**: codex (gpt-6-sol), then itself
- **Consult**: codex (gpt-6-astra), then itself
```

- **Harnesses** lists every name under `harnesses`, one line each. A name with a `harnessConfig`
  block gets its `provider` in plain words, and `credentials from aws` when the block has an `env`
  command. A variant key is its own line under the harness it runs, as
  `**codex-direct**: codex, first-party`. A name with no block is `not set up`.
- **Who `<harness>` asks** shows the `review` and `consult` lists of this harness's entry, falling
  back to `default` per *Resolving the list*. Each item is `<harness> (<model>)`, with the model
  shortened to what a person says: `gpt-6-sol`, not `codex:us.openai.gpt-6-sol`. `current` is
  `itself`. Items are joined with `, then`. A list that is one `current` agent is `itself`.

Nothing else: no file paths, no environment variables, no explanation of what a field means.

## The one question

Directly under the configuration, ask:

```markdown
Would you like to change anything?

1. Set up codex (this harness) so others can call it
2. Add, reorder, or remove reviewers or consultants
3. Nothing, this is fine
```

Name the running harness in item 1. A harness that already has a block keeps item 1, worded
`Change how codex (this harness) is called`. The list is plain prose; no harness-specific prompt
tool, since not every harness has one and the configuration has to stay on screen with the question.

## Choice 1 — set up this harness

Read the session's record, per [Reading this session's record](#reading-this-sessions-record),
and work out the block without narrating the reading:

- **The key.** The harness name when `harnessConfig` has no block for it, or when the block it has
  says the same provider. A different provider gets a variant key, `<harness>-<provider>`, with
  `harness` set inside the block.
- **The block.** `provider` as read. `env` set to `aws configure export-credentials --format env`
  when the provider is Bedrock and the harness reaches it through an AWS SDK. `args` only when the
  user names some. `modelFlag` only when the harness's flag is not `--model`.
- **The lists.** `<key>:<model>` first, `current` kept last, in the `review` and `consult` lists of
  every other harness under `harnesses` and of `default`. Never in this harness's own entry: a
  session in codex reviewing through codex is the second opinion ADR-009 says is worth less.
- **Hooks stay off.** `run-agent.sh` starts codex, claude, and opencode with their hooks off, and
  a headless run must stay that way: one that loaded the project's session hooks followed them
  into a refresh instead of answering. `args` never carries a flag that turns hooks back on. Muse
  and agy have no switch, so a block for either says so in the confirmation, in one clause: `its
  hooks run during headless calls`.

Then show the result in the same shape as the configuration, and ask one yes-or-no:

```markdown
## Set up claude

- **Model**: us.anthropic.claude-opus-5-5[1m] on bedrock, credentials from aws
- **Added first to**: codex, muse, and default, for review and consult

Write this?
```

The model string is shown in full here, once, because it is the one thing the user is confirming.
A record that cannot be read, or has no model in it, means asking for the model string in one
line, and saying in one clause why. On yes, [write](#writing), [prove](#proving-it), and
[report](#the-report). On no, or on an edit, take the edit and ask again.

## Choice 2 — change the lists

Ask which list, in one line: `Which list, and what should it read?` The user answers in words;
render the result as the **Who `<harness>` asks** section and ask `Write this?`. An item that
names a harness with no block is fine to write, and the report says `codefall doctor` will warn
until that harness is set up from inside itself. On yes, [write](#writing) and
[report](#the-report); the probe runs only when the change added an agent this machine can start.

## Choice 3 — nothing

Say `Nothing changed.` and stop. No branch, no commit.

## Reading this session's record

Every harness keeps a record of the session it is running, and most of them put the exact model
string and the provider in it. Read it silently; what the user sees is the block it produced.

**codex.** The shell codex gives its tool calls carries `CODEX_THREAD_ID`. The session's row in
`$CODEX_HOME/state_5.sqlite` (default `~/.codex/state_5.sqlite`), table `threads`, holds `model`
and `model_provider`:

```
sqlite3 "file:${CODEX_HOME:-$HOME/.codex}/state_5.sqlite?mode=ro" \
  "select model, model_provider from threads where id = '$CODEX_THREAD_ID'"
```

Without `sqlite3`, or without the variable, the rollout whose file name ends in that id, under
`$CODEX_HOME/sessions/YYYY/MM/DD/`, holds the same: `model_provider` on its `session_meta` line and
`model` on its last `turn_context` line, readable with jq. The provider name is the one codex's own
config uses, such as `amazon-bedrock-runtime`.

**claude.** The shell carries `CLAUDE_CODE_SESSION_ID`. The transcript is
`$CLAUDE_CONFIG_DIR/projects/*/<id>.jsonl`, default `~/.claude/projects/`, and the last line with a
`message.model` holds Claude Code's normalised name. The string Bedrock, Vertex, or Foundry
actually receives is in the environment when it is set: `ANTHROPIC_DEFAULT_OPUS_MODEL`,
`ANTHROPIC_DEFAULT_SONNET_MODEL`, or `ANTHROPIC_MODEL`, whichever matches the transcript's tier;
prefer it over the transcript's name. Claude Code records no provider name; take it from the
environment: `CLAUDE_CODE_USE_BEDROCK` set means `bedrock`, `CLAUDE_CODE_USE_VERTEX` means
`vertex`, `CLAUDE_CODE_USE_FOUNDRY` means `foundry`, none means the first-party API and no
`provider` field.

**opencode.** Nothing in the shell names the session. `~/.local/share/opencode/opencode.db`, table
`session`, has a `model` column holding `{"id": ..., "providerID": ...}`; the newest row whose
`directory` is the project and whose `parent_id` is null is most likely this session, and that is a
guess when two sessions share a directory. Say it is a guess in one clause of the confirmation.
OpenCode's model string is `providerID/id`, which is what its `--model` flag takes, so the list
item's `model` carries the whole thing and the block needs no `provider`.

**muse.** Muse tells its agent the session log's path in the session-identity note in its context.
The last `run.model.configured` record in that log carries `model_id` and `provider_id`.

**agy.** agy records only a display label for its model, so there is nothing exact to read. Ask
the user for the model string.

## Writing

Before the first write, take the branch step of `../../../../.codefall/shared/landing.md`: standing
on the default branch, `git switch -c equip/agents` and say so in one line; on another branch, ask
once which to use.

Write through the CLI when it is on `PATH`, since it validates the whole file and keeps its
formatting:

```
codefall config harness <key> [--harness <binary>] [--provider <name>] [--arg <a>]... [--env <cmd>]
codefall config agents <activeAgent> review <key>:<model> current
codefall config agents <activeAgent> consult <key>:<model> current
```

The `agents` command replaces a list, so pass the whole list the user confirmed, not only the new
agent. Without the CLI, edit `.codefall/settings.json` directly, keeping every other key and the
file's formatting, and say `codefall doctor` will check it.

## Proving it

Run the agent once, the way a verb will:

```
printf 'Reply with the single word ok.\n' > /tmp/codefall-probe.txt
../../../../.codefall/shared/run-agent.sh <key>:<model> /tmp/codefall-probe.txt \
  ../../../../.codefall/shared/consult.schema.json /tmp/codefall-probe.out
```

Exit `0` and a non-empty out-file is the proof, reported as one line: `claude answered the probe.`
Any other exit is reported in one line with the script's stderr, and the block is revised, not
left: `76` with a credentials error means the `env` command is missing or wrong; `64` names a field
the harness cannot take; `69` means the binary is not on this machine's `PATH`, which `doctor` will
also say. Then run `codefall doctor` when the CLI is on `PATH` and report its **Agents** section in
one line.

## The report

The configuration again, in the shape above, as it now stands. Then the probe's line and doctor's
line. Then the landing, per `../../../../.codefall/shared/landing.md`: settings committed by path
on the branch, and the push and pull request offered in one question. The merge is the user's.
**End with what the user does next**: merge the pull request; run `/codefall-equip agents` inside
each other harness the project wants set up, since each harness knows only its own model strings.
