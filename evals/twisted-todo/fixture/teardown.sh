#!/usr/bin/env bash
#
# Delete the throwaway repository setup.sh created, stop the fixture's server, and, with --purge,
# remove the project checkout too. The run transcripts under evals/twisted-todo/runs/ are kept.
#
# Usage: fixture/teardown.sh [--purge]
# Reads: evals/twisted-todo/.throwaway.env (REPO, PROJECT_DIR)

set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
eval_root=$(cd "$here/.." && pwd)
env_file="$eval_root/.throwaway.env"

[ -f "$env_file" ] || { echo "no $env_file; nothing to tear down" >&2; exit 1; }
# shellcheck disable=SC1090
source "$env_file"

if [ -d "$PROJECT_DIR" ]; then
  (cd "$PROJECT_DIR" && [ -x scripts/local.sh ] && scripts/local.sh stop) || true
fi

scopes=$(gh api user -i 2>/dev/null | awk -F': ' 'tolower($1)=="x-oauth-scopes"{print $2}')
case " ${scopes//,/ } " in *" delete_repo "*) ;; *)
  echo "gh token lacks the 'delete_repo' scope, which deleting the repository needs. Run: gh auth refresh -s delete_repo, then run this script again." >&2
  exit 1 ;;
esac

if gh repo view "$REPO" >/dev/null 2>&1; then
  gh repo delete "$REPO" --yes
  echo "deleted https://github.com/$REPO"
else
  echo "repository $REPO is already gone"
fi

if [ "${1:-}" = "--purge" ]; then
  rm -rf "$PROJECT_DIR"
  echo "removed $PROJECT_DIR"
else
  echo "kept $PROJECT_DIR (pass --purge to remove it)"
fi

rm -f "$env_file"
