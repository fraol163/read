# READ Framework Claim Audit

This document records the empirical verification status of all technical claims made in previous specifications, marketing documents, and architecture reports.

| # | Technical Claim | Original Source Location | Empirical Status | Evidence Reference | Action Taken |
|---|---|---|---|---|---|
| 1 | BMAD stands for Break, Measure, Adapt, Deliver | BMAD_SYNTHESIS.md, Section 1 | Refuted | Official GitHub repo `bmad-code-org/BMAD-METHOD` (tag `v6.12.4`) states Breakthrough Method for Agile AI-Driven Development | Corrected terminology across all documentation |
| 2 | State path is canonically `.method/STATE.json` | METHODOLOGY_TRACK.md, Section 2.1 | Refuted | Directory naming was inconsistent with framework identity | Standardized strictly on `.read/STATE.json` as exclusive canonical path, adding automatic migration and unit test `test_migration_from_legacy_method_path` |
| 3 | CAS state machine executes in sub-millisecond time | ARCHITECTURE.md, Section 1 | Verified | In-memory atomic CAS and file serialization execute in under 0.8 ms on local NVMe disk | Kept claim with empirical benchmark note |
| 4 | Crash recovery completes in under 10 ms | MASTER_REPORT.md, Section 2 | Narrowed | Measured at 7.586 ms to 16.455 ms (mean 11.490 ms) on dual-core hardware when git process spawn is included | Narrowed claim to 7.5 to 16.5 ms |
| 5 | C0 AST extraction completes in under 5 ms | MASTER_REPORT.md, Section 3 | Verified | Measured at 1.762 ms to 2.374 ms across 8 languages (Python, TS, Go, Rust, Java, C#, C++, Ruby) | Kept claim |
| 6 | C0 AST extraction costs zero API tokens | ARCHITECTURE.md, Section 2 | Verified | Extraction runs entirely via local line scanning without external network or LLM API calls | Kept claim |
| 7 | Zero allocation PTY streaming | ARCHITECTURE.md, Section 4 | Refuted | `BenchmarkPtyStreaming` measured 993 B/op and 4 allocations per frame | Removed claim from all documentation |
| 8 | AROP mutation verification strictly enforces 100% kill rate | LIFECYCLE.md, Section 4 | Verified | Reverted 80% threshold back to strictly 100.0%. Surviving mutants trigger abort | Enforced invariant in code and chaos tests |
| 9 | Dynamic RLIMIT_AS 30% RAM ceiling supports all harnesses | SECURITY_MODEL.md, Section 3 | Refuted for Java/.NET | `RLIMIT_AS` limits virtual memory. JVM and .NET CoreCLR abort during initialization | Added documentation explaining `RLIMIT_AS` behavior on managed runtimes |
| 10 | Gateway rate limiter prevents abusive external traffic | SECURITY_MODEL.md, Section 6 | Partially Verified | Rate limiter throttles external IPs, but blind trust in `X-Forwarded-For` allows spoofing loopback | Documented spoofing risk and reverse proxy requirements in THREAT_MODEL.md |
| 11 | Pre-commit hook guarantees unverified code cannot be committed | LIFECYCLE.md, Section 5 | Refuted | Local Git hooks can be bypassed via `git commit --no-verify` | Implemented server-side CI gate in `.github/workflows/read-gate.yml` |
| 12 | Advisory flock guarantees absolute state integrity | ARCHITECTURE.md, Section 1 | Partially Verified | `libc::flock` coordinates cooperating processes, but uncooperating text editors can write directly | Documented lock limitations and added WAL git tag anchoring for tamper detection |
| 13 | Master Override provides cryptographically signed non-repudiation | METHODOLOGY_TRACK.md, Section 2.4 | Refuted | Implemented primitive is SHA-256 hash chaining, not asymmetric digital signatures (Ed25519) | Corrected terminology to hashed and immutably logged tombstone |
| 14 | 30-character rationale enforces operator accountability | METHODOLOGY_TRACK.md, Section 2.4 | Refuted | String length checks verify character count, not forensic justification | Documented governance limitation in THREAT_MODEL.md and GAPS.md |
| 15 | Mutation budget cap of 10 mutants is representative of large diffs | LIFECYCLE.md, Section 4 | Narrowed | Mutant generation takes only the first 10 mutants sequentially, ignoring diff hunks past the cap | Documented mutant truncation in SPECIFICATION.md and GAPS.md |
