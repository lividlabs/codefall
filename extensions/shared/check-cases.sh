#!/usr/bin/env bash
#
# Check every test case file against the case-file contract.
#
# A case is a markdown file at <root>/test-cases/<area>/<slug>.md, where <root> is the testing root
# the project declares as `test.dir` in .codefall/project.json (default: testing). A project's CI
# calls this script; it reads, it never writes, and it needs nothing beyond coreutils — jq only to
# read the declared root.
#
# Usage: check-cases.sh [project-dir] [--root <dir>]
#
#   project-dir   the directory holding .codefall/ (default: .)
#   --root <dir>  the testing root, overriding what settings declare
#
# Rules
#   1  the file opens with a YAML frontmatter block, closed
#   2  `id` equals the file's path minus `.md`, relative to test-cases/
#   3  `modalities` is non-empty, holds only `spec` and `agentic`, and repeats neither
#   4  `variants` is non-empty and every variant has a unique, non-empty `name`
#   5  a case declaring `agentic` carries a `## Steps` section
#   6  a case declaring `spec` has a sibling: a file beside it whose name begins with the slug and
#      is not the `.md` itself, in whatever suffix the project's runner collects
#   7  every non-`.md` file under test-cases/ belongs to a case: some `<slug>.md` beside it is a
#      prefix of its name. A case is written before its spec, so a spec alone is a problem.
#
# Zero case files is a pass: this guards the contract, not the existence of tests. A `fixtures`
# directory holds case assets rather than cases, and is skipped whole.
#
# What the frontmatter parser understands
#   Top-level keys at column 0 (`id: checkout/place-order`), with the value optionally quoted.
#   Lists in either form: flow on one line (`modalities: [spec, agentic]`) or a block of `- value`
#   lines under the key. `variants` as a block sequence of mappings, each item's `name` read from
#   the `- name:` line or any later line of that item.
#
#   What it does not understand, and reports rather than guessing: `variants` written in flow style
#   (`variants: [{name: a}]`), a quoted or indented top-level key, a multi-line or folded scalar,
#   and YAML anchors. A nested mapping inside a variant that carries its own `name` key is read as
#   that variant's name.
#
# The Playwright rules are a separate script, check-cases-playwright.sh, because they need a Node
# toolchain that can fail for reasons no case file causes. This script names it at the end when the
# project declares that runner.
#
# Exit codes
#   0   every case file keeps the contract, or there are none
#   1   at least one problem; all of them are listed
#   2   usage error, or the project directory cannot be read

set -uo pipefail

project_dir=.
root_override=""

while [ $# -gt 0 ]; do
  case $1 in
    --root)
      shift
      [ $# -gt 0 ] || { printf '--root needs a directory\n' >&2; exit 2; }
      root_override=$1
      ;;
    --root=*) root_override=${1#--root=} ;;
    -h|--help) sed -n '2,47p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*) printf 'unknown flag: %s\n' "$1" >&2; exit 2 ;;
    *) project_dir=$1 ;;
  esac
  shift
done

if ! cd "$project_dir" 2>/dev/null; then
  printf 'cannot read project directory: %s\n' "$project_dir" >&2
  exit 2
fi

settings_file=.codefall/project.json
default_root=testing

root=$root_override
if [ -z "$root" ]; then
  root=$default_root
  if [ -f "$settings_file" ] && command -v jq >/dev/null 2>&1; then
    declared=$(jq -r 'if (.test.dir | type) == "string" and ((.test.dir | length) > 0)
                      then .test.dir else empty end' "$settings_file" 2>/dev/null)
    [ -n "$declared" ] && root=$declared
  fi
fi
root=${root%/}
cases_dir="$root/test-cases"

if [ ! -d "$cases_dir" ]; then
  printf 'check-cases OK — no %s directory yet.\n' "$cases_dir"
  exit 0
fi

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

problems=()
problem() { problems+=("$1"); }

# --- frontmatter readers ------------------------------------------------------------------------
#
# Each reads $tmp/front, which holds the frontmatter of the case file being checked.

front_scalar() {
  awk -v k="$1" -v q="'" '
    function clean(s) {
      sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s)
      gsub("^[" q "\"]|[" q "\"]$", "", s)
      return s
    }
    index($0, k ":") == 1 { print clean(substr($0, length(k) + 2)); exit }
  ' "$tmp/front"
}

