#!/usr/bin/env bash
# The H-001 benchmark, one command. Three legs:
#   1. concept:  the minimal client completes the scenario suite (register, two-way
#                RTP call, hold/resume, teardown) against the real pinned Asterisk 20
#                5 times; media counts and wall time are reported with variance.
#   2. baseline: the PJSIP 2.17 incumbent (pjsua, built from source, bench/baseline/)
#                completes the same suite against the same Asterisk 5 times.
#   3. fault:    the Asterisk container is killed mid-call; the client must surface
#                the transaction timeout cleanly (R-003 mitigation).
# Exit 0 only if every run of every leg passes.
#
# Environment declaration (G5): Docker Desktop 29.6.1 daemon on Windows (WSL2
# backend), docker compose v5.3.0; images andrius/asterisk:20.7-cert11_debian-trixie,
# golang:1.22-alpine, minimal-sip-baseline:pjsua-2.17 (built from pjproject tag 2.17).
# The client and PBX share one compose network; no host ports are published.
# Nothing else ran on the host during measurement.
set -euo pipefail
cd "$(dirname "$0")/.."
DOCKER="${DOCKER:-docker}"
RUNS="${RUNS:-5}"

trap 'docker compose down --remove-orphans >/dev/null 2>&1 || true' EXIT

echo "== leg 1/3: concept suite, ${RUNS} runs =="
docker compose up -d asterisk >/dev/null
concept_runs=()
for i in $(seq 1 "$RUNS"); do
  out=$(docker compose run --rm client \
        sh -c "go test -tags integration -v ./internal/sip/ -run TestSuiteIntegration" 2>&1)
  if ! echo "$out" | grep -q -- "--- PASS: TestSuiteIntegration"; then
    echo "FAIL concept run $i"; echo "$out" | tail -20; exit 1
  fi
  # extract media + timing summary lines
  summary=$(echo "$out" | grep -oE "(active sent [0-9]+ / recv [0-9]+|held recv [0-9]+|resumed [0-9]+ / [0-9]+|PASS.*\([0-9.]+s\))" | tr '\n' ' ')
  elapsed=$(echo "$out" | grep -oE "TestSuiteIntegration \([0-9.]+s\)" | head -1)
  concept_runs+=("$elapsed $summary")
  echo "  run $i: $elapsed"
done

echo "== leg 2/3: PJSIP 2.17 baseline (pjsua), ${RUNS} runs =="
docker build -q -t minimal-sip-baseline:pjsua-2.17 bench/baseline >/dev/null
baseline_runs=()
for i in $(seq 1 "$RUNS"); do
  out=$(docker compose run --rm baseline python /app/baseline.py 2>&1)
  # the PASS line can merge onto a pjsua log line (no leading newline)
  if ! echo "$out" | grep -q "PASS register="; then
    echo "FAIL baseline run $i"; echo "$out" | tail -20; exit 1
  fi
  p=$(echo "$out" | grep -o "PASS register=.*bye=ok" | tail -1)
  baseline_runs+=("$p")
  echo "  run $i: $p"
done

echo "== leg 3/3: fault injection — kill the PBX mid-call =="
cid=$(docker compose run -d client \
      sh -c "go test -tags integration -v ./internal/sip/ -run TestKilledPBXMidCall" 2>/dev/null)
# wait for the marker
for _ in $(seq 1 120); do
  if docker logs "$cid" 2>&1 | grep -q READY_FOR_KILL; then break; fi
  sleep 1
done
if ! docker logs "$cid" 2>&1 | grep -q READY_FOR_KILL; then
  echo "FAIL fault leg: client never reached the kill marker"; docker logs "$cid" 2>&1 || true; exit 1
fi
docker kill "$(docker compose ps -q asterisk)" >/dev/null 2>&1 || true
fault_out=""
for _ in $(seq 1 150); do
  code=$(docker wait "$cid" 2>/dev/null || echo "")
  if [ -n "$code" ]; then fault_out=$(docker logs "$cid" 2>&1 || true); break; fi
  sleep 1
done
if ! echo "$fault_out" | grep -q "PASS killed-pbx"; then
  echo "FAIL fault leg"; echo "$fault_out" | tail -20; exit 1
fi
echo "  $(echo "$fault_out" | grep 'PASS killed-pbx')"

echo
echo "== summary =="
echo "concept (${RUNS} runs):"
for r in "${concept_runs[@]}"; do echo "  $r"; done
echo "baseline (${RUNS} runs):"
for r in "${baseline_runs[@]}"; do echo "  $r"; done
echo "ALL LEGS PASS"
