#!/usr/bin/env bash
# Raise file descriptor limit to reduce EMFILE / "too many open files" from Watchpack, then start Next.js dev server.
ulimit -n 10240 2>/dev/null || true
exec npm run dev:next
