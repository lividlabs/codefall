#!/usr/bin/env bash
#
# Run one agent in another harness, headless and read-only.
#
# A verb writes the prompt — the question, the material, the calibration rules,
# and the shape the answer takes. The agent runs inside the repository and reads
# the files itself, so nothing is copied for it. This script starts the harness
# the agent names, with its model when it names one, and puts its final message
# in a file. It parses nothing and decides nothing: what the agent said is the
# verb's to read, and which agent to try next is the verb's to decide. Which
# list to walk, and the walk itself, live in `running-agents.md` beside this
# file; this script runs exactly one entry of a list.
#
# Every harness runs in its own read-only mode, with its own hooks off where it
# has a switch for them: a headless run that loaded the project's session-start
# hooks has been seen to follow them into a refresh instead of answering. An
# agent run this way proposes; a subprocess that edited would bypass the host's
# PreToolUse hooks and its checkpoints, so nothing it did would be guarded or
# reversible.
#
# Usage: run-agent.sh <agent> <prompt-file> <schema-file> <out-file>
#
#   agent        <harness>[:<model>] — codex, claude, opencode, muse, agy, or
#                current, or a key of `harnessConfig` in the project's
#                settings; one entry of a `review` or `consult` list, or a
#                `via=` value
#   prompt-file  the prompt, already written
#   schema-file  the JSON schema the answer follows, when the harness can take one
#   out-file     where the harness's final message is written
#
# How a harness is called comes from `harnessConfig` in `../settings.json`,
# the project's `.codefall/settings.json`, read with jq when both are present.
# The agent's harness is the lookup key. A key that is a harness name configures
# that harness; any other key is a variant whose `harness` field names the
# binary, so a project can call codex two ways. A block may carry:
#
#   modelFlag   the flag the model is passed with (default --model)
#   provider    the harness's own provider name, turned into that harness's
#               switch below: codex takes -c model_provider=<name>; Claude Code
#               takes CLAUDE_CODE_USE_BEDROCK, _VERTEX, or _FOUNDRY from a name
#               containing bedrock, vertex, or foundry; OpenCode takes
#               <provider>/<model>; muse and agy have no switch and refuse it
#   args        extra arguments, appended as given; one may not contain a newline
#   env         a shell command whose standard output is evaluated in this shell
#               before the harness starts, so its `export` lines take effect —
#               `aws configure export-credentials --format env` is the case it
#               exists for, since a harness reaching Bedrock through the AWS SDK
#               cannot refresh an `aws login` session the CLI can
#
# A harness name with no block, or no settings file, or no jq, runs bare, as it
# always has. A variant key with no block cannot run and exits 64. A model is
# passed to the harness untouched, and no model means the harness's own default.
#
# `current` is the harness running the session, and its agent is that harness's
# own subagent. This script cannot start one: it exits 70 and the caller runs the
# subagent itself.
#
# The prompt carries the schema for every harness. Codex and Claude Code also
# take it as a flag, which constrains their output instead of requesting it; the
# other three rely on the prompt alone — OpenCode because it has no such flag,
# Muse because its flag has not been tried since the schema changed (see
# run_muse), agy because its flag has not been tried at all (see run_agy).
#
# Environment
#   CODEFALL_REVIEW_TIMEOUT   seconds before the run is killed (default 900)
#   CODEFALL_SETTINGS         the settings file to read (default ../settings.json
#                             beside this script)
#
# Exit codes — the caller walking a list reads them as three outcomes
#   0    the agent ran and out-file holds its final message         answered
#   70   the agent is on `current`: run a subagent of this harness    yours to run
#   69   the harness is not on PATH                                  not runnable here → skip
#   64   the arguments are wrong, the harness is unknown, or the
#        block asks for something the harness cannot do              not runnable here → skip
#   73   out-file could not be written, or the harness wrote nothing  ran and failed → advance
#   75   the harness was still running at the timeout and was killed ran and failed → advance
#   76   the harness exited non-zero, or the env command did;
#        stderr carries the code                                     ran and failed → advance

set -uo pipefail

readonly USAGE="usage: run-agent.sh <harness>[:<model>] <prompt> <schema> <out>"
readonly HARNESSES="codex claude opencode muse agy"

fail() {
  local code=$1
  shift
  printf 'run-agent: %s\n' "$*" >&2
  exit "$code"
}

if [ "$#" -ne 4 ]; then
  fail 64 "$USAGE"
fi

agent=$1
prompt=$2
schema=$3
out=$4

[ -r "$prompt" ] || fail 64 "cannot read prompt $prompt"
[ -r "$schema" ] || fail 64 "cannot read schema $schema"

is_harness() {
  local candidate
  for candidate in $HARNESSES; do
    [ "$1" = "$candidate" ] && return 0
  done
  return 1
}

# --- read the agent --------------------------------------------------------------------------

