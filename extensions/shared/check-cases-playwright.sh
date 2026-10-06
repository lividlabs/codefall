#!/usr/bin/env bash
#
# The two Playwright-specific case checks. They run only when the project declares that runner in
# `test.runners` in .codefall/settings.json; otherwise the script says so and exits 0, so a CI job
# can call it unconditionally.
#
# Usage: check-cases-playwright.sh [project-dir] [--root <dir>]
#
#   project-dir   the directory holding .codefall/ (default: .)
#   --root <dir>  the testing root, overriding what settings declare
#
# Rules
#   1  every test title begins with its case id. Titles are read from
#      `npx playwright test --list --pass-with-no-tests --reporter=json`, i.e. from the specs as
#      Playwright actually collects them, so a title built from a template literal is checked
#      correctly. A variant-scoped title (`checkout/place-order — [saved-card] …`) satisfies the
#      rule, since the id is its prefix either way. Only specs under <root>/test-cases/ are checked;
#      a project's other Playwright tests are none of this script's business. `--pass-with-no-tests`
#      (Playwright 1.26+) keeps an empty tree a pass: without it Playwright exits 1 with "No tests
#      found", which this script would read as a spec failing to load. A spec that does fail to load
#      still exits non-zero with the flag.
#   2  no `.isVisible().catch(...)` on an un-narrowed locator. `Locator.isVisible()` throws on a
#      strict-mode multi-match, so a `.catch()` around it turns "several elements matched" into
#      "no element is there" — an absence that never happened, reported as a product defect.
#      Narrowing the receiver first (`.first()`, `.last()`, `.nth(n)`) makes the read strict-safe.
#
# Rule 2 is a line-based scan rather than a parse: a receiver that spans lines is read from the
# previous non-blank line, a match after `//` on the same line is ignored, and a match inside a
# block comment or a string is still reported — quote the construct differently there.
#
# check-cases.sh holds the runner-neutral rules and does not call this script. The two are separate
# because these need a Node toolchain, which fails for reasons no case file causes.
#
# Exit codes
#   0   both rules hold, or the project does not declare playwright
#   1   at least one problem; all of them are listed
#   2   usage error, the project directory cannot be read, jq is missing, or Playwright would not
#       list its tests

set -uo pipefail

project_dir=.
root_override=""

while [ $# -gt 0 ]; do
  case $1 in
    --root)
      shift
      [ $# -gt 0 ] || { printf -- '--root needs a directory\n' >&2; exit 2; }
      root_override=$1
      ;;
    --root=*) root_override=${1#--root=} ;;
    -h|--help) sed -n '2,39p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    -*) printf 'unknown flag: %s\n' "$1" >&2; exit 2 ;;
    *) project_dir=$1 ;;
  esac
  shift
done

if ! cd "$project_dir" 2>/dev/null; then
  printf 'cannot read project directory: %s\n' "$project_dir" >&2
  exit 2
fi

settings_file=.codefall/settings.json
default_root=testing

declares_playwright() {
  [ -f "$settings_file" ] || return 1
  if command -v jq >/dev/null 2>&1; then
    jq -e '((.test.runners // []) | index("playwright")) != null' "$settings_file" >/dev/null 2>&1
    return
  fi
  grep -q '"playwright"' "$settings_file" 2>/dev/null
}

if ! declares_playwright; then
  printf 'check-cases-playwright skipped — this project does not declare the playwright runner.\n'
  exit 0
fi

if ! command -v jq >/dev/null 2>&1; then
  printf 'check-cases-playwright needs jq to read Playwright'"'"'s JSON listing.\n' >&2
  exit 2
fi

