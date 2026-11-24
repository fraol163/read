# Active Task Contract (TASK.md)

## Task Metadata
- Task ID: TASK-001
- Created: 2026-11-22T00:00:00Z
- Status: defined
- Risk Level: R1 (Reversible local implementation)

## Task Objective
[Describe the exact functionality, bugfix, or refactoring required. Be concrete and specific.]

## Affected Files & Scope
- Target Files:
  - `[e.g., src/auth/token.py]`
  - `[e.g., tests/test_token.py]`
- Protected / Out-of-Scope Files:
  - `[e.g., src/database/schema.sql]`

## Mandatory Acceptance Criteria
| ID | Criterion Description | Expected Command | Expected Exit |
|---|---|---|---|
| AC-1 | Target unit tests pass | `pytest tests/test_token.py -v` | `0` |
| AC-2 | No regressions in existing test suite | `pytest tests/ -v` | `0` |
| AC-3 | Linting passes with 0 warnings | `ruff check .` | `0` |
| AC-4 | AROP mutation verification kills 100% of mutants (zero survivors) | `read-engine mutate` | `0` |

## Non-Goals for this Task
- Do not alter public API signatures of unrelated modules.
- Do not add external package dependencies.
