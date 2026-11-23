#!/usr/bin/env bash
# READ Framework - Ruby Service Lifecycle Verification Demo
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
echo "READ Lifecycle Demo: Ruby Authentication Service"
echo "================================================================"

ORIG_DIR="$PWD"
DEMO_DIR=$(mktemp -d /tmp/read-ruby-demo-XXXXXX)
trap 'rm -rf "$DEMO_DIR"' EXIT

cp -r "$DIR"/* "$DEMO_DIR"/
cd "$DEMO_DIR"

git init -q
git config user.email "ci@read-framework.local"
git config user.name "READ CI Worker"
git add .
git commit -q -m "Initial commit of ruby auth service"

BASE_COMMIT=$(git rev-parse HEAD)
CONTRACT_FILE="contract.json"
CONTRACT_HASH=$(sha256sum "$CONTRACT_FILE" | awk '{print $1}')

echo "[STEP 1] Initializing READ CAS state engine..."
"$READ_ENGINE" init

echo "[STEP 2] Defining task contract in STATE.json..."
"$READ_ENGINE" define "TASK-RUBY-AUTH-001" "$CONTRACT_HASH" "$BASE_COMMIT"

echo "[STEP 3] Transitioning to Build stage..."
"$READ_ENGINE" build

echo "[STEP 4] Deterministically extracting C0 AST Context (\$0.00 cost)..."
"$READ_ENGINE" ast-context lib/auth_service.rb

echo "[STEP 5] Simulating code implementation & staging diff..."
echo "" >> lib/auth_service.rb
echo "# Verified by READ Engine" >> lib/auth_service.rb
git add lib/auth_service.rb

DIFF_HASH=$("$READ_ENGINE" diff-hash)
echo "Staged Diff Hash: $DIFF_HASH"

echo "[STEP 6] Executing Ruby test suite..."
TEST_OUTPUT=$(bash test.sh)
echo "$TEST_OUTPUT"

echo "[STEP 7] Generating AROP bounded mutants (<4.5s)..."
"$READ_ENGINE" mutate

echo "[STEP 8] Recording verification evidence..."
"$READ_ENGINE" verify "$DIFF_HASH" "$TEST_OUTPUT"

echo "[STEP 9] Sealing task via CAS Handoff..."
"$READ_ENGINE" handoff

echo "[STEP 10] Testing zero-token crash recovery..."
"$READ_ENGINE" recover

echo "================================================================"
echo "DEMO COMPLETE: Ruby service successfully verified and sealed."
echo "================================================================"
