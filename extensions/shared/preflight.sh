#!/usr/bin/env bash
#
# Preconditions and standing for the codefall skills.
#
# Two things, in order. First, where the checkout stands: against the default branch, and whether
# the local environment has been refreshed since HEAD last moved (ADR-005). That part is a report —
# nothing in it changes the exit code, and a skill reads the lines, says what it found, and offers
# `/codefall-refresh`. It never pulls and never runs the local commands. Second, the Beads
# precondition: setup is done or it is not. The script reports which, and never repairs anything —
# a skill reads the key=value lines to tell the user what is missing and which command fixes it.
#
# Usage: preflight.sh [--no-fetch] [project-dir]      (default: .)
#
# --no-fetch skips `git fetch` and emits `fetch=skipped`; `behind` and `ahead` are then answered
# from the refs the checkout already has. Nothing else changes. It is for a caller that cannot
# wait on the network — the session-start notice hook is the one that asks for it — and every
# skill runs the default, which fetches.
#
# Checkout lines (always emitted inside a repository)
#   checkout         ok | not_a_repository | no_remote | no_default_branch
#   branch           the current branch, or `detached`
#   dirty            true when the working tree has changes, staged or not
#   fetch            ok | failed | skipped — `git fetch origin`, without prompting for credentials
#   default_branch   the branch origin/HEAD names, else main or master when one exists
#   behind / ahead   commits on origin/<default_branch> not in HEAD, and the reverse
#   local            declared | undeclared | unknown — whether settings carry a `local` block
#   test             undeclared | unequipped | equipped | unknown — whether settings carry a `test`
#                    block, and whether it names a runner (ADR-007)
#   persona          engineer | product-manager — who `.codefall/user.json` says the person at the
#                    keyboard works as; `engineer` when the file is missing, unreadable, or names
#                    a persona codefall does not know
#   refresh_stamp    the commit `refresh` last succeeded at, or `none`
#   refresh          current | stale | undeclared — the stamp against HEAD, when declared
#
# Exit codes
#   0   beads is installed and this repository has a database
#   10  the `bd` binary is not on PATH
#   11  `bd` is installed but this repository has no beads database
#   12  something else went wrong; `beads_detail` carries bd's own words

set -uo pipefail

emit() { printf '%s\n' "$*"; }

fetching=yes
if [ "${1:-}" = "--no-fetch" ]; then
  fetching=
  shift
fi

project_dir=${1:-.}

if ! cd "$project_dir" 2>/dev/null; then
  emit "beads=blocked"
  emit "beads_reason=project_dir_unreadable"
  emit "project_dir=$project_dir"
  exit 12
fi

emit "project_dir=$PWD"

# --- checkout -----------------------------------------------------------------------------------

# The stamp is machine-local and git-ignored; `refresh` writes it and nothing else does.
settings_file=.codefall/settings.json
stamp_file=.codefall/refresh.stamp
# The user file is one person's and git-ignored, like the stamp; the person writes it, by hand or
# through `codefall config persona`.
user_file=.codefall/user.json

# The default branch is what origin/HEAD names. A clone made before the remote moved its default,
# or a remote that was added by hand, may not have origin/HEAD at all; then main or master, when one
# is on the remote, is the best available answer.
default_branch() {
  local named
  named=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null)
  if [ -n "$named" ]; then
    printf '%s\n' "${named#origin/}"
    return
  fi

  local candidate
  for candidate in main master; do
    if git show-ref --verify --quiet "refs/remotes/origin/$candidate"; then
      printf '%s\n' "$candidate"
      return
    fi
  done
}

# Whether settings declare both local commands. jq reads the shape when it is there; without it, a
# `local` key anywhere in the file is taken as a declaration, which doctor checks properly.
local_declared() {
  if [ ! -f "$settings_file" ]; then
    printf 'undeclared\n'
    return
  fi

  if command -v jq >/dev/null 2>&1; then
    if jq -e '(.local.start | type) == "string" and (.local.update | type) == "string"' \
      "$settings_file" >/dev/null 2>&1; then
      printf 'declared\n'
    else
      printf 'undeclared\n'
    fi
    return
  fi

  if grep -q '"local"[[:space:]]*:' "$settings_file" 2>/dev/null; then
    printf 'declared\n'
  else
    printf 'unknown\n'
  fi
}

# Where the project stands on testing: no `test` block is `undeclared`, a block naming no runner is
# `unequipped`, and a block naming one is `equipped` (ADR-007). A block `Validate` would reject —
# `dir` missing or not a string — is `undeclared`, because nothing can read a root that is not
# there. Without jq a `test` key anywhere counts as declared, but its runners cannot be read, so the
# state is `unknown` — the word `local` falls back to when it cannot see a shape — and a file with no
# `test` key at all is `undeclared`.
test_state() {
  if [ ! -f "$settings_file" ]; then
    printf 'undeclared\n'
    return
  fi

  if command -v jq >/dev/null 2>&1; then
    if ! jq -e '(.test.dir | type) == "string"' "$settings_file" >/dev/null 2>&1; then
      printf 'undeclared\n'
    elif jq -e '((.test.runners // []) | length) > 0' "$settings_file" >/dev/null 2>&1; then
      printf 'equipped\n'
    else
      printf 'unequipped\n'
    fi
    return
  fi

  if grep -q '"test"[[:space:]]*:' "$settings_file" 2>/dev/null; then
    printf 'unknown\n'
  else
    printf 'undeclared\n'
  fi
}

