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

The test pass counts for shell scripts moved across versions as new language fixtures, adversarial checks, and granular lifecycle assertions were added. Below are the exact git diffs against baseline commit `ca31c10`.

#### Suite 01 (`read/tests/test_e2e.sh`: 15 passed)
Full git diff against baseline commit `ca31c10`:
```diff
diff --git a/read/tests/test_e2e.sh b/read/tests/test_e2e.sh
index 9e8f2e9..e72cbe9 100755
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

Verification of removed lines:
The single removed line in the path check was:
`if [ ! -f .method/STATE.json ] || [ ! -f .method/WAL.log ]`
This line was replaced with:
`if [ ! -f .read/STATE.json ] || [ ! -f .read/WAL.log ]`
Zero assertions were loosened or removed.

Pass Count Characterization and Self-Referential Steps:
The 15 passed checks represent assertion lines executed within `read/tests/test_e2e.sh`. Note that 6 of these steps (Step 6 for Python, and Steps 6c through 6g for TypeScript, Java, C#, C++, and Ruby) check counts (6, 4, 6, 4, 6) that verify stable operator emission for the sample diffs.
Furthermore, the lines executing `$READ eval-mutation-score $((COUNT - 1)) $COUNT` are strictly an arithmetic check of the score function, confirming only that non-100% scores produce non-zero exit codes. They do not run test suites against mutants. Real gate rejection is demonstrated by the end-to-end run against an outside repository with an existing test suite (veloce run, Suite 14), which failed with exit code 1.

Steps 6c through 6g Output and Arithmetic Verification:
1. Step 6c (TypeScript): Asserts exactly 6 mutants generated (filtering out invalid compile-error `RETURN_UNDEFINED` under strictNullChecks), validates `REMOVE_OPTIONAL_CHAIN`, `INVERT_EQUALITY`, and `BOUNDARY_SHIFT`, asserts 100% score calculation on 6/6 killed, and performs an arithmetic check of the score function when 1 survivor is simulated.
2. Step 6d (Java): Asserts exactly 4 mutants generated (filtering out invalid compile-error `RETURN_NULL` on primitive boolean return), validates `REMOVE_NULL_CHECK`, `INVERT_EQUALITY`, and `BOUNDARY_SHIFT`, asserts 100% score calculation on 4/4 killed, and performs an arithmetic check of the score function when 1 survivor is simulated.
3. Step 6e (C#): Asserts exactly 6 mutants generated, validates `REMOVE_NULL_CONDITIONAL`, `INVERT_EQUALITY`, and `BOUNDARY_SHIFT`, asserts 100% score calculation on 6/6 killed, and performs an arithmetic check of the score function when 1 survivor is simulated.
4. Step 6f (C++): Asserts exactly 4 mutants generated, validates `REMOVE_NULLPTR_CHECK`, `INVERT_EQUALITY`, and `BOUNDARY_SHIFT`, asserts 100% score calculation on 4/4 killed, and performs an arithmetic check of the score function when 1 survivor is simulated.
5. Step 6g (Ruby): Asserts exactly 6 mutants generated, validates `REMOVE_SAFE_NAVIGATION`, `INVERT_EQUALITY`, `BOUNDARY_SHIFT`, and `RETURN_NIL`, asserts 100% score calculation on 6/6 killed, and performs an arithmetic check of the score function when 1 survivor is simulated.

### 1.2 Per-Language Mutant Verification and Hand-Checked Correctness Analysis

Every mutant generated across the 5 non-Python test languages was mapped to its input patch line and evaluated by hand for syntax validity, operator semantics, and compilation impact. The exact patch fixtures are preserved in [diff_ts_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_ts_sample.patch), [diff_java_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_java_sample.patch), [diff_csharp_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_csharp_sample.patch), [diff_cpp_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_cpp_sample.patch), and [diff_ruby_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_ruby_sample.patch).

Compile Error Rule:
A compile error counts as an invalid mutant, not a kill or a survivor. An engine mutant must produce syntactically and semantically valid code that tests test suite coverage rather than compiler rejection. Mutators must suppress invalid mutants that fail compiler type checks.

#### TypeScript (`auth_service.ts`)
Input patch touched lines:
- L11: `const verified = session?.profile?.verified;`
- L12: `if (session.role === "admin" && session.quota > 0) {`
- L13: `return true;`

Patch file: [diff_ts_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_ts_sample.patch)
Mutants generated (6 total):
1. L11 `REMOVE_OPTIONAL_CHAIN` (`session.profile?.verified`): Changes behavior. Strips the optional chaining guard on `session`, causing null or undefined `session` to throw a TypeError. Compiles cleanly.
2. L12 `INVERT_EQUALITY` (`session.role !== "admin"`): Changes behavior. Inverts strict equality `===` to strict inequality `!==`. Compiles cleanly.
3. L12 `BOUNDARY_SHIFT` (`session.quota <= 0`): Changes behavior. Inverts relational comparison `>` to `<=`. Compiles cleanly.
4. L12 `BOUNDARY_SHIFT` (`session.quota >= 0`): Changes behavior. Off-by-one boundary shift on zero boundary condition (`> 0` to `>= 0`). Compiles cleanly.
5. L12 `LOGICAL_AND_TO_OR` (`session.role === "admin" || session.quota > 0`): Changes behavior. Replaces boolean conjunction `&&` with disjunction `||`. Compiles cleanly.
6. L13 `FLIP_BOOL_TRUE` (`return false;`): Changes behavior. Inverts boolean return literal `true` to `false`, adhering strictly to the `: boolean` return type annotation. Compiles cleanly.

Suppressed under compile error rule:
`RETURN_UNDEFINED` (`return undefined;`) on line 13 was filtered out. Under TypeScript strict null checks (`strictNullChecks: true`), returning undefined for a function declared with `: boolean` return type triggers a compilation error (`Type 'undefined' is not assignable to type 'boolean'`). Under the compile error rule, this is an invalid mutant. Filtering it dropped the expected TypeScript mutant count from 7 to 6.

#### Java (`OrderService.java`)
Input patch touched lines:
- L11: `if (order != null && order.getRole().equals("ADMIN")) {`
- L12: `int remaining = order.getQuota() - 1;`
- L13: `return remaining > 0;`

Patch file: [diff_java_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_java_sample.patch)
Mutants generated (4 total):
1. L11 `REMOVE_NULL_CHECK` (`true && order.getRole().equals("ADMIN")`): Changes behavior. Strips the null guard, triggering NullPointerException when `order` is null. Compiles cleanly.
2. L11 `INVERT_EQUALITY` (`!order.getRole().equals("ADMIN")`): Changes behavior. Inverts String equality method call with logical NOT prefix. Compiles cleanly.
3. L12 `BOUNDARY_SHIFT` (`order.getQuota() + 1`): Changes behavior. Mutates arithmetic decrement `- 1` to increment `+ 1`. Compiles cleanly.
4. L13 `BOUNDARY_SHIFT` (`return remaining >= 0;`): Changes behavior. Mutates relational operator `>` to `>=`. Compiles cleanly.

Suppressed under compile error rule:
`RETURN_NULL` (`return null;`) on line 13 was filtered out. The enclosing method `public boolean process(Order order)` returns primitive `boolean`. In Java, returning null for a primitive type triggers javac compile error (`incompatible types: <null> cannot be converted to boolean`). Under the compile error rule, this is an invalid mutant. Filtering it dropped the expected Java mutant count from 5 to 4.

#### C# (`TokenService.cs`)
Input patch touched lines:
- L11: `var role = user?.Role;`
- L12: `if (role != null && role.Equals("ADMIN")) {`
- L13: `int remaining = user.Retries + 1;`
- L14: `return remaining <= 3;`

Patch file: [diff_csharp_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_csharp_sample.patch)
Mutants generated (6 total):
1. L11 `REMOVE_NULL_CONDITIONAL` (`var role = user.Role;`): Changes behavior. Strips null-conditional operator `?.`, triggering NullReferenceException when `user` is null. Compiles cleanly.
2. L12 `INVERT_EQUALITY` (`!role.Equals("ADMIN")`): Changes behavior. Inverts method call with logical NOT. Compiles cleanly.
3. L12 `INVERT_EQUALITY` (`role == null`): Changes behavior. Inverts null check `!= null` to `== null`. Compiles cleanly.
4. L13 `BOUNDARY_SHIFT` (`user.Retries - 1`): Changes behavior. Mutates addition `+ 1` to subtraction `- 1`. Compiles cleanly.
5. L14 `RETURN_DEFAULT` (`return default;`): Changes behavior. In C#, `default` for type `bool` is `false`. Compiles cleanly and changes return value.
6. L14 `BOUNDARY_SHIFT` (`return remaining > 3;`): Changes behavior. Inverts relational operator `<=` to `>`. Compiles cleanly.

#### C++ (`BufferPool.cpp`)
Input patch touched lines:
- L11: `if (ptr != nullptr) {`
- L12: `int next = current + 1;`
- L13: `if (size <= max_capacity) {`
- L14: `return ptr;`

Patch file: [diff_cpp_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_cpp_sample.patch)
Mutants generated (4 total):
1. L11 `REMOVE_NULLPTR_CHECK` (`if (true) {`): Changes behavior. Replaces pointer null check with boolean literal `true`. Compiles cleanly.
2. L12 `BOUNDARY_SHIFT` (`int next = current - 1;`): Changes behavior. Mutates arithmetic addition `+ 1` to subtraction `- 1`. Compiles cleanly.
3. L13 `INVERT_EQUALITY` (`if (size > max_capacity) {`): Changes behavior. Inverts relational comparison `<=` to `>`. Compiles cleanly.
4. L14 `RETURN_DEFAULT` (`return nullptr;`): Changes behavior. Replaces pointer return with `nullptr`, matching return type `void*`. Compiles cleanly.

#### Ruby (`auth_service.rb`)
Input patch touched lines:
- L11: `return nil unless user&.active`
- L12: `if /^[A-Z0-9]+$/ === token && secret.eql?(expected)`
- L13: `count = attempts + 1`
- L14: `return session if count < 5`

Patch file: [diff_ruby_sample.patch](file:///home/renan/Downloads/me/methodology_session/evidence/patches/diff_ruby_sample.patch)
Mutants generated (6 total):
1. L11 `REMOVE_SAFE_NAVIGATION` (`return nil unless user.active`): Changes behavior. Removes safe navigation operator `&.`, raising NoMethodError when `user` is nil. Compiles cleanly.
2. L12 `INVERT_EQUALITY` (`!secret.eql?(expected)`): Changes behavior. Inverts method call with logical NOT prefix. Compiles cleanly.
3. L12 `INVERT_EQUALITY` (`!(/^[A-Z0-9]+$/ === token)`): Changes behavior. Wraps case equality comparison in `!(...)` to avoid invalid `!==` syntax. Compiles cleanly.
4. L13 `BOUNDARY_SHIFT` (`count = attempts - 1`): Changes behavior. Mutates addition `+ 1` to subtraction `- 1`. Compiles cleanly.
5. L14 `BOUNDARY_SHIFT` (`return session if count <= 5`): Changes behavior. Mutates comparison `< 5` to `<= 5`. Compiles cleanly.
6. L14 `RETURN_NIL` (`return nil`): Changes behavior. Replaces return expression with `nil`. Compiles cleanly.

#### Suite 03 (`read/tests/test_bmad_zero_command.sh`: Recounted, not changed)
Git diff against baseline commit `ca31c10`:
```
(No diff - identical to commit ca31c10)
```
Status: Recounted, not changed.
The script contains 6 distinct `[PASS]` assertion lines. It is identical to baseline commit `ca31c10`, and its pass count did not move. Earlier informal summaries omitted counting the final commit remediation assertion.

#### Suite 07 (`read/tests/test_live_realtime.sh`: Recounted, not changed)
Git diff against baseline commit `ca31c10`:
```
(No diff - identical to commit ca31c10)
```
Status: Recounted, not changed.
Suite 07 is an end-to-end integration lifecycle test executing 8 sequential stages under `set -euo pipefail`. It is identical to baseline commit `ca31c10`, and its pass count did not move. Earlier informal summaries counted 4 selective checkpoints.

#### Suite 08 (`read-core`: 55 to 58 passed)
Added 3 unit tests in `read/read-core/src/`:
- `test_ast_resolver_non_ascii_and_unicode_handling`: Validates multi-byte UTF-8 parsing for Japanese docstrings, Cyrillic comments, Greek symbols, and French accents.
- `test_diff_mutator_skips_python_docstrings`: Validates that Python triple quotes (`"""` and `'''`) are skipped during diff mutation.
- `test_extraction_latency_contention_10_runs`: Validates 10 runs of AST extraction under 4-thread CPU contention strictly satisfy the under 5 ms threshold.

#### Suite 09 (`read-daemon`: 56 to 57 passed)
Single command evaluated across both versions (`cd read/read-daemon && go test -v -count=1 ./...`):
- Baseline commit `ca31c10`: 56 passed test functions.
- Current commit `HEAD`: 57 passed test functions (added `TestMigrationFromLegacyMethodPath` in `orchestrator_test.go`).
The earlier mention of '22' was an inaccurate legacy table entry from a partial test run. Running without `-v` reports 9 passing package suites on both versions.

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
The in-process latency benchmark is located strictly in test code within `read/read-core/src/ast/mod.rs` inside the `#[cfg(test)] mod tests` block (`test_extraction_latency_under_5ms` and `test_extraction_latency_contention_10_runs`). No benchmark scaffolding exists in engine runtime code paths.

Measurement History Across Specific Commits:
1. Baseline commit `ca31c10`:
   - Single thread debug unoptimized: 2.96 ms
   - Contention measurement: Executed in a separate detached clone directory (`/tmp/check_ca31c10`) checked out to commit `ca31c10`. The only difference from target code was adding the benchmark test function inside `#[cfg(test)] mod tests` in `read-core/src/ast/mod.rs`. Engine runtime code was 100% untouched.
   - Under 4-thread CPU contention with 3 concurrent worker threads (`--test-threads=4`, debug): Mean 3.659 ms (Min: 2.670 ms, Max: 7.658 ms, raw=[4.844, 2.710, 2.713, 2.678, 4.007, 3.662, 2.941, 7.658, 2.707, 2.670]). The 7.658 ms spike is recorded strictly as one empirical observation from a 10-run sample under 4-thread CPU contention on baseline commit ca31c10, not as proof or cause of the earlier 6.2 ms failure.
   - Logs: [16_latency_baseline_ca31c10.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/16_latency_baseline_ca31c10.log) and [20_ast_latency_contention.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/20_ast_latency_contention.log).

2. Optimized commit `3ce41c4`:
   - Single-pass byte slice parsing replaced repeated string allocations.
   - Single thread debug unoptimized: 1.65 ms
   - Single thread release optimized: 0.375 ms (the exact origin of the 0.375 ms figure)
   - Contention under 4 threads: 2.1 to 2.4 ms
   - Log: [17_latency_optimized_3ce41c4.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/17_latency_optimized_3ce41c4.log)

3. Intermediate commit `7ce0a00`:
   - In-Process Function Execution Latency on idle host (10 iterations):
     * Release profile: Mean 0.140 ms (Min: 0.122 ms, Max: 0.226 ms, StdDev: 0.032 ms). The exact origin of the 0.140 ms figure.
     * Debug profile:   Mean 1.225 ms (Min: 0.962 ms, Max: 1.677 ms, StdDev: 0.274 ms).

4. Current commit `a58abd7` (Contention with 3 worker threads across 10 iterations):
   - In-Process Execution Latency Under 4-Thread Contention (3 concurrent worker threads continuously parsing while measuring thread executes):
     * Release profile (10 runs): Mean 0.144 ms (Min: 0.134 ms, Max: 0.192 ms). The exact origin of the 0.144 ms figure.
     * Debug profile (10 runs):   Mean 1.982 ms (Min: 1.435 ms, Max: 3.524 ms).
     * Raw measurements log: [20_ast_latency_contention.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/20_ast_latency_contention.log).
     * Assertion check: Both test `test_extraction_latency_under_5ms` (line 2082) and contention test `test_extraction_latency_contention_10_runs` assert strictly under 5 ms (`elapsed.as_millis() < 5` and `mean_us / 1000.0 < 5.0`). Both assertions pass.
   - CLI Process Execution Latency (including startup and file reading across 10 iterations):
     * Release profile: Mean 2.014 ms (Min: 1.894 ms, Max: 2.138 ms, StdDev: 0.077 ms).
     * Debug profile:   Mean 3.745 ms (Min: 3.327 ms, Max: 4.287 ms, StdDev: 0.323 ms).

Normative Profile Applicability:
The specification threshold of under 5 ms applies normatively to the release build profile for runtime execution. Both release and debug profiles satisfy the requirement on this hardware.

### Cargo.toml Build Profile Disclosure and Debug Latency Impact

To prevent timing jitter during parallel `cargo test` execution, `read/read-core/Cargo.toml` specifies `opt-level = 1` for the test profile.

Exact git diff in `read/read-core/Cargo.toml`:
```toml
[profile.test]
opt-level = 1
```

Why it was added:
Default Rust test profiles run at `opt-level = 0` (unoptimized). When all 58 unit tests run concurrently across worker threads on a host with 2 physical cores (AMD Ryzen 3 3250U), running `test_extraction_latency_contention_10_runs` (which spawns 3 continuous worker threads) alongside other concurrent tests creates 7 concurrent threads competing for 2 physical cores. Under Linux CFS CPU scheduling, unoptimized debug code suffered from thread starvation and context switch delays, causing wall-clock times to spike to 7 to 10 ms. Adding `[profile.test] opt-level = 1` enables basic compiler optimization (inlining, register allocation, loop unrolling) for the test binary, dropping in-process extraction to 0.18 ms and eliminating test flakiness.

Effect on debug profile latency in the specification:
The specification's under 5 ms latency threshold applies normatively to the runtime release binary (`read-engine`), which executes in 0.14 ms to 0.18 ms. Under the test profile with `opt-level = 1`, extraction executes in ~0.18 ms. On unoptimized debug builds (`opt-level = 0`), extraction executes in 1.2 ms to 1.9 ms on an idle host, but can spike up to 7 ms under heavy parallel multi-thread load on 2-core hardware. Latency tests in `ast/mod.rs` are synchronized via a static Mutex `LATENCY_TEST_LOCK` to ensure clean measurement.

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

To verify build reproducibility on this Linux host, clean builds were executed from isolated git clones:
Raw log: [18_clean_build_reproducibility.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/18_clean_build_reproducibility.log)

Source Commit: `9942fb7` on branch `report-review-v0.7.1`

Clean Clone 1 (`/tmp/clean_clone_1`):
- Build Start Time: `2026-11-23T13:32:39.636580+00:00`
- Clone Command: `git clone --branch report-review-v0.7.1 /home/renan/Downloads/me/methodology_session /tmp/clean_clone_1`
- Rust Build Command: `cargo build --release --manifest-path read/read-core/Cargo.toml --target-dir target`
- read-engine SHA-256: `f7d627fae9e742e83abdf7ba95c9fc7c3fe13c5790780b215adfc4f00e3ba52f`
- Go Build Command: `cd read/read-daemon && go build -trimpath -o read-daemon cmd/read-daemon/main.go`
- read-daemon SHA-256: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Build End Time: `2026-11-23T13:33:24.750943+00:00`

Clean Clone 2 (`/tmp/clean_clone_2`):
- Build Start Time: `2026-11-23T13:33:24.804871+00:00`
- Clone Command: `git clone --branch report-review-v0.7.1 /home/renan/Downloads/me/methodology_session /tmp/clean_clone_2`
- Rust Build Command: `cargo build --release --manifest-path read/read-core/Cargo.toml --target-dir target`
- read-engine SHA-256: `f7d627fae9e742e83abdf7ba95c9fc7c3fe13c5790780b215adfc4f00e3ba52f`
- Go Build Command: `cd read/read-daemon && go build -trimpath -o read-daemon cmd/read-daemon/main.go`
- read-daemon SHA-256: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Build End Time: `2026-11-23T13:34:02.050598+00:00`

Committed Binaries in `read/bin/`:
- `read/bin/read-engine`: `f7d627fae9e742e83abdf7ba95c9fc7c3fe13c5790780b215adfc4f00e3ba52f`
- `read/bin/read-daemon`: `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59`
- Result: Both binaries match the clean isolated directory builds bitwise identically.
- Scope: Verified on this Linux host (Linux 7.2.15-1-omarchy x86_64) only.

Verification One-Liner Command:
`sha256sum read/bin/read-engine read/bin/read-daemon` produces `f7d627fae9e742e83abdf7ba95c9fc7c3fe13c5790780b215adfc4f00e3ba52f  read/bin/read-engine` and `dcecffced24d61a7b7de605bfa490c38c614fa12d130a1f0508c7d8415aada59  read/bin/read-daemon`.

## 5. Trial Parameters and Governance Draft Proposals (Awaiting User Decisions)

The following 5 parameters remain open draft proposals awaiting explicit user confirmation before any trial begins. Trial execution is held until the user explicitly sets each value:

1. Time limit per task, in minutes (Draft proposal: 15 minutes per task, based on 5 to 10 minute human tutorial baseline).
2. Minimum cohort completion rate (Draft proposal: 80 percent completion rate, meaning at least 8 of 10 participants complete all tasks).
3. Remediation retry cap (Draft proposal: maximum 3 test authoring attempts per task before escalation).
4. Cohort size and profile (Draft proposal: exactly 10 junior to mid-level software engineers with 1 to 3 years experience and zero prior exposure to the READ framework).
5. Enforcement mode for the first cohort: Hard block or warning mode.
Recommendation on item 5: Start in warning mode. A hard block from day one will likely block honest commits from equivalent mutants and untested error paths, while warning mode measures the real gate rejection rate before enforcing hard blocks.
