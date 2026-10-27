#!/usr/bin/env bash
# SessionStart notice: one short notice naming what needs attention, and nothing at all when
# the project is current.
#
# It reads the shared preflight and reports. It never pulls, never runs the project's `update`,
# and never writes a file: a hook cannot ask, and a pull moves the working tree under whatever
# the session is doing (ADR-005). It always exits 0 — a session that failed to start over a
# notice would cost more than the notice is worth — and it says nothing when every line it reads
# is current, so a session that has nothing to fix opens silently.
#
# Output: plain text by default, which is what the OpenCode plugin passes to the session.
# --hook-json selects the JSON contract instead, the same shape `bd prime --hook-json` emits and
# the shape Claude Code's and Codex's SessionStart entries register this beside. Selecting a
# contract by flag is how the merge guard's --antigravity works. Antigravity has no session
# event, so nothing registers this there.
#
# The fetch is bounded rather than waited on. preflight's lines all arrive at once except the one
# the fetch precedes, so waiting a few seconds for the next line is waiting on the network and
# nothing else — `read -t` is the bound, because `timeout` and `gtimeout` are not on every machine
# and neither is a dependency worth taking for this. A run that does not finish inside the bound is
# run again with --no-fetch, which answers from the refs the checkout already has; the fetch is
# treated as failed, the same as a fetch that could not reach the remote.
#
# A headless run says nothing. The runners that start another harness for a review or a build
# (run-agent.sh and run-worker.sh) set CODEFALL_HEADLESS=1 for every harness, and a run that was
# given a prompt to answer has been seen to follow a notice into a refresh instead. The variable
# means the same in every harness, so the script stands down for it whichever harness runs it.
# The merge guard has no such switch: it stays on in every run.
#
# Usage: codefall-session-notice.sh [--hook-json] [project-dir]      (default: .)
set -u

[ -n "${CODEFALL_HEADLESS:-}" ] && exit 0

# How long to wait for one line of the preflight report before giving up on the network and
# reading what the checkout already knows.
bound=5

as_json=
if [ "${1:-}" = "--hook-json" ]; then
  as_json=yes
  shift
fi

project_dir=${1:-.}

# Both scripts are installed under .codefall/ and both live under extensions/ in the repository
# codefall is developed in, so the path from this file's own directory is the same either way.
here=$(cd "$(dirname "${BASH_SOURCE[0]}")" 2>/dev/null && pwd) || exit 0
preflight="$here/../../shared/preflight.sh"

# A missing preflight means the extension is not installed here, or the project moved. The notice
# stands down rather than saying something about a project it cannot read.
[ -f "$preflight" ] || exit 0

report=

exec 3< <(bash "$preflight" "$project_dir" 2>/dev/null)
while IFS= read -r -t "$bound" -u 3 line; do
  report="$report$line
"
done
exec 3<&-

# preflight emits a beads line last, whatever it found, so a report without one is a run the bound
# cut off partway. The preflight still running is a read-only fetch that ends on its own.
if ! printf '%s' "$report" | grep -q '^beads'; then
  report=$(bash "$preflight" --no-fetch "$project_dir" 2>/dev/null)
fi

behind=
default_branch=
refresh=
test_state=
beads=
beads_remedy=

while IFS='=' read -r key value; do
  case $key in
    behind) behind=$value ;;
    default_branch) default_branch=$value ;;
    refresh) refresh=$value ;;
    test) test_state=$value ;;
    beads) beads=$value ;;
    beads_remedy) beads_remedy=$value ;;
  esac
done <<REPORT
$report
REPORT

lines=

say() {
  lines="${lines}${lines:+
}- $1"
}

# The checkout is behind only when the count is a number: `unknown` is git declining to answer,
# which is not something to send a session off to fix.
if printf '%s' "$behind" | grep -qE '^[0-9]+$' && [ "$behind" -gt 0 ]; then
  commits=commits
  [ "$behind" -eq 1 ] && commits=commit
  say "$behind $commits behind ${default_branch:-the default branch}: run /codefall-refresh"
fi

case $refresh in
  stale) say "the local environment has not been refreshed since HEAD moved: run /codefall-refresh" ;;
  undeclared) say "no local environment scripts are declared: run /codefall-equip local" ;;
esac

case $test_state in
  undeclared) say "no testing root is declared: run codefall init" ;;
  unequipped) say "no test runner is declared: run /codefall-equip test" ;;
esac

if [ "$beads" = blocked ]; then
  if [ -n "$beads_remedy" ]; then
    say "beads is not ready: $beads_remedy"
  else
    say "beads is not ready, and bd did not say why: run bd info"
  fi
fi

[ -n "$lines" ] || exit 0

notice="codefall:
$lines"

if [ -n "$as_json" ] && command -v python3 >/dev/null 2>&1; then
  python3 -c 'import json, sys; print(json.dumps({"hookSpecificOutput": {"hookEventName": "SessionStart", "additionalContext": sys.argv[1]}}))' \
    "$notice"
  exit 0
fi

# Without python3 there is nothing to build the JSON with, and Claude Code adds a SessionStart
# hook's plain stdout to the session context as well, so the text is the fallback rather than
# nothing.
printf '%s\n' "$notice"
exit 0
