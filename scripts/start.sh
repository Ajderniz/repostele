#!/usr/bin/env bash
# Starts the staff and user servers in the background from bin/.
# Usage: scripts/start.sh   (ports: STAFF_PORT=8081 USER_PORT=8080)
#
# Only the staff binary creates bin/data.db, so staff always starts first.
# When the database does not exist yet, the user server waits until staff
# is listening (schema created) before it starts.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin"
STAFF_PORT="${STAFF_PORT:-8081}"
USER_PORT="${USER_PORT:-8080}"
WAIT_SECS=15

running() {
  local pidfile="$BIN/$1.pid"
  [[ -f "$pidfile" ]] && kill -0 "$(cat "$pidfile")" 2>/dev/null
}

listening() {
  (: </dev/tcp/127.0.0.1/"$1") 2>/dev/null
}

start() {
  local name="$1" port="$2"
  if running "$name"; then
    echo "$name already running (pid $(cat "$BIN/$name.pid"))"
    return
  fi
  if [[ ! -x "$BIN/$name" ]]; then
    echo "missing $BIN/$name; run 'make all' first" >&2
    exit 1
  fi
  mkdir -p "$BIN/log"
  # The servers write their own log files; stderr keeps startup failures.
  (cd "$BIN"
   nohup "./$name" -port "$port" >/dev/null 2>>"log/$name.err" &
   echo $! >"$name.pid")
  echo "$name started on port $port (pid $(cat "$BIN/$name.pid"))"
}

wait_listening() {
  local name="$1" port="$2"
  for ((i = 0; i < WAIT_SECS * 4; i++)); do
    listening "$port" && return
    if ! running "$name"; then
      echo "$name exited; see $BIN/log/$name.err" >&2
      exit 1
    fi
    sleep 0.25
  done
  echo "$name not listening on port $port after ${WAIT_SECS}s" >&2
  exit 1
}

fresh_db=0
[[ -f "$BIN/data.db" ]] || fresh_db=1

start staff "$STAFF_PORT"
if ((fresh_db)); then
  echo "no database yet; waiting for staff to create it"
  wait_listening staff "$STAFF_PORT"
fi
start user "$USER_PORT"
