#!/usr/bin/env bash
#
# Run one worker in another harness, headless, with write access to one worktree.
#
# `codefall-implement` writes the prompt — the bead, the criteria, the branch, the
# checks — and creates the worktree the worker builds in. This script starts the
# harness the agent names, with its model when it names one, inside that
# worktree, and puts the worker's final message in one file and the run's
# transcript in another. It parses nothing and decides nothing: whether the
# branch is any good is the root's to judge, and the root reviews the diff,
# re-runs the checks, pushes, and opens the pull request itself. Which harness
# to use for a bead is the root's to choose from the `work` list, per
# `running-agents.md` beside this file; this script runs exactly one entry.
#
# This is the one runner that may write, and the reason `run-agent.sh` gives for
# never writing still holds: a subprocess that edits bypasses the host's
# PreToolUse hooks and its checkpoints. What makes it acceptable here is where
# the writes land and what guards them. They land in a worktree on a branch of
# its own, which the root reads in full before anything leaves the machine, so
# a bad run costs a branch and nothing else. And four guards stand where the
# host's hooks would have:
#
#   - each harness runs in its own build mode with scoped permissions, never a
#     blanket approval: edits allowed, the shell allowed except `git push`,
#     `gh`, and `bd`, and the network tools and other directories denied, in
#     whatever form that harness takes them (see run_opencode, run_claude,
#     run_codex);
#   - a directory of shims goes first on the worker's PATH: `git` refuses
#     `push` and `send-pack` and passes everything else to the real git, `gh`
#     refuses everything, and `bd` refuses everything except `bd hooks run`,
#     because the beads git hooks call `bd hooks run pre-commit` and
#     `bd hooks run prepare-commit-msg` on every commit, and refusing them
#     stopped a worker from committing at all. A shim guards a shell that keeps
#     the PATH it was given; a login shell that rebuilds PATH can put the real
#     binaries first, which is why the harness's own rules deny the same
#     commands;
#   - the harness's hooks are off, as in `run-agent.sh`: the project's
#     session-start hooks run `bd prime` and the refresh notice, and a headless
#     run has been seen to follow them instead of the prompt;
#   - the root's duties after the run, in `codefall-implement`, which no
#     script can do for it.
#
# Muse and agy have read-only modes and no write mode scoped this way yet, so
# this script refuses them.
#
# `run-agent.sh` and this script share the agent syntax, the `harnessConfig`
# read, the provider switch, the env command, and the polled timeout, copied
# rather than shared so that `run-agent.sh` stayed as it was. A change to any of
# those in one script is made in the other in the same change.
#
# Usage: run-worker.sh <agent> <worktree> <prompt-file> <out-file>
#
#   agent        <harness>[:<model>] — codex, claude, or opencode, or current,
#                or a key of `harnessConfig` in the project's settings; one
#                entry of a `work` list
#   worktree     the git worktree the worker builds in, already on its branch
#                with its dependencies fetched; the harness runs with it as its
#                working directory
#   prompt-file  the prompt, already written
#   out-file     where the worker's final message is written; the transcript
#                goes beside it, in <out-file>.transcript. Keep both outside the
#                worktree, or the worker may commit them
#
# How a harness is called comes from `harnessConfig` in `../settings.json`,
# the project's `.codefall/settings.json`, read with jq when both are present,
# exactly as `run-agent.sh` reads it: the agent's harness is the lookup key, a
# key that is a harness name configures that harness, any other key is a
# variant whose `harness` field names the binary, and a block may carry
# `modelFlag`, `provider`, `args`, and `env`. Its header says what each does.
#
# `current` is the harness running the session, and its worker is that
# harness's own subagent. This script cannot start one: it exits 70 and the
# caller runs the subagent itself.
#
# After an OpenCode run, when sqlite3 is present, the script reads the session
# OpenCode recorded for the worktree and prints the model and variant it ran on
# to stderr and the transcript, since a model or variant OpenCode did not take
# is otherwise visible only in its own database. It never changes the exit code.
#
# Environment
#   CODEFALL_WORK_TIMEOUT   seconds before the run is killed (default 3600)
#   CODEFALL_SETTINGS       the settings file to read (default ../settings.json
#                           beside this script)
#
# Exit codes — the same as run-agent.sh's, read as the same three outcomes
#   0    the worker ran and out-file holds its final message           answered
#   70   the agent is on `current`: run a subagent of this harness      yours to run
#   69   the harness is not on PATH                                    not runnable here → skip
#   64   the arguments are wrong, the harness is unknown or has no
#        write mode, or the block asks for something the harness
#        cannot do                                                     not runnable here → skip
#   73   out-file or the shims could not be written, or the harness
#        wrote nothing                                                 ran and failed → advance
#   75   the harness was still running at the timeout and was killed   ran and failed → advance
#   76   the harness exited non-zero, or the env command did;
#        stderr carries the code                                       ran and failed → advance