key=${agent%%:*}
model=""
case $agent in
  *:*) model=${agent#*:} ;;
esac

if [ "$key" = current ]; then
  fail 70 "agent \"$agent\" is on current: run a subagent of this harness"
fi

# --- read the harness block ------------------------------------------------------------------

# The block is read into plain variables, and only when there is one to read.
# Every read below tolerates an absent field, so a block naming one field is as
# good as a block naming five.
settings=${CODEFALL_SETTINGS:-$(dirname "$0")/../settings.json}
harness=$key
model_flag_name=--model
provider=""
env_command=""
extra_args=()

have_block=false
if [ -f "$settings" ] && command -v jq >/dev/null 2>&1; then
  if jq -e --arg key "$key" '.harnessConfig[$key] | type == "object"' "$settings" >/dev/null 2>&1; then
    have_block=true
  fi
fi

if $have_block; then
  # The four fields come joined by the unit separator, not a tab: `read` treats
  # a tab as whitespace and collapses the empty fields a partial block leaves.
  block=$(jq -r --arg key "$key" '
    .harnessConfig[$key]
    | [(.harness // ""), (.modelFlag // ""), (.provider // ""), (.env // "")]
    | join("\u001f")' "$settings")
  IFS=$'\x1f' read -r block_harness block_model_flag provider env_command <<<"$block"

  if ! is_harness "$key"; then
    [ -n "$block_harness" ] || fail 64 "harnessConfig.$key names no harness in $settings"
    harness=$block_harness
  fi

  [ -n "$block_model_flag" ] && model_flag_name=$block_model_flag

  while IFS= read -r line; do
    extra_args+=("$line")
  done < <(jq -r --arg key "$key" '.harnessConfig[$key].args // [] | .[]' "$settings")
elif ! is_harness "$key"; then
  if [ ! -f "$settings" ]; then
    fail 64 "unknown harness \"$key\": not one of $HARNESSES current, and there is no $settings to define it"
  elif ! command -v jq >/dev/null 2>&1; then
    fail 64 "unknown harness \"$key\": not one of $HARNESSES current, and jq is needed to read harnessConfig from $settings"
  fi
  fail 64 "unknown harness \"$key\": not one of $HARNESSES current, and not a harnessConfig key in $settings"
fi

is_harness "$harness" || fail 64 "harnessConfig.$key names harness \"$harness\": not one of $HARNESSES"

command -v "$harness" >/dev/null 2>&1 || fail 69 "$harness is not on PATH"

timeout_seconds=${CODEFALL_REVIEW_TIMEOUT:-900}

# The harness may write out-file itself (codex) or have its stdout redirected
# into it (the rest), so the directory has to exist either way.
out_dir=$(dirname "$out")
mkdir -p "$out_dir" || fail 73 "cannot create $out_dir"
: >"$out" || fail 73 "cannot write $out"

# --- the provider switch ---------------------------------------------------------------------

# Each harness has its own way to be pointed at a provider, and this is the one
# place that knows them. A harness with no way refuses the field rather than
# running against the wrong endpoint with a model name only the right one knows.
if [ -n "$provider" ]; then
  case $harness in
    codex)
      extra_args+=(-c "model_provider=$provider")
      ;;
    claude)
      case $provider in
        *bedrock*) export CLAUDE_CODE_USE_BEDROCK=1 ;;
        *vertex*) export CLAUDE_CODE_USE_VERTEX=1 ;;
        *foundry*) export CLAUDE_CODE_USE_FOUNDRY=1 ;;
        *) fail 64 "harnessConfig.$key: claude knows the providers bedrock, vertex, and foundry, not \"$provider\"" ;;
      esac
      ;;
    opencode)
      [ -n "$model" ] || fail 64 "harnessConfig.$key: opencode takes a provider only with a model, as <provider>/<model>"
      model="$provider/$model"
      ;;
    *)
      fail 64 "harnessConfig.$key: $harness has no provider switch; drop the provider field for it"
      ;;
  esac
fi

# Building the flag as an array keeps the no-model case from becoming an empty
# argument, which some CLIs read as a model named "".
#
# Every expansion below is written `"${model_flag[@]+"${model_flag[@]}"}"` rather
# than `"${model_flag[@]}"`. Bash before 4.4 — which is what macOS ships as
# /bin/bash — treats an empty array as unset under `set -u` and aborts, and this
# script is run by whichever bash is first on PATH. `extra_args` is expanded the
# same way, for the same reason.
model_flag=()
if [ -n "$model" ]; then
  model_flag=("$model_flag_name" "$model")
fi

# --output-schema constrains the final message to the findings shape.
# --output-last-message is codex writing its own output, which the read-only
# sandbox does not govern: the sandbox covers the model's tool calls, not the
# program's plumbing. features.hooks=false keeps the project's .codex/hooks.json
# out of the run.
run_codex() {
  codex exec \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    -c features.hooks=false \
    --sandbox read-only \
    --output-schema "$schema" \
    --output-last-message "$out" \
    - <"$prompt"
}

