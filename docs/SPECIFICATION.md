# READ Framework Technical Specification (v0.1.0)

This specification defines the normative operational rules for the READ Framework.
The key words MUST, MUST NOT, REQUIRED, SHALL, SHALL NOT, SHOULD, SHOULD NOT, RECOMMENDED, and MAY in this document are to be interpreted as described in RFC 2119.

## 1. Canonical State Path and Structure

1.1 Canonical State File and Migration:
- The system MUST store its runtime state exclusively in `.read/STATE.json` and its log in `.read/WAL.log`.
- If an existing legacy `.method` directory is detected in the repository without `.read`, the engine MUST automatically migrate `.method` to `.read`.
- Implementation: `read-core/src/state/mod.rs` and `read-daemon/config/config.go`.
- Enforcing Test: `read-core/src/state/mod.rs` (`test_migration_from_legacy_method_path`).

1.2 State Schema:
- The state file MUST contain the fields: `task_id`, `contract_hash`, `base_commit`, `status`, `state_revision`, `staged_diff_hash`, `verify_hash`, and `updated_at`.
- The `status` field MUST take one of the following enumerated values: `Defined`, `Executing`, `Verifying`, `Verified`, `HandoffComplete`.
- Implementation: `read-core/src/state/mod.rs` (struct `State`).
- Enforcing Test: `tests/test_e2e.sh` (Step 1-10 Lifecycle Validation).

## 2. Compare-And-Swap (CAS) State Transitions

2.1 Concurrency and Advisory Locking:
- The state engine MUST acquire an exclusive POSIX advisory lock via `libc::flock` on `.read/STATE.json` before reading or mutating state.
- The engine MUST release the lock immediately upon completing the disk write.
- Implementation: `read-core/src/state/mod.rs` (functions `flock_acquire` and `flock_release`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_flock_concurrency`).

2.2 State Revision Increment:
- Every successful state transition MUST increment `state_revision` by exactly 1.
- If an expected revision is supplied during a CAS transition, the engine MUST abort with `CasLockViolation` if the actual revision does not match.
- Implementation: `read-core/src/state/mod.rs` (`transition_handoff_cas`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_cas_lock_violation_on_stale_revision`).

2.3 Staged Diff Integrity:
- The engine MUST record the SHA-256 hash of the staged git diff in `staged_diff_hash` during the verification phase.
- During the handoff transition, the engine MUST recalculate the current staged git diff hash.
- If the recalculated diff hash differs from the recorded `staged_diff_hash`, the engine MUST reject the handoff with `DiffHashMismatch`.
- Implementation: `read-core/src/state/mod.rs` (`transition_handoff_cas`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_handoff_cas_diff_hash_mismatch`).

## 3. Cryptographic Write-Ahead Logging (WAL)

3.1 Append-Only Log Chaining:
- The engine MUST record every valid state transition as a single line in `.read/WAL.log`.
- Each WAL entry MUST contain: timestamp, previous entry hash, event name, payload string, and entry hash.
- The entry hash MUST be calculated as `SHA256(timestamp + "|" + prev_hash + "|" + event + "|" + payload)`.
- For the initial entry, `prev_hash` MUST be 64 zero hex digits (`0000000000000000000000000000000000000000000000000000000000000000`).
- Implementation: `read-core/src/state/mod.rs` (`append_wal`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_wal_chain_integrity`).

3.2 Crash Recovery:
- The engine MUST provide a crash recovery routine that parses the entire WAL from beginning to end.
- The recovery routine MUST recalculate and verify the hash chain of every record.
- If any record fails hash verification, recovery MUST abort with `TamperedWalChain`.
- The crash recovery routine executes without API token cost. Measured crash recovery latency ranges from 7.5 ms to 16.5 ms (mean: 11.49 ms across 10 iterations).
- Implementation: `read-core/src/state/mod.rs` (`recover`).
- Enforcing Test: `tests/test_chaos.sh` (Scenario 2: WAL Tamper Detection).

