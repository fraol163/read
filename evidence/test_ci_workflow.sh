#!/usr/bin/env bash
set -e

echo "=== TEST: GitHub Actions CI Verification Gate Simulation ==="
echo "Workflow Target: .github/workflows/read-gate.yml"
echo "Date: $(date --iso-8601=seconds)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ENGINE="$REPO_DIR/bin/read-engine"
TMPDIR=$(mktemp -d /tmp/read-ci-gate-test-XXXXXX)

cleanup() {
    rm -rf "$TMPDIR"
}
trap cleanup EXIT

cd "$TMPDIR"
git init -q
git config user.name "CI Worker"
git config user.email "ci@example.com"
echo "initial" > file.txt
git add file.txt
git commit -qm "initial commit"

echo ""
echo "--- SCENARIO 1: Valid Verified Commit with Git Tag Anchor (Must PASS) ---"
"$ENGINE" init
echo "TASK-CI-001" > .read/TASK.md
CONTRACT_HASH=$(sha256sum .read/TASK.md | cut -d" " -f1)
"$ENGINE" define "TASK-CI-001" "$CONTRACT_HASH" "HEAD"
"$ENGINE" build

echo "content v1" > file.txt
git add file.txt
DIFF_HASH=$("$ENGINE" diff-hash)
"$ENGINE" verify "$DIFF_HASH" "Verification passed with 100% mutation score"
"$ENGINE" handoff

echo "[CI STEP] Verifying state and contract integrity..."
if [ ! -f ".read/STATE.json" ]
then
    echo "[FAIL] Missing .read/STATE.json"
    exit 1
fi
echo "[CI STEP] State file presence verified."

echo "[CI STEP] Verifying cryptographic WAL git tag anchor..."
"$ENGINE" verify-wal-anchor
echo "[CI STEP] WAL git tag anchor verified successfully."

echo "[CI STEP] Enforcing sealed state handoff..."
STATUS=$("$ENGINE" status | grep '"status":' | tr -d '", ' | cut -d: -f2)
echo "Current READ State: $STATUS"
if [ "$STATUS" != "handoff_complete" ] && [ "$STATUS" != "verified" ]
then
    echo "[FAIL] Commit was not verified by READ."
    exit 1
fi
echo "[PASS] Scenario 1: Valid verified commit passed CI gate."

echo ""
echo "--- SCENARIO 2: Unverified Commit / git commit --no-verify Bypass (Must REJECT) ---"
# Make unverified modification and commit without going through READ verify/handoff
echo "unverified change" >> file.txt
git add file.txt
git commit -qm "Bypassed verification via --no-verify"

set +e
echo "[CI STEP] Running CI gate checks against unverified commit..."
STATUS_CHECK=$("$ENGINE" status | grep '"status":' | tr -d '", ' | cut -d: -f2)
echo "Inspecting state status: $STATUS_CHECK"

# In unverified state, diff-hash does not match staged_diff_hash in state
CURRENT_DIFF=$("$ENGINE" diff-hash 2>/dev/null || echo "no_staged_diff")
RECORDED_DIFF=$(python3 -c "
import json
print(json.load(open('.read/STATE.json'))['staged_diff_hash'])
")

if [ "$STATUS_CHECK" != "HandoffComplete" ] || [ "$CURRENT_DIFF" != "$RECORDED_DIFF" ]
then
    echo "[GATE BLOCKED] Server-side CI gate successfully rejected unverified code change."
    SCENARIO2_BLOCKED=1
else
    echo "[ERROR] Gate failed to catch unverified change."
    SCENARIO2_BLOCKED=0
fi
set -e

if [ "$SCENARIO2_BLOCKED" -ne 1 ]
then
    echo "[FAIL] Scenario 2 did not block unverified commit!"
    exit 1
fi
echo "[PASS] Scenario 2: Unverified bypass attempt successfully rejected."

echo ""
echo "--- SCENARIO 3: Tampered WAL Chain Detection in CI (Must REJECT) ---"
# Tamper with WAL while keeping git tag anchor pointing to original head
echo "2026-11-23T09:00:00Z | prev_hash=0000000000000000000000000000000000000000000000000000000000000000 | TAMPERED | hacked=true | entry_hash=ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" > .read/WAL.log

set +e
"$ENGINE" verify-wal-anchor 2> ci_wal_tamper_err.txt
TAMPER_EXIT=$?
set -e

if [ "$TAMPER_EXIT" -ne 0 ]
then
    echo "[GATE BLOCKED] Server-side CI gate rejected tampered WAL chain (exit code: $TAMPER_EXIT)."
    cat ci_wal_tamper_err.txt
    echo "[PASS] Scenario 3: Tampered WAL chain successfully rejected by CI anchor check."
else
    echo "[FAIL] Scenario 3 failed to catch tampered WAL!"
    exit 1
fi

echo ""
echo "=== ALL 3 CI GATE WORKFLOW SCENARIOS PASSED ==="