# Plan mode is Claude Code's read-only mode. Text output puts the final message
# on stdout with no envelope, which is what keeps this script free of a JSON
# dependency, and --json-schema constrains that message to the findings shape.
# --bare skips the hooks defined in settings and by plugins.
#
# Two differences from codex's --output-schema. The flag takes the schema itself
# rather than a path to it, so the file is read in here; and its validator does
# not resolve the 2020-12 meta-schema, rejecting the document outright while
# `$schema` and `$id` are present — so those two lines are dropped first. They
# identify the published artifact and say nothing about the shape, so a schema
# without them constrains exactly the same output.
run_claude() {
  claude --print \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --bare \
    --permission-mode plan \
    --output-format text \
    --json-schema "$(sed -e '/^  *"\$schema": /d' -e '/^  *"\$id": /d' "$schema")" \
    <"$prompt" >"$out"
}

# OpenCode's `plan` agent is its read-only one. It has no schema flag, so the
# copy of the schema in the prompt is all it gets. --pure runs without external
# plugins, which is where OpenCode's hooks live.
run_opencode() {
  opencode run \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --pure \
    --agent plan \
    --file "$prompt" \
    "Answer as the attached prompt says. Reply with the JSON it asks for and nothing else." >"$out"
}

# Muse's headless mode is `exec`. Its read-only mode is three flags rather than
# one: --disable-write refuses the workspace file tools, --disable-web-tools the
# network, and --approval-mode never keeps a run with nobody at the terminal from
# waiting on a prompt; the OS sandbox stays on by default and covers the shell.
# Muse has an --output-schema flag like codex's, and it is not used yet: the API
# behind it rejected an `if`/`then` clause the findings schema used to carry,
# and the run failed before the model read a file. The clause is gone (it made
# `reason` required on a dismissed finding, which the review skill now checks
# itself, since Bedrock's structured output rejected it too), and the flag has
# not been tried since. The copy in the prompt is what Muse gets. Muse has been
# seen to print its final object twice in a row; the skill's parse retry covers
# that, and this script does not. Muse has no switch for its hooks.
run_muse() {
  muse exec \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --prompt-file "$prompt" \
    --disable-write \
    --disable-web-tools \
    --approval-mode never \
    --user-input-auto-resolve \
    >"$out"
}

# agy's headless mode is --print, which takes the prompt as its own value and
# not from stdin or a file, so the prompt file is read into the argument here.
# --mode plan is its read-only mode, and --disable-slash-commands keeps a line in
# the prompt that begins with a slash from being read as a command. agy has a
# --json-schema flag like Claude Code's; it has not been tried against this
# schema, so the copy in the prompt is what it gets until someone has. agy has
# no switch for its hooks.
run_agy() {
  agy \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --mode plan \
    --output-format text \
    --disable-slash-commands \
    --print="$(cat "$prompt")" \
    >"$out"
}

# The env command runs in the same background job as the harness, so the
# timeout covers a credential helper that hangs as well as a harness that does.
# Its output is evaluated here, in the job's shell, which the harness inherits.
# Exit 78 is this job's own signal that the env command failed, distinct from
# anything a harness exits with.
run_agent() {
  if [ -n "$env_command" ]; then
    local exported
    if ! exported=$(eval "$env_command"); then
      printf 'run-agent: env command failed: %s\n' "$env_command" >&2
      exit 78
    fi
    eval "$exported"
  fi

  "run_$harness"
}

# A run that hangs is the failure this guards against. There is no portable
# `timeout` on macOS, so the run goes to the background and is polled — and the
# whole process group is killed, because the harnesses spawn children of their
# own that outlive a signal sent to the parent alone.
run_with_timeout() {
  set -m

  run_agent &
  local pid=$!
  local waited=0

  while kill -0 "$pid" 2>/dev/null; do
    if [ "$waited" -ge "$timeout_seconds" ]; then
      kill -TERM -"$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null
      sleep 2
      kill -KILL -"$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null
      wait "$pid" 2>/dev/null
      return 75
    fi

    sleep 1
    waited=$((waited + 1))
  done

  wait "$pid"
}

run_with_timeout
status=$?

if [ "$status" -eq 75 ]; then
  fail 75 "$harness did not finish within ${timeout_seconds}s"
fi

if [ "$status" -eq 78 ]; then
  fail 76 "env command for $key exited non-zero; nothing was run"
fi

if [ "$status" -ne 0 ]; then
  fail 76 "$harness exited $status"
fi

if [ ! -s "$out" ]; then
  fail 73 "$harness exited 0 and wrote nothing to $out"
fi

exit 0
