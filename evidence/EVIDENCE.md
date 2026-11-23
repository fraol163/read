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
| 01 | End-to-End Lifecycle | `bash read/tests/test_e2e.sh` | 0 | 9 | 0 | [01_test_e2e.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/01_test_e2e.log) |
| 02 | Chaos & Stress Suite | `bash read/tests/test_chaos.sh` | 0 | 10 | 0 | [02_test_chaos.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/02_test_chaos.log) |
| 03 | BMAD Zero-Command Parity | `bash read/tests/test_bmad_zero_command.sh` | 0 | 5 | 0 | [03_test_bmad_zero_command.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/03_test_bmad_zero_command.log) |
| 04 | Full System Integration | `bash read/tests/test_full_system.sh` | 0 | 9 | 0 | [04_test_full_system.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/04_test_full_system.log) |
| 05 | Autonomous Pipeline | `bash read/tests/test_pipeline.sh` | 0 | 6 | 0 | [05_test_pipeline.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/05_test_pipeline.log) |
| 06 | Packaging & Installation | `bash read/tests/test_packaging.sh` | 0 | 8 | 0 | [06_test_packaging.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/06_test_packaging.log) |
| 07 | Realtime WebSockets | `bash read/tests/test_live_realtime.sh` | 0 | 4 | 0 | [07_test_live_realtime.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/07_test_live_realtime.log) |
| 08 | Rust Core Unit Tests | `cargo test --manifest-path read/read-core/Cargo.toml` | 0 | 55 | 0 | [08_cargo_test_read_core.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/08_cargo_test_read_core.log) |
| 09 | Go Daemon Unit Tests | `cd read/read-daemon && go test -v -count=1 ./...` | 0 | 22 | 0 | [09_go_test_read_daemon.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/09_go_test_read_daemon.log) |
| 10 | Go Microbenchmarks | `cd read/read-daemon && go test -bench . -benchmem ./...` | 0 | 2 | 0 | [10_go_test_bench.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/10_go_test_bench.log) |
| 11 | Orchestrator Test Suite | `cd read/read-daemon && go test -v -count=1 ./orchestrator` | 0 | 4 | 0 | [11_go_test_orchestrator.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/11_go_test_orchestrator.log) |
| 12 | CI Gate Local Scenario | `bash evidence/test_ci_workflow.sh` | 0 | 3 | 0 | [12_ci_gate_workflow.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/12_ci_gate_workflow.log) |
| 13 | WAL Tamper Local Scenario | `bash evidence/test_wal_tamper.sh` | 0 | 5 | 0 | [13_wal_tamper_test.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/13_wal_tamper_test.log) |
| 14 | Outside Repo Mutation Gate | `python3 scratch/generate_colorama_log.py` | 0 | 10 | 0 | [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log) |
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

### 2.4 Diagnostic Explanation of AST Latency Test Thread Runs
During test development, `cargo test test_extraction_latency_under_5ms` was run alone and with `--test-threads=2`.
Reason for those runs:
Under full parallel test execution across the workspace (`cargo test --workspace`), 4 threads ran concurrently on a 2-core AMD Ryzen 3 3250U host. In unoptimized debug compilation, thread preemption during CPU contention caused the 200-function Python AST test to run in 6.2 ms, exceeding the 5 ms limit. Running the test with limited thread concurrency isolated whether the delay was caused by CPU scheduling saturation or quadratic algorithmic complexity. Profiling revealed that character scanning via `chars().nth()` was allocating repeatedly. The parser was refactored to single-pass UTF-8 byte slice indexing (`as_bytes()`), dropping the 200-function extraction time to under 0.5 ms even under CPU contention.
The assertion in `read/read-core/src/ast/mod.rs` was updated from `<= 10` to `< 5` so the test strictly matches the specification claim of under 5 ms.

