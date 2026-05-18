#!/bin/sh
set -e

# Bind-mount preserves host sources but node_modules lives in a Docker volume.
# Re-install when lockfile changes or a required package is missing.
STAMP="/app/node_modules/.package-lock.hash"
LOCK_HASH="$(sha256sum package-lock.json 2>/dev/null | cut -d' ' -f1 || echo none)"

if [ ! -d node_modules/next-intl ] || [ ! -f "$STAMP" ] || [ "$(cat "$STAMP" 2>/dev/null)" != "$LOCK_HASH" ]; then
  echo "[frontend] Installing dependencies (package-lock changed or deps missing)..."
  if [ -f package-lock.json ]; then
    npm ci
  else
    npm install
  fi
  echo "$LOCK_HASH" > "$STAMP"
fi

exec "$@"
