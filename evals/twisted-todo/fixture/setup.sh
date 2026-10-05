#!/usr/bin/env bash
#
# Create the throwaway project for one run of the twisted-todo eval.
#
# What it does, in order:
#   1. checks the tools the run needs (gh with the right scopes, the gh-stack extension, bd, jq,
#      node 24, go);
#   2. builds the codefall binary from this checkout of the codefall repository;
#   3. copies the fixture into a fresh directory, makes it a git repository on `main`, and creates a
#      private GitHub repository for it (`gh repo create --private`), pushed;
#   4. runs `codefall init` for Claude Code with the GitHub tracker, which also runs `bd init`;
#   5. sets the persona to `product-manager`;
#   6. installs the document-landing Action by copying the template codefall ships
#      (extensions/skills/codefall-equip/templates/codefall-land-documents.yml) to
#      .github/workflows/, and creates the `land` label the Action listens for, so the throwaway
#      lands documents the way a project that ran `/codefall-equip landing` does; a throwaway has
#      no branch rule, so the default token merges;
#   7. commits, pushes, adopts the git origin as the Beads Dolt remote, and writes the paths the
#      driver reads into evals/twisted-todo/.throwaway.env.
#
# It declares nothing under `local` or `test.runners`: the chain's first two sessions are
# `/codefall-equip local` and `/codefall-equip test`, and they are under test too.
#
# Usage: fixture/setup.sh [repo-name]      default: twisted-todo-<date>-<time>
# Env:   EVAL_WORK_DIR  where the project checkout goes (default: ~/tmp/twisted-todo)
#
# Nothing here touches the codefall repository except `go build` into the work directory.

set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
eval_root=$(cd "$here/.." && pwd)
codefall_repo=$(cd "$eval_root/../.." && pwd)
action_template="$codefall_repo/extensions/skills/codefall-equip/templates/codefall-land-documents.yml"

name=${1:-twisted-todo-$(date +%Y%m%d-%H%M%S)}
work_root=${EVAL_WORK_DIR:-$HOME/tmp/twisted-todo}
project="$work_root/$name"
bin_dir="$work_root/bin"

say() { printf '\n== %s\n' "$*"; }
need() { command -v "$1" >/dev/null 2>&1 || { echo "missing tool: $1 ($2)" >&2; exit 1; }; }

say "Checking tools"
need gh "brew install gh"
need bd "brew install beads"
need jq "brew install jq"
need node "Node 24 or newer, for node:sqlite"
need go "to build codefall from this branch"
need curl "used by scripts/local.sh"
node -e 'const [maj] = process.versions.node.split("."); if (Number(maj) < 24) { console.error(`Node ${process.versions.node} is too old; the fixture needs Node 24 or newer.`); process.exit(1) }'
gh auth status >/dev/null 2>&1 || { echo "gh is not logged in; run gh auth login" >&2; exit 1; }
scopes=$(gh api user -i 2>/dev/null | awk -F': ' 'tolower($1)=="x-oauth-scopes"{print $2}')
for scope in repo workflow project delete_repo; do
  case " ${scopes//,/ } " in *" $scope "*) ;; *)
    echo "gh token lacks the '$scope' scope (has: $scopes). Run: gh auth refresh -s repo -s workflow -s project -s read:project -s delete_repo" >&2
    exit 1 ;;
  esac
done
gh extension list 2>/dev/null | grep -q 'github/gh-stack' || gh extension install github/gh-stack
owner=$(gh api user -q .login)
[ -f "$action_template" ] || { echo "missing $action_template; is this checkout on a branch that ships the landing track?" >&2; exit 1; }

say "Building codefall from $codefall_repo ($(git -C "$codefall_repo" branch --show-current))"
mkdir -p "$bin_dir"
(cd "$codefall_repo" && go build -o "$bin_dir/codefall" ./cli/cmd/codefall)
codefall="$bin_dir/codefall"
"$codefall" --version

say "Copying the fixture to $project"
[ -e "$project" ] && { echo "$project already exists; pick another name or remove it" >&2; exit 1; }
mkdir -p "$project"
(cd "$here" && tar --exclude=setup.sh --exclude=teardown.sh --exclude=README.md -cf - .) | (cd "$project" && tar -xf -)
chmod +x "$project/scripts/local.sh"
cat >"$project/README.md" <<'EOF'
# Forfeit

A todo list where tasks come in pairs and finishing one forfeits the other. One person, one
laptop, no accounts. `AGENTS.md` has the stack and the rules for working here.

- `scripts/local.sh update` then `scripts/local.sh start`, and open http://localhost:3000.
- `npm test` runs the unit tests; `npm run e2e` runs the Playwright specs against the running server.
EOF

say "Creating the private repository $owner/$name"
cd "$project"
git init -q -b main
git add -A
git commit -qm "chore: fixture skeleton for the twisted-todo eval"
gh repo create "$owner/$name" --private --source=. --remote=origin --push \
  --description "Throwaway repository for a codefall eval; deleted after the run."
git remote set-head origin -a >/dev/null
gh repo edit "$owner/$name" --enable-squash-merge --delete-branch-on-merge --enable-merge-commit=false --enable-rebase-merge=false >/dev/null

say "Running codefall init (it runs bd init too)"
"$codefall" init --harness claude --tracker github --issues-repo "$owner/$name" --test-dir testing
bd config get issue_prefix

say "Setting the persona to product-manager"
"$codefall" config persona product-manager

say "Installing the document-landing Action from codefall's template"
mkdir -p .github/workflows
cp "$action_template" .github/workflows/codefall-land-documents.yml

say "Creating the land label the Action listens for, as /codefall-equip landing would"
gh label create land --description "merge this document pull request" --color 0E8A16 --repo "$owner/$name"

say "Committing the install and pushing"
git add -A
git commit -qm "chore: codefall init, persona, and the document-landing Action"
git push -q origin main

say "Adopting the git origin as the Beads Dolt remote"
bd dolt push --yes || echo "bd dolt push did not adopt a remote; the verbs will say so once and carry on" >&2

say "Doctor"
"$codefall" doctor || true

cat >"$eval_root/.throwaway.env" <<EOF
# Written by fixture/setup.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ). Read by the driver and teardown.sh.
PROJECT_DIR=$project
REPO=$owner/$name
CODEFALL_BIN=$codefall
EOF

say "Done"
echo "project:  $project"
echo "repo:     https://github.com/$owner/$name"
echo "env file: $eval_root/.throwaway.env"
echo "next:     cd $eval_root/driver && npm install && node run.ts"
