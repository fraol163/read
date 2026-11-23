#!/usr/bin/env bash
# READ Framework Autonomous Agent Execution Pipeline Integration Tests
# Verifies: read run CLI and daemon REST endpoint /api/v1/pipeline/run
# Stages 1-5: Contract Freeze, AST Context, Dispatch, Diff & Mutation, Sealing

set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
READ_BIN="$WORKSPACE_ROOT/read/bin/read"
READ_DAEMON="$WORKSPACE_ROOT/read/bin/read-daemon"
READ_ENGINE="$WORKSPACE_ROOT/read/bin/read-engine"

SANDBOX_DIR=$(mktemp -d -t read-pipeline-test-XXXXXX)
MOCK_DIR=$(mktemp -d -t read-mock-harness-XXXXXX)
DAEMON_PID=""

cleanup() {
    [ -n "$DAEMON_PID" ] && kill "$DAEMON_PID" 2>/dev/null || true
    rm -rf "$SANDBOX_DIR" "$MOCK_DIR"
}
trap cleanup EXIT

echo "================================================================"
echo "=== [TEST SUITE] Starting READ Autonomous Pipeline Tests ==="
echo "Sandbox: $SANDBOX_DIR"
echo "================================================================"

cd "$SANDBOX_DIR"
git init -q
git config user.name "READ Orchestrator Test"
git config user.email "orchestrator@read.framework"

# 1. Verify read run --help output
echo "[TEST 1] Verifying read run CLI help flags..."
HELP_OUT=$("$READ_BIN" run --help 2>&1 || true)
if ! echo "$HELP_OUT" | grep -q -- "-harness" || ! echo "$HELP_OUT" | grep -q -- "-auto-approve" || ! echo "$HELP_OUT" | grep -q -- "-timeout"; then
    echo "FAIL: read run help missing required flags" >&2
    exit 1
fi
echo "[PASS] 1. read run CLI supports --harness, --auto-approve, --timeout"

# 2. Set up mock agent harness and custom configuration
MOCK_AGENT="$MOCK_DIR/mock_agent.sh"
cat << 'EOF' > "$MOCK_AGENT"
#!/bin/bash
cat << 'INNER_EOF' > token_service.py
def verify_token(raw_token: str) -> bool:
    return len(raw_token) >= 12
INNER_EOF
echo "[MOCK AGENT] Updated token_service.py"
EOF
chmod +x "$MOCK_AGENT"

CUSTOM_HARNESS_CFG="$MOCK_DIR/harnesses.json"
cat << EOF > "$CUSTOM_HARNESS_CFG"
{
  "harnesses": [
    {
      "id": "mock_agent",
      "name": "Mock Pipeline Agent",
      "type": "cli",
      "path": "$MOCK_AGENT",
      "enabled": true,
      "description": "Mock agent for pipeline automated test"
    }
  ]
}
EOF

# Initial source file and commit
cat << 'EOF' > token_service.py
def verify_token(raw_token: str) -> bool:
    pass
EOF

cat << 'EOF' > test_tokens.py
import unittest
from token_service import verify_token

class TestTokenService(unittest.TestCase):
    def test_short_token(self):
        self.assertFalse(verify_token("short"))
        self.assertFalse(verify_token("12345678901"))
    def test_valid_token(self):
        self.assertTrue(verify_token("123456789012"))
        self.assertTrue(verify_token("very_long_secure_token_12345"))
    def test_boundary(self):
        self.assertIs(verify_token("123456789012"), True)
        self.assertIs(verify_token("12345678901"), False)

if __name__ == '__main__':
    unittest.main()
EOF

git add token_service.py test_tokens.py
git commit -qm "Initial commit with stub and tests"

# 3. Test CLI Autonomous Execution: read run "prompt"
echo "[TEST 2] Executing autonomous pipeline: read run --harness mock_agent..."
"$READ_BIN" run \
    --harness mock_agent \
    --custom-harness "$CUSTOM_HARNESS_CFG" \
    --test-cmd "python3 -m unittest test_tokens.py" \
    --timeout 20 \
    "Implement token verification logic with minimum length 12"

# Verify STATE.json sealed handoff
if [ ! -f .read/STATE.json ]; then
    echo "FAIL: .read/STATE.json missing" >&2
    exit 1
fi

STATE_STATUS=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['status'])")
VERIFY_HASH=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['verify_hash'])")
DIFF_HASH=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['staged_diff_hash'])")

if [ "$STATE_STATUS" != "handoff_complete" ]; then
    echo "FAIL: State status is '$STATE_STATUS', expected 'handoff_complete'" >&2
    exit 1
fi
if [ -z "$VERIFY_HASH" ] || [ "$VERIFY_HASH" = "None" ]; then
    echo "FAIL: verify_hash is null in STATE.json" >&2
    exit 1
fi
if [ -z "$DIFF_HASH" ] || [ "$DIFF_HASH" = "None" ]; then
    echo "FAIL: staged_diff_hash is null in STATE.json" >&2
    exit 1
fi
echo "[PASS] 2. read run CLI completed 5 stages: sealed handoff_complete (verify_hash: $VERIFY_HASH)"

