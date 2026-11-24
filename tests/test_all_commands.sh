#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="$(cd "$SCRIPT_DIR/.." && pwd)"
READ="$REPO/bin/read"
READ_MCP="$REPO/bin/read-mcp"
READ_ENGINE="$REPO/bin/read-engine"
READ_DAEMON="$REPO/bin/read-daemon"

TEST_DIR=$(mktemp -d /tmp/read-verify-commands-XXXXXX)
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

cd "$TEST_DIR"
git init -q
git config user.name "Tester"
git config user.email "test@test.org"

echo "1. Testing read --version and read -v"
"$READ" --version | grep "READ Framework Unified CLI"
"$READ" -v | grep "READ Framework Unified CLI"

echo "2. Testing read --help and read -h"
"$READ" --help | grep "Resource-conscious"
"$READ" -h | grep "Resource-conscious"

echo "3. Testing read init"
"$READ" init
test -d ".read"
test -f ".read/STATE.json"
test -f ".read/WAL.log"

echo "4. Testing read status"
"$READ" status | grep '"status": "idle"'

echo "5. Testing read status --yaml"
"$READ" status --yaml | grep "status: idle"

echo "6. Testing read config"
"$READ" config | grep '"version": "0.1.0"'

echo "7. Testing read diff-hash"
echo "hello world" > sample.txt
git add sample.txt
DIFF_HASH=$("$READ" diff-hash | tr -d '[:space:]')
test ${#DIFF_HASH} -eq 64

echo "8. Testing read ast-context"
cat > sample.py << 'EOF'
class Calculator:
    def add(self, a: int, b: int) -> int:
        return a + b
EOF
"$READ" ast-context sample.py | grep "class Calculator"

echo "9. Testing read define"
BASE_COMMIT=$(git rev-parse --verify HEAD 2>/dev/null || echo "0000000000000000000000000000000000000000")
"$READ" define TASK-001 "hash1234567890123456789012345678901234567890123456789012345678901234" "$BASE_COMMIT"
"$READ" status | grep '"status": "defined"'

echo "10. Testing read build"
"$READ" build
"$READ" status | grep '"status": "executing"'

echo "11. Testing read eval-mutation-score"
# 10/10 killed -> exit 0
"$READ" eval-mutation-score 10 10 | grep "PASSED"
# 9/10 killed -> exit 1
if "$READ" eval-mutation-score 9 10 2>/dev/null; then
    echo "eval-mutation-score should fail on 90%"
    exit 1
fi

echo "12. Testing read mutate"
git diff --staged > /tmp/sample.diff
"$READ" mutate /tmp/sample.diff
rm -f /tmp/sample.diff

echo "13. Testing read verify"
echo "PASS" > /tmp/verify.log
"$READ" verify "$DIFF_HASH" /tmp/verify.log
"$READ" status | grep '"status": "verified"'
rm -f /tmp/verify.log

echo "14. Testing read handoff"
"$READ" handoff
"$READ" status | grep '"status": "handoff_complete"'

echo "15. Testing read recover"
"$READ" recover | grep "RECOVER SUCCESS"

echo "16. Testing read override"
"$READ" define TASK-002 "hash1234567890123456789012345678901234567890123456789012345678901234" "$BASE_COMMIT"
"$READ" build
"$READ" override "OP-42" "Emergency deployment hotfix rationale with more than 30 characters verified"
"$READ" status | grep '"status": "verified"'

echo "17. Testing read asm"
"$READ" handoff
"$READ" asm "DEF TASK-003 aabbccdd 1234567 | BUILD | STATUS" | grep "DEF task=TASK-003"

echo "18. Testing read report --html and --yaml"
"$READ" report --html "$TEST_DIR/report.html"
test -f "$TEST_DIR/report.html"
grep -q "READ Evidence" "$TEST_DIR/report.html"
"$READ" report --yaml | grep "status:"

echo "19. Testing read scan"
"$READ" scan | grep "READ Framework Harness Scanner"

echo "20. Testing read learn (init, scenario, diagnose, progress)"
"$READ" learn init | grep "Initialized"
"$READ" learn scenario 1 | grep "Loaded Scenario 01"
"$READ" learn progress | grep "Mastery Score"
"$READ" learn diagnose | grep "DIAGNOSIS"

echo "21. Testing read-mcp --check"
"$READ_MCP" --check | grep "READ MCP Server Diagnostic"

echo "22. Testing read-mcp detect"
"$READ_MCP" detect | grep "Detected AI Agent Harnesses"

echo "23. Testing read-mcp --version"
"$READ_MCP" --version | grep "read-mcp 0.1.0"

echo "24. Testing read-mcp JSON-RPC initialize and tools/list"
INIT_REQ='{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
TOOLS_REQ='{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
CALL_REQ='{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_status","arguments":{"workspace_path":"'"$TEST_DIR"'"}}}'

RESPONSES=$(printf "%s\n%s\n%s\n" "$INIT_REQ" "$TOOLS_REQ" "$CALL_REQ" | "$READ_MCP")
echo "$RESPONSES" | grep "protocolVersion"
echo "$RESPONSES" | grep "read_status"
echo "$RESPONSES" | grep "active_task_id"

echo "ALL COMMANDS VERIFIED AND WORKING 100%!"
