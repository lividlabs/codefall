# Setting up an agent from inside its harness

The agents track in full: which harness this is, where its own session records the model and
provider it is using, the one question, what is written where, and how the result is proven. The
track runs inside the harness being set up, because that harness is the one that knows the exact
model string its provider accepts, and nobody should have to remember that `gpt-6-astra` is
`us.openai.gpt-6-astra` on Bedrock. The shape being written is *How a harness is called* in
`../../../../.codefall/shared/running-agents.md`.

## Contents

- [The declaration](#the-declaration)
- [Which harness this is](#which-harness-this-is)
- [Reading this session's record](#reading-this-sessions-record)
- [The one question](#the-one-question)
- [Writing the block and the lists](#writing-the-block-and-the-lists)
- [Proving it](#proving-it)
- [What the user owes](#what-the-user-owes)

## The declaration

Read `harnessConfig` and `agents` out of `.codefall/settings.json`, and say what they hold today:
each block by key with its fields, and each active agent's `review` and `consult` lists. No
`.codefall/settings.json` at all: stop and say `codefall init` comes first.

## Which harness this is

Answer it per *Which harness this is* in `../../../../.codefall/shared/running-agents.md`: one of
`claude`, `codex`, `muse`, `opencode`, `agy`, checked against the directory this skill was loaded
from. This track cannot run as `unknown`: the whole point is to read the session's own record, and
a session that cannot say what it is has no record to read. Say so and stop.

## Reading this session's record

Every harness keeps a record of the session it is running, and most of them put the exact model
string and the provider in it. Read it, and show what was read before asking anything. Where the
record is missing, ambiguous, or has no model in it, ask the user for the model string instead and
say why. Whatever the source, the user confirms the value before it is written.

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
`message.model` holds the model string. Claude Code records no provider name; take it from the
environment: `CLAUDE_CODE_USE_BEDROCK` set means `bedrock`, `CLAUDE_CODE_USE_VERTEX` means
`vertex`, `CLAUDE_CODE_USE_FOUNDRY` means `foundry`, none means the first-party API and no
`provider` field.

**opencode.** Nothing in the shell names the session. `~/.local/share/opencode/opencode.db`, table
`session`, has a `model` column holding `{"id": ..., "providerID": ...}`; the newest row whose
`directory` is the project and whose `parent_id` is null is most likely this session, and that is a
guess when two sessions share a directory. Say it is a guess, show the row, and confirm. OpenCode's
model string is `providerID/id`, which is what its `--model` flag takes, so the list item's `model`
carries the whole thing and the block needs no `provider`.

**muse.** Muse tells its agent the session log's path in the session-identity note in its context.
The last `run.model.configured` record in that log carries `model_id` and `provider_id`.

**agy.** agy records only a display label for its model, so there is nothing exact to read. Ask
the user for the model string, and say that is why.

## The one question

Show what was read, then ask one question with three parts, each with a proposed answer:

1. **The key.** The harness name when `harnessConfig` has no block for it, or when the block it has
   says the same provider. A different provider gets a variant key, proposed as
   `<harness>-<provider>`, with `harness` set inside the block.
2. **The block.** `provider` as read. `env` when the provider is Bedrock and the harness reaches it
   through an AWS SDK: propose `aws configure export-credentials --format env`, and say why — a
   headless run has no interactive shell to export credentials for it, and the SDK cannot refresh
   an `aws login` session the CLI can. `args` only when the user names some; the reasoning-effort
   argument is the usual one. `modelFlag` only when the harness's flag is not `--model`.
3. **The lists.** Which active agents' `review` and `consult` lists gain `<key>:<model>`, and where
   in each. Propose the entries for every other harness the project has under `harnesses`, and
   `default`, with the new agent first and `current` kept last, and let the user trim it.

Never propose an entry for the active agent that is this harness: a session in codex reviewing
through codex is the second opinion ADR-009 says is worth less.

## Writing the block and the lists

Before the first write, take the branch step of `../../../../.codefall/shared/landing.md`: standing
on the default branch, `git switch -c equip/agents`; on another branch, ask once which to use.

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

Exit `0` and a non-empty out-file is the proof. Any other exit is reported with the script's stderr
and the block is revised, not left: `76` with a credentials error means the `env` command is
missing or wrong; `64` names a field the harness cannot take; `69` means the binary is not on this
machine's `PATH`, which `doctor` will also say. Then run `codefall doctor` when the CLI is on `PATH`
and read its **Agents** section.

## What the user owes

The report says what was written: the block by key, each list changed, and the proof's result. Then
the landing, per `../../../../.codefall/shared/landing.md`: settings committed by path on the
branch, and the push and pull request offered — its own pull request, never another verb's. The
merge is the user's. **End with what the user does next**: merge the pull request; run the same
track inside each other harness the project wants configured, since each harness knows only its
own model strings.
