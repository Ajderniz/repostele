#!/usr/bin/env bash
# Stops the servers started by scripts/start.sh.
# Usage: scripts/stop.sh

set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin"
WAIT_SECS=5

stop() {
  local name="$1" pidfile="$BIN/$1.pid" pid
  if [[ ! -f "$pidfile" ]]; then
    echo "$name not running"
    return
  fi
  pid="$(cat "$pidfile")"
  if ! kill -0 "$pid" 2>/dev/null; then
    echo "$name not running (stale pidfile)"
    rm -f "$pidfile"
    return
  fi
  kill "$pid"
  for ((i = 0; i < WAIT_SECS * 4; i++)); do
    kill -0 "$pid" 2>/dev/null || break
    sleep 0.25
  done
  if kill -0 "$pid" 2>/dev/null; then
    echo "$name did not stop in ${WAIT_SECS}s; killing"
    kill -9 "$pid"
  fi
  rm -f "$pidfile"
  echo "$name stopped (pid $pid)"
}

stop user
stop staff
