# READ Framework Master Architecture and Verification Report

**Framework Identity:** READ (Resource-conscious, Evidence-driven AI Development)  
**Date of Record:** 2026-11-23  
**Release Target:** v0.7.1  
**Working Git Branch:** `report-review-v0.7.1`  
**Host Environment:** Linux 7.2.15-1-omarchy x86_64, AMD Ryzen 3 3250U, 5.7 GiB RAM

CRITICAL GATE STATUS:
The outside repository mutation verification gate on veloce-acceptance/github-repo strictly FAILED (Exit Code: 1). Score: 75.0% (6 killed, 2 survived). The framework invariant requires a 100% kill rate, so verification was rejected.

---

## 1. Executive Summary and Vision

READ replaces conversational multi-agent chat loops with mechanical systems verification. Rather than relying on large language model personas to critique code and declare tasks done in prompts, READ enforces machine-verifiable contracts, local AST analysis, 100% mutation testing, and append-only cryptographic logging.

Detailed vision and philosophical foundations are documented in [docs/VISION.md](file:///home/renan/Downloads/me/methodology_session/read/docs/VISION.md).

### Four Core Contrasts with BMAD
BMAD stands for Breakthrough Method for Agile AI-Driven Development (`bmad-code-org/BMAD-METHOD`, release `v6.12.4`). READ contrasts with BMAD across four foundational dimensions:

1. Persona Theater vs Deterministic Routing: BMAD relies on conversational prompts between simulated roles. READ routes tasks through deterministic AST analysis and Compare-And-Swap (CAS) state machine transitions.
2. Narrative PRDs vs Machine-Verifiable Contracts: BMAD uses natural language markdown stories. READ freezes machine-readable contracts anchored by the SHA-256 hash of `TASK.md`.
3. Conversational QA vs Cryptographic Evidence: BMAD relies on agent conversational approval. READ requires staged git diff hashes, write-ahead logging (`WAL.log`), and a strictly 100% mutation verification kill score.
4. Unmetered Abundance vs Strict Resource Finitude: BMAD assumes unbounded context windows and cloud compute. READ bounds AST context to local dependencies, limits mutants to a maximum of 10 targeted alterations, and calibrates process memory to 30% of available RAM.

---

## 2. Technical Specification Reference

The operational specification for the framework is codified in [docs/SPECIFICATION.md](file:///home/renan/Downloads/me/methodology_session/read/docs/SPECIFICATION.md) using RFC 2119 normative language. Every rule links directly to its implementing source file and enforcing test.

Key systems include:
- Canonical State Path: Runtime state is stored exclusively in `.read/STATE.json`. Legacy `.method/` directories are automatically migrated and removed on detection.
- Compare-And-Swap State Engine (`read-core`): Coordinates transitions across Defined, Executing, Verifying, Verified, and Handoff states. Prevents race conditions using revision checks and POSIX advisory locking.
- Cryptographic Write-Ahead Log (`WAL.log`): Chained SHA-256 record entries. Recovers consistent state following process crashes.
- Cryptographic Git Tag Anchor: Subcommands `anchor-wal` and `verify-wal-anchor` bind the latest WAL entry hash to git tag `read-wal-head`, detecting rewritten log chains.
- AST Context Resolver: Extracts declarations and dependencies across 8 languages (Python, TypeScript/JavaScript, Go, Rust, Java, C#, C++, Ruby) using local line scanning.
- AROP Mutation Verification Gate: Mutates added or modified diff lines up to a 10-mutant ceiling. Demands a strictly 100.0% mutation kill score before allowing handoff.
- Process Supervisor (`read-daemon`): Dynamically discovers CLI agent harnesses, manages PTY sessions over WebSockets, enforces dynamic memory limits, and terminates child process trees on timeout.

---

## 3. Empirical Evidence and Test Logs

All test suites and benchmark measurements were executed directly on the host machine. Output logs are preserved in `evidence/logs/`.
The evidence dossier is documented in [evidence/EVIDENCE.md](file:///home/renan/Downloads/me/methodology_session/evidence/EVIDENCE.md).
Suites 12 and 13 represent local shell scenarios executing real binary steps rather than cloud GitHub Actions runners. The CI gate remains UNTESTED on cloud runners because no git remote is configured.

### Automated Test Suite Execution Summary

| Suite # | Target Component | Command | Exit Code | Pass Count | Fail Count | Evidence Log Link |
|---|---|---|---|---|---|---|
| 01 | Lifecycle Verification | `bash read/tests/test_e2e.sh` | 0 | 15 | 0 | [01_test_e2e.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/01_test_e2e.log) |
| 02 | Adversarial Chaos Suite | `bash read/tests/test_chaos.sh` | 0 | 10 | 0 | [02_test_chaos.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/02_test_chaos.log) |
| 03 | BMAD Zero-Command Parity | `bash read/tests/test_bmad_zero_command.sh` | 0 | 6 | 0 | [03_test_bmad_zero_command.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/03_test_bmad_zero_command.log) |
| 04 | Full System Integration | `bash read/tests/test_full_system.sh` | 0 | 9 | 0 | [04_test_full_system.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/04_test_full_system.log) |
| 05 | Autonomous Pipeline | `bash read/tests/test_pipeline.sh` | 0 | 6 | 0 | [05_test_pipeline.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/05_test_pipeline.log) |
| 06 | Packaging and Distribution | `bash read/tests/test_packaging.sh` | 0 | 8 | 0 | [06_test_packaging.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/06_test_packaging.log) |
| 07 | Realtime WebSockets | `bash read/tests/test_live_realtime.sh` | 0 | 8 | 0 | [07_test_live_realtime.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/07_test_live_realtime.log) |
| 08 | Rust Core Unit Tests | `cargo test --manifest-path read/read-core/Cargo.toml` | 0 | 58 | 0 | [08_cargo_test_read_core.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/08_cargo_test_read_core.log) |
| 09 | Go Daemon Unit Tests | `cd read/read-daemon && go test -v -count=1 ./...` | 0 | 57 | 0 | [09_go_test_read_daemon.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/09_go_test_read_daemon.log) |
| 10 | Go Microbenchmarks | `cd read/read-daemon && go test -bench . -benchmem ./...` | 0 | 2 | 0 | [10_go_test_bench.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/10_go_test_bench.log) |
| 11 | Orchestrator Test Suite | `cd read/read-daemon && go test -v -count=1 ./orchestrator` | 0 | 4 | 0 | [11_go_test_orchestrator.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/11_go_test_orchestrator.log) |
| 12 | CI Gate Local Scenario (untested on cloud runners) | `bash evidence/test_ci_workflow.sh` | 0 | 3 | 0 | [12_ci_gate_workflow.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/12_ci_gate_workflow.log) |
| 13 | WAL Tamper Local Scenario (local shell simulation) | `bash evidence/test_wal_tamper.sh` | 0 | 3 | 0 | [13_wal_tamper_test.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/13_wal_tamper_test.log) |
| 14 | Outside Repo Mutation Gate (veloce-acceptance) | `python3 evidence/eval_outside_mutation.py` | 1 | 6 | 2 | [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log) |
| 14a | Outside Repo Integration Check (colorama) | `python3 scratch/generate_colorama_log.py` | 0 | 10 | 0 | [14a_colorama_integration_check.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14a_colorama_integration_check.log) |
| 15 | Tutorial Machine Timing | `python3 evidence/time_tutorial.py` | 0 | 8 | 0 | [15_tutorial_timing.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/15_tutorial_timing.log) |

Suite 01 note: The 15 passed checks represent assertion lines executed within `read/tests/test_e2e.sh`. 6 of these steps (Step 6 for Python, and Steps 6c through 6g for TypeScript, Java, C#, C++, and Ruby) check counts chosen based on the mutator's output. The arithmetic checks (`eval-mutation-score $((N-1)) N`) verify only the scoring arithmetic, but they do not test real gate rejection. Real gate rejection was tested by the end-to-end run against an outside repository with an existing test suite (veloce run, Suite 14), which failed with exit code 1.

### Measured Benchmark Metrics (10 Runs Each)

- Staged Diff Hashing: Mean 8.835 ms (Min 8.053 ms, Max 10.500 ms, StdDev 0.706 ms). Dominated by git process spawn overhead.
- Crash Recovery: Mean 11.490 ms (Min 7.586 ms, Max 16.455 ms, StdDev 3.113 ms). Replays WAL entries without token re-spend.
- AST Context Extraction Across Languages:
  - Python: Mean 2.243 ms
  - TypeScript: Mean 1.921 ms
  - Go: Mean 1.762 ms
  - Rust: Mean 1.828 ms
  - Java: Mean 2.286 ms
  - C#: Mean 2.374 ms
  - C++: Mean 1.962 ms
  - Ruby: Mean 1.927 ms
- 200 Functions Fixture Variance and Contention:
  - Benchmark Location: Located strictly in test code within `read/read-core/src/ast/mod.rs` inside `#[cfg(test)] mod tests`. Zero benchmark code exists in engine runtime paths.
  - Measurement History Across Commits: Single-run release in-process extraction was 0.375 ms on commit `3ce41c4`. Across 10 repeated runs on commit `7ce0a00`, release in-process extraction on an idle host averaged 0.140 ms mean (Min 0.122 ms, Max 0.226 ms, StdDev 0.032 ms) and debug in-process averaged 1.225 ms mean.
  - Baseline Contention on Commit `ca31c10`: Executed in a separate detached clone directory (`/tmp/check_ca31c10`) with benchmark test added inside `mod tests`. Under 4-thread CPU contention (3 concurrent worker threads), baseline unoptimized extraction averaged 3.659 ms mean (Min 2.670 ms, Max 7.658 ms). The 7.658 ms spike is recorded strictly as one empirical observation showing latency variance under parallel CPU load.
  - Optimized Contention on Current Commit `a58abd7`: Under 4-thread CPU contention (3 concurrent worker threads), release profile averaged 0.144 ms mean (Min 0.134 ms, Max 0.192 ms) and debug profile averaged 1.982 ms mean (Min 1.435 ms, Max 3.524 ms). Both assertions in `test_extraction_latency_under_5ms` and `test_extraction_latency_contention_10_runs` strictly require under 5 ms and pass.
  - CLI Process: Mean 2.014 ms in release profile (StdDev 0.077 ms) and Mean 3.745 ms in debug profile (StdDev 0.323 ms).
  - Normative Threshold: The under 5 ms threshold applies normatively to the release build profile for runtime execution.
- Large Test Suite Hook (500 tests): Mean 127.431 ms.
- PTY Stream Allocation: 56,628 ns/op, 974 B/op, 4 allocs/op.
- Real External Repo Mutation Test: Evaluated against `veloce-acceptance/github-repo` (commit `7732578`) on `backend/resources/preview.py`. Reverted bug fix commit `7a34571`, staged the diff (33 additions, 13 deletions across preview.py only, zero test file changes), and evaluated mutants using `python3 evidence/eval_outside_mutation.py`. Result: 6 killed, 2 survived on line 259 error handling path (75.0% score). Gate status: strictly FAILED. Empirical line coverage (`pytest --cov`) proves lines 258-259 had zero executions during the test run, proving the survivors represent genuine test gaps.
- Tutorial Completion Timing: Machine execution 176.72 ms (0.18 seconds across 8 steps). Human student trial completion time: Not measured (machine execution: 0.18s, human trial unconducted).
- Reproducible Binary Builds: Verified identical SHA-256 hashes across clean git clones from commit `a58abd7` on this Linux host for Rust release binary `75b0e382...` and Go daemon binary with `-trimpath` `dcecffce...`.

### Trial Parameters and Governance Draft Proposals (Awaiting User Decisions)

The following parameters are draft proposals submitted for user decision. Trial execution is held until the user explicitly sets each value:

1. Draft Proposal 1 (Numeric Trial Thresholds): 15-minute task completion ceiling, strictly 100.0% mutation kill rate, maximum 3 remediation retries, 80% cohort completion rate.
2. Draft Proposal 2 (Enforcement Policy): Hard block vs warning mode during trial (hard block strictly enforces gate but may block honest student commits, while warning mode records block rate without aborting commits).
3. Draft Proposal 3 (User Group and Cohort Size): 10 participants, junior to mid-level engineers (1 to 3 years experience) with Python proficiency and git familiarity who have zero prior exposure to READ. 3 evaluation tasks.

---

## 4. Claim Audit Summary

The empirical status of all technical claims is tracked in [evidence/CLAIM_AUDIT.md](file:///home/renan/Downloads/me/methodology_session/evidence/CLAIM_AUDIT.md).
Key claim adjustments include:
- Refuted: BMAD acronym was corrected from "Break, Measure, Adapt, Deliver" to "Breakthrough Method for Agile AI-Driven Development".
- Refuted: "Zero allocation PTY streaming" was refuted by microbenchmarks (974 B/op, 4 allocs/op) and removed.
- Refuted for Managed Runtimes: Dynamic `RLIMIT_AS` (30% RAM ceiling) causes 64-bit JVM and .NET CoreCLR to abort on startup because it restricts virtual memory address space.
- Narrowed: Crash recovery claims were narrowed from "under 10 ms" to 7.5 ms through 16.5 ms based on empirical measurements.
- Refuted: "Master Override provides non-repudiation" was corrected to "provides cryptographic integrity and auditability via SHA-256 hash chaining".
- Refuted: "Outside codebase achieves 100% mutation kill score" was refuted by testing on `veloce-acceptance/github-repo` (75.0% score, gate strictly failed on unexercised error path).

---

## 5. Security Analysis and Threat Model

A dedicated security evaluation is documented in [read/docs/THREAT_MODEL.md](file:///home/renan/Downloads/me/methodology_session/read/docs/THREAT_MODEL.md).

Key findings:
1. PTY WebSocket Exposure: `/api/v1/pty` allows arbitrary shell execution under daemon user privileges. Child processes are terminated via process group signals on disconnect.
2. Static Token Risk: Bearer tokens are static and lack granular scopes. Passing `--auth-token` via CLI exposes secrets in `/proc`.
3. Reverse Proxy XFF Spoofing: The daemon trusts incoming `X-Forwarded-For` unconditionally. Sending `X-Forwarded-For: 127.0.0.1` bypasses rate limits. Reverse proxies must strip client-supplied headers.
4. Advisory Lock Limits: POSIX `libc::flock` coordinates only cooperating processes. Uncooperating text editors can write directly to state files. Integrity is guarded by WAL hash verification.
5. Server-Side CI Enforcement: Developers can bypass client-side pre-commit hooks via `git commit --no-verify`. Server-side CI enforcement has been implemented in [.github/workflows/read-gate.yml](file:///home/renan/Downloads/me/methodology_session/.github/workflows/read-gate.yml), but remains UNTESTED on cloud runners until a remote repository is configured and pushed.

---

## 6. Architecture Review and Known Gaps

- Independent Architecture Review: [read/docs/ARCHITECTURE_REVIEW.md](file:///home/renan/Downloads/me/methodology_session/read/docs/ARCHITECTURE_REVIEW.md).
- System Gaps and Roadmap: [read/docs/GAPS.md](file:///home/renan/Downloads/me/methodology_session/read/docs/GAPS.md).
- Student Onboarding Guide: [read/docs/TUTORIAL.md](file:///home/renan/Downloads/me/methodology_session/read/docs/TUTORIAL.md).
- Detailed Version History: [CHANGELOG.md](file:///home/renan/Downloads/me/methodology_session/CHANGELOG.md).

---

## 7. Open User Decisions

The following 5 parameters remain open draft proposals awaiting explicit user confirmation before any trial begins. Trial execution is held until the user explicitly sets each value:

1. Time limit per task, in minutes (Draft proposal: 15 minutes per task, based on 5 to 10 minute human tutorial baseline).
2. Minimum cohort completion rate (Draft proposal: 80 percent completion rate, meaning at least 8 of 10 participants complete all tasks).
3. Remediation retry cap (Draft proposal: maximum 3 test authoring attempts per task before escalation).
4. Cohort size and profile (Draft proposal: exactly 10 junior to mid-level software engineers with 1 to 3 years experience and zero prior exposure to the READ framework).
5. Enforcement mode for the first cohort: Hard block or warning mode.
Recommendation on item 5: Start in warning mode. A hard block from day one will likely block honest commits from equivalent mutants and untested error paths, while warning mode measures the real gate rejection rate before enforcing hard blocks.

Additional Infrastructure Note:
CI Gate Cloud Status: The repository has no configured git remote (`git remote -v` outputs nothing). The workflow remains labeled as UNTESTED on cloud infrastructure until a remote is configured and pushed.
