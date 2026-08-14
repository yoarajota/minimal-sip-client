#!/usr/bin/env bash
# One command: brings up Asterisk 20 + the component integration test, runs
# the scenario suite against the real PBX, and tears everything down.
# Exit code is the test's.
set -euo pipefail
cd "$(dirname "$0")"
trap 'docker compose down --remove-orphans >/dev/null 2>&1 || true' EXIT
docker compose up --build --abort-on-container-exit --exit-code-from client
