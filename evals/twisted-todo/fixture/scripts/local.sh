#!/usr/bin/env bash
#
# The local environment for Forfeit: `start` brings the server up, and `update` makes the local
# environment match the checkout. codefall-refresh runs them in that order; anyone can run either
# from a terminal. `stop` is a convenience for people and for the eval's teardown.
#
# Both are safe to run at any time and cheap when nothing changed. Neither drops, resets, or
# deletes anything. A step that fails says what to do about it on stderr.
#
# Usage: scripts/local.sh start | update | stop

set -euo pipefail

cd "$(dirname "$0")/.."

PORT=${PORT:-3000}
BASE_URL=${BASE_URL:-http://localhost:$PORT}
PID_FILE=data/server.pid
LOG_FILE=data/server.log

healthy() { curl -fsS "$BASE_URL/health" >/dev/null 2>&1; }

start() {
  mkdir -p data
  if healthy; then
    echo "server already up at $BASE_URL"
    return 0
  fi
  nohup node src/main.js >"$LOG_FILE" 2>&1 &
  echo $! >"$PID_FILE"
  for _ in $(seq 1 20); do
    if healthy; then
      echo "server up at $BASE_URL (pid $(cat "$PID_FILE"))"
      return 0
    fi
    sleep 0.5
  done
  echo "server did not answer at $BASE_URL/health within 10s; see $LOG_FILE" >&2
  return 1
}

update() {
  # Runtime: Node 24 or newer, for node:sqlite.
  node -e 'const [maj] = process.versions.node.split("."); if (Number(maj) < 24) { console.error(`Node ${process.versions.node} is too old; Forfeit needs Node 24 or newer.`); process.exit(1) }'

  # Dependencies: reinstall only when the manifest or the lockfile changed.
  mkdir -p node_modules
  stamp=$(cat package.json package-lock.json 2>/dev/null | shasum | cut -d' ' -f1)
  if [ "$stamp" != "$(cat node_modules/.codefall-stamp 2>/dev/null || true)" ]; then
    npm install --no-audit --no-fund
    echo "$stamp" >node_modules/.codefall-stamp
  fi

  # Browsers for the Playwright runner. Exits 0 quickly when they are already present.
  npx playwright install chromium

  # Migrations: applies what is pending, prints "nothing to do" otherwise.
  node src/migrate.js
}

stop() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    kill "$(cat "$PID_FILE")" && rm -f "$PID_FILE"
    echo "server stopped"
  else
    echo "server not running"
  fi
}

case ${1:-} in
  start) start ;;
  update) update ;;
  stop) stop ;;
  *)
    echo "usage: $0 start | update | stop" >&2
    exit 2
    ;;
esac
