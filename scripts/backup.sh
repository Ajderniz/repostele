#!/usr/bin/env bash
# Backs up the database and uploaded images into bin/backup/<timestamp>/.
# Usage: scripts/backup.sh [dest-dir]   (default dest: bin/backup)
#
# Safe while the servers run: sqlite3's .backup takes a consistent copy,
# including what is still in data.db-wal (a plain cp of data.db is not).
# Restore: see README.md.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin"
DEST="${1:-$BIN/backup}/$(date +%Y%m%d-%H%M%S)"

if [[ ! -f "$BIN/data.db" ]]; then
  echo "no database at $BIN/data.db" >&2
  exit 1
fi
command -v sqlite3 >/dev/null || { echo "sqlite3 not installed" >&2; exit 1; }

mkdir -p "$DEST"
sqlite3 "$BIN/data.db" ".backup '$DEST/data.db'"
[[ "$(sqlite3 "$DEST/data.db" "PRAGMA integrity_check")" == ok ]] \
  || { echo "backup failed its integrity check: $DEST" >&2; exit 1; }
[[ -d "$BIN/dyn" ]] && cp -r "$BIN/dyn" "$DEST/dyn"

echo "backup in $DEST ($(du -sh "$DEST" | cut -f1))"