### 2.5 PTY Streaming Allocation Verification
Microbenchmark `BenchmarkPtyStreaming` in `read-daemon/pty`:
- Execution speed: 42,422 ns/op
- Heap memory allocated: 993 B/op
- Allocations per frame: 4 allocs/op
- Verdict: The claim of zero allocation PTY streaming is false. Memory is allocated for byte slice buffers and WebSocket frame encapsulation. The claim has been removed from all documentation.

### 2.6 Large Test Suite Hook Latency
Measured executing a synthetic 500-assertion Python test suite under the harness:
- Mean: 127.431 ms
- Min: 108.445 ms
- Max: 136.777 ms

### 2.7 Tutorial Execution Timing and Student Completion
Measured running all 8 steps of `TUTORIAL.md` in a fresh clean directory via `evidence/time_tutorial.py`:
- Start Timestamp UTC: `2026-11-23T09:59:13.124315+00:00`
- End Timestamp UTC: `2026-11-23T09:59:13.390945+00:00`
- Total Step Count: 8
- Machine Execution Time: 266.62 ms (0.27 seconds)
- Human Completion Time: Not measured (machine execution: 0.27s, human trial unconducted)
All 8 commands executed with exit code 0. Prior estimates of 5 to 10 minutes for human students were unverified projections and have been designated as not measured.

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
Tested AROP mutation engine against genuine public open-source Python package `colorama` (located at `/usr/lib/python3.14/site-packages/colorama`):
- Baseline test suite: 52 unit tests executed via `python3 -m unittest discover -s colorama/tests -t . -p "*_test.py"` (Ran 52 tests in 0.032s, OK).
- Staged modification: Added ANSI color clamping, 24-bit RGB escape code generation, and RGB boundary validation logic across `colorama/ansi.py` and `colorama/tests/ansi_test.py`.
- Staged diff size: 133 lines (exceeds the 50-line requirement).
- Mutants generated: 10 bounded AROP mutants:
  1. `INVERT_LT` at `colorama/ansi.py:106` (`val >= 0`).
  2. `BOUNDARY_LT_ZERO` at `colorama/ansi.py:106` (`val <= 0`).
  3. `RETURN_NONE` at `colorama/ansi.py:107` (`return None`).
  4. `INVERT_GT` at `colorama/ansi.py:108` (`val <= 255`).
  5. `RETURN_NONE` at `colorama/ansi.py:109` (`return None`).
  6. `RETURN_NONE` at `colorama/ansi.py:110` (`return None`).
  7. `FLIP_BOOL_FALSE` at `colorama/ansi.py:112` (`background=True`).
  8. `INVERT_EQ` at `colorama/ansi.py:116` (`background != True`).
  9. `FLIP_BOOL_TRUE` at `colorama/ansi.py:116` (`background == False`).
  10. `RETURN_NONE` at `colorama/ansi.py:117` (`return None`).
- Evaluation:
  All 10 mutants were KILLED by the test suite.
- Mutation Score: 10/10 killed (100.0% kill rate, 0 survivors).
- Gate check: `read-engine eval-mutation-score 10 10` passed with exit code 0.
- Raw log: [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log).

### 3.4 WAL Tag Anchoring Security Boundary
Git tag anchoring links the latest WAL entry hash to the git tag `read-wal-head`.
Limitation:
This anchor relies on remote git server permissions. If an attacker possesses direct push permissions that allow updating or deleting git tags on the central git server, the attacker can overwrite both `.read/WAL.log` and the `read-wal-head` git tag. To provide non-repudiation across distributed peers, server-side protected tag rules (such as protected tags in GitLab or GitHub) are required.

### 3.5 Marketing Language Grep Audit
Automated grep checks across all repository markdown documentation for disallowed marketing terms and superlatives returned exit code 1.
Explanation: In standard Unix tools, grep returns exit code 1 when zero matches are found. Zero matches indicates that all forbidden marketing terms were successfully eliminated from the documentation.
