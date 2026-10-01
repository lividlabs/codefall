#!/usr/bin/env bash
#
# The delivery chart: one line on where an epic's work stands, read from Beads.
#
# A delivery is the work on one epic from the design's graph to the human merge, taken in rounds of
# implement, review, and test; it ends when the epic has no open children and the last test run
# passed. Every verb in a delivery prints this line at its start and in its close-out, so the person
# running the chain sees the same count wherever they are. The script reads and never writes.
#
# Usage: delivery.sh <epic-or-bead-id> [project-dir]      (default: .)
#
# For an epic, the epic, the round, and the children closed over the children in all:
#   booking-DESIGN-007 · round 2 · 10/12 ██████████████░░░
#   - the counts are `bd epic status --json`'s own, every child included, so this line and
#     `bd show <epic>` agree
#   - the round is `metadata.round` on the epic, 1 when unset; `codefall-implement` writes it
#
# For any other bead, the same bar over what was discovered from it:
#   booking-DESIGN-007-T1 · 1/2 █████████░░░░░░░░
#   booking-DESIGN-007-T3 · nothing discovered from it
#
# Prints nothing and exits 0 when `bd` is not on PATH or the id resolves to nothing, so a skill can
# call it unconditionally. Without jq it prints the id and says what is missing.

set -uo pipefail

id=${1:-}
project_dir=${2:-.}

[ -n "$id" ] || exit 0
cd "$project_dir" 2>/dev/null || exit 0
command -v bd >/dev/null 2>&1 || exit 0

if ! command -v jq >/dev/null 2>&1; then
  printf '%s · (install jq for the chart)\n' "$id"
  exit 0
fi

# Seventeen cells, done on the left. Scaled to the total; an empty total is an empty bar.
bar() {
  local done=$1 total=$2 cells=17 filled=0 out="" i
  if [ "$total" -gt 0 ]; then
    filled=$(( (done * cells + total / 2) / total ))
  fi
  for ((i = 0; i < cells; i++)); do
    if [ "$i" -lt "$filled" ]; then out+="█"; else out+="░"; fi
  done
  printf '%s' "$out"
}

# An epic: Beads counts the children itself.
entry=$(bd epic status --json 2>/dev/null | jq -c --arg id "$id" '
  (if type == "array" then . else [] end) | map(select(.epic.id == $id)) | .[0] // empty
' 2>/dev/null)

if [ -n "$entry" ]; then
  read -r round done total < <(printf '%s' "$entry" | jq -r '
    "\(.epic.metadata.round // 1) \(.closed_children // 0) \(.total_children // 0)"
  ' 2>/dev/null)
  printf '%s · round %s · %s/%s %s\n' "$id" "${round:-1}" "${done:-0}" "${total:-0}" \
    "$(bar "${done:-0}" "${total:-0}")"
  exit 0
fi

# Any other bead: the work discovered from it, if any.
bd show "$id" --json >/dev/null 2>&1 || exit 0

read -r done total < <(bd dep list "$id" --direction=up -t discovered-from --json 2>/dev/null | jq -r '
  (if type == "array" then . else [] end) as $all
  | "\([$all[] | select(.status == "closed")] | length) \($all | length)"
' 2>/dev/null)
done=${done:-0}; total=${total:-0}

if [ "$total" -eq 0 ]; then
  printf '%s · nothing discovered from it\n' "$id"
  exit 0
fi

printf '%s · %s/%s %s\n' "$id" "$done" "$total" "$(bar "$done" "$total")"
