# Setting up an agent

The agents track in full: what to show, the one question, and what each answer leads to. Any
harness can be set up from any other: this is settings, and the track's job is to find the
parameters so nobody has to remember that `gpt-6-astra` is `us.openai.gpt-6-astra` on Bedrock. The
shape being written is *How a harness is called* in `../../../../.codefall/shared/running-agents.md`.

**Every reply on this track fits on one screen.** Reasons, paths, environment variables, and
alternatives stay in this file. The user sees the configuration, one question, and then the one
change they chose.

## Contents

- [Which harness this is](#which-harness-this-is)
- [Showing the configuration](#showing-the-configuration)
- [The one question](#the-one-question)
- [Choice 1 — set up a harness](#choice-1--set-up-a-harness)
- [Choice 2 — change the lists](#choice-2--change-the-lists)
- [Choice 3 — nothing](#choice-3--nothing)
- [Finding a harness's parameters](#finding-a-harnesss-parameters)
- [Writing](#writing)
- [Proving it](#proving-it)
- [The report](#the-report)
- [Rules](#rules)

## Which harness this is

Answer it per *Which harness this is* in `../../../../.codefall/shared/running-agents.md`: one of
`claude`, `codex`, `muse`, `opencode`, `agy`, or `unknown`. It decides whose lists **Who
`<harness>` asks** shows, and it is the one harness whose parameters can come from the running
session. `unknown` shows the `default` entry and reads no session.

## Showing the configuration

No `codefall` on `PATH`: stop and say to install codefall. Read `harnesses`, `harnessConfig`, and
`agents` out of `.codefall/settings.json`. No `.codefall/settings.json` at all: stop and say
`codefall init` comes first.

When `codefall doctor`'s settings checks say the file does not match the schema, show nothing of
it. Say one line and go straight to choice 1:

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

1. Set up a harness
2. Add, reorder, or remove reviewers or consultants
3. Nothing, this is fine
```

The list is plain prose; no harness-specific prompt tool, since not every harness has one and the
configuration has to stay on screen with the question.

## Choice 1 — set up a harness

Ask which, as a numbered list of the names under `harnesses`, each with its state from the
configuration: `codex (not set up)`, `claude (bedrock)`. One name only skips the question.

Find that harness's parameters per [Finding a harness's parameters](#finding-a-harnesss-parameters)
without narrating the search, and work out the block:

- **The key.** The harness name when `harnessConfig` has no block for it, or when the block it has
  says the same provider. A different provider gets a variant key, `<harness>-<provider>`, with
  `harness` set inside the block.
- **The block.** `provider` as found. `env` set to `aws configure export-credentials --format env`
  when the provider is Bedrock and the harness reaches it through an AWS SDK. `args` only when the
  user names some. `modelFlag` only when the harness's flag is not `--model`.
- **The lists.** `<key>:<model>` first, `current` kept last, in the `review` and `consult` lists of
  every other harness under `harnesses` and of `default`. Never in the entry of the harness being
  set up: codex reviewing through codex is the second opinion ADR-009 says is worth less.
- **Hooks stay off.** `run-agent.sh` starts codex, claude, and opencode with their hooks off, and
  a headless run must stay that way: one that loaded the project's session hooks followed them
  into a refresh instead of answering. `args` never carries a flag that turns hooks back on. Muse
  and agy have no switch, so a block for either says so in the confirmation, in one clause: `its
  hooks run during headless calls`.

Then show the result in the same shape as the configuration, and ask one yes-or-no:

```markdown
## Set up codex

- **Model**: us.openai.gpt-6-sol on amazon-bedrock-runtime, from codex's config
- **Credentials**: from aws
- **Added first to**: claude, muse, and default, for review and consult

Write this?
```

The model string is shown in full here, once, because it is the one thing the user is confirming.
The `from` clause names the source in a few words: `from this session`, `from codex's config`,
`from codex's last session here`, `from the review settings`. Nothing found means asking for the
model string in one line. On yes, [write](#writing), [prove](#proving-it), and
[report](#the-report). On no, or on an edit, take the edit and ask again.

## Choice 2 — change the lists

Ask which list, in one line: `Which list, and what should it read?` The user answers in words;
render the result as the **Who `<harness>` asks** section and ask `Write this?`. An item that
names a harness with no block is fine to write, and the report says `codefall doctor` will warn
until that harness is set up. On yes, [write](#writing) and [report](#the-report); the probe runs
only when the change added an agent this machine can start.

## Choice 3 — nothing

Say `Nothing changed.` and stop. No branch, no commit.

## Finding a harness's parameters

Every place below is a clue. Look in this order and take the first place that has a model; the
provider comes from the same place. Read silently; what the user sees is the block it produced and
the `from` clause.

1. **This session**, when the harness being set up is the one running.
2. **That harness's own config**: the project's copy first, then the one in the home directory.
3. **Its most recent session in this project.**
4. **The review settings**: `.codefall/skills/codefall-review/CUSTOMIZE.md` in the project, which
   older projects used to name a reviewer's harness, model, or provider.
5. **The internet**, when this harness can search or fetch: the provider's own documentation for
   the model ID's current format. Use it to fill what the places above left out, and to check a
   string they gave when its format looks wrong for the provider.
6. **Ask.**

**codex.** This session: the shell carries `CODEX_THREAD_ID`, and the session's row in
`${CODEX_HOME:-~/.codex}/state_5.sqlite`, table `threads`, holds `model` and `model_provider`.
Config: `.codex/config.toml` in the project, then `${CODEX_HOME:-~/.codex}/config.toml`, keys
`model` and `model_provider`; a top-level `profile` key names a `[profiles.<name>]` table whose keys
win. Recent session: the same table, the newest row by `updated_at` whose `cwd` is the project root. Read the database read-only:

```
sqlite3 "file:${CODEX_HOME:-$HOME/.codex}/state_5.sqlite?mode=ro" \
  "select model, model_provider from threads where id = '$CODEX_THREAD_ID'"
```

The provider name is the one codex's own config uses, such as `amazon-bedrock-runtime`.

**claude.** The config directory is `$CLAUDE_CONFIG_DIR`, else `~/.claude`. This session: the
shell carries `CLAUDE_CODE_SESSION_ID`, and the last `message.model` in
`<config>/projects/*/<id>.jsonl` holds Claude Code's own name for the model. Config:
`.claude/settings.local.json` and `.claude/settings.json` in the project, then
`<config>/settings.json`; the `model` key and the `env` block may carry the provider and the model.
Recent session: the newest transcript under `<config>/projects/` for the project. Claude Code's own
name is not always the string the provider takes; when the environment or the `env` block sets the
model, prefer that. The provider: `CLAUDE_CODE_USE_BEDROCK` set means `bedrock`, `CLAUDE_CODE_USE_VERTEX`
means `vertex`, `CLAUDE_CODE_USE_FOUNDRY` means `foundry`, none means the first-party API and no
`provider` field.

**opencode.** Config: `opencode.json` or `opencode.jsonc` in the project, then in
`~/.config/opencode/`, key `model`, already `providerID/id`. Next, `~/.local/state/opencode/model.json`,
the first item of `recent`, as `providerID/modelID`. Recent session:
`~/.local/share/opencode/opencode.db`, table `session`, the newest row whose `directory` is the
project and whose `parent_id` is null; its `model` column holds `{"id", "providerID"}`. OpenCode's
`--model` takes `providerID/id` whole, so the list item's `model` carries it and the block needs no
`provider`.

**muse.** This session: Muse tells its agent the session log's path in the session-identity note
in its context, and the last `run.model.configured` record carries `model_id` and `provider_id`.
Config: `~/.config/muse/settings.json`, keys `model` and `provider`. Recent session: the newest log
under `~/.local/share/muse/sessions/` for the project, read the same way.

**agy.** agy records only a display label in `~/.gemini/antigravity-cli/settings.json`. Ask for
the model string, offering the label as the hint.

## Writing

Before the first write, take the branch step of `../../../../.codefall/shared/landing.md`: standing
on the default branch, `git switch -c equip/agents` and say so in one line; on another branch, ask
once which to use.

Write through the CLI, since it validates the whole file and keeps its formatting:

```
codefall config harness <key> [--harness <binary>] [--provider <name>] [--arg <a>]... [--env <cmd>]
codefall config agents <activeAgent> review <key>:<model> current
codefall config agents <activeAgent> consult <key>:<model> current
```

The `agents` command replaces a list, so pass the whole list the user confirmed, not only the new
agent.

A file that does not match the schema is the one exception: the CLI refuses to write to it. Write
the `harnessConfig` block and the `agents` lists into `.codefall/settings.json` directly, keeping
every other key and the file's formatting, then run `codefall doctor` and confirm its settings
checks pass before going on.

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
also say. Then run `codefall doctor` and report its **Agents** section in one line.

## The report

The configuration again, in the shape above, as it now stands. Then the probe's line and doctor's
line. Then the landing, per `../../../../.codefall/shared/landing.md`: settings committed by path
on the branch, pushed, and the pull request opened and named by its URL, none of it a question. A
person merges it, because it changes settings.
**End with what the user does next**: merge the pull request, and nothing after it. A harness that
still says `not set up` is named as a fact, with `/codefall-equip agents` as what sets it up once
the merge is done, never as a second step.

## Rules

The rules that hold for every track are in `../SKILL.md`. This one holds for this track:

- **The agents track sets up any harness from any harness.** Parameters come from that harness's
  own records and config, and are written once the user confirms them.