set -uo pipefail

readonly USAGE="usage: run-worker.sh <harness>[:<model>] <worktree> <prompt> <out>"
readonly HARNESSES="codex claude opencode muse agy"

fail() {
  local code=$1
  shift
  printf 'run-worker: %s\n' "$*" >&2
  exit "$code"
}

if [ "$#" -ne 4 ]; then
  fail 64 "$USAGE"
fi

agent=$1
worktree=$2
prompt=$3
out=$4

[ -r "$prompt" ] || fail 64 "cannot read prompt $prompt"
[ -d "$worktree" ] || fail 64 "worktree $worktree is not a directory"

is_harness() {
  local candidate
  for candidate in $HARNESSES; do
    [ "$1" = "$candidate" ] && return 0
  done
  return 1
}

# Every path is made absolute before the harness starts in the worktree, since a
# relative one would then be read from the wrong directory.
absolute_dir() {
  (cd "$1" 2>/dev/null && pwd -P)
}

absolute_file() {
  local dir
  dir=$(absolute_dir "$(dirname "$1")") || return 1
  printf '%s/%s\n' "$dir" "$(basename "$1")"
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

# Copied from run-agent.sh; the two change together.
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

case $harness in
  muse | agy) fail 64 "$harness has no write mode yet: it can review and consult through run-agent.sh, and cannot build" ;;
esac

command -v "$harness" >/dev/null 2>&1 || fail 69 "$harness is not on PATH"

# Claude Code's transcript is its JSON stream, and the final message is read out
# of it, so a Claude worker needs jq where the other two do not.
if [ "$harness" = claude ] && ! command -v jq >/dev/null 2>&1; then
  fail 64 "claude: jq is needed to read the final message out of its stream"
fi

real_git=$(type -P git) || fail 64 "git is not on PATH"
"$real_git" -C "$worktree" rev-parse --is-inside-work-tree >/dev/null 2>&1 ||
  fail 64 "$worktree is not a git worktree"

timeout_seconds=${CODEFALL_WORK_TIMEOUT:-3600}

worktree=$(absolute_dir "$worktree") || fail 64 "cannot enter $worktree"
prompt=$(absolute_file "$prompt") || fail 64 "cannot read prompt $prompt"

out_dir=$(dirname "$out")
mkdir -p "$out_dir" || fail 73 "cannot create $out_dir"
out=$(absolute_file "$out") || fail 73 "cannot write $out"
transcript=$out.transcript
: >"$out" || fail 73 "cannot write $out"
: >"$transcript" || fail 73 "cannot write $transcript"

# --- the provider switch ---------------------------------------------------------------------

# Copied from run-agent.sh; the two change together. Muse and agy were refused
# above, so their case is gone.
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
  esac
fi

# Every expansion of an array below is written `"${a[@]+"${a[@]}"}"`, because
# bash before 4.4, which macOS ships as /bin/bash, treats an empty array as unset
# under `set -u`. run-agent.sh says the same at greater length.
model_flag=()
if [ -n "$model" ]; then
  model_flag=("$model_flag_name" "$model")
fi

# --- the shims -------------------------------------------------------------------------------

# Each shim names the real binary by the absolute path found here, before the
# shim directory is on PATH, so a shim never finds itself. A bd shim is written
# only when bd is installed: the beads hooks skip themselves when `command -v bd`
# finds nothing, and a shim with nothing behind it would turn that skip into a
# failed commit.
shim_dir=""
job=""

cleanup() {
  [ -n "$shim_dir" ] && rm -rf "$shim_dir"
}

# A signal to this script stops the run as well, since the run is in a process
# group of its own and would otherwise outlive it.
interrupted() {
  if [ -n "$job" ]; then
    kill -TERM -"$job" 2>/dev/null || kill -TERM "$job" 2>/dev/null
  fi
  exit "$1"
}

trap cleanup EXIT
trap 'interrupted 130' INT
trap 'interrupted 143' TERM

shim_dir=$(mktemp -d "${TMPDIR:-/tmp}/codefall-worker.XXXXXX") || fail 73 "cannot create the shim directory"

