#!/usr/bin/env bash
#
# Report every skill against the length guidelines and the supporting-file rules.
#
# The guidelines are Anthropic's published ones for a SKILL.md body, adopted as
# guidelines rather than hard limits in docs/adrs/ADR-004.2-skill-length-guidelines.md;
# the placement rules are in extensions/skills/AGENTS.md. This script reports. It
# fails only when asked to.
#
# Usage: skill-health.sh [--strict] [skill-dir ...]
#
#   skill-dir   a directory holding a SKILL.md; default is every one under
#               extensions/skills/
#   --strict    exit 1 when any skill is outside a guideline, for CI
#
# Checks, per skill
#   lines         SKILL.md body, after the frontmatter: under 500
#   tokens        estimated two ways, chars/4 and words*1.33; the larger is judged: under 5000
#   description   under 1024 characters
#   name          frontmatter name matches the directory
#   invocation    disable-model-invocation: true, except for the skills listed in
#                 MODEL_INVOCABLE, which carry no such line
#   references    every ./ or ../ path SKILL.md names resolves
#   depth         a referenced file names no further file that SKILL.md does not also name
#   contents      a referenced file over 100 lines has a table of contents
#   unreferenced  files in the skill directory nothing names — informational, never a failure
#
# A reference is a markdown link target or a backticked path with a file
# extension. Bare paths that do not resolve are taken to be the user's project
# and ignored; paths with placeholders are ignored. Files under templates/ are
# installed into projects, not read as instruction, so depth and contents skip
# them, and a supporting file may name one without counting as nested.
# Another skill's SKILL.md, named by a skill that follows its procedure, is
# checked on its own row, so depth and contents skip it too.
# NOTES.md is never loaded by a skill and is exempt from everything.
#
# Token counts are estimates. The tokenizer is not available to a script, so
# two heuristics are reported and the larger one is judged.
#
# Exit codes
#   0   report printed; with --strict, every skill inside the guidelines
#   1   --strict and at least one skill outside a guideline
#   2   usage error

set -uo pipefail

LINES_MAX=500
TOKENS_MAX=5000
DESC_MAX=1024
# Skills an agent may invoke on its own; every other skill carries disable-model-invocation: true.
MODEL_INVOCABLE="codefall-refresh codefall-design codefall-implement codefall-test codefall-review"
TOC_LINES=100

strict=0
dirs=()
for arg in "$@"; do
  case "$arg" in
    --strict) strict=1 ;;
    -h|--help) sed -n '2,40p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*) printf 'unknown flag: %s\n' "$arg" >&2; exit 2 ;;
    *) dirs+=("$arg") ;;
  esac
done