# 4. Verify WAL.log contains cryptographic records
echo "[TEST 3] Verifying cryptographic WAL audit trail..."
if ! grep -q "DEFINE" .read/WAL.log || ! grep -q "BUILD_START" .read/WAL.log || ! grep -q "VERIFIED" .read/WAL.log || ! grep -q "HANDOFF" .read/WAL.log; then
    echo "FAIL: WAL.log missing expected lifecycle transitions" >&2
    exit 1
fi
echo "[PASS] 3. Cryptographic WAL audit chain intact (DEFINE, BUILD_START, VERIFIED, HANDOFF)"

# 5. Verify Verification Gate Failure when mutants survive without --auto-approve
echo "[TEST 4] Testing verification gate failure on inadequate test suite..."
# Mock agent introduces multiple untracked bypass branches that survive mutation
cat << 'EOF' > "$MOCK_AGENT"
#!/bin/bash
cat << 'INNER_EOF' > token_service.py
def verify_token(raw_token: str) -> bool:
    if raw_token == "admin_bypass_one":
        return True
    if raw_token == "admin_bypass_two":
        return True
    if raw_token == "admin_bypass_three":
        return True
    return len(raw_token) >= 12
INNER_EOF
EOF

# Reset state to idle for new task
"$READ_ENGINE" init 2>/dev/null || true
# Overwrite STATE.json with idle
python3 -c "
import json
st = json.load(open('.read/STATE.json'))
st['status'] = 'idle'
st['verify_hash'] = None
st['staged_diff_hash'] = None
json.dump(st, open('.read/STATE.json', 'w'))
"

# Run without auto-approve: should fail verification gate or fail tests
set +e
"$READ_BIN" run \
    --harness mock_agent \
    --custom-harness "$CUSTOM_HARNESS_CFG" \
    --test-cmd "python3 -m unittest test_tokens.py" \
    --timeout 15 \
    "Add backdoor token check"
FAIL_EXIT=$?
set -e

GATE_STATUS=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['status'])")
if [ "$GATE_STATUS" = "handoff_complete" ]; then
    echo "FAIL: Gate should have blocked handoff when mutants survived!" >&2
    exit 1
fi
echo "[PASS] 4. Verification gate blocked premature handoff (status: $GATE_STATUS, exit: $FAIL_EXIT)"

# 6. Verify Master Override with --auto-approve
echo "[TEST 5] Testing Master Override application with --auto-approve..."
"$READ_BIN" run \
    --harness mock_agent \
    --custom-harness "$CUSTOM_HARNESS_CFG" \
    --test-cmd "python3 -m unittest test_tokens.py" \
    --auto-approve \
    --timeout 15 \
    "Add backdoor token check with auto approve"

OVR_STATUS=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['status'])")
OVR_HASH=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['verify_hash'])")

if [ "$OVR_STATUS" != "handoff_complete" ] || [[ "$OVR_HASH" != OVR-* ]]; then
    echo "FAIL: Master override failed: status=$OVR_STATUS, hash=$OVR_HASH" >&2
    exit 1
fi
echo "[PASS] 5. Master Override applied via --auto-approve (sealed: $OVR_HASH)"

# 7. Test Daemon REST Endpoint: /api/v1/pipeline/run
echo "[TEST 6] Testing Daemon REST endpoint /api/v1/pipeline/run..."
DAEMON_PORT=19485
"$READ_DAEMON" -port "$DAEMON_PORT" -workspace "$SANDBOX_DIR" -engine-bin "$READ_ENGINE" -custom-harness "$CUSTOM_HARNESS_CFG" > /dev/null 2>&1 &
DAEMON_PID=$!
for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# Reset state to idle
python3 -c "
import json
st = json.load(open('.read/STATE.json'))
st['status'] = 'idle'
st['verify_hash'] = None
st['staged_diff_hash'] = None
json.dump(st, open('.read/STATE.json', 'w'))
"

API_RESPONSE=$(curl -s -X POST "http://127.0.0.1:$DAEMON_PORT/api/v1/pipeline/run" \
    -H "Content-Type: application/json" \
    -d "{
        \"prompt\": \"Verify tokens via REST API\",
        \"harness\": \"mock_agent\",
        \"auto_approve\": true,
        \"timeout\": 20,
        \"test_command\": \"python3 -m unittest test_tokens.py\"
    }")

API_SUCCESS=$(echo "$API_RESPONSE" | jq -r .success)
API_STATE=$(echo "$API_RESPONSE" | jq -r .final_state)
API_TASK_ID=$(echo "$API_RESPONSE" | jq -r .task_id)

if [ "$API_SUCCESS" != "true" ] || [ "$API_STATE" != "handoff_complete" ]; then
    echo "FAIL: Daemon REST API failed: $API_RESPONSE" >&2
    exit 1
fi
echo "[PASS] 6. Daemon REST endpoint /api/v1/pipeline/run successfully executed pipeline (task: $API_TASK_ID)"

echo "================================================================"
echo "=== [ALL 6 PIPELINE INTEGRATION TESTS PASSED] ==="
echo "================================================================"
