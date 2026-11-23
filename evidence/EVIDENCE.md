# READ Framework Empirical Evidence Dossier

Date: 2026-11-23
Review Branch: report-review-v0.7.1
Host Environment: Linux 7.2.15-1-omarchy x86_64
Hardware: AMD Ryzen 3 3250U with Radeon Graphics (2 cores, 4 threads), 5.7 GiB RAM
Compilers and Runtimes: rustc 1.99.1, go 1.27.3, python 3.15.2, dotnet 10.0.403, openjdk 21.0.9

CRITICAL GATE STATUS:
The outside repository mutation verification gate on veloce-acceptance/github-repo strictly FAILED (Exit Code: 1). Score: 75.0% (6 killed, 2 survived). The framework invariant requires a 100% kill rate, so verification was rejected.

## 1. Automated Test Suite Executions

Every suite was executed directly on the host machine. Output logs are preserved in `evidence/logs/`.
Suites 12 and 13 represent local shell scenarios executing real binary steps rather than cloud GitHub Actions runners. The CI gate remains UNTESTED on cloud runners because no git remote is configured.

| Suite # | Test Target | Command Executed | Exit Code | Pass Count | Fail Count | Raw Evidence Log |
|---|---|---|---|---|---|---|
| 01 | End-to-End Lifecycle | `bash read/tests/test_e2e.sh` | 0 | 15 | 0 | [01_test_e2e.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/01_test_e2e.log) |
| 02 | Chaos and Stress Suite | `bash read/tests/test_chaos.sh` | 0 | 10 | 0 | [02_test_chaos.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/02_test_chaos.log) |
| 03 | BMAD Zero-Command Parity | `bash read/tests/test_bmad_zero_command.sh` | 0 | 6 | 0 | [03_test_bmad_zero_command.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/03_test_bmad_zero_command.log) |
| 04 | Full System Integration | `bash read/tests/test_full_system.sh` | 0 | 9 | 0 | [04_test_full_system.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/04_test_full_system.log) |
| 05 | Autonomous Pipeline | `bash read/tests/test_pipeline.sh` | 0 | 6 | 0 | [05_test_pipeline.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/05_test_pipeline.log) |
| 06 | Packaging and Installation | `bash read/tests/test_packaging.sh` | 0 | 8 | 0 | [06_test_packaging.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/06_test_packaging.log) |
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

### 1.1 Traceability and Git Diffs of Shell Test Files

The test pass counts for shell scripts moved across versions as new language fixtures, adversarial checks, and granular lifecycle assertions were added. Below are the exact git diffs against baseline commit `ca31c10` proving no assertions were weakened.

#### Suite 01 (`read/tests/test_e2e.sh`: 9 to 15 passed)
Git diff against baseline commit `ca31c10`:
```diff
diff --git a/read/tests/test_e2e.sh b/read/tests/test_e2e.sh
index 9e8f2e9..11da58e 100755
--- a/read/tests/test_e2e.sh
+++ b/read/tests/test_e2e.sh
@@ -14,8 +14,8 @@ cd "$SANDBOX"
 # 1. Initialize git repo and READ framework
 git init -q
 "$READ" init
-if [ ! -f .method/STATE.json ] || [ ! -f .method/WAL.log ]
-then
+if [ ! -f .read/STATE.json ] || [ ! -f .read/WAL.log ]
+then
     echo "[FAIL] .read state files not created" >&2
     exit 1
 fi
 echo "[PASS] 1. read init successfully initialized atomic state and WAL"
```
Explanation of count shift:
In the earliest prototype, the script tested 9 sequential steps (numbered 1 through 9) and concluded with `echo "=== [ALL 9 TESTS PASSED] ==="`. During implementation of Directives 003 through 006, 6 new verification assertions were added to test strict mutation gates for additional languages:
- Step 6b: Mechanical rejection of 90% mutation score (9/10 killed, 1 survivor)
- Step 6c: TypeScript AST Diff Mutation (INVERT_EQUALITY, BOUNDARY_SHIFT, REMOVE_OPTIONAL_CHAIN)
- Step 6d: Java AST Diff Mutation (REMOVE_NULL_CHECK, INVERT_EQUALITY, BOUNDARY_SHIFT)
- Step 6e: C# AST Diff Mutation (REMOVE_NULL_CONDITIONAL, INVERT_EQUALITY, BOUNDARY_SHIFT)
- Step 6f: C++ AST Diff Mutation (REMOVE_NULLPTR_CHECK, INVERT_EQUALITY, BOUNDARY_SHIFT)
- Step 6g: Ruby AST Diff Mutation (REMOVE_SAFE_NAVIGATION, INVERT_EQUALITY, BOUNDARY_SHIFT, RETURN_NIL)
The script now outputs 15 explicit `[PASS]` markers. The only change between `ca31c10` and current HEAD is standardizing the path check from `.method/` to `.read/`. Zero assertions were removed or weakened.

