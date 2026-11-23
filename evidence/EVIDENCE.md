# READ Framework Empirical Evidence Dossier

Date: 2026-11-23
Review Branch: report-review-v0.7.1
Host Environment: Linux 7.2.15-1-omarchy x86_64
Hardware: AMD Ryzen 3 3250U with Radeon Graphics (2 cores, 4 threads), 5.7 GiB RAM
Compilers and Runtimes: rustc 1.99.1, go 1.27.3, python 3.15.2, dotnet 10.0.403, openjdk 21.0.9

## 1. Automated Test Suite Executions

Every suite was executed directly on the host machine. Output logs are preserved in `evidence/logs/`.
Suites 12 and 13 represent local shell scenarios executing real binary steps rather than cloud GitHub Actions runners.

| Suite # | Test Target | Command Executed | Exit Code | Pass Count | Fail Count | Raw Evidence Log |
|---|---|---|---|---|---|---|
| 01 | End-to-End Lifecycle | `bash read/tests/test_e2e.sh` | 0 | 15 | 0 | [01_test_e2e.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/01_test_e2e.log) |
| 02 | Chaos and Stress Suite | `bash read/tests/test_chaos.sh` | 0 | 10 | 0 | [02_test_chaos.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/02_test_chaos.log) |
| 03 | BMAD Zero-Command Parity | `bash read/tests/test_bmad_zero_command.sh` | 0 | 6 | 0 | [03_test_bmad_zero_command.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/03_test_bmad_zero_command.log) |
| 04 | Full System Integration | `bash read/tests/test_full_system.sh` | 0 | 9 | 0 | [04_test_full_system.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/04_test_full_system.log) |
| 05 | Autonomous Pipeline | `bash read/tests/test_pipeline.sh` | 0 | 6 | 0 | [05_test_pipeline.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/05_test_pipeline.log) |
| 06 | Packaging and Installation | `bash read/tests/test_packaging.sh` | 0 | 8 | 0 | [06_test_packaging.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/06_test_packaging.log) |
| 07 | Realtime WebSockets | `bash read/tests/test_live_realtime.sh` | 0 | 8 | 0 | [07_test_live_realtime.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/07_test_live_realtime.log) |
| 08 | Rust Core Unit Tests | `cargo test --manifest-path read/read-core/Cargo.toml` | 0 | 56 | 0 | [08_cargo_test_read_core.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/08_cargo_test_read_core.log) |
| 09 | Go Daemon Unit Tests | `cd read/read-daemon && go test -v -count=1 ./...` | 0 | 57 | 0 | [09_go_test_read_daemon.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/09_go_test_read_daemon.log) |
| 10 | Go Microbenchmarks | `cd read/read-daemon && go test -bench . -benchmem ./...` | 0 | 2 | 0 | [10_go_test_bench.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/10_go_test_bench.log) |
| 11 | Orchestrator Test Suite | `cd read/read-daemon && go test -v -count=1 ./orchestrator` | 0 | 4 | 0 | [11_go_test_orchestrator.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/11_go_test_orchestrator.log) |
| 12 | CI Gate Local Scenario (untested on cloud runners) | `bash evidence/test_ci_workflow.sh` | 0 | 3 | 0 | [12_ci_gate_workflow.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/12_ci_gate_workflow.log) |
| 13 | WAL Tamper Local Scenario (local shell simulation) | `bash evidence/test_wal_tamper.sh` | 0 | 3 | 0 | [13_wal_tamper_test.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/13_wal_tamper_test.log) |
| 14 | Outside Repo Mutation Gate (veloce-acceptance) | `read-engine mutate staged diff + pytest` | 1 | 6 | 2 | [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log) |
| 14a | Outside Repo Integration Check (colorama) | `python3 scratch/generate_colorama_log.py` | 0 | 10 | 0 | [14a_colorama_integration_check.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14a_colorama_integration_check.log) |
| 15 | Tutorial Machine Timing | `python3 evidence/time_tutorial.py` | 0 | 8 | 0 | [15_tutorial_timing.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/15_tutorial_timing.log) |