# Single-quoted for sh, with any single quote in the path closed, escaped, and
# reopened.
sh_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

write_shim() {
  local name=$1 body=$2
  printf '#!/bin/sh\n# codefall run-worker: %s for a worker, removed when the run ends.\n%s\n' "$name" "$body" \
    >"$shim_dir/$name" || return 1
  chmod +x "$shim_dir/$name"
}

# git takes options before its subcommand, some with a value of their own, so
# the subcommand is the first word that is neither an option nor an option's
# value. send-pack is refused beside push because it is the plumbing push runs.
write_shim git "real=$(sh_quote "$real_git")
sub=
skip=
for arg in \"\$@\"; do
  if [ -n \"\$skip\" ]; then skip=; continue; fi
  case \$arg in
    -C | -c | --git-dir | --work-tree | --namespace | --super-prefix | --config-env) skip=1 ;;
    -*) ;;
    *) sub=\$arg; break ;;
  esac
done
case \$sub in
  push | send-pack)
    echo \"git \$sub is refused for this worker: the root pushes once it has reviewed the branch.\" >&2
    exit 1
    ;;
esac
exec \"\$real\" \"\$@\"" || fail 73 "cannot write the git shim"

write_shim gh "echo \"gh is refused for this worker: the root opens the pull request.\" >&2
exit 1" || fail 73 "cannot write the gh shim"

if real_bd=$(type -P bd); then
  write_shim bd "real=$(sh_quote "$real_bd")
if [ \"\${1:-}\" = hooks ] && [ \"\${2:-}\" = run ]; then
  exec \"\$real\" \"\$@\"
fi
echo \"bd is refused for this worker: every bd command is the root's. Only the git hooks' bd hooks run passes.\" >&2
exit 1" || fail 73 "cannot write the bd shim"
fi

export PATH="$shim_dir:$PATH"

# --- the harnesses ---------------------------------------------------------------------------

# OpenCode's `build` agent is its editing one, and the permission block narrows
# it: edits allowed; the shell allowed except git push, gh, and bd; web fetch,
# web search, and directories outside the worktree denied. The block is set at
# the top level and on the build agent, because an agent's own permissions win
# over the top level's and a person's config may give build some. No --auto,
# which approves every permission a rule does not deny: one run that used it
# had the host session's next command refused as creating an unsafe agent.
#
# --standalone runs a private server, which inherits this process's PATH,
# working directory, and OPENCODE_CONFIG_CONTENT; the default background service
# was started elsewhere and would run the tools with none of them. --pure runs
# without external plugins, which is where OpenCode's hooks live, as in
# run-agent.sh.
#
# The message comes before --file because --file takes every value up to the
# next flag (#215). Every text part goes to stdout, so out-file holds the
# worker's narration with its final message last, and the transcript holds the
# same with stderr beside it.
readonly OPENCODE_PERMISSION='{
  "edit": "allow",
  "bash": {
    "*": "allow",
    "git push*": "deny",
    "gh": "deny",
    "gh *": "deny",
    "bd": "deny",
    "bd *": "deny"
  },
  "webfetch": "deny",
  "websearch": "deny",
  "external_directory": "deny"
}'

# shellcheck disable=SC2094 # both writers append to the transcript, and neither reads it
run_opencode() {
  OPENCODE_CONFIG_CONTENT="{\"permission\": $OPENCODE_PERMISSION, \"agent\": {\"build\": {\"permission\": $OPENCODE_PERMISSION}}}" \
    opencode run \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --standalone \
    --pure \
    --agent build \
    "You are the worker the attached prompt describes. Work only in the current directory, follow the prompt exactly, and end with the single JSON object it asks for." \
    --file "$prompt" \
    2>>"$transcript" | tee -a "$transcript" >"$out"
}

# Claude Code's acceptEdits mode lets the file tools edit without asking. The
# shell is allowed by rule, and the deny rules, which win over allow rules, take
# git push, gh, bd, and the web tools back out. --permission-prompts none denies
# anything else that would have asked, since nobody is there to answer.
# --bare skips the hooks defined in settings and by plugins, as in run-agent.sh.
#
# The stream is the transcript, and the final message is the `result` event's
# text, read out of it once the run ends.
run_claude() {
  claude --print \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    --bare \
    --permission-mode acceptEdits \
    --permission-prompts none \
    --allowedTools "Bash" "Read" "Edit" "Write" "Glob" "Grep" \
    --disallowedTools "Bash(git push)" "Bash(git push *)" "Bash(gh)" "Bash(gh *)" \
    "Bash(bd)" "Bash(bd *)" "WebFetch" "WebSearch" \
    --output-format stream-json \
    --verbose \
    <"$prompt" >>"$transcript" 2>&1
}

