#!/usr/bin/env bash
# Real-Time Live Execution Verification of READ Framework
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
READ_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
export PATH="$READ_ROOT/bin:$PATH"
enable -n read 2>/dev/null || true

DEMO_DIR=$(mktemp -d -t read-live-realtime-XXXXXX)
trap 'rm -rf "$DEMO_DIR"' EXIT

echo "================================================================"
echo "READ FRAMEWORK: LIVE REAL-TIME EXECUTION AUDIT"
echo "Workspace Sandbox: $DEMO_DIR"
echo "================================================================"
echo ""

cd "$DEMO_DIR"
git init -q
git config user.name "Live Engineer"
git config user.email "engineer@read.live"

echo "=== STEP 1: Interactive Onboarding Wizard (read setup) ==="
"$READ_ROOT/bin/read" setup --batch
echo ""

echo "=== STEP 2: Declarative Contract Definition (.read/TASK.md) ==="
cat << 'EOF' > .read/TASK.md
# Task: Token Authorization Service
Status: Defined
Requirement: Implement validate_bearer_token(header, secret) -> bool.
Acceptance: Returns true if header is 'Bearer <secret>', false otherwise.
EOF
echo "[READ Contract] TASK.md created with cryptographic requirements."
echo ""

echo "=== STEP 3: Implementation Code & Test Suite ==="
cat << 'EOF' > auth.py
def validate_bearer_token(header: str, secret: str) -> bool:
    if not header or not header.startswith("Bearer "):
        return False
    token = header[7:].strip()
    if not token:
        return False
    return token == secret
EOF

cat << 'EOF' > test_auth.py
import unittest
from auth import validate_bearer_token

class TestAuth(unittest.TestCase):
    def test_valid(self):
        self.assertTrue(validate_bearer_token("Bearer secret-key-xyz", "secret-key-xyz"))

    def test_invalid_secret(self):
        self.assertFalse(validate_bearer_token("Bearer wrong-key", "secret-key-xyz"))

    def test_missing_bearer_prefix(self):
        self.assertFalse(validate_bearer_token("Basic dXNlcjpwYXNz", "secret-key-xyz"))

    def test_empty_token(self):
        self.assertFalse(validate_bearer_token("Bearer ", "secret-key-xyz"))

    def test_empty_header(self):
        self.assertFalse(validate_bearer_token("", "secret-key-xyz"))

if __name__ == '__main__':
    unittest.main()
EOF
echo "[DEV] auth.py and test_auth.py authored."
echo ""

echo "=== STEP 4: Developer Commits (Zero Commands - Hook Intercepts) ==="
git add .
git commit -m "feat: implement bearer token authentication"
echo ""

echo "=== STEP 5: Verification of Automatically Sealed State ==="
python3 -c '
import json
st = json.load(open(".read/STATE.json"))
ev = json.load(open(".read/EVIDENCE.json"))
print("STATE.json Status:        " + str(st.get("status")))
print("STATE.json Contract Hash:  " + str(st.get("contract_hash", ""))[:16] + "...")
print("STATE.json Verify Hash:    " + str(st.get("verify_hash", ""))[:16] + "...")
print("EVIDENCE.json Status:      " + str(ev.get("status")))
print("EVIDENCE.json Score:       " + str(ev.get("mutation_score")) + "%")
print("EVIDENCE.json Mutants:     " + str(ev.get("mutants_killed")) + "/" + str(ev.get("mutants_total")) + " killed")
'
echo ""

echo "=== STEP 6: Adversarial Code Injection (Untested Code) ==="
cat << 'EOF' >> auth.py

def divide(a: int, b: int) -> float:
    if b == 0:
        raise ValueError("Cannot divide by zero")
    return a / b
EOF

git add auth.py
python3 -c '
import json
with open(".read/config.json") as f:
    c = json.load(f)
c["enforcement_mode"] = "block"
with open(".read/config.json", "w") as f:
    json.dump(c, f, indent=2)
'
echo "Attempting commit without tests (should be rejected by gate)..."
set +e
git commit -m "feat: add divide function without tests"
COMMIT_CODE=$?
set -e

if [ "$COMMIT_CODE" -ne 0 ]; then
    echo "[PASS] Pre-commit hook mechanically BLOCKED commit (Exit Code: $COMMIT_CODE)."
else
    echo "[FAIL] Pre-commit hook failed to block unverified code!"
    exit 1
fi

python3 -c "
import json
ev = json.load(open('.read/EVIDENCE.json'))
print(f'EVIDENCE.json Gate Status: {ev.get(\"status\")} (Score: {ev.get(\"mutation_score\")}%)')
"
echo ""

echo "=== STEP 7: Test Remediation (Killing All Mutants) ==="
cat << 'EOF' > test_auth.py
import unittest
from auth import validate_bearer_token, divide

class TestAuth(unittest.TestCase):
    def test_valid(self):
        self.assertTrue(validate_bearer_token("Bearer secret-key-xyz", "secret-key-xyz"))

    def test_invalid_secret(self):
        self.assertFalse(validate_bearer_token("Bearer wrong-key", "secret-key-xyz"))

    def test_missing_bearer_prefix(self):
        self.assertFalse(validate_bearer_token("Basic dXNlcjpwYXNz", "secret-key-xyz"))

    def test_empty_token(self):
        self.assertFalse(validate_bearer_token("Bearer ", "secret-key-xyz"))

    def test_empty_header(self):
        self.assertFalse(validate_bearer_token("", "secret-key-xyz"))

    def test_divide_valid(self):
        self.assertEqual(divide(10, 2), 5.0)

    def test_divide_zero(self):
        with self.assertRaises(ValueError):
            divide(10, 0)

if __name__ == '__main__':
    unittest.main()
EOF

git add test_auth.py
echo "Retrying commit with complete tests..."
git commit -m "feat: complete divide with full unit tests"
echo ""

echo "=== STEP 8: Final Proof and Sealed Commit Chain ==="
python3 -c '
import json
st = json.load(open(".read/STATE.json"))
ev = json.load(open(".read/EVIDENCE.json"))
print("Final Status:          " + str(st.get("status")))
print("Final Evidence Status: " + str(ev.get("status")))
print("Final Mutation Score:  " + str(ev.get("mutation_score")) + "% (" + str(ev.get("mutants_killed")) + "/" + str(ev.get("mutants_total")) + " killed)")
'
git log --oneline -n 2

echo ""
echo "================================================================"
echo "LIVE REAL-TIME AUDIT COMPLETE: 100% OPERATIONAL WITH ZERO COMMANDS"
echo "================================================================"
