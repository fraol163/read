#!/usr/bin/env bash
# READ Framework - Python Service Lifecycle Verification Demo
# Resource-conscious, Evidence-driven AI Development
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

READ_ENGINE="$DIR/../../bin/read-engine"
if [ ! -x "$READ_ENGINE" ]; then
    echo "[ERROR] read-engine binary not found at $READ_ENGINE" >&2
    exit 1
fi

echo "================================================================"
echo "READ Lifecycle Demo: Python Microservice Rate Limiter"
echo "================================================================"

# 1. Initialize temporary Git environment if not already in one
ORIG_DIR="$PWD"
DEMO_DIR=$(mktemp -d /tmp/read-py-demo-XXXXXX)
trap 'rm -rf "$DEMO_DIR"' EXIT

cp -r "$DIR"/* "$DEMO_DIR"/
cd "$DEMO_DIR"

git init -q
git config user.email "ci@read-framework.local"
git config user.name "READ CI Worker"
git add .
git commit -q -m "Initial commit of python rate limiter service"

BASE_COMMIT=$(git rev-parse HEAD)
CONTRACT_FILE="contract.json"
CONTRACT_HASH=$(sha256sum "$CONTRACT_FILE" | awk '{print $1}')

echo "[STEP 1] Initializing READ CAS state engine..."
"$READ_ENGINE" init

echo "[STEP 2] Defining task contract in STATE.json..."
"$READ_ENGINE" define "TASK-PY-RATELIMITER-001" "$CONTRACT_HASH" "$BASE_COMMIT"

echo "[STEP 3] Transitioning to Build stage..."
"$READ_ENGINE" build

echo "[STEP 4] Deterministically extracting C0 AST Context (\$0.00 cost)..."
"$READ_ENGINE" ast-context src/rate_limiter.py

echo "[STEP 5] Simulating code implementation & staging diff..."
# Introduce an intentional addition to test diff hashing
echo "" >> src/rate_limiter.py
echo "# Verified by READ Engine" >> src/rate_limiter.py
git add src/rate_limiter.py

DIFF_HASH=$("$READ_ENGINE" diff-hash)
echo "Staged Diff Hash: $DIFF_HASH"

echo "[STEP 6] Executing unit test suite..."
TEST_OUTPUT=$(pytest tests/ -v)
echo "$TEST_OUTPUT" | head -n 15

echo "[STEP 7] Generating AROP bounded mutants (<4.5s)..."
"$READ_ENGINE" mutate --staged

echo "[STEP 8] Recording verification evidence..."
"$READ_ENGINE" verify "$DIFF_HASH" "$TEST_OUTPUT"

echo "[STEP 9] Sealing task via CAS Handoff..."
"$READ_ENGINE" handoff

echo "[STEP 10] Testing zero-token crash recovery..."
"$READ_ENGINE" recover

echo "================================================================"
echo "DEMO COMPLETE: Python service successfully verified and sealed."
echo "================================================================"
