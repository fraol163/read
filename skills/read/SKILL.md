---
name: read-framework
description: >-
  Resource-conscious, Evidence-driven AI Development (READ).
  Guides the agent through deterministic C0-AST context extraction,
  Compare-And-Swap state machine locking, cryptographic WAL verification,
  and AROP 100% mutation gating before code commits.
---

# READ Framework: Agent Operational Guide

The READ Framework replaces conversational QA assumptions with deterministic, mechanical systems verification.

## Core Operational Protocol

### 1. Read Before Touching Code (Define Stage)
- Never load entire 10,000+ token files into context.
- Call the `read_ast_context` MCP tool (or run `read-engine ast-context <file>`).
- Inspect the deterministic symbol signatures and dependency closures for free.
- Verify task contracts in `TASK.md` and check `read_status`.

### 2. Implement Within Bounded Limits (Build Stage)
- Keep changes scoped to the contract definition.
- Transition state to executing via `read_build`.
- Run local unit tests to ensure happy path and error cases execute cleanly.

### 3. Mechanical Verification (Verify Stage)
- Plain passing tests are not sufficient to claim correctness.
- Compute the staged diff hash with `read_diff_hash`.
- Generate targeted AST mutants on the staged diff with `read_mutate`.
- Run the test suite against the mutants.
- Evaluate the kill ratio with `read_eval_mutation_score`.
- If any mutant survives, strengthen the test assertions until 100% of mutants are killed.
- Record verification with `read_verify`.

### 4. Zero-Token Crash Recovery
- If a process dies or state is interrupted, call `read_recover`.
- The engine replays the cryptographic `WAL.log` hash chain and restores consistent state without re-spending model tokens.

### 5. Evidence Audit and Reporting
- Generate machine-readable YAML state with `read_status_yaml`.
- Generate sealed HTML verification reports with `read_report_html`.
- Run micro-DSL assembly sequences for token efficiency with `read_asm`.