root=$root_override
if [ -z "$root" ]; then
  root=$default_root
  declared=$(jq -r 'if (.test.dir | type) == "string" and ((.test.dir | length) > 0)
                    then .test.dir else empty end' "$settings_file" 2>/dev/null)
  [ -n "$declared" ] && root=$declared
fi
root=${root%/}
cases_dir="$root/test-cases"

if [ ! -d "$cases_dir" ]; then
  printf 'check-cases-playwright OK — no %s directory yet.\n' "$cases_dir"
  exit 0
fi

cases_abs=$(cd "$cases_dir" && pwd -P)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

problems=()
problem() { problems+=("$1"); }

# --- rule 1 — every test title begins with its case id ------------------------------------------

if ! npx playwright test --list --pass-with-no-tests --reporter=json > "$tmp/list.json" 2> "$tmp/list.err"; then
  printf 'check-cases-playwright: playwright would not list its tests (a spec may fail to load):\n' >&2
  cat "$tmp/list.err" >&2
  exit 2
fi

jq -r '
  def titles($anc):
    ((.specs // [])[] | [ .file, (($anc + [.title]) | join(" ")) ]),
    ((.suites // [])[]
      | titles(
          if ((.title // "") == "") or (.title == (.file // ""))
          then $anc
          else $anc + [.title]
          end));
  (.config.rootDir // ".") as $root
  | [ .suites[]? | titles([]) ]
  | .[]
  | [ (if (.[0] | startswith("/")) then .[0] else ($root + "/" + .[0]) end), .[1] ]
  | @tsv
' "$tmp/list.json" > "$tmp/titles" 2> "$tmp/titles.err" || {
  printf 'check-cases-playwright: could not read the JSON listing:\n' >&2
  cat "$tmp/titles.err" >&2
  exit 2
}

checked=0
while IFS=$'\t' read -r file title; do
  [ -n "$file" ] || continue
  case $file in
    "$cases_abs"/*) rel=${file#"$cases_abs"/} ;;
    *) continue ;;
  esac
  dir=$(dirname "$rel")
  base=$(basename "$rel")
  slug=${base%%.*}
  if [ "$dir" = "." ]; then id=$slug; else id="$dir/$slug"; fi
  checked=$((checked + 1))
  case $title in
    "$id"*) ;;
    *) problem "$rel: test title does not begin with the case id \"$id\": \"$title\"" ;;
  esac
done < "$tmp/titles"

# --- rule 2 — no .isVisible().catch(...) on an un-narrowed locator ------------------------------

scanned=0
while IFS= read -r f; do
  scanned=$((scanned + 1))
  awk '
    {
      line = $0
      if (match(line, /\.isVisible\(\)[ \t]*\.catch\(/)) {
        before = substr(line, 1, RSTART - 1)
        stripped = before
        sub(/^[ \t]+/, "", stripped)
        if (index(before, "//") == 0) {
          receiver = (stripped == "") ? prev : before
          if (receiver !~ /\.(first|last|nth)\([^()]*\)[ \t]*$/) {
            tail = receiver
            gsub(/[ \t]+/, " ", tail)
            sub(/^ /, "", tail)
            if (length(tail) > 60) tail = substr(tail, length(tail) - 59)
            printf "%s:%d: `.isVisible().catch(...)` on an un-narrowed locator (`%s`) — isVisible() throws on a strict-mode multi-match and the catch reads that throw as an absence. Narrow the receiver with .first(), .last(), or .nth(n) first.\n", FILENAME, FNR, tail
          }
        }
      }
      if ($0 ~ /[^ \t]/) prev = $0
    }
  ' "$f" >> "$tmp/isvisible"
done < <(find "$cases_abs" -type d -name fixtures -prune -o -type f \
  \( -name '*.ts' -o -name '*.tsx' -o -name '*.js' -o -name '*.jsx' -o -name '*.mjs' -o -name '*.cjs' \) \
  -print | sort)

if [ -s "$tmp/isvisible" ]; then
  while IFS= read -r p; do
    problem "${p#"$cases_abs"/}"
  done < "$tmp/isvisible"
fi

# --- report -------------------------------------------------------------------------------------

if [ ${#problems[@]} -gt 0 ]; then
  printf 'check-cases-playwright FAILED (%d problem(s)):\n' "${#problems[@]}"
  for p in "${problems[@]}"; do
    printf '  - %s\n' "$p"
  done
  exit 1
fi

printf 'check-cases-playwright OK — %d test title(s), %d spec file(s) scanned under %s.\n' \
  "$checked" "$scanned" "$cases_dir"
exit 0