#### Suite 03 (`read/tests/test_bmad_zero_command.sh`: 5 to 6 passed)
Git diff against baseline commit `ca31c10`:
```
(No diff - identical to commit ca31c10)
```
Explanation of count shift:
The script has always contained exactly 6 `[PASS]` assertions:
1. Line 104: STATE.json status transitioned to 'handoff_complete' with zero manual commands
2. Line 113: EVIDENCE.json sealed with verified mutation proof
3. Line 121: Git commit atomically sealed .read/STATE.json and .read/EVIDENCE.json
4. Line 157: Pre-commit hook mechanically blocked commit when mutation score fell below 100% threshold
5. Line 165: EVIDENCE.json accurately diagnosed and recorded 'gate_failed'
6. Line 205: Commit succeeded automatically once test suite killed all mutants
Earlier informal summaries that reported 5 passed had mistakenly omitted counting the final commit remediation assertion. Zero assertions were altered or weakened.

#### Suite 07 (`read/tests/test_live_realtime.sh`: 4 to 8 stages)
Git diff against baseline commit `ca31c10`:
```
(No diff - identical to commit ca31c10)
```
Explanation of count shift:
Suite 07 is an end-to-end integration lifecycle test rather than a unit test runner counting assertions. It executes 8 sequential execution stages:
- Stage 1: Interactive Onboarding Wizard (`read setup`)
- Stage 2: Declarative Contract Definition (`.read/TASK.md`)
- Stage 3: Implementation Code and Test Suite Authoring
- Stage 4: Developer Commit Intercept (Zero Commands)
- Stage 5: Verification of Automatically Sealed State
- Stage 6: Adversarial Code Injection (Untested Code Blocked)
- Stage 7: Test Remediation (Killing All Mutants)
- Stage 8: Final Proof and Sealed Commit Chain
Earlier informal summaries counted 4 selective state checkpoints (Stages 1, 5, 6, and 8). The script executes all 8 sequential stages under `set -euo pipefail`. Zero assertions were altered or weakened.

#### Suite 08 (`read-core`: 55 to 58 passed)
Added 3 targeted unit tests:
- `test_ast_resolver_non_ascii_and_unicode_handling`: Validates parsing Japanese identifiers and docstrings, Cyrillic comments, Greek symbols, and French accents without panic or degraded fallback.
- `test_diff_mutator_skips_python_docstrings`: Validates that Python triple quotes (`"""` and `'''`) are skipped during diff mutation.
- `test_extraction_latency_contention_10_runs`: Validates 10 runs of AST extraction under 4-thread CPU contention strictly satisfy the under 5 ms threshold.

#### Suite 09 (`read-daemon`: 22 to 57 passed)
Earlier summaries counted 22 top-level test groups. Running `go test -v` executes 57 individual test functions across packages config, gateway, orchestrator, pty, scanner, and security.

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

### 2.4 AST Latency Benchmarks, Contention Variance, and Code Location

Target fixture: Synthetic Python file containing 200 functions (800 lines).
Location in codebase:
The in-process latency benchmark is located strictly in test code within `read/read-core/src/ast/mod.rs` inside the `#[cfg(test)] mod tests` block (`test_extraction_latency_under_5ms` and `test_extraction_latency_contention_10_runs`). No benchmark scaffolding exists in production code paths.

