#!/usr/bin/env bash
#
# Hold extensions/shared/workflow.md to docs/workflow.md.
#
# docs/workflow.md is the description of how codefall works, edited in this
# repository. extensions/shared/workflow.md is the copy `codefall project init` installs
# into a project as .codefall/shared/workflow.md, which the Codefall section of
# the project's AGENTS.md points at. The copy carries its own opening, and then
# the sections a project needs, byte for byte from the source. This script
# reports where the two differ; it fails only when asked to, and rewrites the
# copy only when asked to.
#
# Usage: workflow-sync.sh [--strict] [--write]
#
#   --strict    exit 1 when a shared section differs, for CI
#   --write     replace the copy's shared sections with the source's; the copy's
#               own opening, everything before its first shared heading, is kept
#
# Shared sections, by heading, in the order the copy holds them
#   ## The chain
#   ## Keeping the project current
#   ## Who is authoritative for what
#
# A section runs from its heading to the line before the next `## ` heading or
# the end of the file, with trailing blank lines dropped. Anything in the source
# outside those sections — what init puts in place — is the CLI's business and is
# not copied.
#
# Exit codes
#   0   report printed; with --strict, every shared section is identical
#   1   --strict and at least one section differs, or --write could not write
#   2   usage error, a file missing, or a section missing from the source

set -uo pipefail

strict=0
write=0
for arg in "$@"; do
  case "$arg" in
    --strict) strict=1 ;;
    --write) write=1 ;;
    -h|--help) sed -n '2,33p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$arg" >&2; exit 2 ;;
  esac
done

here="$(cd "$(dirname "$0")" && pwd -P)"
ext_root="$(cd "$here/.." && pwd -P)"
source="$ext_root/../docs/workflow.md"
copy="$ext_root/shared/workflow.md"

headings=(
  "## The chain"
  "## Keeping the project current"
  "## Who is authoritative for what"
)

for f in "$source" "$copy"; do
  if [ ! -f "$f" ]; then
    printf 'missing: %s\n' "$f" >&2
    exit 2
  fi
done

# The section under a heading: the heading line through the line before the next
# `## ` heading, trailing blank lines dropped. Empty when the heading is absent.
section() {
  awk -v want="$2" '
    $0 == want { on = 1 }
    on && $0 != want && /^## / { exit }
    on { print }
  ' "$1" | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}'
}

# What comes before the first shared heading in the copy: its own opening.
opening() {
  awk -v first="${headings[0]}" '$0 == first { exit } { print }' "$1" \
    | sed -e :a -e '/^\n*$/{$d;N;ba' -e '}'
}

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

differs=0
for h in "${headings[@]}"; do
  section "$source" "$h" > "$tmp/source"
  if [ ! -s "$tmp/source" ]; then
    printf 'missing from the source: %s\n' "$h" >&2
    exit 2
  fi

  section "$copy" "$h" > "$tmp/copy"
  if cmp -s "$tmp/source" "$tmp/copy"; then
    printf 'ok       %s\n' "$h"
  else
    differs=1
    printf 'differs  %s\n' "$h"
    if [ "$write" -eq 0 ]; then
      diff -u --label "docs/workflow.md" --label "extensions/shared/workflow.md" \
        "$tmp/source" "$tmp/copy" | sed 's/^/    /'
    fi
  fi
done

if [ "$write" -eq 1 ] && [ "$differs" -eq 1 ]; then
  {
    opening "$copy"
    for h in "${headings[@]}"; do
      printf '\n'
      section "$source" "$h"
    done
  } > "$tmp/next" || exit 1
  cp "$tmp/next" "$copy" || exit 1
  printf 'wrote    extensions/shared/workflow.md\n'
  differs=0
fi

if [ "$strict" -eq 1 ] && [ "$differs" -eq 1 ]; then
  printf '\nextensions/shared/workflow.md has drifted from docs/workflow.md; run %s --write\n' \
    "extensions/scripts/workflow-sync.sh" >&2
  exit 1
fi

exit 0
