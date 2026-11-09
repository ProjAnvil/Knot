#!/usr/bin/env bash
# Playwright webServer bootstrap for the Knot E2E suite.
#
# Playwright runs this script with cwd = frontend/ (the directory that holds
# playwright.config.ts). It builds the real production stack and then serves
# it in the foreground:
#   1. build the frontend bundle (frontend/dist)
#   2. (re)populate backend/internal/embedded/frontend_dist -- the directory
#      is untracked, so without this step the `embedfrontend`-tagged build
#      fails on a fresh checkout and serves a stale UI on dev machines
#   3. build the CLI binary with the embedded frontend into e2e/bin/knot-e2e
#   4. exec the CLI's hidden __serve subcommand on port 3999 with an
#      isolated SQLite database
#
# The script itself echoes nothing on success (chatty build tools have their
# stdout silenced; their stderr still surfaces if Playwright reports the
# webServer as failed).

set -euo pipefail

# Always run from frontend/ regardless of how the script was invoked.
cd "$(dirname "$0")/.."

# Isolated database directory: a fresh temp dir per server start by default.
# Pin E2E_DB_DIR externally to keep the database in a known location.
E2E_DB_DIR="${E2E_DB_DIR:-$(mktemp -d)}"
mkdir -p "$E2E_DB_DIR"

# Wipe stale database files so every fresh server start begins empty (SQLite
# WAL/SHM sidecars included). Redundant for the default temp dir, but vital
# when E2E_DB_DIR is pinned and a previous run left data behind.
rm -f "$E2E_DB_DIR"/knot-e2e.db "$E2E_DB_DIR"/knot-e2e.db-wal "$E2E_DB_DIR"/knot-e2e.db-shm

# 1. Frontend bundle.
bun run build >/dev/null

# 2. Populate the embedded frontend directory.
(
  cd ../backend
  rm -rf internal/embedded/frontend_dist
  mkdir -p internal/embedded/frontend_dist
  cp -r ../frontend/dist/* internal/embedded/frontend_dist/
) >/dev/null

# 3. CLI binary with the frontend embedded (embedfrontend build tag).
mkdir -p e2e/bin
(cd ../backend && go build -tags embedfrontend -o ../frontend/e2e/bin/knot-e2e ./cmd/cli)

# 4. Foreground server. `env -u PORT` because a preexisting PORT variable
#    would override KNOT_PORT inside serve.go.
exec env -u PORT \
  KNOT_PORT=3999 \
  KNOT_SQLITE_PATH="$E2E_DB_DIR/knot-e2e.db" \
  ./e2e/bin/knot-e2e __serve
