#!/usr/bin/env bash
#
# Run one configured agent in another harness, headless and read-only.
#
# A verb writes the prompt — the question, the material, the calibration rules,
# and the shape the answer takes. The agent runs inside the repository and reads
# the files itself, so nothing is copied for it. This script resolves which
# harness and model the agent means, starts that harness, and puts its final
# message in a file. It parses nothing and decides nothing: what the agent said
# is the verb's to read, and which agent to try next is the verb's to decide.
# The walk over an order lives in `running-agents.md` beside this file; this script runs
# exactly one entry of it.
#
# Every harness runs in its own read-only mode. An agent run this way proposes;
# a subprocess that edited would bypass the host's PreToolUse hooks and its
# checkpoints, so nothing it did would be guarded or reversible.
#
# Usage: run-agent.sh [--settings <path>] <agent> <prompt-file> <schema-file> <out-file>
#
#   --settings   the project's settings file (default .codefall/settings.json)
#   agent        a name from the settings file's `agents` list, or the raw form
#                <harness>[:<model>] — codex, claude, opencode, gemini, muse, agy
#   prompt-file  the prompt, already written
#   schema-file  the JSON schema the answer follows, when the harness can take one
#   out-file     where the harness's final message is written
#
# A name is looked up first: when the settings file has an `agents` entry by that
# name, its harness and model are used, and a model on the entry is passed to the
# harness untouched. A name the file does not define is read as the raw form. The
# raw form needs no settings file and no jq, which is what `via=codex:gpt-5-codex`
# relies on. An absent or empty `agents` list means the one default entry,
# `subagent` on `current`.
#
# `current` is the harness running the session, and its agent is that harness's
# own subagent. This script cannot start one: it exits 70 and the caller runs the
# subagent itself.
#
# The prompt carries the schema for every harness. Codex and Claude Code also
# take it as a flag, which constrains their output instead of requesting it; the
# other four rely on the prompt alone — OpenCode and Gemini because they have no
# such flag, Muse because its flag rejects this schema (see run_muse), agy because
# its flag has not been tried against this schema (see run_agy).
#
# Environment
#   CODEFALL_REVIEW_TIMEOUT   seconds before the run is killed (default 900)
#
# Exit codes — the caller walking an order reads them as three outcomes
#   0    the agent ran and out-file holds its final message         answered
#   70   the agent is on `current`: run a subagent of this harness    yours to run
#   69   the harness is not on PATH                                  not runnable here → skip
#   64   the arguments are wrong, the agent is unknown, or a name
#        needs jq and jq is missing                                  not runnable here → skip
#   73   out-file could not be written, or the harness wrote nothing  ran and failed → advance
#   75   the harness was still running at the timeout and was killed ran and failed → advance
#   76   the harness exited non-zero; stderr carries its code         ran and failed → advance

set -uo pipefail

readonly USAGE="usage: run-agent.sh [--settings <path>] <agent> <prompt> <schema> <out>"
readonly HARNESSES="codex claude opencode gemini muse agy"

fail() {
  local code=$1
  shift
  printf 'run-agent: %s\n' "$*" >&2
  exit "$code"
}

settings=.codefall/settings.json

while [ "$#" -gt 0 ]; do
  case $1 in
    --settings)
      [ "$#" -ge 2 ] || fail 64 "--settings needs a value ($USAGE)"
      settings=$2
      shift 2
      ;;
    --settings=*)
      settings=${1#--settings=}
      shift
      ;;
    --)
      shift
      break
      ;;
    -*)
      fail 64 "unknown flag \"$1\" ($USAGE)"
      ;;
    *)
      break
      ;;
  esac
done

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
  for candidate in $HARNESSES current; do
    [ "$1" = "$candidate" ] && return 0
  done
  return 1
}

# --- resolve the agent -----------------------------------------------------------------------

# A configured name wins over the raw form, so a project that names an agent `codex` gets that
# entry's model when someone types `via=codex`. The lookup needs jq; without it a name cannot be
# read, and the raw form is the way through.
harness=""
model=""

