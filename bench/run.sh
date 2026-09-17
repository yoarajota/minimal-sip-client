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
# Per-run data. The default is scratch: re-running the benchmark must not overwrite the
# committed record the ledger's checksums point at. To publish a run, point DATA at
# evidence-data/ explicitly, as the ledger's entries did.
DATA="${DATA:-bench/out/bench-runs.json}"
# Which legs to run: concept, baseline, fault (comma-separated, or "all").
LEGS="${LEGS:-concept,baseline,fault}"
want() { [[ ",$LEGS," == *",$1,"* ]] || [[ "$LEGS" == "all" ]]; }

trap 'docker compose down --remove-orphans >/dev/null 2>&1 || true' EXIT

# Declared here, not inside the legs: a skipped leg must leave an empty collector rather than
# an unset variable, which `set -u` turns into an error while writing the data file.
concept_runs=(); concept_seconds=(); baseline_runs=(); fault_out=""

if want concept; then
echo "== leg 1/3: concept suite, ${RUNS} runs =="
docker compose up -d asterisk >/dev/null
for i in $(seq 1 "$RUNS"); do
  # `if ! out=$(...)` rather than a bare assignment: under `set -e` a failing command
  # substitution exits the script immediately, so the failure branch below never ran and a
  # failing leg died silently at its own header.
  if ! out=$(docker compose run --rm client \
        sh -c "go test -tags integration -v ./internal/sip/ -run TestSuiteIntegration" 2>&1); then
    echo "FAIL concept run $i (client exited non-zero)"; echo "$out" | tail -20; exit 1
  fi
  if ! echo "$out" | grep -q -- "--- PASS: TestSuiteIntegration"; then
    echo "FAIL concept run $i"; echo "$out" | tail -20; exit 1
  fi
  # extract media + timing summary lines
  summary=$(echo "$out" | grep -oE "(active sent [0-9]+ / recv [0-9]+|held recv [0-9]+|resumed [0-9]+ / [0-9]+|PASS.*\([0-9.]+s\))" | tr '\n' ' ')
  elapsed=$(echo "$out" | grep -oE "TestSuiteIntegration \([0-9.]+s\)" | head -1)
  concept_runs+=("$elapsed $summary")
  concept_seconds+=("$(echo "$elapsed" | grep -oE "[0-9.]+" | head -1)")
  echo "  run $i: $elapsed"
done

fi

if want baseline; then
echo "== leg 2/3: PJSIP 2.17 baseline (pjsua), ${RUNS} runs =="
docker build -q -t minimal-sip-baseline:pjsua-2.17 bench/baseline >/dev/null
for i in $(seq 1 "$RUNS"); do
  if ! out=$(docker compose run --rm baseline python /app/baseline.py 2>&1); then
    echo "FAIL baseline run $i (baseline exited non-zero)"; echo "$out" | tail -20; exit 1
  fi
  # the PASS line can merge onto a pjsua log line (no leading newline)
  if ! echo "$out" | grep -q "PASS register="; then
    echo "FAIL baseline run $i"; echo "$out" | tail -20; exit 1
  fi
  p=$(echo "$out" | grep -o "PASS register=.*bye=ok" | tail -1)
  baseline_runs+=("$p")
  echo "  run $i: $p"
done

fi

if want fault; then
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
for _ in $(seq 1 150); do
  code=$(docker wait "$cid" 2>/dev/null || echo "")
  if [ -n "$code" ]; then fault_out=$(docker logs "$cid" 2>&1 || true); break; fi
  sleep 1
done
if ! echo "$fault_out" | grep -q "PASS killed-pbx"; then
  echo "FAIL fault leg"; echo "$fault_out" | tail -20; exit 1
fi
echo "  $(echo "$fault_out" | grep 'PASS killed-pbx')"

fi

echo
echo "== summary =="
if want concept; then
  echo "concept (${RUNS} runs):"
  for r in "${concept_runs[@]}"; do echo "  $r"; done
fi
if want baseline; then
  echo "baseline (${RUNS} runs):"
  for r in "${baseline_runs[@]}"; do echo "  $r"; done
fi
echo "ALL LEGS PASS"

mkdir -p "$(dirname "$DATA")"
escape() { printf '%s' "$1" | sed 's/"/\\"/g'; }
{
  printf '{\n'
  printf '  "runs_per_leg": %s,\n' "$RUNS"
  printf '  "legs": "%s",\n' "$LEGS"
  printf '  "concept": {\n'
  printf '    "ran": %s,\n' "$(want concept && echo true || echo false)"
  printf '    "runs_passed": %s,\n' "${#concept_seconds[@]}"
  printf '    "wall_time_s": [%s],\n' "$(IFS=,; echo "${concept_seconds[*]}")"
  printf '    "summary": "%s"\n' "$(escape "${concept_runs[*]}")"
  printf '  },\n'
  printf '  "baseline": {\n'
  printf '    "ran": %s,\n' "$(want baseline && echo true || echo false)"
  printf '    "runs_passed": %s,\n' "${#baseline_runs[@]}"
  printf '    "pass_lines": ["%s"]\n' "$(escape "$(IFS='","'; echo "${baseline_runs[*]}")")"
  printf '  },\n'
  printf '  "fault": {\n'
  printf '    "ran": %s,\n' "$(want fault && echo true || echo false)"
  printf '    "passed": %s,\n' "$(want fault && echo true || echo false)"
  printf '    "line": "%s"\n' "$(escape "$(echo "$fault_out" | grep 'PASS killed-pbx' | head -1)")"
  printf '  },\n'
  printf '  "all_legs_pass": true\n'
  printf '}\n'
} > "$DATA"
echo "wrote $DATA"
