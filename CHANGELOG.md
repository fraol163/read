# Changelog

All notable changes to the READ Framework will be documented in this file.

## [v0.7.1] - 2026-11-23

### Changed
- Reverted AROP mutation verification gate from 80% to strictly 100.0% kill threshold. Any surviving mutant triggers an immediate commit abort.
- Standardized canonical state storage path on `.read/STATE.json` and `.read/WAL.log` with backward compatible fallback for `.method/`.
- Replaced hardcoded 2 GB `RLIMIT_AS` default with dynamic calibration based on 30% of detected system available memory. Default fallback set to 512 MB.
- Corrected Master Override documentation to state that overrides provide cryptographic integrity and auditability via SHA-256 hash chaining, not non-repudiation.
- Narrowed crash recovery latency specification from under 10 ms to 7.5 ms through 16.5 ms based on empirical multi-run benchmarks including git process creation.
- Removed zero-allocation PTY streaming claims after microbenchmarks measured 993 B/op and 4 allocations per frame.
- Relabeled demo directories in `examples/` as test fixtures and reference implementations.
- Removed mobile-specific scaffolding and terminology (`phone_only`, `RolePhone`) across core code, configurations, and documentation.
- Corrected BMAD acronym across all documents to Breakthrough Method for Agile AI-Driven Development (`bmad-code-org/BMAD-METHOD`, tag `v6.12.4`).

### Added
- Multi-language AST operators and test harness runners for TypeScript, JavaScript, Java, C#, C++, and Ruby.
- Cryptographic WAL git tag anchoring (`read-wal-head`) via subcommands `anchor-wal` and `verify-wal-anchor` to detect rewritten WAL logs.
- Daemon network security surface: `--tls-cert` and `--tls-key` enforcement, constant-time bearer token authentication, and token bucket rate limiting (10 burst, 2 sustained per second).
- Server-side CI [UNTESTED] enforcement workflow in `.github/workflows/read-gate.yml` to reject client-side `git commit --no-verify` hook bypasses.
- Threat modeling documentation in `read/docs/THREAT_MODEL.md` detailing PTY shell exposure, reverse proxy XFF spoofing, and advisory flock limits.
- Empirical evidence dossier in `evidence/EVIDENCE.md` linking directly to raw execution logs in `evidence/logs/`.
- Claim audit ledger in `evidence/CLAIM_AUDIT.md` tracking the status of 15 technical claims.
- Ten-minute student onboarding guide in `read/docs/TUTORIAL.md`.
- Gaps and architectural limitations catalog in `read/docs/GAPS.md`.
- Independent architectural evaluation in `read/docs/ARCHITECTURE_REVIEW.md`.
- Dedicated vision and philosophical divergence documentation in `read/docs/VISION.md`.
- Normative RFC 2119 operational rules in `read/docs/SPECIFICATION.md`.

### Fixed
- Fixed template initialization in `read/template/.read/STATE.json` where a stale lease held by `declarative_engine` blocked CLI state transitions.
- Fixed orphan process handling in `read-daemon/pty` by terminating process trees via `KillProcessTree` before closing master PTY file descriptors.
- Fixed mutation generation in `read-core/src/mutation/mod.rs` to ignore documentation, logs, and framework metadata files located in `.read/` or `.method/`.
- Optimized Python AST extraction to single-pass byte scanning, eliminating multi-pass latency spikes under thread contention.