Measurement History Across Specific Commits:
1. Baseline commit `ca31c10`:
   - Single thread debug unoptimized: 2.96 ms
   - Under 4-thread CPU contention (`--test-threads=4`, debug): 6.2 ms (failed the `< 5ms` assertion with exit code 101)
   - Log: [16_latency_baseline_ca31c10.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/16_latency_baseline_ca31c10.log)

2. Optimized commit `3ce41c4`:
   - Single-pass byte slice parsing replaced repeated string allocations.
   - Single thread debug unoptimized: 1.65 ms
   - Single thread release optimized: 0.375 ms (the exact origin of the 0.375 ms figure)
   - Contention under 4 threads: 2.1 to 2.4 ms
   - Log: [17_latency_optimized_3ce41c4.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/17_latency_optimized_3ce41c4.log)

3. Current HEAD commit (10-run variance and 4-thread contention):
   - In-Process Function Execution Latency (10 iterations on idle host):
     * Release profile: Mean 0.140 ms (Min: 0.122 ms, Max: 0.226 ms, StdDev: 0.032 ms)
     * Debug profile:   Mean 1.225 ms (Min: 0.962 ms, Max: 1.677 ms, StdDev: 0.274 ms)
   - In-Process Execution Latency Under 4-Thread Contention (3 concurrent worker threads continuously parsing while measuring thread executes):
     * Release profile (10 runs): Mean 0.144 ms (Min: 0.134 ms, Max: 0.192 ms)
     * Debug profile (10 runs):   Mean 1.982 ms (Min: 1.435 ms, Max: 3.524 ms)
     * Raw measurements log: [20_ast_latency_contention.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/20_ast_latency_contention.log)
     * Both release and debug profiles strictly satisfy the under 5 ms threshold under 4-thread contention.
   - CLI Process Execution Latency (including startup and file reading across 10 iterations):
     * Release profile: Mean 2.014 ms (Min: 1.894 ms, Max: 2.138 ms, StdDev: 0.077 ms)
     * Debug profile:   Mean 3.745 ms (Min: 3.327 ms, Max: 4.287 ms, StdDev: 0.323 ms)

Normative Profile Applicability:
The specification threshold of under 5 ms applies normatively to the release build profile for runtime execution. Both release and debug profiles satisfy the requirement on this hardware.

Correction Note:
An earlier summary reported an unverified projection of 0.05 ms in debug and 0.01 ms in release. Real measured execution times on this hardware are 1.225 ms mean in debug and 0.140 ms mean in release for idle in-process extraction, and 1.982 ms debug / 0.144 ms release under 4-thread CPU contention.

UTF-8 Handling Verification:
Test `test_ast_resolver_non_ascii_and_unicode_handling` in `read/read-core/src/ast/mod.rs` validates parsing Japanese identifiers and docstrings, Cyrillic comments, Greek symbols, and French accents without panics or degraded fallback states.

### 2.5 Root Cause Analysis of Previous Test Timeout
During optimization development, `test_learn_lifecycle` timed out with exit code 124.
Root cause:
In `read/read-core/src/ast/mod.rs` inside `extract_python`, when checking for an optional function docstring, the loop `while k < lines.len()` inspected candidate lines. If the first non-comment and non-empty line was ordinary Python code rather than a triple-quoted docstring, the loop failed to advance `k` and lacked an exit break. This caused an infinite loop when analyzing functions without docstrings.
Remediation:
Added an unconditional loop exit break after evaluating the first candidate line. The test now finishes in under 0.01 seconds. All 58 tests in `read-core` pass cleanly in 1.12 seconds.

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
- Target Repository Commit Hash: `7732578c560572cb0e60d3aa83fdd2fdac6a85cf`
- Target File: `backend/resources/preview.py`
- Methodology: Reverted past bug fix commit `7a34571f9d6e29bcaeb0226765dde34720ba0982` on a scratch branch, staged the diff, and ran `read-engine mutate` on the staged diff.
- Staged diff stat: 33 additions, 13 deletions across `backend/resources/preview.py` only. Zero test files were staged.
- Exact Execution Command:
  `python3 evidence/eval_outside_mutation.py`
  (invokes `read-engine mutate` in `/tmp/test_outside_repo` and evaluates mutants against `pytest tests/test_m7_vector_preview.py`)