## 2. Quantitative Performance Measurements

Detailed measurement scripts were executed via `evidence/measure_all.py` (10 iterations per benchmark).
Raw log: [measurements.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/measurements.log)
Structured data: [measurements.json](file:///home/renan/Downloads/me/methodology_session/evidence/logs/measurements.json)

### 2.1 Staged Diff Hash Latency
Measures spawning `git diff --cached` and computing SHA-256 over 10 consecutive runs:
- Mean: 8.835 ms
- Standard Deviation: 0.706 ms
- Min: 8.053 ms
- Max: 10.500 ms
- Observation: Pure memory hashing takes under 50 microseconds. The 8.8 ms overhead is dominated by spawning the `git` process on Linux.

### 2.2 Crash Recovery Latency
Measures state engine recovery across 10 fresh repositories with uncommitted staged diffs:
- Mean: 11.490 ms
- Standard Deviation: 3.113 ms
- Min: 7.586 ms
- Max: 16.455 ms
- Observation: Measured crash recovery latency ranges from 7.5 ms to 16.5 ms. The previous marketing claim of strictly under 10ms fails on 2-core hardware when git process creation is measured. The claim has been narrowed accordingly.

### 2.3 AST Context Extraction Latency
Measures `read-engine ast-context` across 8 supported programming languages (10 runs each):
- Python: Mean 2.243 ms (Min: 1.549 ms, Max: 3.297 ms, StdDev: 0.582 ms)
- TypeScript: Mean 1.921 ms (Min: 1.567 ms, Max: 2.764 ms, StdDev: 0.358 ms)
- Go: Mean 1.762 ms (Min: 1.519 ms, Max: 2.436 ms, StdDev: 0.314 ms)
- Rust: Mean 1.828 ms (Min: 1.587 ms, Max: 2.320 ms, StdDev: 0.256 ms)
- Java: Mean 2.286 ms (Min: 1.873 ms, Max: 3.443 ms, StdDev: 0.574 ms)
- C#: Mean 2.374 ms (Min: 1.955 ms, Max: 2.818 ms, StdDev: 0.257 ms)
- C++: Mean 1.962 ms (Min: 1.518 ms, Max: 2.954 ms, StdDev: 0.482 ms)
- Ruby: Mean 1.927 ms (Min: 1.659 ms, Max: 2.582 ms, StdDev: 0.278 ms)
- Python (100 functions fixture): Mean 2.033 ms (Min: 1.731 ms, Max: 3.012 ms, StdDev: 0.484 ms)
- Verdict: All 8 languages satisfy the under 5ms latency threshold on single files.

### 2.4 Diagnostic Explanation of AST Latency Variance and Thread Contention
Target fixture: Synthetic Python file containing 200 functions (800 lines).
Measured across 10 repeated benchmark iterations on an AMD Ryzen 3 3250U host:
- CLI process execution (including binary startup and file reading):
  - Debug profile (10 runs): Mean 3.745 ms (Min 3.327 ms, Max 4.287 ms, StdDev 0.323 ms).
  - Release profile (10 runs): Mean 2.014 ms (Min 1.894 ms, Max 2.138 ms, StdDev 0.077 ms).
- Pure in-process function execution:
  - Debug unoptimized: 1.65 ms.
  - Release optimized: 0.375 ms.
  - Baseline commit `ca31c10` unoptimized: 2.96 ms single thread, 6.2 ms under 4-thread CPU contention (`--test-threads=4`).
- Profile applicability: The specification threshold of under 5 ms applies normatively to the release build profile for runtime use. The optimized implementation satisfies the threshold in both release and debug profiles.
- Raw logs:
  - Baseline log: [16_latency_baseline_ca31c10.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/16_latency_baseline_ca31c10.log).
  - Optimized log: [17_latency_optimized_3ce41c4.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/17_latency_optimized_3ce41c4.log).

UTF-8 Handling Verification:
Test `test_ast_resolver_non_ascii_and_unicode_handling` in `read/read-core/src/ast/mod.rs` validates parsing Japanese identifiers and docstrings, Cyrillic comments, Greek symbols, and French accents without panics or degraded fallback states.

### 2.5 Root Cause Analysis of Previous Test Timeout
During optimization development, `test_learn_lifecycle` timed out with exit code 124.
Root cause:
In `read/read-core/src/ast/mod.rs` inside `extract_python`, when checking for an optional function docstring, the loop `while k < lines.len()` inspected candidate lines. If the first non-comment and non-empty line was ordinary Python code rather than a triple-quoted docstring, the loop failed to advance `k` and lacked an exit break. This caused an infinite loop when analyzing functions without docstrings.
Remediation:
Added an unconditional loop exit break after evaluating the first candidate line. The test now finishes in under 0.01 seconds. All 56 tests in `read-core` pass cleanly in 1.11 seconds.

### 2.6 PTY Streaming Allocation Verification
Microbenchmark `BenchmarkPtyStreaming` in `read-daemon/pty`:
- Execution speed: 56,628 ns/op
- Heap memory allocated: 974 B/op
- Allocations per frame: 4 allocs/op
- Verdict: The claim of zero allocation PTY streaming is false. Memory is allocated for byte slice buffers and WebSocket frame encapsulation. The claim has been removed from all documentation.

### 2.7 Large Test Suite Hook Latency
Measured executing a synthetic 500-assertion Python test suite under the harness:
- Mean: 127.431 ms
- Min: 108.445 ms
- Max: 136.777 ms

### 2.8 Tutorial Execution Timing and Student Completion
Measured running all 8 steps of `TUTORIAL.md` in a fresh clean directory via `evidence/time_tutorial.py`:
- Start Timestamp UTC: `2026-11-23T11:21:37.256742+00:00`
- End Timestamp UTC: `2026-11-23T11:21:37.433474+00:00`
- Total Step Count: 8
- Machine Execution Time: 176.72 ms (0.18 seconds)
- Human Completion Time: Not measured (machine execution: 0.18s, human trial unconducted)
All 8 commands executed with exit code 0.

## 3. Security and Vulnerability Empirical Verifications

### 3.1 Rate Limiter and X-Forwarded-For Spoofing
Tested `read-daemon` gateway under simulated proxy and client headers:
- Loopback connections (no XFF, 15 requests): 15/15 succeeded (200 OK). Whitelisted by design.
- External IP via XFF (`198.51.100.55`, 15 requests): First 10 requests succeeded (burst capacity). Next 5 requests returned HTTP 429 Too Many Requests with Retry-After.
- Spoofed loopback via XFF (`127.0.0.1`, 15 requests): 15/15 succeeded (200 OK).
- Finding: The daemon trusts incoming `X-Forwarded-For` headers unconditionally. An external client sending `X-Forwarded-For: 127.0.0.1` bypasses rate limiting. This risk is codified in `read/docs/THREAT_MODEL.md`.

### 3.2 Dynamic RLIMIT_AS Memory Constraints
Tested Java and .NET process execution under virtual memory constraints:
- Java (OpenJDK 21) under 512 MB `RLIMIT_AS`: Exit code 1. Stderr: `Error occurred during initialization of VM: Could not reserve enough space for 262144 KB object heap`.
- Dotnet 10 under 512 MB `RLIMIT_AS`: Exit code 137. Stderr: `GC: Reserving 268435456 bytes (0 GiB) for the regions range failed`.
- Java under 30% host RAM (1.71 GB) `RLIMIT_AS`: Exit code 1.
- Dotnet under 30% host RAM (1.71 GB) `RLIMIT_AS`: Exit code -6 (SIGABRT).
- Finding: Setting `RLIMIT_AS` limits virtual address space rather than resident memory (RSS). 64-bit managed runtimes reserve large virtual address regions on startup and crash when `RLIMIT_AS` is constrained to 30% RAM on systems with under 6 GB memory.

### 3.3 Mutation Testing on External Public Repository
Tested AROP mutation engine against genuine public repository `veloce-acceptance/github-repo`:
- Target File: `backend/resources/preview.py`
- Methodology: Reverted past bug fix commit `7a34571` on a scratch branch, staged the diff, and ran `read-engine mutate` on the staged diff. Staged diff stat confirmed 33 insertions, 13 deletions across `backend/resources/preview.py` only. Zero test files were staged.
- Mutants evaluated:
  - Initial run: Generated 10 mutants with 4 survivors (60.0% score).
    - Survivor 1 [L259 FLIP_BOOL_FALSE]: Real test gap. The test suite does not exercise `ascii_from_png` with corrupted pixel input that triggers `ValueError`.
    - Survivor 2 [L259 RETURN_NONE]: Real test gap. Same unexercised error handling path.
    - Survivor 3 [L264 LOGICAL_OR_TO_AND]: Equivalent mutant from an operator design flaw. Mutated "or" to "and" inside a triple-quoted Python docstring.
    - Survivor 4 [L267 LOGICAL_AND_TO_OR]: Equivalent mutant from an operator design flaw. Mutated "and" to "or" inside a triple-quoted Python docstring.
  - Corrected run (with docstrings skipped): Generated 8 mutants.
    - 6 mutants killed by `pytest tests/test_m7_vector_preview.py`.
    - 2 mutants survived on line 259 (the unasserted `ValueError` handling path).
- Gate Status: FAILED. The score of 75.0% (6/8 killed) fails the strictly required 100.0% threshold.
- Raw log: [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log).
- Integration check log: [14a_colorama_integration_check.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14a_colorama_integration_check.log).

### 3.4 WAL Tag Anchoring Security Boundary
Git tag anchoring links the latest WAL entry hash to the git tag `read-wal-head`.
Limitation:
This anchor relies on remote git server permissions. If an attacker possesses direct push permissions that allow updating or deleting git tags on the central git server, the attacker can overwrite both `.read/WAL.log` and the `read-wal-head` git tag. To provide non-repudiation across distributed peers, server-side protected tag rules (such as protected tags in GitLab or GitHub) are required.

### 3.5 Marketing Language Grep Audit
Automated grep checks across all repository markdown documentation for disallowed marketing terms and superlatives returned exit code 1.
Explanation: In standard Unix tools, grep returns exit code 1 when zero matches are found. Zero matches indicates that all forbidden marketing terms were successfully eliminated from the documentation.

## 4. Reproducible Binary Build Verification

To verify build reproducibility on this Linux host, clean builds were executed across independent isolated directories:

### 4.1 Rust Engine Reproducibility (`read-engine`)
Built using `cargo build --release --manifest-path read/read-core/Cargo.toml`.
- Binary SHA-256 in `read/bin/read-engine`: `d1504e35fa9ad89b32a531331a258ed0ebc28e76b68cbb101fec1c0887a990c0`
- Clean isolated build SHA-256 (`/tmp/reproduce_clean1`): `d1504e35fa9ad89b32a531331a258ed0ebc28e76b68cbb101fec1c0887a990c0`
- Result: SHA-256 hashes match identically. Rust release builds are bitwise reproducible on this Linux host.

### 4.2 Go Host Daemon Reproducibility (`read-daemon`)
Built using `go build -trimpath -o bin/read-daemon cmd/read-daemon/main.go`.
- Standard `go build` embeds local directory paths into debug symbols, producing different hashes across different directories.
- With `-trimpath` and building `cmd/read-daemon/main.go`, file system paths are stripped from the compiled binary.
- Binary SHA-256 in `read/bin/read-daemon`: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Clean isolated build SHA-256 (`/tmp/reproduce_clean1`): `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Result: SHA-256 hashes match identically. With `-trimpath`, Go daemon builds are bitwise reproducible on this Linux host.
