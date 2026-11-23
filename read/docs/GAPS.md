# Gaps, Trade-offs, and Roadmap

This document outlines architectural limitations, technical debt, and missing features in the READ Framework (v0.7.1).

## 1. Empirical Baseline Comparison Plan (BMAD vs READ)

While READ is designed as the deterministic antithesis to BMAD, formal side-by-side empirical benchmarks have not yet been completed.
To validate claims of token and cost reduction, the following benchmark protocol is planned:

### 1.1 Methodology
- Benchmark Tasks: Select ten standard coding tasks from SWE-bench Lite (ranging from bug fixes to new feature additions).
- Evaluation Arms:
  - Arm A (BMAD v6.12.4): Execute using standard BMAD multi-agent prompt flows with Claude 3.5 Sonnet.
  - Arm B (READ v0.7.1): Execute using READ pipeline with the same underlying model invoked purely for code generation in Stage 3.
- Metrics to Measure:
  1. Total Input and Output Tokens consumed.
  2. Total API Dollar Cost per completed task.
  3. Total Wall-Clock Execution Latency in seconds.
  4. Code Quality via Surviving Mutant Count (using external mutation tools such as mutmut or cosmic-ray).
  5. Task Success Rate (percentage of tasks passing ground-truth acceptance tests).

## 2. TASK.md Contract Authoring Overhead

Writing formal `TASK.md` contracts manually imposes cognitive load on developers.
- Current Reality: The developer must manually define preconditions, postconditions, and test invariants in markdown before initiating a task.
- Risk: Developers may write superficial or trivial contracts to bypass the friction, weakening contract effectiveness.
- Planned Solution: Implement `read contract scaffold` to generate initial contracts automatically from GitHub issues, Jira tickets, or existing unit test suites.

## 3. Partial Staging (git add -p) Blind Spot

- Current Reality: AROP diff hashing relies on `git diff --cached`. If a developer stages only part of their changes (using `git add -p` or staging individual files), the verification gate tests the staged index against the workspace filesystem.
- Risk: If workspace files contain unstaged modifications that the test suite imports, the test suite may pass against unstaged code, leading to state inconsistencies.
- Planned Solution: Implement git stash or temporary shadow tree execution during Stage 4 mutation verification so tests execute against staged files exclusively.

## 4. Binary Distribution and Cryptographic Release Signing

- Current Reality: Release tarballs are packaged and compressed into `read/dist/read-v0.7.0-linux-x86_64.tar.gz`. They lack cryptographic signatures (such as Minisign, Cosign, or GPG).
- Risk: Users downloading binaries have no cryptographic mechanism to verify binary authenticity against supply-chain tampering.
- Planned Solution: Integrate Minisign or Cosign key signing into GitHub release workflows with published SHA-256 checksums.

## 5. Master Override Governance Limitations

- Current Reality: Applying a Master Override requires only a non-empty operator name and a 30-character rationale string.
- Risk: A single developer or an automated agent script can easily generate 30 characters of filler text to force handoff completion on failing code.
- Planned Solution: Introduce multi-party authorization requiring cryptographic signatures from two distinct team members before an override transitions state.

## 6. Operating System Portability (Linux Exclusivity)

- Current Reality: READ currently supports only Linux x86_64 and aarch64.
- Technical Blockers on Other Platforms:
  1. Dynamic RLIMIT: `syscall.RLIMIT_AS` and `/proc/meminfo` are specific to Linux. macOS and Windows do not support `RLIMIT_AS` in the same manner.
  2. Advisory File Locking: `libc::flock` exhibits different semantics on Windows network filesystems.
  3. PTY Handling: PTY allocation relies on Linux termios and `/dev/ptmx`.
- Planned Solution: Create platform abstraction layers for macOS (`kqueue`, BSD resource limits) and Windows (Job Objects, Named Pipes).

## 7. Large Diff Mutation Truncation

- Current Reality: The AROP engine caps mutants at 10 sequentially.
- Risk: In a commit touching 500 lines across 10 files, only the first file or hunk receives mutants. The remaining 9 files receive zero mutation testing.
- Planned Solution: Replace first-fit sequential truncation with stratified sampling across all modified files and functions.