front_has_key() {
  awk -v k="$1" 'index($0, k ":") == 1 { found = 1; exit } END { exit !found }' "$tmp/front"
}

front_list() {
  awk -v k="$1" -v q="'" '
    function clean(s) {
      sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s)
      gsub("^[" q "\"]|[" q "\"]$", "", s)
      return s
    }
    index($0, k ":") == 1 {
      rest = substr($0, length(k) + 2)
      sub(/^[ \t]+/, "", rest)
      if (rest ~ /^\[/) {
        sub(/^\[/, "", rest); sub(/\].*$/, "", rest)
        n = split(rest, a, ",")
        for (i = 1; i <= n; i++) { v = clean(a[i]); if (v != "") print v }
        exit
      }
      if (rest != "") { print "!!scalar"; exit }
      inblock = 1
      next
    }
    inblock && /^[^ \t#-]/ { exit }
    inblock && /^[ \t]*-[ \t]*/ {
      v = $0; sub(/^[ \t]*-[ \t]*/, "", v); v = clean(v)
      if (v != "") print v
      next
    }
  ' "$tmp/front"
}

# One line per variant: its name, or !!unnamed when the item carries none. !!unreadable when
# `variants` is written in a shape this parser does not read.
front_variant_names() {
  awk -v q="'" '
    function clean(s) {
      sub(/^[ \t]+/, "", s); sub(/[ \t]+$/, "", s)
      gsub("^[" q "\"]|[" q "\"]$", "", s)
      return s
    }
    function flush() {
      if (item) { if (name == "") print "!!unnamed"; else print name }
      item = 0; name = ""
    }
    index($0, "variants:") == 1 {
      rest = substr($0, 10); sub(/^[ \t]+/, "", rest)
      if (rest != "") { print "!!unreadable"; exit }
      inb = 1
      next
    }
    inb && /^[^ \t#-]/ { flush(); inb = 0 }
    inb && /^[ \t]*-/ {
      flush(); item = 1
      line = $0; sub(/^[ \t]*-[ \t]*/, "", line)
      if (index(line, "name:") == 1) name = clean(substr(line, 6))
      next
    }
    inb && item && /^[ \t]*name:[ \t]*/ {
      line = $0; sub(/^[ \t]*name:[ \t]*/, "", line); name = clean(line)
      next
    }
    END { flush() }
  ' "$tmp/front"
}

# --- collect ------------------------------------------------------------------------------------

find "$cases_dir" -type d -name fixtures -prune -o -type f -print | sort > "$tmp/all"
grep -E '\.md$' "$tmp/all" > "$tmp/cases"
grep -v -E '\.md$' "$tmp/all" > "$tmp/siblings"

case_count=$(wc -l < "$tmp/cases" | tr -d ' ')
sibling_count=$(wc -l < "$tmp/siblings" | tr -d ' ')

if [ "$case_count" -eq 0 ] && [ "$sibling_count" -eq 0 ]; then
  printf 'check-cases OK — no case files yet under %s.\n' "$cases_dir"
  exit 0
fi

# --- rules 1–6, per case file -------------------------------------------------------------------

while IFS= read -r f; do
  rel=${f#"$cases_dir"/}
  dir=$(dirname "$f")
  slug=$(basename "$f" .md)

  tr -d '\r' < "$f" > "$tmp/file"

  if [ "$(head -n 1 "$tmp/file")" != "---" ]; then
    problem "$f: no frontmatter block — a case file opens with \`---\`"
    continue
  fi
  if ! awk 'NR > 1 && /^---[ \t]*$/ { found = 1; exit } END { exit !found }' "$tmp/file"; then
    problem "$f: the frontmatter block is never closed"
    continue
  fi

  awk 'NR == 1 { next } /^---[ \t]*$/ { exit } { print }' "$tmp/file" > "$tmp/front"
  awk 'NR == 1 { infm = 1; next }
       infm == 1 && /^---[ \t]*$/ { infm = 2; next }
       infm == 2 { print }' "$tmp/file" > "$tmp/body"

  # 2 — id against the path.
  expected=${rel%.md}
  id=$(front_scalar id)
  if [ -z "$id" ]; then
    problem "$f: \`id\` must be a non-empty string"
  elif [ "$id" != "$expected" ]; then
    problem "$f: \`id\` is \"$id\" but the file's path says \"$expected\""
  fi

  # 3 — modalities.
  modalities=()
  while IFS= read -r m; do [ -n "$m" ] && modalities+=("$m"); done < <(front_list modalities)

  has_spec=0
  has_agentic=0
  if ! front_has_key modalities; then
    problem "$f: \`modalities\` is missing (allowed: spec, agentic)"
  elif [ ${#modalities[@]} -eq 0 ] || [ "${modalities[0]}" = "!!scalar" ]; then
    problem "$f: \`modalities\` must be a non-empty list of spec and agentic"
  else
    seen=""
    for m in "${modalities[@]}"; do
      case $m in
        spec) has_spec=1 ;;
        agentic) has_agentic=1 ;;
        *) problem "$f: unknown modality \"$m\" (allowed: spec, agentic)" ;;
      esac
      case " $seen " in
        *" $m "*) problem "$f: \`modalities\` repeats \"$m\"" ;;
        *) seen="$seen $m" ;;
      esac
    done
  fi

  # 4 — variants.
  names=()
  while IFS= read -r n; do [ -n "$n" ] && names+=("$n"); done < <(front_variant_names)

  if ! front_has_key variants; then
    problem "$f: \`variants\` is missing — a case runs once per named variant"
  elif [ ${#names[@]} -eq 0 ]; then
    problem "$f: \`variants\` must be a non-empty list of mappings, each with a \`name\`"
  elif [ "${names[0]}" = "!!unreadable" ]; then
    problem "$f: \`variants\` is written in a shape this check cannot read — use a block sequence of mappings"
  else
    seen=""
    for n in "${names[@]}"; do
      if [ "$n" = "!!unnamed" ]; then
        problem "$f: a variant has no \`name\`"
        continue
      fi
      case " $seen " in
        *" $n "*) problem "$f: duplicate variant name \"$n\"" ;;
        *) seen="$seen $n" ;;
      esac
    done
  fi

  # 5 — agentic implies Steps.
  if [ "$has_agentic" -eq 1 ] && ! grep -q -E '^##[ \t]+Steps([ \t]|$)' "$tmp/body"; then
    problem "$f: modality \"agentic\" but no \`## Steps\` section"
  fi

  # 6 — spec implies a sibling.
  if [ "$has_spec" -eq 1 ]; then
    found_sibling=0
    for g in "$dir"/*; do
      [ -f "$g" ] || continue
      b=$(basename "$g")
      case $b in *.md) continue ;; esac
      case $b in "$slug"*) found_sibling=1; break ;; esac
    done
    if [ "$found_sibling" -eq 0 ]; then
      problem "$f: modality \"spec\" but no sibling file beside it whose name begins with \"$slug\""
    fi
  fi
done < "$tmp/cases"

# --- rule 7 — every sibling belongs to a case ---------------------------------------------------

while IFS= read -r f; do
  dir=$(dirname "$f")
  b=$(basename "$f")
  belongs=0
  for m in "$dir"/*.md; do
    [ -f "$m" ] || continue
    s=$(basename "$m" .md)
    case $b in "$s"*) belongs=1; break ;; esac
  done
  if [ "$belongs" -eq 0 ]; then
    problem "$f: no case file beside it — a case is written before its spec"
  fi
done < "$tmp/siblings"

# --- report -------------------------------------------------------------------------------------

playwright_note() {
  local declared=""
  if [ -f "$settings_file" ]; then
    if command -v jq >/dev/null 2>&1; then
      declared=$(jq -r 'if ((.test.runners // []) | index("playwright")) then "yes" else empty end' \
        "$settings_file" 2>/dev/null)
    elif grep -q '"playwright"' "$settings_file" 2>/dev/null; then
      declared=yes
    fi
  fi
  [ -n "$declared" ] && printf 'check-cases — this project declares playwright; run check-cases-playwright.sh for its own two rules.\n'
  return 0
}

if [ ${#problems[@]} -gt 0 ]; then
  printf 'check-cases FAILED (%d problem(s)):\n' "${#problems[@]}"
  for p in "${problems[@]}"; do
    printf '  - %s\n' "$p"
  done
  playwright_note
  exit 1
fi

printf 'check-cases OK — %d case file(s), %d sibling file(s) under %s.\n' \
  "$case_count" "$sibling_count" "$cases_dir"
playwright_note
exit 0
