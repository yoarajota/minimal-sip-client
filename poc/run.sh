#!/usr/bin/env bash
# One command: brings up Asterisk 20 + the minimal client, runs the scenario
# suite, and tears everything down. Exit code is the client's.
set -euo pipefail
cd "$(dirname "$0")"
trap 'docker compose down --remove-orphans >/dev/null 2>&1 || true' EXIT
# Start from a clean stack: the legs share container names and the compose network, and an
# aborted run leaves both behind, which makes `up` fail for a reason unrelated to the claim.
# The wait is the point — `down` returns before the network is released, and `up` immediately
# after it races that teardown and exits non-zero (observed: silent exit 1).
docker compose down --remove-orphans >/dev/null 2>&1 || true
for _ in $(seq 1 20); do
  docker network inspect minimal-sip-client_default >/dev/null 2>&1 || break
  sleep 0.5
done
# Only the services this leg needs. The project also defines the PJSIP baseline service, and
# starting it here would let an unrelated environment limitation (its media assertion) abort the
# client's suite through --abort-on-container-exit.
docker compose up --build --abort-on-container-exit --exit-code-from client asterisk client
