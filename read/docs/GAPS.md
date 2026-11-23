# Gaps, Trade-offs, and Roadmap

This document outlines architectural limitations, technical debt, and missing features in the READ Framework (v0.7.1).
Each gap item is assigned an explicit engineering owner.

## 1. Empirical Baseline Comparison Plan (BMAD vs READ)
- Owner: Benchmarking and Evaluation Lead
- Current Status: Planned benchmark protocol.
- Metrics to Measure:
  1. Total Input and Output Tokens consumed per completed task.
  2. Total API Dollar Cost per completed task.
  3. Total Wall-Clock Execution Latency in seconds.
  4. Code Quality via Surviving Mutant Count.
  5. Task Success Rate against ground-truth acceptance tests.
- Tooling for Quality Measurement:
  - Python tasks: `mutmut` mutation engine.
  - TypeScript tasks: `@stryker-mutator/core` (Stryker) mutation engine.
- Benchmark Cost Note:
  Token counts, execution latency, and dollar costs have not been measured empirically for this comparison. Estimating token numbers in advance without measurement violates empirical standards. The protocol requires capturing actual token tallies and billing metrics directly from provider API responses during benchmark runs.

## 2. TASK.md Contract Authoring Overhead
- Owner: Developer Experience Lead
- Current Reality: The developer must manually define preconditions, postconditions, and test invariants in markdown before initiating a task.
- Risk: Developers may write superficial or trivial contracts to bypass the friction, weakening contract effectiveness.
- Planned Solution: Implement `read contract scaffold` to generate initial contracts automatically from GitHub issues, Jira tickets, or existing unit test suites.

## 3. Partial Staging (git add -p) Blind Spot
- Owner: Git Integration Lead
- Current Reality: AROP diff hashing relies on `git diff --cached`. If a developer stages only part of their changes (using `git add -p` or staging individual files), the verification gate tests the staged index against the workspace filesystem.
- Risk: If workspace files contain unstaged modifications that the test suite imports, the test suite may pass against unstaged code, leading to state inconsistencies.
- Planned Solution: Implement git stash or temporary shadow tree execution during Stage 4 mutation verification so tests execute against staged files exclusively.

## 4. Binary Distribution and Cryptographic Release Signing
- Owner: DevOps and Release Security Lead
- Current Reality: Release tarballs are packaged and compressed into `read/dist/read-v0.7.0-linux-x86_64.tar.gz`. They lack cryptographic signatures (such as Minisign, Cosign, or GPG).
- Risk: Users downloading binaries have no cryptographic mechanism to verify binary authenticity against supply-chain tampering.
- Planned Solution: Integrate Minisign or Cosign key signing into GitHub release workflows with published SHA-256 checksums.

## 5. Master Override Governance Limitations
- Owner: Security and Governance Lead
- Current Reality: Applying a Master Override requires only a non-empty operator name and a 30-character rationale string.
- Risk: A single developer or an automated agent script can easily generate 30 characters of filler text to force handoff completion on failing code.
- Planned Solution: Introduce multi-party authorization requiring cryptographic signatures from two distinct team members before an override transitions state.

## 6. Operating System Portability (Linux Exclusivity)
- Owner: Systems Portability Lead
- Current Reality: READ currently supports only Linux x86_64 and aarch64.
- Technical Blockers on Other Platforms:
  1. Dynamic RLIMIT: `syscall.RLIMIT_AS` and `/proc/meminfo` are specific to Linux. macOS and Windows do not support `RLIMIT_AS` in the same manner.
  2. Advisory File Locking: `libc::flock` exhibits different semantics on Windows network filesystems.
  3. PTY Handling: PTY allocation relies on Linux termios and `/dev/ptmx`.
- Planned Solution: Create platform abstraction layers for macOS (`kqueue`, BSD resource limits) and Windows (Job Objects, Named Pipes).

## 7. Large Diff Mutation Truncation
- Owner: Mutation Verification Lead
- Current Reality: The AROP engine caps mutants at 10 sequentially.
- Risk: In a commit touching 500 lines across 10 files, only the first file or hunk receives mutants. The remaining 9 files receive zero mutation testing.
- Planned Solution: Replace first-fit sequential truncation with stratified sampling across all modified files and functions.
