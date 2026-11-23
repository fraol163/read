# READ Framework Empirical Evidence Dossier

Date: 2026-11-23
Host Environment: Linux 7.2.15-1-omarchy x86_64
Hardware: AMD Ryzen 3 3250U with Radeon Graphics (2 cores, 4 threads), 5.7 GiB RAM
Compilers and Runtimes: rustc 1.99.1, go 1.27.3, python 3.15.2, dotnet 10.0.403, openjdk 21.0.9

## 1. Automated Test Suite Executions

All ten test suites were executed directly on the host machine. Output logs are preserved in `evidence/logs/`.

| Suite # | Test Target | Command Executed | Exit Code | Raw Evidence Log |
|---|---|---|---|---|
| 01 | End-to-End Lifecycle | `bash read/tests/test_e2e.sh` | 0 | [01_test_e2e.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/01_test_e2e.log) |
| 02 | Chaos and Stress Tests | `bash read/tests/test_chaos.sh` | 0 | [02_test_chaos.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/02_test_chaos.log) |
| 03 | BMAD Command Parity | `bash read/tests/test_bmad_zero_command.sh` | 0 | [03_test_bmad_zero_command.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/03_test_bmad_zero_command.log) |
| 04 | Full System Integration | `bash read/tests/test_full_system.sh` | 0 | [04_test_full_system.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/04_test_full_system.log) |
| 05 | Autonomous Pipeline | `bash read/tests/test_pipeline.sh` | 0 | [05_test_pipeline.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/05_test_pipeline.log) |
| 06 | Packaging and Distribution | `bash read/tests/test_packaging.sh` | 0 | [06_test_packaging.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/06_test_packaging.log) |
| 07 | Realtime WebSockets | `bash read/tests/test_live_realtime.sh` | 0 | [07_test_live_realtime.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/07_test_live_realtime.log) |
| 08 | Rust Core Unit Tests | `cargo test --manifest-path read/read-core/Cargo.toml` | 0 (54 passed) | [08_cargo_test_read_core.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/08_cargo_test_read_core.log) |
| 09 | Go Daemon Unit Tests | `cd read/read-daemon && go test -v -count=1 ./...` | 0 | [09_go_test_read_daemon.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/09_go_test_read_daemon.log) |
| 10 | Go Microbenchmarks | `cd read/read-daemon && go test -bench . -benchmem ./...` | 0 | [10_go_test_bench.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/10_go_test_bench.log) |

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
- Observation: The previous marketing claim of "strictly under 10ms" fails on 2-core hardware when process creation is measured. Latency ranges between 7.5 ms and 16.5 ms. The claim has been narrowed accordingly.

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

### 2.4 PTY Streaming Allocation Verification
Microbenchmark `BenchmarkPtyStreaming` in `read-daemon/pty`:
- Execution speed: 42,422 ns/op
- Heap memory allocated: 993 B/op
- Allocations per frame: 4 allocs/op
- Verdict: The claim of "zero allocation PTY streaming" is false. Memory is allocated for byte slice buffers and WebSocket frame encapsulation. The claim has been removed from all documentation.

### 2.5 Large Test Suite Hook Latency
Measured executing a synthetic 500-assertion Python test suite under the harness:
- Mean: 127.431 ms
- Min: 108.445 ms
- Max: 136.777 ms

## 3. Security and Vulnerability Empirical Verifications

### 3.1 Rate Limiter and X-Forwarded-For Spoofing
Tested `read-daemon` gateway under simulated proxy and client headers:
- Loopback connections (no XFF, 15 requests): 15/15 succeeded (200 OK). Whitelisted by design.
- External IP via XFF (`198.51.100.55`, 15 requests): First 10 requests succeeded (burst capacity). Next 5 requests returned HTTP 429 Too Many Requests with Retry-After.
- Spoofed loopback via XFF (`127.0.0.1`, 15 requests): 15/15 succeeded (200 OK).
- Finding: The daemon trusts incoming `X-Forwarded-For` unconditionally. An external client sending `X-Forwarded-For: 127.0.0.1` completely bypasses rate limiting. This risk is codified in `read/docs/THREAT_MODEL.md`.

### 3.2 Dynamic RLIMIT_AS Memory Constraints
Tested Java and .NET process execution under virtual memory constraints:
- Java (OpenJDK 21) under 512 MB `RLIMIT_AS`: Exit code 1. Stderr: `Error occurred during initialization of VM: Could not reserve enough space for 262144 KB object heap`.
- Dotnet 10 under 512 MB `RLIMIT_AS`: Exit code 137. Stderr: `GC: Reserving 268435456 bytes (0 GiB) for the regions range failed`.
- Java under 30% host RAM (1.71 GB) `RLIMIT_AS`: Exit code 1.
- Dotnet under 30% host RAM (1.71 GB) `RLIMIT_AS`: Exit code -6 (SIGABRT).
- Finding: Setting `RLIMIT_AS` limits virtual address space rather than resident memory (RSS). 64-bit managed runtimes reserve large virtual address regions on startup and crash when `RLIMIT_AS` is constrained to 30% RAM on systems with under 6 GB memory.

### 3.3 Mutation Testing on External Codebase
Tested AROP mutation engine against real Python code from `veloce-acceptance/github-repo/backend/version_catalog.py`:
- Baseline test suite: `tests/test_version_catalog.py` (2 tests, 0.001s).
- Staged modification: Added length validation check in `update_toml_version`.
- Mutants generated: 3 bounded mutants:
  1. `INVERT_EQ` at line 396 (`==` to `!=`).
  2. `FLIP_BOOL_FALSE` at line 397 (`return False` to `return True`).
  3. `RETURN_NONE` at line 397 (`return False` to `return None`).
- Evaluation:
  - Mutant #1: KILLED (test suite failed).
  - Mutant #2: KILLED (test suite failed).
  - Mutant #3: KILLED (test suite failed).
- Mutation Score: 3/3 killed (100.0%).
- Result: Gate passed.
