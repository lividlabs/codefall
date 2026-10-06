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
# actual base branch; a `git push` is judged by its own arguments, not by the rest of the
# line. A destination named through a variable or a command substitution is not resolved.
# It prefers a rare false denial over a false allow, and it is a
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
  # A merge into an epic branch or another non-default base is allowed; the rest of the
  # line still goes through the checks below.
fi

current=$(git branch --show-current 2>/dev/null)

# --- git merge while standing on the protected branch. ---
if printf '%s' "$cmd" | grep -qE '(^|[;&|[:space:]])git[[:space:]]+merge([[:space:]]|$)'; then
  [ "$current" = "$protected" ] && \
    deny "denied: 'git merge' while on '$protected'." "$protected"
fi

# --- git push whose destination is the protected branch. ---
# The line is split into commands at ;, &&, ||, |, & and newlines, and each `git push` is read
# from its own arguments: the first operand is the remote, every later one a refspec, and a
# refspec's destination is what follows its colon, or the source itself when there is none.
# So `git push origin feat && gh pr create --base main` is allowed, and `origin main`,
# `HEAD:main`, `+main`, `feat:refs/heads/main`, `:main`, `HEAD` while on main, and `--all` or
# `--mirror` are denied. A push with no refspec while standing on the protected branch pushes
# it, and is denied too. The script of `sh -c`/`bash -c` and the words of `eval` are read the
# same way. A line shlex cannot tokenize (an unbalanced quote) is split on whitespace instead.
# The program is read into a variable first: a quoted heredoc inside $(...) trips the parser
# of the bash 3.2 that macOS ships.
IFS= read -r -d '' judge_py <<'PY'
import fnmatch, os, re, shlex, sys

cmd, protected, current = sys.argv[1:4]
OPS = "();<>|&\n"
SHELLS = {"sh", "bash", "zsh", "dash", "ksh"}
GIT_VALUE_OPTS = {"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--config-env", "--super-prefix"}
PUSH_VALUE_OPTS = {"-o", "--push-option", "--repo", "--receive-pack", "--exec"}


def tokenize(text):
    text = text.replace("\\\n", "")
    try:
        lex = shlex.shlex(text, posix=True, punctuation_chars=OPS)
        lex.whitespace = " \t\r"
        lex.whitespace_split = True
        lex.commenters = ""
        return list(lex)
    except ValueError:
        return [w.strip("'\"") for w in re.findall(r"[();<>|&\n]+|[^\s();<>|&]+", text)]


def commands(tokens):
    seg = []
    for tok in tokens:
        if tok and all(c in OPS for c in tok):
            if seg:
                yield seg
            seg = []
        else:
            seg.append(tok)
    if seg:
        yield seg


def protected_destination(spec):
    spec = spec[1:] if spec.startswith("+") else spec
    if spec.startswith("^"):
        return False  # a negative refspec excludes, it pushes nothing
    src, colon, dst = spec.partition(":")
    if colon and not src and not dst:
        return True  # ":" pushes every matching branch
    dst = dst or src
    if dst in ("HEAD", "@"):
        dst = current
    for prefix in ("refs/heads/", "heads/"):
        if dst.startswith(prefix):
            dst = dst[len(prefix):]
            break
    if "*" in dst:
        return fnmatch.fnmatchcase(protected, dst)
    return dst == protected


def judge_push(args):
    operands, every_branch, options_done, skip = [], None, False, False
    for arg in args:
        if skip:
            skip = False
        elif options_done or not arg.startswith("-") or arg == "-":
            operands.append(arg)
        elif arg == "--":
            options_done = True
        elif arg in PUSH_VALUE_OPTS:
            skip = True
        elif arg in ("--all", "--branches", "--mirror"):
            every_branch = arg
    if every_branch:
        return "denied: 'git push %s' pushes every branch, '%s' among them." % (every_branch, protected)
    refspecs, after_tag = operands[1:], False
    if not refspecs:
        if current == protected:
            return "denied: bare 'git push' while on '%s'." % protected
        return None
    for spec in refspecs:
        if after_tag:
            after_tag = False  # `tag <name>` pushes refs/tags/<name>
        elif spec == "tag":
            after_tag = True
        elif protected_destination(spec):
            return "denied: 'git push' targeting '%s'." % protected
    return None


def judge(text, depth=0):
    for seg in commands(tokenize(text)):
        for i, tok in enumerate(seg):
            nested = None
            if depth < 3 and os.path.basename(tok) in SHELLS:
                for k in range(i + 1, len(seg) - 1):
                    if re.fullmatch(r"-[A-Za-z]*c[A-Za-z]*", seg[k]):
                        nested = seg[k + 1]
                        break
            elif depth < 3 and tok == "eval":
                nested = " ".join(seg[i + 1:])
            if nested is not None:
                reason = judge(nested, depth + 1)
                if reason:
                    return reason
            if os.path.basename(tok) != "git":
                continue
            j = i + 1
            while j < len(seg) and seg[j].startswith("-"):
                j += 2 if seg[j] in GIT_VALUE_OPTS else 1
            if j < len(seg) and seg[j] == "push":
                reason = judge_push(seg[j + 1:])
                if reason:
                    return reason
    return None


print(judge(cmd) or "")
PY
push=$(python3 -c "$judge_py" "$cmd" "$protected" "$current" 2>/dev/null)
[ -n "$push" ] && deny "$push" "$protected"

exit 0
