#!/usr/bin/env bash
set -euo pipefail

# Simple secret scanner for tracked files or provided file list.
# NOTE: This is a heuristic scan; use dedicated tooling for deeper coverage.

PATTERN='AKIA[0-9A-Z]{16}|ASIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{35}|gh[pousr]_[A-Za-z0-9]{36,}|xox[baprs]-[A-Za-z0-9-]{10,}|sk_(live|test)_[A-Za-z0-9]{16,}|sk-[A-Za-z0-9_-]{20,}|-----BEGIN (RSA|EC|OPENSSH|PRIVATE) KEY-----|DATABASE_URL=.*://[^[:space:]/:]+:[^[:space:]@]+@|(JWT_SECRET|API_KEY|CLIENT_SECRET|PASSWORD|TOKEN)=[A-Za-z0-9+/=._-]{16,}'

emit_scan_targets() {
  if [ "$#" -gt 0 ]; then
    printf '%s\0' "$@"
  else
    git ls-files -z
  fi
}

filter_scan_targets() {
  while IFS= read -r -d '' file; do
    case "$file" in
      *.example|*.sample|*.template|scripts/secret_scan.sh)
        continue
        ;;
      *)
        printf '%s\0' "$file"
        ;;
    esac
  done
}

# Build file list (null-delimited) from args or git ls-files, then scan once.
MATCHES=$(emit_scan_targets "$@" | filter_scan_targets | xargs -0 grep -nEI -I "$PATTERN" | grep -vE '\$\{[^}]+\}' || true)

if [ -n "$MATCHES" ]; then
  echo "Potential secrets detected (values redacted):" >&2
  echo "$MATCHES" | awk -F: '{print " - " $1 ":" $2}' >&2
  exit 1
fi

exit 0
