# READ 4-Stage Gated Lifecycle Specification

## 1. Overview and Core Philosophy

In conversational AI workflows, developers rely on conversational feedback loops: an LLM generates code, runs a unit test, claims success, and hands the work back to the user. This workflow fails frequently because:
1. LLMs often write trivial or tautological tests that pass regardless of whether business invariants hold.
2. Code may introduce regressions in unasserted edge cases.
3. Code can drift between the time tests are executed and the time code is merged.
4. There is no cryptographic audit trail connecting requirements to implementation evidence.

The READ framework enforces a strictly gated, 4-stage mechanical lifecycle:
```
+-----------------------------------------------------------------------------------------+
|                                4-STAGE GATED LIFECYCLE                                  |
|                                                                                         |
|   STAGE 1: DEFINE                                                                       |
|   [Task Contract] ---> Hash Normalization ---> Freeze Contract Hash into STATE.json     |
|                                                                                         |
|                                          |                                              |
|                                          v                                              |
|   STAGE 2: BUILD                                                                        |
|   Acquire CAS Lease ---> AST Context Extraction ---> Agent Code Construction            |
|                                                                                         |
|                                          |                                              |
|                                          v                                              |
|   STAGE 3: VERIFY                                                                       |
|   Stage Changes ---> SHA-256 Diff Hash ---> AROP Bounded Mutation Test (Score = 100%)    |
|                                                                                         |
|                                          |                                              |
|                                          v                                              |
|   STAGE 4: HANDOFF                                                                      |
|   CAS Invariant Check ---> WAL Proof Audit ---> Diff Drift Check ---> Handoff Sealed    |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Stage 1: Define

### 2.1 The Task Contract
A task cannot begin without an explicit, machine-readable task contract. The contract specifies:
- `task_id`: Unique alphanumeric identifier (e.g., `TASK-AUTH-042`).
- `specification`: Unambiguous description of functional requirements.
- `invariants`: Absolute system invariants that must never be violated.
- `target_files`: Scope of source files authorized for modification.
- `base_commit`: The exact Git commit hash from which work originates.

Example contract file (`contract.json`):
```json
{
  "task_id": "TASK-METRICS-001",
  "base_commit": "4b825dc642cb6eb9a060e54bf8d69288fbee4904",
  "target_files": [
    "src/metrics/collector.py",
    "tests/test_collector.py"
  ],
  "invariants": [
    "Latency percentiles (p50, p95, p99) must be non-negative",
    "Empty metric samples must return an error rather than division-by-zero",
    "Memory overhead of sample ring buffer must not exceed 10 MB"
  ],
  "verification_command": "pytest tests/test_collector.py"
}
```

### 2.2 Contract Hashing and Freezing
The contract is normalized and hashed using SHA-256:
$$\text{contract\_hash} = \text{SHA-256}(\text{CanonicalJSON}(\text{contract}))$$

The transition is recorded using the CLI:
```bash
read-engine define TASK-METRICS-001 <CONTRACT_HASH> <BASE_COMMIT>
```

Under the hood:
1. An exclusive POSIX file lock is acquired on `.method/.state.lock`.
2. `.method/STATE.json` is checked: the task status must be `Idle` or `HandoffComplete`. Attempting to define a new task while an existing task is in progress triggers `CasLockViolation`.
3. `state_revision` is incremented.
4. Status transitions to `Defined`.
5. An entry is appended to `.method/WAL.log` with event type `DEFINE`.

---

## 3. Stage 2: Build

### 3.1 CAS Lease Acquisition and Workspace Setup
To prevent concurrent workers or multi-agent collisions from corrupting state:
```bash
read-engine build
```

The engine validates:
- Status is strictly `Defined`.
- The current working directory is clean relative to `base_commit`.
- Sets status to `Executing` and increments `state_revision`.
- Appends `BUILD_START` to `.method/WAL.log`.

### 3.2 AST Context Provisioning
Before the external AI agent or human begins coding, `read-core` provides context via deterministic AST extraction:
```bash
read-engine ast-context <TARGET_FILE>
```
This produces the complete public interface of local dependencies in under 5 milliseconds with zero model tokens consumed.

---

## 4. Stage 3: Verify

### 4.1 Staged Diff Hashing
When implementation is complete, the changes are staged in Git:
```bash
git add <TARGET_FILES>
```
The diff hash is computed across `git diff --staged`:
```bash
read-engine diff-hash
```
The diff hasher streams the unified diff directly through SHA-256, finishing in under 0.1 milliseconds. This hash represents the exact cryptographic identity of the code changes.

### 4.2 Asymmetric Bounded Mutation Testing (AROP)

#### Why Full Mutation Testing Fails for AI Agents
Standard mutation testing tools (e.g., Mutmut, Stryker, PIT) generate hundreds or thousands of mutants across the entire codebase. Running the full test suite against every mutant takes 5 to 45 minutes of CPU time. This makes interactive agent supervision impossible.

#### AROP Formulation and Constraints
READ introduces AROP (Asymmetric Bounded Mutation Testing). AROP restricts mutation to a strictly bounded space:
1. **Target Boundary**: Mutants are generated **only** on lines newly added or modified in `git diff --staged`. Untouched legacy code is never mutated.
2. **Population Ceiling**: A maximum of 10 mutants ($M \le 10$) are generated per verification run.
3. **Execution Ceiling**: Mutation generation is capped at a hard timeout of 4.5 seconds ($T \le 4500\text{ms}$).
4. **Memory Ceiling**: Input diff buffer is capped at 50 MB, guaranteeing RAM usage stays well below 1.2 GB.

```
Unified Git Diff (+ lines only)
            |
            v
   Filter non-code lines (comments, blanks)
            |
            v
   Apply Mutation Operators (up to 10 mutants)
            |
            +---> 1. Relational Inversions (== -> !=, < -> >=)
            +---> 2. Boundary Shifts (+ 1, - 1)
            +---> 3. Boolean Flips (True -> False)
            +---> 4. Return Value Inversions (return 0 -> return 1)
            |
            v
   Run Test Suite Against Each Mutant