3.3 Git Tag Anchoring:
- Upon completing a valid handoff transition, the engine MUST anchor the latest WAL entry hash into the repository by creating or updating the git tag `read-wal-head`.
- The engine MUST verify this anchor during recovery. If the git tag `read-wal-head` points to a hash that does not match the latest entry in `.read/WAL.log`, recovery MUST fail.
- Tag Anchoring Security Boundary: Git tag anchoring requires remote server-side tag protection (e.g. protected tags on GitHub or GitLab). If an attacker possesses write permissions allowing force-pushing tags to the remote repository, they can overwrite both the local log and the tag. Server-side branch and tag protection rules are strictly required to enforce non-repudiation across distributed peers.
- Implementation: `read-core/src/state/mod.rs` (`anchor_wal_in_git` and `verify_wal_anchor`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_wal_git_anchor_verification_and_rewritten_chain_detection`).

## 4. AST Context Extraction (C0-C3)

4.1 Supported Languages:
- The AST engine MUST extract symbol signatures and dependencies for: Python, TypeScript, JavaScript, Go, Rust, Java, C#, C++, and Ruby.
- Implementation: `read-core/src/ast/mod.rs` (enum `SourceLanguage`).
- Enforcing Test: `read-core/src/ast/mod.rs` (`test_extract_python_symbols`, `test_extract_typescript_symbols`, etc.).

4.2 Token Footprint and Resource Bounds:
- The AST engine MUST parse source files line-by-line without invoking full compilation toolchains.
- Context extraction MUST NOT exceed 20 MB file size limit per file. Files exceeding this ceiling MUST be skipped.
- Context traversal MUST NOT exceed 500 dependent files.
- Implementation: `read-core/src/ast/mod.rs`.
- Enforcing Test: `read-core/src/ast/mod.rs` (`test_ast_context_resolver_large_file`).

4.3 Graceful Degradation:
- If a source file contains syntax errors or partial edits, the parser MUST extract all valid surrounding declarations and set `degraded: true` in the output manifest.
- Implementation: `read-core/src/ast/mod.rs` (method `extract_symbols_and_imports`).
- Enforcing Test: `read-core/src/ast/mod.rs` (`test_graceful_degradation_on_malformed_syntax`).

4.4 Latency Threshold and Build Profile:
- The AST engine MUST extract deterministic symbol signatures and dependencies with an in-process execution latency of under 5 ms per file on release builds.
- Normative Applicability: The shipped binary is built in release mode (`cargo build --release -p read-core --bin read-engine`). The binary hashes do not change between builds, and the under 5 ms release latency threshold is the sole binding requirement. In release mode, extraction executes in 0.14 ms to 0.18 ms across test workloads.
- Measurement Methodology: The latency test methodology executes an unmeasured warm-up run followed by calculating the median duration of 10 runs under a serialized mutex lock. This replaces earlier unbuffered single-run measurements to guard against operating system CPU scheduler context-switch jitter on multi-threaded hosts while avoiding the bias of minimum-of-three filtering.
- Test Profile Configuration: The unit test suite in `read/read-core/Cargo.toml` executes under `[profile.test] opt-level = 1` to ensure compiler optimizations are active during concurrent test runner execution on 2-core hardware. Under baseline unoptimized debug code, measured contention spiked to 7.658 ms in [20_ast_latency_contention.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/20_ast_latency_contention.log).
- Implementation: `read-core/src/ast/mod.rs` (`extract_symbols_and_imports`).
- Enforcing Test: `read-core/src/ast/mod.rs` (`test_extraction_latency_under_5ms` and `test_extraction_latency_contention_10_runs`).

## 5. AROP Mutation Verification Gate

5.1 Target Scope:
- The mutation engine MUST target only lines added or modified in the staged git diff.
- The mutation engine MUST ignore non-source files, documentation files, and framework metadata files located in `.read/` or `.method/`.
- Implementation: `read-core/src/mutation/mod.rs` (`generate_diff_mutants`).
- Enforcing Test: `tests/test_pipeline.sh` (Pipeline End-to-End Execution).

5.2 Mutant Budget Cap:
- The engine MUST cap mutant generation at a maximum of 10 mutants per diff (the default ceiling).
- Mutants MUST be selected sequentially from diff hunks until the ceiling is reached.
- Implementation: `read-core/src/mutation/mod.rs` (`DEFAULT_MAX_MUTANTS`).
- Enforcing Test: `read-core/src/mutation/mod.rs` (`test_bounded_mutant_cap`).

5.3 Mutation Operators:
- The engine MUST support the following operator classes:
  1. INVERT_EQUALITY (`==` to `!=`, `!=` to `==`).
  2. INVERT_COMPARISON (`<` to `<=`, `>` to `>=`, etc.).
  3. FLIP_BOOLEAN (`true` to `false`, `false` to `true`).
  4. RETURN_NULL / RETURN_NONE (`return x` to `return null` or `return None`).
  5. BOUNDARY_SHIFT (`n` to `n+1` or `n-1`).
- Implementation: `read-core/src/mutation/` (`mod.rs`, `python.rs`, `ts_js.rs`, `java.rs`, `csharp.rs`, `cpp.rs`, `ruby.rs`).
- Enforcing Test: `read-core/src/mutation/` unit test suites.

5.4 Compile-Error-is-Invalid Rule and Outcome Evaluation:
- A mutant is KILLED if executing the target project test suite produces a non-zero exit code caused by test assertion failures, exceptions, crashes, or timeouts.
- A mutant SURVIVES if executing the target project test suite returns exit code 0.
- Compile-Error-is-Invalid Rule: Mutants that fail compiler type-checking or syntax validation are INVALID mutants, not kills or survivors. Generating uncompilable mutants tests compiler diagnostics rather than test suite assertion coverage. Therefore, mutator operators MUST suppress invalid mutants:
  1. Java: `RETURN_NULL` MUST be suppressed for primitive return types (`boolean`, `int`, etc.) and relational expressions.
  2. TypeScript: `RETURN_UNDEFINED` MUST be suppressed for functions returning boolean literals under `strictNullChecks`.
- Implementation: `read-core/src/mutation/java.rs` and `read-core/src/mutation/ts_js.rs`.
- Enforcing Test: `read-core/src/mutation/java.rs` (`test_java_diff_mutator_skips_primitive_boolean_return_null`) and `read-core/src/mutation/ts_js.rs` (`test_ts_diff_mutator_skips_boolean_return_undefined`).

5.5 Pass Threshold Invariant:
- The verification gate MUST require a strictly 100.0% mutation kill score (all generated mutants must be killed).
- If any mutant survives, the verification gate MUST reject the commit and mark the stage as failed.
- Implementation: `read-daemon/orchestrator/orchestrator.go` (`stage4MutationVerification`).
- Enforcing Test: `tests/test_chaos.sh` (Scenario 6: Mutation Gate Rejection on Weak Tests).

## 6. Process Supervision and Resource Limits

6.1 Dynamic RLIMIT Calibration:
- The daemon MUST query available host memory at startup (via `/proc/meminfo` or system APIs).
- The daemon MUST set `RLIMIT_AS` (virtual address space) for child processes to 30% of detected available RAM.
- If memory detection fails, the daemon MUST default child process `RLIMIT_AS` to 512 MB.
- Implementation: `read-daemon/security/process.go` (`CalibrateMemoryLimit`).
- Enforcing Test: `read-daemon/security/process_test.go` (`TestDynamicRLIMITCalibration`).

6.2 Process Group Termination:
- The daemon MUST assign each spawned process a dedicated process group (`setpgid`).
- Upon timeout or client cancellation, the daemon MUST send `SIGTERM` followed by `SIGKILL` to `-pgid`.
- Implementation: `read-daemon/security/process.go` (`KillProcessTree`).
- Enforcing Test: `read-daemon/security/process_test.go` (`TestProcessGroupTermination`).

## 7. Daemon Gateway Security

7.1 Network Binding:
- If `--tls-cert` and `--tls-key` flags are absent, the daemon MUST bind strictly to `127.0.0.1`.
- The daemon MUST reject binding to `0.0.0.0` or external network interfaces without TLS configuration.
- Implementation: `read-daemon/gateway/gateway.go`.
- Enforcing Test: `read-daemon/gateway/gateway_test.go` (`TestNonLoopbackWithoutTLSRejected`).

7.2 Bearer Token Authentication:
- If `--auth-token` is configured, all API endpoints except `/api/v1/status` MUST require `Authorization: Bearer <token>`.
- Token comparison MUST use `crypto/subtle.ConstantTimeCompare` to prevent timing attacks.
- Implementation: `read-daemon/gateway/gateway.go` (`AuthMiddleware`).
- Enforcing Test: `read-daemon/gateway/gateway_test.go` (`TestAuthEnforcement`).

7.3 Rate Limiting:
- Non-whitelisted client IP addresses MUST be limited to a token bucket capacity of 10 burst requests and 2 sustained requests per second.
- Requests exceeding the limit MUST return HTTP status code 429 with a `Retry-After` header.
- Implementation: `read-daemon/gateway/ratelimit.go`.
- Enforcing Test: `read-daemon/gateway/gateway_test.go` (`TestRateLimiting`).

## 8. Master Override Protocol

8.1 Override Execution:
- If verification fails or is blocked, an operator MAY apply a manual override via `read-engine override <OPERATOR> <RATIONALE>`.
- The operator identifier MUST NOT be empty.
- The rationale string MUST contain at least 30 characters.
- Implementation: `read-core/src/state/mod.rs` (`master_override`).
- Enforcing Test: `read-core/src/state/mod.rs` (`test_master_override_requires_rationale_length`).

8.2 Override Hashing and Auditability:
- The override entry MUST prefix the verification hash with `OVR-`.
- The override MUST record the operator identity and full rationale into `WAL.log`.
- Implementation: `read-core/src/state/mod.rs`.
- Enforcing Test: `tests/test_chaos.sh` (Scenario 5: Master Override Recovery).
