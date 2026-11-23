# Ten-Minute Student Quickstart Guide

This tutorial guides you through the fundamental workflow of the READ Framework using a simple Python service in a clean workspace.

## Prerequisites
- Linux x86_64 or aarch64
- Git
- Python 3.8 or higher
- Compiled `read` binary available in your `PATH` or located at `read/bin/read`

---

## Step 1: Initialize Workspace

Create a new directory and initialize git and the READ state engine:

```bash
mkdir my-read-project
cd my-read-project
git init
git config user.name "Student Developer"
git config user.email "student@example.com"

# Initialize READ directory structure and pre-commit hooks
read init
```

Output:
```
[READ] Initialized state engine in .read/
[READ] Initialized declarative methodology contracts in .read/
[READ] Activated declarative zero-command pre-commit hook (core.hooksPath=.read/hooks)
```

---

## Step 2: Define a Task Contract

Create or edit `.read/TASK.md` to specify your machine-verifiable acceptance criteria:

```markdown
# TASK CONTRACT: Math Service
TaskID: TASK-001
Preconditions:
- Python 3 is installed
Postconditions:
- Function is_positive(n) returns True when n > 0 and False otherwise
- All unit tests pass with 100% mutation kill score
```

Freeze the contract into the state machine:

```bash
# Calculate contract hash and bind to initial task state
CONTRACT_HASH=$(sha256sum .read/TASK.md | cut -d" " -f1)
read define TASK-001 "$CONTRACT_HASH" HEAD
```

Output:
```
[READ] Task defined. State revision: 2
```

---

## Step 3: Transition to Build Stage

Notify the state machine that implementation work is starting:

```bash
read build
```

Output:
```
[READ] Build started. Status: Executing
```

---

## Step 4: Implement Code and Tests

Create the application logic in `math_service.py`:

```python
def is_positive(n: int) -> bool:
    return n > 0
```

Create the unit test suite in `test_math.py`:

```python
import unittest
from math_service import is_positive

class TestMath(unittest.TestCase):
    def test_positive(self):
        self.assertTrue(is_positive(5))

    def test_negative(self):
        self.assertFalse(is_positive(-5))

    def test_zero(self):
        self.assertFalse(is_positive(0))

if __name__ == "__main__":
    unittest.main()
```

Run tests locally to verify baseline functionality:

```bash
python3 -m unittest test_math.py
```

---

## Step 5: Stage Changes and Compute Diff Hash

Stage your code changes in git:

```bash
git add math_service.py test_math.py
```

Compute the cryptographic hash of your staged git diff:

```bash
DIFF_HASH=$(read diff-hash)
echo "Staged Diff Hash: $DIFF_HASH"
```

---

## Step 6: Verify and Record Test Results

Record verification proof into the state engine:

```bash
read verify "$DIFF_HASH" "All 3 tests passed successfully."
```

Output:
```
[READ] Verification recorded. verify_hash: <sha256_hash>
```

---

## Step 7: Execute CAS Handoff

Complete the task lifecycle by executing the Compare-And-Swap handoff transition:

```bash
read handoff
```

Output:
```
[READ] CAS Handoff successful. State: HandoffComplete
```

Verify final status:

```bash
read status
```

---

## Step 8: Observe Mutation Gate Rejection

To see how READ stops weak test suites, introduce a weak test that checks only positive inputs:

```python
# test_weak.py
import unittest
from math_service import is_positive

class TestWeak(unittest.TestCase):
    def test_only_positive(self):
        self.assertTrue(is_positive(10))

if __name__ == "__main__":
    unittest.main()
```

If you mutate `is_positive` (e.g. changing `n > 0` to `n >= 0`), the weak test still passes because it never tested zero.
The AROP mutation engine detects that the mutant survived and immediately rejects the commit.
To pass the gate, you must strengthen the test suite until all generated mutants are killed.
