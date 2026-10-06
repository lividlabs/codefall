#!/usr/bin/env bash
# PreToolUse guard: deny any shell command that would merge or push to the default branch.
#
# No codefall verb merges to main. A person merges a code pull request, and the
# land-documents workflow merges a document pull request on the `auto-merge` label or an
# approving review when the project has it installed. A denial from this hook is the system working as designed. For Claude Code and Codex a deny is exit 2 with
# the reason on stderr, shown to the model. Antigravity needs a JSON decision on stdout,
# which --antigravity selects. Exit 0 raises no objection; the normal permission flow
# still applies.
#
# The input shape differs by harness: Claude Code and Codex put the command in
# .tool_input.command, Antigravity in .toolCall.args.CommandLine. The deny shape is the
# only other difference, so one script serves all three.
#
# Deliberate limits: this inspects the command string plus, for `gh pr merge`, the PR's
# actual base branch. It prefers a rare false denial over a false allow, and it is a
# guard, not the only line — a repository ruleset protecting the default branch remains
# the backstop. `gh stack merge` is denied on the command string alone, without reading
# the stack, because a stack's trunk is the default branch in every codefall use.
set -u

antigravity=
if [ "${1:-}" = "--antigravity" ]; then
  antigravity=yes
fi

input=$(cat)

if [ -n "$antigravity" ]; then
  cmd=$(printf '%s' "$input" | python3 -c \
    'import json,sys; print(json.load(sys.stdin).get("toolCall",{}).get("args",{}).get("CommandLine",""))' \
    2>/dev/null) || exit 0
else
  cmd=$(printf '%s' "$input" | python3 -c \
    'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("command",""))' \
    2>/dev/null) || exit 0
fi
[ -z "$cmd" ] && exit 0

deny() {
  reason="codefall: $1 No verb merges to '$2': a person merges a code pull request, and the land-documents workflow merges a document pull request on the 'auto-merge' label or an approving review. Report and stop."
  if [ -n "$antigravity" ]; then
    python3 -c 'import json,sys; print(json.dumps({"decision": "deny", "reason": sys.argv[1]}))' "$reason"
    exit 0
  fi
  echo "$reason" >&2
  exit 2
}

# The protected branch: origin's default, else main.
protected=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's|^origin/||')
[ -z "$protected" ] && protected=main

# --- gh stack merge: lands every layer of a stack on its trunk, which is the default branch. ---
if printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])gh[[:space:]]+stack[[:space:]]+merge([[:space:]]|$)'; then
  deny "denied: 'gh stack merge' lands a stack on '$protected'." "$protected"
fi

# --- gh pr merge: the PR's base decides; an undetermined base is denied, not allowed. ---
if printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])gh[[:space:]]+pr[[:space:]]+merge([[:space:]]|$)'; then
  ref=$(printf '%s' "$cmd" | awk '
    {for (i = 2; i <= NF; i++)
       if ($(i-1) == "pr" && $i == "merge") {
         for (j = i + 1; j <= NF; j++) if ($j !~ /^-/) { print $j; exit }
       }}')
  # A bare `gh pr merge` names no PR and means the current branch's: gh answers that
  # question only when asked without a branch, because `gh pr view ""` looks up a branch
  # called nothing and always fails.
  if [ -z "$ref" ]; then
    base=$(gh pr view --json baseRefName --jq .baseRefName 2>/dev/null)
  else
    base=$(gh pr view "$ref" --json baseRefName --jq .baseRefName 2>/dev/null)
  fi
  if [ -z "$base" ] || [ "$base" = "$protected" ]; then
    deny "denied: 'gh pr merge' into '$protected' (or into a base this hook could not determine)." "$protected"
  fi
  exit 0  # merge into an epic branch or another non-default base: allowed
fi

current=$(git branch --show-current 2>/dev/null)

# --- git merge while standing on the protected branch. ---
if printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])git[[:space:]]+merge([[:space:]]|$)'; then
  [ "$current" = "$protected" ] && \
    deny "denied: 'git merge' while on '$protected'." "$protected"
fi

# --- git push with a refspec naming the protected branch (origin main, HEAD:main). ---
# The name is regex-escaped first: a branch with a dot in it must not match what isn't there.
if printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])git[[:space:]]+push([[:space:]]|$)'; then
  escaped=$(printf '%s' "$protected" | sed 's/[][\\.*^$/]/\\&/g')
  if printf '%s' "$cmd" | grep -qE "[[:space:]:]${escaped}([[:space:]]|\$)"; then
    deny "denied: 'git push' targeting '$protected'." "$protected"
  fi
  # A bare push (no refspec) while standing on the protected branch pushes it.
  if [ "$current" = "$protected" ] && \
     printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])git[[:space:]]+push([[:space:]]+-[[:alnum:]=-]+)*([[:space:]]+[[:alnum:]._-]+)?[[:space:]]*$'; then
    deny "denied: bare 'git push' while on '$protected'." "$protected"
  fi
fi

exit 0
