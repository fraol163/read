#!/usr/bin/env bash
# ==============================================================================
# READ Framework - Zero-Command Declarative BMAD Workflow Test Suite
# Verifies that READ operates purely through declarative markdown contracts,
# file edits, and git hooks with ZERO manual CLI commands needed by the user.
# ==============================================================================
set -euo pipefail

WORKSPACE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE_DIR="$WORKSPACE_ROOT/read/template/.read"
ENGINE_BIN="$WORKSPACE_ROOT/read/bin/read-engine"

SANDBOX_DIR=$(mktemp -d -t read-bmad-sandbox-XXXXXX)
trap 'rm -rf "$SANDBOX_DIR"' EXIT

echo "================================================================================"
echo "STARTING BMAD ZERO-COMMAND DECLARATIVE WORKFLOW TEST"
echo "Sandbox Directory: $SANDBOX_DIR"
echo "Template:          $TEMPLATE_DIR"
echo "Engine Binary:     $ENGINE_BIN"
echo "================================================================================"

cd "$SANDBOX_DIR"

# Step 1: Initialize fresh git repository (standard developer setup)
git init -q
git config user.name "Declarative Developer"
git config user.email "dev@bmad-style.org"

# Step 2: Install READ template and autonomous pre-commit hook
cp -r "$TEMPLATE_DIR" .read
mkdir -p .git/hooks
cp .read/hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit

echo "[TEST 1] Setup repository with .read/ template and git pre-commit hook."

# Configure PROJECT.md to use python unittest
cat << 'EOF' > .read/PROJECT.md
# Project Configuration
name: bmad_calculator_service
test_command: python3 -m unittest test_calculator.py
EOF

# Step 3: Write code and comprehensive tests
cat << 'EOF' > calculator.py
def add(a: int, b: int) -> int:
    return a + b

def multiply(a: int, b: int) -> int:
    return a * b
EOF

cat << 'EOF' > test_calculator.py
import unittest
from calculator import add, multiply

class TestCalculator(unittest.TestCase):
    def test_add(self):
        self.assertEqual(add(2, 3), 5)
        self.assertEqual(add(-1, 1), 0)
        self.assertEqual(add(0, 0), 0)

    def test_multiply(self):
        self.assertEqual(multiply(2, 3), 6)
        self.assertEqual(multiply(-2, 3), -6)
        self.assertEqual(multiply(0, 5), 0)
        self.assertEqual(multiply(1, 7), 7)

if __name__ == "__main__":
    unittest.main()
EOF

# Step 4: Write declarative TASK.md (Zero commands!)
cat << 'EOF' > .read/TASK.md
# TASK: Arithmetic Engine
Status: defined
Target Files:
- calculator.py
- test_calculator.py

Acceptance Criteria:
- add(a, b) returns correct sum
- multiply(a, b) returns correct product
EOF

echo "[TEST 2] Executing standard git workflow: git add && git commit (ZERO read commands)..."

# Ensure PATH includes engine binary directory so pre-commit hook finds it
export PATH="$(dirname "$ENGINE_BIN"):$PATH"

# User or Agent only runs standard git commands:
git add .
git commit -m "feat: implement arithmetic functions with complete tests"

echo "[TEST 3] Verifying that pre-commit hook automatically sealed state..."

# Assert that STATE.json was automatically updated to handoff_complete
STATE_STATUS=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['status'])")
if [ "$STATE_STATUS" != "handoff_complete" ]; then
    echo "FAIL: Expected status 'handoff_complete', got '$STATE_STATUS'"
    exit 1
fi
echo "[PASS] 1. STATE.json status transitioned to 'handoff_complete' with zero manual commands."

# Assert that EVIDENCE.json was automatically written with mutation proof
EVIDENCE_STATUS=$(python3 -c "import json; print(json.load(open('.read/EVIDENCE.json'))['status'])")
MUTATION_SCORE=$(python3 -c "import json; print(json.load(open('.read/EVIDENCE.json'))['mutation_score'])")
if [ "$EVIDENCE_STATUS" != "verified" ]; then
    echo "FAIL: Expected evidence status 'verified', got '$EVIDENCE_STATUS'"
    exit 1
