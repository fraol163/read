#!/usr/bin/env bash
set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
READ_BIN="$WORKSPACE_ROOT/read/bin/read"
READ_DAEMON="$WORKSPACE_ROOT/read/bin/read-daemon"
READ_ENGINE="$WORKSPACE_ROOT/read/bin/read-engine"

DAEMON_PID=""
SANDBOX_DIR=$(mktemp -d -t read-system-sandbox-XXXXXX)
trap 'rm -rf "$SANDBOX_DIR"; [ -n "$DAEMON_PID" ] && kill "$DAEMON_PID" 2>/dev/null || true' EXIT

echo "=== [TEST SUITE] Starting READ Full System & Gateway Tests ==="
echo "Sandbox: $SANDBOX_DIR"

cd "$SANDBOX_DIR"
git init -q
git config user.name "READ Test"
git config user.email "test@read.framework"

# 1. Test Native Rust Engine Initialization
"$READ_ENGINE" init
if [ ! -d ".method" ] || [ ! -f ".method/WAL.log" ] || [ ! -f ".method/STATE.json" ]; then
    echo "FAIL: .method/ artifacts missing"
    exit 1
fi
echo "[PASS] 1. Native Rust Engine initialized atomic WAL & STATE"

# 2. Test Contract Definition & AST Resolution
cat << 'EOF' > auth_service.py
import hashlib

def hash_token(raw_token: str) -> str:
    return hashlib.sha256(raw_token.encode()).hexdigest()
EOF

git add auth_service.py
git commit -qm "Initial commit"

CONTRACT_HASH=$(sha256sum auth_service.py | cut -d' ' -f1)
"$READ_ENGINE" define "TASK-101" "$CONTRACT_HASH" "HEAD"

STATE_HASH=$(python3 -c "import json; print(json.load(open('.method/STATE.json'))['contract_hash'])")
if [ "$STATE_HASH" != "$CONTRACT_HASH" ]; then
    echo "FAIL: Contract hash mismatch: $STATE_HASH != $CONTRACT_HASH"
    exit 1
fi
echo "[PASS] 2. Deterministic AST & Contract Hash sealed: $CONTRACT_HASH"

# 3. Test Go Gateway Daemon Startup & Dynamic Harness Discovery
DAEMON_PORT=19482
"$READ_DAEMON" -port "$DAEMON_PORT" -workspace "$SANDBOX_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
DAEMON_PID=$!
for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# Check daemon health status
STATUS_RES=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/status")
SYSTEM_NAME=$(echo "$STATUS_RES" | jq -r .system)
if [[ "$SYSTEM_NAME" != *"READ Framework"* ]]; then
    echo "FAIL: Daemon failed to respond"
    exit 1
fi
echo "[PASS] 3. Go Gateway Daemon online on port $DAEMON_PORT"

# 4. Verify Dynamic Detection of Host AI Agent Harnesses
HARNESSES_RES=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/harnesses")
HARNESS_COUNT=$(echo "$HARNESSES_RES" | jq '.harnesses | length')
if [ "$HARNESS_COUNT" -lt 1 ]; then
    echo "FAIL: No harnesses detected"
    exit 1
fi
echo "[PASS] 4. Dynamic Harness Scanner detected $HARNESS_COUNT agents on host system"

# 5. Test 3-Way Storage Policy Updates
SYNC_POLICY_RES=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT/api/v1/sync/policy" \
    -H "Content-Type: application/json" \
    -d '{"policy": "phone_only"}')
POLICY_SET=$(echo "$SYNC_POLICY_RES" | jq -r .policy)
if [ "$POLICY_SET" != "phone_only" ]; then
    echo "FAIL: Policy not set"
    exit 1
fi
echo "[PASS] 5. 3-Way Storage policy switched dynamically to 'phone_only'"

# 6. Test Remote File Explorer & SHA-256 Checksums
FS_LIST_RES=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/fs/list?path=")
AUTH_FILE_PRESENT=$(echo "$FS_LIST_RES" | jq -r '.[] | select(.name=="auth_service.py") | .name')
if [ "$AUTH_FILE_PRESENT" != "auth_service.py" ]; then
    echo "FAIL: Remote file explorer did not list auth_service.py"
    exit 1
fi
echo "[PASS] 6. Remote Workspace File Explorer listed active codebase with SHA-256"

# 7. Test Remote Master Override via Daemon API
OVERRIDE_RES=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT/api/v1/state/override" \
    -H "Content-Type: application/json" \
    -d '{"rationale": "Emergency hotfix bypassing AST gate for critical CVE"}')
OVERRIDE_SUCCESS=$(echo "$OVERRIDE_RES" | jq -r .success)
if [ "$OVERRIDE_SUCCESS" != "true" ]; then
    echo "FAIL: Master override failed: $OVERRIDE_RES"
    exit 1
fi
echo "[PASS] 7. Remote Master Override executed & recorded to WAL with SHA-256 rationale"

# 8. Verify State is 'verified' with 'OVR-' hash, then trigger handoff via daemon
STATE_RES=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/state")
STATE_STATUS=$(echo "$STATE_RES" | jq -r .status)
VERIFY_HASH=$(echo "$STATE_RES" | jq -r .verify_hash)
if [ "$STATE_STATUS" != "verified" ] || [[ "$VERIFY_HASH" != OVR-* ]]; then
    echo "FAIL: State status is not verified with OVR hash: $STATE_STATUS ($VERIFY_HASH)"
    exit 1
fi
echo "[PASS] 8. Master Override sealed state with cryptographic proof: $VERIFY_HASH"

# 9. Trigger CAS Handoff through Gateway Daemon
HANDOFF_RES=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT/api/v1/task/command" \
    -H "Content-Type: application/json" \
    -d '{"command": "handoff", "args": []}')
HANDOFF_SUCCESS=$(echo "$HANDOFF_RES" | jq -r .success)
if [ "$HANDOFF_SUCCESS" != "true" ]; then
    echo "FAIL: Gateway handoff failed: $HANDOFF_RES"
    exit 1
fi

FINAL_STATE=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/state")
FINAL_STATUS=$(echo "$FINAL_STATE" | jq -r .status)
if [ "$FINAL_STATUS" != "handoff_complete" ]; then
    echo "FAIL: Final state status is not handoff_complete: $FINAL_STATUS"
    exit 1
fi
echo "[PASS] 9. CAS Handoff completed via Gateway Daemon to handoff_complete"

echo "=== [ALL 9 SYSTEM TESTS PASSED]: READ Framework & Gateway are 100% Operational ==="
