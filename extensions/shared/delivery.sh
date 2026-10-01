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
# For an epic:
#   booking-DESIGN-007 · round 2 · 14/17 done ██████████████░░░ · 3 open · 0/9 PRs merged
#   - children from `bd list --parent <epic> --all`, less the landed bead `<epic>-MERGED` and any
#     child labelled `design-amended`, which record an amendment and are never work
#   - done is `closed`; open is every other status, `deferred` included
#   - the round is `metadata.round` on the epic, 1 when unset; `codefall-implement` writes it
#   - PRs are the `gh:pr` gates on `<epic>-MERGED`; one gate reads as the aggregate PR, none
#     omits the segment
#
# For any other bead, the same counts over what was discovered from it:
#   booking-gh-131 · 1/2 done █████████░░░░░░░░ · 1 open
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

# bd show accepts several ids and may answer with an array; take the one object either way.
one='if type == "array" then .[0] else . end'

shown=$(bd show "$id" --json 2>/dev/null) || exit 0
[ -n "$shown" ] || exit 0
kind=$(printf '%s' "$shown" | jq -r "$one | (.issue_type // .type // \"\")" 2>/dev/null)
[ -n "$kind" ] || exit 0
round=$(printf '%s' "$shown" | jq -r "$one | (.metadata.round // empty)" 2>/dev/null)

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

# Counts over a JSON array of issues: "<done> <open>".
counts() {
  jq -r '
    (if type == "array" then . else [] end) as $all
    | [$all[] | select(.status == "closed")] | length as $done
    | ($all | length) - $done as $open
    | "\($done) \($open)"
  ' 2>/dev/null
}

if [ "$kind" = "epic" ]; then
  landed="$id-MERGED"
  children=$(bd list --parent "$id" --all --json --limit 0 2>/dev/null)
  [ -n "$children" ] || children='[]'
  read -r done open < <(printf '%s' "$children" | jq --arg landed "$landed" '
    (if type == "array" then . else [] end)
    | map(select(.id != $landed and ((.labels // []) | index("design-amended") | not)))
  ' 2>/dev/null | counts)
  done=${done:-0}; open=${open:-0}
  total=$((done + open))

  line="$id · round ${round:-1} · $done/$total done $(bar "$done" "$total") · $open open"

  gates=$(bd gate list "$landed" --all --json 2>/dev/null)
  if [ -n "$gates" ]; then
    read -r merged pending < <(printf '%s' "$gates" | counts)
    merged=${merged:-0}; pending=${pending:-0}
    gate_total=$((merged + pending))
    if [ "$gate_total" -eq 1 ]; then
      if [ "$merged" -eq 1 ]; then line+=" · aggregate PR merged"; else line+=" · aggregate PR open"; fi
    elif [ "$gate_total" -gt 1 ]; then
      line+=" · $merged/$gate_total PRs merged"
    fi
  fi

  printf '%s\n' "$line"
  exit 0
fi

dependents=$(bd dep list "$id" --direction=up -t discovered-from --json 2>/dev/null)
[ -n "$dependents" ] || dependents='[]'
read -r done open < <(printf '%s' "$dependents" | jq '
  (if type == "array" then . else [] end)
  | map(select((.labels // []) | index("design-amended") | not))
' 2>/dev/null | counts)
done=${done:-0}; open=${open:-0}
total=$((done + open))

if [ "$total" -eq 0 ]; then
  printf '%s · nothing discovered from it\n' "$id"
  exit 0
fi

line="$id"
[ -n "$round" ] && line+=" · round $round"
line+=" · $done/$total done $(bar "$done" "$total") · $open open"
printf '%s\n' "$line"
