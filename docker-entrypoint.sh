#!/bin/sh
set -e

for dir in /config /logs /storage; do
  mkdir -p "$dir"
  if ! su-exec chat test -w "$dir"; then
    chown -R chat:chat "$dir" 2>/dev/null || true
  fi
done

exec su-exec chat "$@"