```

#### Mutation Operator Classes
AROP implements five deterministic operator families:

1. **Relational and Equality Flips**:
   - `==` replaced by `!=`
   - `!=` replaced by `==`
   - `===` replaced by `!==` (JS/TS)
   - `<` replaced by `>=`
   - `<=` replaced by `>`
   - `>` replaced by `<=`
   - `>=` replaced by `<`

2. **Boundary and Arithmetic Shifts**:
   - `+` replaced by `-`
   - `-` replaced by `+`
   - `*` replaced by `/`
   - `+= 1` replaced by `-= 1`
   - `x + 1` replaced by `x - 1`

3. **Boolean Logic Inversions**:
   - `true` replaced by `false`
   - `false` replaced by `true`
   - `and` / `&&` replaced by `or` / `||`
   - `or` / `||` replaced by `and` / `&&`

4. **Return Value Mutations**:
   - `return True` replaced by `return False`
   - `return 0` replaced by `return 1`
   - `return nil` / `return None` replaced by error/sentinel return

5. **Assignment and Nil Replacements**:
   - `nil` replaced by non-nil pointer dummy
   - `err != nil` inverted to `err == nil`

#### Mutation Kill Score Calculation
For each mutant $m_i \in \{m_1, \dots, m_M\}$, the target test suite is executed:
- **Killed ($K$)**: The test suite fails (non-zero exit code or assertion failure). This proves the tests effectively guarded against the defect.
- **Surviving ($S$)**: The test suite passes. This proves the test suite failed to test the logic, indicating inadequate test coverage.
- **Timed Out ($T_{out}$)**: The mutant induced an infinite loop and was killed by process timeout. Counted as killed.

$$\text{Mutation Score} = \left( \frac{K + T_{out}}{M} \right) \times 100\%$$

To clear the verification gate, the score must satisfy:
$$\text{Mutation Score} = 100.0\% \quad (S = 0)$$

### 4.3 Recording Verification
Once tests kill all mutants, the verification proof is recorded:
```bash
read-engine verify <DIFF_HASH> "<VERIFICATION_LOG_OUTPUT>"
```
The engine:
1. Validates status is `Executing`.
2. Hashes the verification log using SHA-256 to form `verify_hash`.
3. Records `staged_diff_hash` and `verify_hash` into `.method/STATE.json`.
4. Sets status to `Verified`.
5. Logs `VERIFIED staged_diff_hash=... verify_hash=...` into `WAL.log`.

---

## 5. Master Override: The Escape Hatch

### 5.1 Purpose and Invariants
In production engineering, certain legitimate changes produce surviving mutants that cannot or should not be tested:
- Pure logging additions (e.g., adding debug log lines).
- Non-functional refactoring or metric annotations.
- Emergency security hotfixes under time-critical incidents.

Rather than weakening verification thresholds, READ provides the **Master Override**:
- An authorized human operator can force state into `Verified`.
- Bypassing the gate requires cryptographic accountability.

### 5.2 Master Override Cryptographic Proof
The operator executes:
```bash
read-engine override "<OPERATOR_ID>" "<RATIONALE>"
```

The engine enforces strict preconditions:
1. **Rationale Minimum Length**: The rationale must be at least 30 characters explaining the engineering justification. Vague entries such as "override" or "tests pass" are rejected.
2. **Override Hash Generation**:
   $$\text{override\_hash} = \text{"OVR-"} \parallel \text{SHA-256}(\text{rationale} \parallel \text{operator\_id} \parallel \text{timestamp})$$
3. **WAL Tombstone**:
   A `MASTER_OVERRIDE` event is appended to `WAL.log` containing operator identity, rationale string, and the generated override hash. Provides integrity and auditability via WAL chaining; does NOT provide non-repudiation.
4. **State Transition**:
   `verify_hash` in `STATE.json` is set to `override_hash`, and status transitions to `Verified`.

---

## 6. Stage 4: Handoff

### 6.1 Mechanical Invariant Verification
The final transition to `HandoffComplete` is mechanically gated:
```bash
read-engine handoff
```

The CAS engine executes the following verification checks:

1. **Status Check**: Status must be `Verified`. If status is `Executing` or `Defined`, handoff fails immediately with `CasLockViolation`.
2. **Verify Hash Non-Null**: `verify_hash` must be present and non-empty.
3. **WAL Audit Integrity**:
   - The engine parses `WAL.log` from start to finish.
   - For standard verifications, a `VERIFIED` event with matching `verify_hash` must exist in the WAL.
   - For overrides, a `MASTER_OVERRIDE` event with matching `OVR-...` hash must exist.
   - If `STATE.json` was manually edited to forge a `verify_hash` without a matching WAL record, handoff is blocked:
     `CasLockViolation: State tampering detected! verify_hash has no matching cryptographic audit record in WAL.log`
4. **Diff Drift Detection**:
   - The engine computes the current `git diff --staged`.
   - The current diff hash must match the `staged_diff_hash` recorded during verification.
   - If files were modified or staged after running verification, handoff is blocked:
     `CasLockViolation: Staged diff hash drift detected! Verified diff is 'A' but current diff is 'B'. Re-verification required.`

### 6.2 Sealing the Task
Upon passing all checks:
- Status transitions to `HandoffComplete`.
- `state_revision` is incremented.
- `HANDOFF` event is logged to `WAL.log`.
- The task contract is fulfilled and safe for Git commit and branch merge.

---

## 7. Declarative Zero-Command Operational Model (BMAD-Style)

READ can be operated entirely without manual CLI commands, functioning through declarative repository-native markdown contracts and git hooks, mirroring the BMAD user experience while enforcing deterministic verification:

### 7.1 How It Works
1. **Contract Declaration**:
   The developer or AI agent edits `.read/TASK.md` to define scope and acceptance criteria.
2. **Code Implementation**:
   Source files and unit tests are edited normally.
3. **Autonomous Pre-Commit Gate**:
   Running standard `git commit` automatically triggers `.read/hooks/pre-commit`:
   - Hashes `.read/TASK.md` to establish the contract hash.
   - Computes `git diff --staged` SHA-256.
   - Auto-discovers and executes the workspace test runner.
   - Generates up to 10 AROP AST mutants on the staged diff.
   - Verifies test suite kill score ($\ge 80\%$).
   - Automatically writes `.read/STATE.json` and `.read/EVIDENCE.json`.
   - Stages both files into the commit and seals handoff.
4. **Adversarial Failure Rejection**:
   If tests fail or any mutant survives, the pre-commit hook automatically blocks the commit, logs exact failure diagnostics into `.read/EVIDENCE.json`, and outputs the surviving lines in the terminal.

Zero CLI commands are typed by the user or agent during daily development. The methodology is enforced natively by git.