# Who the person at the keyboard works as. Anything this cannot read as a known persona — no file, a
# file it cannot parse, a value codefall does not know — is `engineer`, the default; doctor is what
# reports a file that is wrong. Without jq, the first `"persona": "<value>"` in the file is taken as
# written, and anything else is the default.
persona() {
  local value=""

  if [ -f "$user_file" ]; then
    if command -v jq >/dev/null 2>&1; then
      value=$(jq -r 'if (.persona | type) == "string" then .persona else "" end' \
        "$user_file" 2>/dev/null)
    else
      value=$(grep -o '"persona"[[:space:]]*:[[:space:]]*"[^"]*"' "$user_file" 2>/dev/null |
        head -n 1 | sed 's/.*"\([^"]*\)"$/\1/')
    fi
  fi

  case $value in
    engineer|product-manager) printf '%s\n' "$value" ;;
    *) printf 'engineer\n' ;;
  esac
}

checkout() {
  if [ "$(git rev-parse --is-inside-work-tree 2>/dev/null)" != "true" ]; then
    emit "checkout=not_a_repository"
    return
  fi

  emit "branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || printf 'detached')"

  if [ -n "$(git status --porcelain 2>/dev/null)" ]; then
    emit "dirty=true"
  else
    emit "dirty=false"
  fi

  if git remote get-url origin >/dev/null 2>&1; then
    # Never wait on a credential prompt: a skill runs this with nobody at the keyboard for it.
    if [ -z "$fetching" ]; then
      emit "fetch=skipped"
    elif GIT_TERMINAL_PROMPT=0 GIT_SSH_COMMAND="ssh -o BatchMode=yes" \
      git fetch --quiet origin >/dev/null 2>&1; then
      emit "fetch=ok"
    else
      emit "fetch=failed"
    fi

    local default
    default=$(default_branch)

    if [ -z "$default" ]; then
      emit "checkout=no_default_branch"
    else
      emit "default_branch=$default"
      emit "behind=$(git rev-list --count "HEAD..origin/$default" 2>/dev/null || printf 'unknown')"
      emit "ahead=$(git rev-list --count "origin/$default..HEAD" 2>/dev/null || printf 'unknown')"
      emit "checkout=ok"
    fi
  else
    emit "checkout=no_remote"
  fi

  local declared
  declared=$(local_declared)
  emit "local=$declared"
  emit "test=$(test_state)"
  emit "persona=$(persona)"

  local stamped=""
  if [ -f "$stamp_file" ]; then
    stamped=$(head -n 1 "$stamp_file" | tr -d '[:space:]')
    emit "refresh_stamp=${stamped:-none}"
  else
    emit "refresh_stamp=none"
  fi

  if [ "$declared" != declared ]; then
    emit "refresh=undeclared"
    return
  fi

  local head_commit
  head_commit=$(git rev-parse HEAD 2>/dev/null)

  if [ -n "$head_commit" ] && [ "$stamped" = "$head_commit" ]; then
    emit "refresh=current"
  else
    emit "refresh=stale"
  fi
}

checkout

# --- beads --------------------------------------------------------------------------------------

if ! command -v bd >/dev/null 2>&1; then
  emit "beads_binary=missing"
  emit "beads=blocked"
  emit "beads_reason=not_installed"
  emit "beads_remedy=brew install beads"
  exit 10
fi

emit "beads_binary=$(command -v bd)"
emit "beads_version=$(bd version 2>/dev/null | head -1)"

# `bd info` exercises the binary, workspace discovery (or BEADS_DIR), and the
# database in one call. Never test for a literal .beads/ directory instead:
# BEADS_DIR relocates it, so that test passes on setups that do not work and
# fails on setups that do.
info_output=$(bd info 2>&1)
info_status=$?

if [ "$info_status" -eq 0 ]; then
  emit "beads=ok"
  exit 0
fi

# Only on the failure path, and only because it separates "never initialized"
# from "initialized but not found from here" — a subdirectory, a worktree, or a
# wrong BEADS_DIR.
emit "beads_workspace=$(bd where 2>&1 | tr '\n' ' ')"

lowered=$(printf '%s' "$info_output" | tr '[:upper:]' '[:lower:]')
case $lowered in
  *"no beads database"*|*"not initialized"*|*"no .beads"*)
    emit "beads=blocked"
    emit "beads_reason=not_initialized"
    emit "beads_remedy=bd init"
    exit 11
    ;;
esac

emit "beads_detail=$(printf '%s' "$info_output" | tr '\n' ' ')"
emit "beads=blocked"
emit "beads_reason=unreadable"
exit 12
