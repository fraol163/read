#!/usr/bin/env bash
set -e

echo "=== TEST: Cryptographic WAL Git Tag Anchoring and Rewritten Chain Attack ==="
echo "Target Command: read-engine verify-wal-anchor & recover"
echo "Date: $(date --iso-8601=seconds)"

ENGINE="/home/renan/Downloads/me/methodology_session/read/bin/read-engine"
TMPDIR=$(mktemp -d /tmp/read-wal-anchor-test-XXXXXX)

cleanup() {
    rm -rf "$TMPDIR"
}
trap cleanup EXIT

cd "$TMPDIR"
git init -q
git config user.name "Security Tester"
git config user.email "security@read.local"
git commit --allow-empty -qm "Initial root commit"

# 1. Initialize READ and perform valid transitions
"$ENGINE" init
echo "TASK-TAMPER-001" > .read/TASK.md
CONTRACT_HASH=$(sha256sum .read/TASK.md | cut -d" " -f1)
"$ENGINE" define "TASK-TAMPER-001" "$CONTRACT_HASH" "HEAD"
"$ENGINE" build

echo "print('safe code')" > app.py
git add app.py
DIFF_HASH=$("$ENGINE" diff-hash)
"$ENGINE" verify "$DIFF_HASH" "All 10 mutants killed"
"$ENGINE" handoff

echo ""
echo "[STEP 1] Valid State and Anchored Git Tag"
ANCHOR_TAG_MSG=$(git tag -l --format="%(contents)" read-wal-head | tr -d '\n')
echo "Git Tag read-wal-head contains: $ANCHOR_TAG_MSG"

echo "[STEP 2] Running initial anchor verification..."
"$ENGINE" verify-wal-anchor
echo "[PASS] Initial anchor verification succeeded."

echo ""
echo "[STEP 3] Simulating Adversarial Attack: Rewriting Entire WAL Chain"
echo "Adversary rewrites WAL.log with self-consistent internal SHA-256 hashes"
python3 -c "
import hashlib
t = '2026-11-23T09:30:00Z'
p = '0000000000000000000000000000000000000000000000000000000000000000'
ev = 'FORGED_ATTACK'
payload = 'malicious_override=true'
raw = f'{t}|{p}|{ev}|{payload}'
h = hashlib.sha256(raw.encode('utf-8')).hexdigest()
with open('.read/WAL.log', 'w') as f:
    f.write(f'{t} | prev_hash={p} | {ev} | {payload} | entry_hash={h}\n')
print(f'Rewritten WAL entry_hash: {h}')
"

echo ""
echo "[STEP 4] Executing read-engine verify-wal-anchor against rewritten WAL..."
set +e
"$ENGINE" verify-wal-anchor 2> verify_err.txt
VERIFY_EXIT=$?
set -e

if [ "$VERIFY_EXIT" -ne 0 ]
then
    echo "[PASS] verify-wal-anchor correctly detected rewritten chain! Exit code: $VERIFY_EXIT"
    cat verify_err.txt
else
    echo "[FAIL] verify-wal-anchor failed to detect rewritten chain!"
    exit 1
fi

echo ""
echo "[STEP 5] Executing read-engine recover against rewritten WAL..."
set +e
"$ENGINE" recover 2> recover_err.txt
RECOVER_EXIT=$?
set -e

if [ "$RECOVER_EXIT" -ne 0 ]
then
    echo "[PASS] read-engine recover correctly aborted on anchor mismatch! Exit code: $RECOVER_EXIT"
    cat recover_err.txt
else
    echo "[FAIL] read-engine recover failed to abort on anchor mismatch!"
    exit 1
fi

echo ""
echo "=== WAL ANCHOR AND REWRITTEN CHAIN TEST PASSED ==="