#### Empirical Line Coverage Proof on preview.py
To prove empirically that surviving mutants on line 259 represent genuine test gaps rather than an inference, test coverage was executed via `pytest --cov`:
Command:
```bash
cd /tmp/test_outside_repo && pytest --cov=backend.resources.preview --cov-report=term-missing tests/test_m7_vector_preview.py
```
Output Summary:
```
Name                           Stmts   Miss  Cover   Missing
backend/resources/preview.py     200     29    86%   109-118, 129, 131, 137, 139, 142, 145, 148, 188, 224, 226, 231, 251, 255, 258-259, 276, 279, 281, 295
TOTAL                            200     29    86%
38 passed in 2.58s
```
Lines 258-259 are explicitly reported under Missing. Line 259 had zero executions during the entire test suite run. This proves empirically that the surviving mutants on line 259 represent genuine test gaps.
Raw coverage log: [19_preview_coverage.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/19_preview_coverage.log)

#### Detailed Mutant Comparison: Run A vs Run B

Run A (Historical Only - Initial Run with Docstring Operator Flaw):
- Binary SHA-256: `d1504e35fa9ad89b32a531331a258ed0ebc28e76b68cbb101fec1c0887a990c0`
- Status: HISTORICAL ONLY (conducted prior to docstring tracking fix at commit `18bc274`)
- Generated Mutants (10 total):
  * Mutant #1 [L259 FLIP_BOOL_FALSE]: SURVIVED (Empirical test gap on ValueError handling)
  * Mutant #2 [L259 RETURN_NONE]: SURVIVED (Empirical test gap on ValueError handling)
  * Mutant #3 [L260 FLIP_BOOL_TRUE]: KILLED (test failure)
  * Mutant #4 [L260 RETURN_NONE]: KILLED (test failure)
  * Mutant #5 [L264 LOGICAL_OR_TO_AND]: SURVIVED (Equivalent mutant in docstring: 'or' changed to 'and')
  * Mutant #6 [L267 LOGICAL_AND_TO_OR]: SURVIVED (Equivalent mutant in docstring: 'and' changed to 'or')
  * Mutant #7 [L272 LOGICAL_OR_TO_AND]: KILLED (test failure)
  * Mutant #8 [L285 INVERT_LT]: KILLED (test failure)
  * Mutant #9 [L294 INVERT_NEQ]: KILLED (test failure)
  * Mutant #10 [L296 RETURN_NONE]: KILLED (test failure)
- Score: 6/10 killed, 4 survived (60.0% score). Gate status: FAILED.

Run B (Corrected Run with Docstring Operator Fix):
- Binary SHA-256: `75b0e382ad58de809386f078c066165b1836c4f87357a2c0d01a6d943507f479`
- Status: REPRODUCIBLE CURRENT RUN
- Why total dropped from 10 to 8 without test changes:
  The mutator was updated in `read/read-core/src/mutation/mod.rs` to track Python triple quotes (`"""` and `'''`) and skip docstrings. Mutants #5 and #6 from Run A were generated from words inside comments on lines 266 and 271 of `pixel_grid_from_txt`. The mutator fix eliminated those 2 spurious docstring mutants without changing any target repository code or test suites.
