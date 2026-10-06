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
START_TIMEOUT=${START_TIMEOUT:-10}

# Each check gives up after 2s, so a process that holds the port and never answers cannot stall it.
healthy() { curl -fsS --max-time 2 "$BASE_URL/health" >/dev/null 2>&1; }

start() {
  mkdir -p data
  if healthy; then
    echo "server already up at $BASE_URL"
    return 0
  fi
  nohup node src/main.js >"$LOG_FILE" 2>&1 &
  pid=$!
  echo "$pid" >"$PID_FILE"
  deadline=$((SECONDS + START_TIMEOUT))
  while [ "$SECONDS" -lt "$deadline" ]; do
    if healthy; then
      echo "server up at $BASE_URL (pid $pid)"
      return 0
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      rm -f "$PID_FILE"
      echo "server exited before answering at $BASE_URL/health; port $PORT may be in use by another process (lsof -i :$PORT shows it); see $LOG_FILE" >&2
      return 1
    fi
    sleep 0.5
  done
  echo "server did not answer at $BASE_URL/health within ${START_TIMEOUT}s; port $PORT may be in use by another process (lsof -i :$PORT shows it); see $LOG_FILE" >&2
  return 1
}

# The manifest and, when there is one, the lockfile, hashed together.
deps_stamp() {
  {
    cat package.json
    if [ -f package-lock.json ]; then cat package-lock.json; fi
  } | shasum | cut -d' ' -f1
}

update() {
  # Runtime: Node 24 or newer, for node:sqlite.
  node -e 'const [maj] = process.versions.node.split("."); if (Number(maj) < 24) { console.error(`Node ${process.versions.node} is too old; Forfeit needs Node 24 or newer.`); process.exit(1) }'

  # Dependencies: reinstall only when the manifest or the lockfile changed. Without a lockfile the
  # stamp covers the manifest alone, and npm install writes the lockfile; the stamp is taken after
  # the install, so it covers the lockfile the install wrote.
  mkdir -p node_modules
  if [ ! -f package-lock.json ]; then
    echo "no package-lock.json; npm install will write one"
  fi
  if [ "$(deps_stamp)" != "$(cat node_modules/.codefall-stamp 2>/dev/null || true)" ]; then
    npm install --no-audit --no-fund
    deps_stamp >node_modules/.codefall-stamp
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
