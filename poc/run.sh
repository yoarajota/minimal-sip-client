#!/usr/bin/env bash
# One command: brings up Asterisk 20 + the minimal client, runs the scenario
# suite, and tears everything down. Exit code is the client's.
set -euo pipefail
cd "$(dirname "$0")"
trap 'docker compose down --remove-orphans >/dev/null 2>&1 || true' EXIT
docker compose up --build --abort-on-container-exit --exit-code-from client