fi
echo "[PASS] 2. EVIDENCE.json sealed with verified mutation proof (Score: ${MUTATION_SCORE}%)."

# Assert that the git commit actually includes the sealed STATE.json and EVIDENCE.json
COMMITTED_FILES=$(git show --name-only --oneline HEAD | tr '\n' ' ')
if [[ "$COMMITTED_FILES" != *".read/STATE.json"* ]] || [[ "$COMMITTED_FILES" != *".read/EVIDENCE.json"* ]]; then
    echo "FAIL: Git commit does not contain sealed .read/STATE.json and .read/EVIDENCE.json"
    exit 1
fi
echo "[PASS] 3. Git commit atomically sealed .read/STATE.json and .read/EVIDENCE.json."

# Step 5: Test adversarial rejection on weak test coverage (No manual commands)
echo "[TEST 4] Testing adversarial rejection: adding function without tests..."

cat << 'EOF' >> calculator.py

def divide(a: int, b: int) -> float:
    if b == 0:
        raise ValueError("Cannot divide by zero")
    return a / b
EOF

# Update TASK.md
cat << 'EOF' > .read/TASK.md
# TASK: Arithmetic Engine Extension
Status: defined
Target Files:
- calculator.py
- test_calculator.py

Acceptance Criteria:
- divide(a, b) returns quotient or raises ValueError
EOF

git add .

# Attempt commit without adding tests for divide: should be REJECTED by mutation gate!
COMMIT_FAILED=0
git commit -m "feat: add divide without tests" > /tmp/read_commit_reject.log 2>&1 || COMMIT_FAILED=1

if [ "$COMMIT_FAILED" -ne 1 ]; then
    echo "FAIL: Commit should have been rejected due to surviving mutants, but it succeeded!"
    exit 1
fi

echo "[PASS] 4. Pre-commit hook mechanically blocked commit when mutation score fell below strictly required 100% threshold."

# Verify that EVIDENCE.json records the gate failure
GATE_STATUS=$(python3 -c "import json; print(json.load(open('.read/EVIDENCE.json'))['status'])")
if [ "$GATE_STATUS" != "gate_failed" ]; then
    echo "FAIL: Expected evidence status 'gate_failed', got '$GATE_STATUS'"
    exit 1
fi
echo "[PASS] 5. EVIDENCE.json accurately diagnosed and recorded 'gate_failed'."

# Step 6: Fix tests to satisfy mutation gate and commit again
echo "[TEST 5] Fixing test suite to kill divide mutants..."

cat << 'EOF' > test_calculator.py
import unittest
from calculator import add, multiply, divide

class TestCalculator(unittest.TestCase):
    def test_add(self):
        self.assertEqual(add(2, 3), 5)
        self.assertEqual(add(-1, 1), 0)
        self.assertEqual(add(0, 0), 0)

    def test_multiply(self):
        self.assertEqual(multiply(2, 3), 6)
        self.assertEqual(multiply(-2, 3), -6)
        self.assertEqual(multiply(0, 5), 0)
        self.assertEqual(multiply(1, 7), 7)

    def test_divide(self):
        self.assertEqual(divide(6, 2), 3.0)
        self.assertEqual(divide(5, 2), 2.5)
        self.assertEqual(divide(-6, 2), -3.0)
        with self.assertRaises(ValueError):
            divide(5, 0)

if __name__ == "__main__":
    unittest.main()
EOF

git add .
git commit -m "feat: add divide with complete test assertions"

FINAL_STATUS=$(python3 -c "import json; print(json.load(open('.read/STATE.json'))['status'])")
if [ "$FINAL_STATUS" != "handoff_complete" ]; then
    echo "FAIL: Expected final status 'handoff_complete', got '$FINAL_STATUS'"
    exit 1
fi
echo "[PASS] 6. Commit succeeded automatically once test suite killed all mutants."

echo "================================================================================"
echo "ALL 6 BMAD ZERO-COMMAND DECLARATIVE WORKFLOW TESTS PASSED SUCCESSFULLY"
echo "Verified: Zero commands required. Pure git & markdown operation like BMAD."
echo "================================================================================"