# Codex's workspace-write sandbox lets the model write in the working directory
# and nowhere else, with the network off, so neither a push nor gh can reach a
# remote even past the shims. A worktree's commits land in the repository's
# common git directory, which is outside the worktree, so that directory is
# added as writable; without it the worker could edit and never commit.
# approval_policy=never keeps a run with nobody at the terminal from waiting on
# a prompt. features.hooks=false keeps the project's .codex/hooks.json out, as in
# run-agent.sh. --output-last-message is codex writing its own output, which the
# sandbox does not govern.
run_codex() {
  local common
  common=$("$real_git" -C "$worktree" rev-parse --path-format=absolute --git-common-dir 2>/dev/null) ||
    common=$(cd "$worktree" && absolute_dir "$("$real_git" rev-parse --git-common-dir)")

  codex exec \
    "${model_flag[@]+"${model_flag[@]}"}" \
    "${extra_args[@]+"${extra_args[@]}"}" \
    -c features.hooks=false \
    -c approval_policy=never \
    -c sandbox_workspace_write.network_access=false \
    --sandbox workspace-write \
    --add-dir "$common" \
    --output-last-message "$out" \
    - <"$prompt" >>"$transcript" 2>&1
}

# Copied from run-agent.sh; the two change together. The job enters the
# worktree first, so every harness starts there.
run_agent() {
  cd "$worktree" || exit 78

  if [ -n "$env_command" ]; then
    local exported
    if ! exported=$(eval "$env_command"); then
      printf 'run-worker: env command failed: %s\n' "$env_command" >&2
      exit 78
    fi
    eval "$exported"
  fi

  "run_$harness"
}

# Copied from run-agent.sh; the two change together. There is no portable
# `timeout` on macOS, so the run goes to the background and is polled, and the
# whole process group is killed, because the harnesses spawn children of their
# own that outlive a signal sent to the parent alone.
run_with_timeout() {
  set -m

  run_agent &
  job=$!
  local waited=0

  while kill -0 "$job" 2>/dev/null; do
    if [ "$waited" -ge "$timeout_seconds" ]; then
      kill -TERM -"$job" 2>/dev/null || kill -TERM "$job" 2>/dev/null
      sleep 2
      kill -KILL -"$job" 2>/dev/null || kill -KILL "$job" 2>/dev/null
      wait "$job" 2>/dev/null
      job=""
      return 75
    fi

    sleep 1
    waited=$((waited + 1))
  done

  wait "$job"
  local status=$?
  job=""
  return "$status"
}

# The model and variant an OpenCode run used, from the session it recorded for
# the worktree since the run started. Informational: nothing here changes the
# exit code.
report_opencode_model() {
  local db=$HOME/.local/share/opencode/opencode.db
  command -v sqlite3 >/dev/null 2>&1 && [ -r "$db" ] || return 0

  local quoted row
  quoted=$(printf '%s' "$worktree" | sed "s/'/''/g")
  row=$(sqlite3 -readonly "$db" \
    "select id || ' ' || coalesce(agent, '') || ' ' || coalesce(model, '') from session_v2
     where directory = '$quoted' and parent_id is null and time_created >= $started_ms
     order by time_created desc limit 1" 2>/dev/null) || return 0
  [ -n "$row" ] || return 0

  printf 'run-worker: opencode session %s\n' "$row" | tee -a "$transcript" >&2
}

started_ms=$(($(date +%s) * 1000))

run_with_timeout
status=$?

if [ "$harness" = opencode ]; then
  report_opencode_model
fi

if [ "$status" -eq 75 ]; then
  fail 75 "$harness did not finish within ${timeout_seconds}s"
fi

if [ "$status" -eq 78 ]; then
  fail 76 "env command for $key exited non-zero, or the worktree could not be entered; nothing was run"
fi

if [ "$status" -ne 0 ]; then
  fail 76 "$harness exited $status; the transcript is $transcript"
fi

if [ "$harness" = claude ]; then
  jq -Rr 'fromjson? | select(.type == "result") | .result // empty' "$transcript" >"$out" 2>/dev/null
fi

if [ ! -s "$out" ]; then
  fail 73 "$harness exited 0 and wrote nothing to $out"
fi

exit 0