here="$(cd "$(dirname "$0")" && pwd -P)"
ext_root="$(cd "$here/.." && pwd -P)"
skills_root="$ext_root/skills"
if [ ${#dirs[@]} -eq 0 ]; then
  for d in "$skills_root"/*/; do
    [ -f "$d/SKILL.md" ] && dirs+=("${d%/}")
  done
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# Every relative reference a markdown file names, one per line, unique.
refs() {
  {
    grep -o -E '\]\([^)]+\)' "$1" 2>/dev/null | sed -E 's/^\]\(//; s/\)$//; s/ .*$//; s/#.*$//'
    grep -o -E '`[^` ]+\.(md|sh|json|skeleton|txt|yml|yaml)`' "$1" 2>/dev/null | tr -d '`'
  } | grep -v -E '^(https?:|/|~|$)' | grep -v -E '[<>{}*$]' | sort -u
}

# Absolute path of a reference relative to a directory, or nothing when it does not exist.
#
# A skill names a shared file by the path it has once installed —
# ../../../.codefall/shared/<file> from a skill directory, one level deeper from a supporting
# file. `codefall init` writes that copy into the project's .codefall/; in this repository the
# same file is under extensions/, so the .codefall/ prefix resolves there instead.
resolve() {
  local base="$1" ref="$2" target
  case "$ref" in
    */.codefall/*) target="$ext_root/${ref#*.codefall/}" ;;
    *) target="$base/$ref" ;;
  esac
  [ -e "$target" ] || return 1
  if [ -d "$target" ]; then
    (cd "$target" && pwd -P)
  else
    (cd "$(dirname "$target")" && printf '%s/%s\n' "$(pwd -P)" "$(basename "$target")")
  fi
}

has_toc() {
  grep -q -i -E '^#{1,3} +(contents|table of contents)' "$1"
}

row() { printf '  %-13s %s\n' "$1" "$2"; }

# A path for the report: relative to the skill when inside it, to extensions/ otherwise.
show() {
  case "$1" in
    "$dir"/*) printf '%s\n' "${1#$dir/}" ;;
    "$ext_root"/*) printf '%s\n' "${1#$ext_root/}" ;;
    *) printf '%s\n' "$1" ;;
  esac
}

failed_skills=()

for dir in "${dirs[@]}"; do
  dir="$(cd "$dir" && pwd -P)"
  skill="$dir/SKILL.md"
  name_dir="$(basename "$dir")"
  bad=0

  if [ ! -f "$skill" ]; then
    printf '%s\n' "$name_dir"
    row "SKILL.md" "missing"
    failed_skills+=("$name_dir")
    continue
  fi

  awk 'NR==1 && /^---$/ {f=1; next} f==1 && /^---$/ {f=2; next} f==1 {print}' "$skill" > "$tmp/front"
  awk 'NR==1 && /^---$/ {f=1; next} f==1 && /^---$/ {f=2; next} f==2 {print}' "$skill" > "$tmp/body"

  fm_name="$(sed -n -E 's/^name: *//p' "$tmp/front" | head -1 | sed -E 's/^"(.*)"$/\1/')"
  fm_desc="$(sed -n -E 's/^description: *//p' "$tmp/front" | head -1 | sed -E 's/^"(.*)"$/\1/')"
  fm_dmi="$(sed -n -E 's/^disable-model-invocation: *//p' "$tmp/front" | head -1)"

  lines=$(wc -l < "$tmp/body" | tr -d ' ')
  words=$(wc -w < "$tmp/body" | tr -d ' ')
  chars=$(wc -c < "$tmp/body" | tr -d ' ')
  tok_c=$((chars / 4))
  tok_w=$((words * 133 / 100))
  tok=$tok_c; [ "$tok_w" -gt "$tok" ] && tok=$tok_w
  desc_len=${#fm_desc}

  printf '%s\n' "$name_dir"

  if [ "$lines" -lt "$LINES_MAX" ]; then
    row "lines" "$(printf '%5d   ok     (under %d)' "$lines" "$LINES_MAX")"
  else
    row "lines" "$(printf '%5d   over   (under %d)' "$lines" "$LINES_MAX")"; bad=1
  fi

  if [ "$tok" -lt "$TOKENS_MAX" ]; then
    row "tokens" "$(printf '~%4d   ok     (under %d; chars/4 = %d, words*1.33 = %d)' "$tok" "$TOKENS_MAX" "$tok_c" "$tok_w")"
  else
    row "tokens" "$(printf '~%4d   over   (under %d; chars/4 = %d, words*1.33 = %d)' "$tok" "$TOKENS_MAX" "$tok_c" "$tok_w")"; bad=1
  fi
  row "words" "$(printf '%5d' "$words")"

  if [ "$desc_len" -eq 0 ]; then
    row "description" "missing"; bad=1
  elif [ "$desc_len" -le "$DESC_MAX" ]; then
    row "description" "$(printf '%5d   ok     (max %d)' "$desc_len" "$DESC_MAX")"
  else
    row "description" "$(printf '%5d   over   (max %d)' "$desc_len" "$DESC_MAX")"; bad=1
  fi

  if [ "$fm_name" = "$name_dir" ]; then
    row "name" "ok"
  else
    row "name" "frontmatter says '$fm_name', directory is '$name_dir'"; bad=1
  fi

  case " $MODEL_INVOCABLE " in
    *" $name_dir "*)
      if [ -z "$fm_dmi" ]; then
        row "invocation" "ok     (model-invocable)"
      else
        row "invocation" "disable-model-invocation is set; $name_dir is model-invocable and carries no such line"; bad=1
      fi ;;
    *)
      if [ "$fm_dmi" = "true" ]; then
        row "invocation" "ok"
      else
        row "invocation" "disable-model-invocation is not true"; bad=1
      fi ;;
  esac

  # References named by SKILL.md.
  : > "$tmp/resolved"
  unresolved=()
  named=0
  while IFS= read -r ref; do
    [ -n "$ref" ] || continue
    if abs="$(resolve "$dir" "$ref")"; then
      [ "$abs" = "$skill" ] && continue
      named=$((named + 1))
      printf '%s\n' "$abs" >> "$tmp/resolved"
    elif [[ "$ref" == ./* || "$ref" == ../* ]]; then
      named=$((named + 1))
      unresolved+=("$ref")
    fi
  done < <(refs "$skill")

  if [ ${#unresolved[@]} -eq 0 ]; then
    row "references" "$(printf '%5d   ok' "$named")"
  else
    row "references" "$(printf '%5d   %d unresolved:' "$named" "${#unresolved[@]}")"
    for u in "${unresolved[@]}"; do printf '                %s\n' "$u"; done
    bad=1
  fi

  # Depth and contents, over the referenced files the skill reads.
  nested=()
  notoc=()
  while IFS= read -r abs; do
    [ -f "$abs" ] || continue
    case "$abs" in *.md) ;; *) continue ;; esac
    case "$abs" in */templates/*) continue ;; esac
    case "$abs" in */SKILL.md) continue ;; esac
    base="$(dirname "$abs")"
    while IFS= read -r ref; do
      [ -n "$ref" ] || continue
      if sub="$(resolve "$base" "$ref")"; then
        [ "$sub" = "$abs" ] && continue
        [ "$sub" = "$skill" ] && continue
        case "$sub" in */templates/*) continue ;; esac
        if ! grep -q -x -F "$sub" "$tmp/resolved"; then
          nested+=("$(show "$abs") -> $ref")
        fi
      fi
    done < <(refs "$abs")
    n=$(wc -l < "$abs" | tr -d ' ')
    if [ "$n" -gt "$TOC_LINES" ] && ! has_toc "$abs"; then
      notoc+=("$(show "$abs") ($n lines)")
    fi
  done < <(sort -u "$tmp/resolved")

  if [ ${#nested[@]} -eq 0 ]; then
    row "depth" "ok"
  else
    row "depth" "${#nested[@]} nested, not named by SKILL.md:"
    for x in "${nested[@]}"; do printf '                %s\n' "$x"; done
    bad=1
  fi

  if [ ${#notoc[@]} -eq 0 ]; then
    row "contents" "ok"
  else
    row "contents" "${#notoc[@]} over $TOC_LINES lines without a table of contents:"
    for x in "${notoc[@]}"; do printf '                %s\n' "$x"; done
    bad=1
  fi

  # Files nothing names. Informational.
  orphans=()
  while IFS= read -r f; do
    case "$f" in
      "$skill"|"$dir/NOTES.md"|*/templates/*) continue ;;
    esac
    if ! grep -q -x -F "$f" "$tmp/resolved"; then
      orphans+=("${f#$dir/}")
    fi
  done < <(find "$dir" -type f | sort)

  if [ ${#orphans[@]} -eq 0 ]; then
    row "unreferenced" "none"
  else
    row "unreferenced" "${#orphans[@]} (informational):"
    for x in "${orphans[@]}"; do printf '                %s\n' "$x"; done
  fi

  [ "$bad" -eq 1 ] && failed_skills+=("$name_dir")
  printf '\n'
done

total=${#dirs[@]}
if [ ${#failed_skills[@]} -eq 0 ]; then
  printf '%d skills, all inside the guidelines\n' "$total"
  exit 0
fi

printf '%d skills, %d outside the guidelines: %s\n' "$total" "${#failed_skills[@]}" "${failed_skills[*]}"
[ "$strict" -eq 1 ] && exit 1
exit 0