- Generated Mutants (8 total):
  * Mutant #1 [L259 FLIP_BOOL_FALSE]: SURVIVED (Empirical test gap on ValueError handling)
  * Mutant #2 [L259 RETURN_NONE]: SURVIVED (Empirical test gap on ValueError handling)
  * Mutant #3 [L260 FLIP_BOOL_TRUE]: KILLED (test failure)
  * Mutant #4 [L260 RETURN_NONE]: KILLED (test failure)
  * Mutant #5 [L272 LOGICAL_OR_TO_AND]: KILLED (test failure)
  * Mutant #6 [L285 INVERT_LT]: KILLED (test failure)
  * Mutant #7 [L294 INVERT_NEQ]: KILLED (test failure)
  * Mutant #8 [L296 RETURN_NONE]: KILLED (test failure)
- Score: 6/8 killed, 2 survived (75.0% score). Gate status: FAILED (Exit Code: 1).

Why Survivors Remain Open:
Mutants #1 and #2 mutate line 259 inside `ascii_from_png`:
`except ValueError as e: return TextPreview(False, metrics=metrics, message=f"text preview failed: {e}")`
The target repository test suite does not include a test that feeds malformed pixel output into `ascii_from_png` to assert this error path. This gap remains open. The mutation gate strictly failed and mechanically blocked handoff.

Docstring Unit Test and Repository Impact:
Unit test `test_diff_mutator_skips_python_docstrings` in `read/read-core/src/mutation/mod.rs` verifies that multi-line and single-line docstrings produce zero mutants while active code lines are mutated normally. Across the target diff, this filter eliminated 2 spurious mutants. Across the `read` repository's own Python codebase, all docstrings are protected from text mutation.

Raw logs:
- Evaluated log: [14_external_repo_mutation.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14_external_repo_mutation.log)
- Integration check log: [14a_colorama_integration_check.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/14a_colorama_integration_check.log)

### 3.4 WAL Tag Anchoring Security Boundary
Git tag anchoring links the latest WAL entry hash to the git tag `read-wal-head`.
Limitation:
This anchor relies on remote git server permissions. If an attacker possesses direct push permissions that allow updating or deleting git tags on the central git server, the attacker can overwrite both `.read/WAL.log` and the `read-wal-head` git tag. To provide non-repudiation across distributed peers, server-side protected tag rules (such as protected tags in GitLab or GitHub) are required.

### 3.5 Marketing Language Grep Audit
Automated grep checks across all repository markdown documentation for disallowed marketing terms and superlatives returned exit code 1.
Explanation: In standard Unix tools, grep returns exit code 1 when zero matches are found. Zero matches indicates that all forbidden marketing terms were successfully eliminated from the documentation.

## 4. Reproducible Binary Build Verification

To verify build reproducibility on this Linux host, clean builds were executed across independent isolated directories:
Raw log: [18_clean_build_reproducibility.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/18_clean_build_reproducibility.log)

Clean Directory 1 (`/tmp/clean_build_1`):
- Build Start: `2026-11-23T11:57:43.012698110Z`
- Build End:   `2026-11-23T11:58:13.365523578Z`
- Duration:    30.35 seconds
- Rust Engine SHA-256: `75b0e382ad58de809386f078c066165b1836c4f87357a2c0d01a6d943507f479`
- Go Host Daemon SHA-256: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`

Clean Directory 2 (`/tmp/clean_build_2`):
- Build Start: `2026-11-23T11:58:19.872800549Z`
- Build End:   `2026-11-23T11:58:50.389144770Z`
- Duration:    30.51 seconds
- Rust Engine SHA-256: `75b0e382ad58de809386f078c066165b1836c4f87357a2c0d01a6d943507f479`
- Go Host Daemon SHA-256: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`

Shipped Binaries in `read/bin/`:
- `read/bin/read-engine`: `75b0e382ad58de809386f078c066165b1836c4f87357a2c0d01a6d943507f479`
- `read/bin/read-daemon`: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Result: Both binaries match the clean isolated directory builds bitwise identically.
- Scope: Verified on this Linux host (Linux 7.2.15-1-omarchy x86_64) only.

Verification One-Liner Command:
`sha256sum read/bin/read-engine read/bin/read-daemon` produces `75b0e382ad58de809386f078c066165b1836c4f87357a2c0d01a6d943507f479  read/bin/read-engine` and `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59  read/bin/read-daemon`.