lookup_name() {
  local name=$1 found

  if [ ! -f "$settings" ]; then
    if [ "$name" = subagent ]; then
      harness=current
      return 0
    fi
    return 1
  fi

  if ! command -v jq >/dev/null 2>&1; then
    # Without jq the default entry is still known, because it is the same in every project.
    if [ "$name" = subagent ] && ! grep -q '"agents"[[:space:]]*:' "$settings" 2>/dev/null; then
      harness=current
      return 0
    fi
    fail 64 "jq is needed to read an agent by name from $settings; pass <harness>[:<model>] instead"
  fi

  found=$(jq -r --arg name "$name" '
    (if ((.agents // []) | length) == 0 then [{name: "subagent", harness: "current"}] else .agents end)
    | map(select(.name == $name)) | first // empty
    | [.harness, (.model // "")] | @tsv' "$settings" 2>/dev/null) || return 1

  [ -n "$found" ] || return 1

  harness=${found%%	*}
  model=${found#*	}
  [ "$model" = "$harness" ] && model=""
  return 0
}

if ! lookup_name "$agent"; then
  case $agent in
    *:*)
      harness=${agent%%:*}
      model=${agent#*:}
      ;;
    *)
      harness=$agent
      ;;
  esac
fi

is_harness "$harness" || fail 64 "unknown agent \"$agent\": not a name in $settings and not one of $HARNESSES"

if [ "$harness" = current ]; then
  fail 70 "agent \"$agent\" is on current: run a subagent of this harness"
fi

command -v "$harness" >/dev/null 2>&1 || fail 69 "$harness is not on PATH"

timeout_seconds=${CODEFALL_REVIEW_TIMEOUT:-900}

# The harness may write out-file itself (codex) or have its stdout redirected
# into it (the rest), so the directory has to exist either way.
out_dir=$(dirname "$out")
mkdir -p "$out_dir" || fail 73 "cannot create $out_dir"
: >"$out" || fail 73 "cannot write $out"

# Building the flag as an array keeps the no-model case from becoming an empty
# argument, which some CLIs read as a model named "".
#
# Every expansion below is written `"${model_flag[@]+"${model_flag[@]}"}"` rather
# than `"${model_flag[@]}"`. Bash before 4.4 — which is what macOS ships as
# /bin/bash — treats an empty array as unset under `set -u` and aborts, and this
# script is run by whichever bash is first on PATH.
model_flag=()
if [ -n "$model" ]; then
  model_flag=(--model "$model")
fi

# --output-schema constrains the final message to the findings shape.
# --output-last-message is codex writing its own output, which the read-only
# sandbox does not govern: the sandbox covers the model's tool calls, not the
# program's plumbing.
run_codex() {
  codex exec \
    "${model_flag[@]+"${model_flag[@]}"}" \
    --sandbox read-only \
    --output-schema "$schema" \
    --output-last-message "$out" \
    - <"$prompt"
}

# Plan mode is Claude Code's read-only mode. Text output puts the final message
# on stdout with no envelope, which is what keeps this script free of a JSON
# dependency, and --json-schema constrains that message to the findings shape.
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
    --permission-mode plan \
    --output-format text \
    --json-schema "$(sed -e '/^  *"\$schema": /d' -e '/^  *"\$id": /d' "$schema")" \
    <"$prompt" >"$out"
}

# OpenCode's `plan` agent is its read-only one. It has no schema flag, so the
# copy of the schema in the prompt is all it gets.
run_opencode() {
  opencode run \
    "${model_flag[@]+"${model_flag[@]}"}" \
    --agent plan \
    --file "$prompt" \
    "Answer as the attached prompt says. Reply with the JSON it asks for and nothing else." >"$out"
}

# Gemini appends -p to whatever arrived on stdin, so the prompt file is the
# input and -p is the instruction that closes it. No schema flag here either.
run_gemini() {
  gemini \
    "${model_flag[@]+"${model_flag[@]}"}" \
    --approval-mode plan \
    --output-format text \
    -p "Answer as the text above says. Reply with the JSON it asks for and nothing else." \
    <"$prompt" >"$out"
}

# Muse's headless mode is `exec`. Its read-only mode is three flags rather than
# one: --disable-write refuses the workspace file tools, --disable-web-tools the
# network, and --approval-mode never keeps a run with nobody at the terminal from
# waiting on a prompt; the OS sandbox stays on by default and covers the shell.
# Muse has an --output-schema flag like codex's, and it is not used: the API
# behind it rejects the `if`/`then` clause that makes `reason` required on a
# dismissed finding, and the run fails before the model reads a file. The copy
# in the prompt is what it gets. Muse has been seen to print its final object
# twice in a row; the skill's parse retry covers that, and this script does not.
run_muse() {
  muse exec \
    "${model_flag[@]+"${model_flag[@]}"}" \
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
# schema, so the copy in the prompt is what it gets until someone has.
run_agy() {
  agy \
    "${model_flag[@]+"${model_flag[@]}"}" \
    --mode plan \
    --output-format text \
    --disable-slash-commands \
    --print="$(cat "$prompt")" \
    >"$out"
}

# A run that hangs is the failure this guards against. There is no portable
# `timeout` on macOS, so the run goes to the background and is polled — and the
# whole process group is killed, because the harnesses spawn children of their
# own that outlive a signal sent to the parent alone.
run_with_timeout() {
  set -m

  "run_$harness" &
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

if [ "$status" -ne 0 ]; then
  fail 76 "$harness exited $status"
fi

if [ ! -s "$out" ]; then
  fail 73 "$harness exited 0 and wrote nothing to $out"
fi

exit 0
