# Threat Model and Security Assessment

This document outlines the security boundaries, threat vectors, and architectural mitigations of the READ Framework.

## 1. System Trust Boundaries

The framework operates across three distinct privilege zones:
1. Workspace Filesystem: Unprivileged repository files subject to modification by developers, AI agents, and build tools.
2. READ Core Engine (read-engine): Local Rust binary running with developer user privileges, executing CAS transitions and managing cryptographic logs.
3. READ Daemon (read-daemon): Long running Go process managing PTY sessions, file synchronization, API gateways, and child process execution.

## 2. Threat Analysis and Known Vectors

### 2.1 WebSocket PTY Shell Execution
- Threat: The `/api/v1/pty` endpoint exposes an interactive pseudoterminal session. Any client with bearer token access can execute arbitrary shell commands under the daemon user account.
- Mitigation: The daemon creates isolated process groups via `setpgid`. On client disconnect, the daemon issues `SIGTERM` followed by `SIGKILL` to the entire process group (`kill(-pgid)`).
- Residual Risk: If an agent spawns double-forked background daemons that decouple their process group, processes may survive session termination.

### 2.2 Static Bearer Token Authentication
- Threat: Authentication relies on a single static bearer token passed via CLI argument `--auth-token` or configuration file.
- Risk Analysis:
  1. Process Table Exposure: Passing `--auth-token` on the command line exposes the secret to local unprivileged users via `/proc` or `ps aux`.
  2. Lack of Rotation: The token is static for the lifetime of the daemon process.
  3. Lack of Scopes: All authorized clients receive full root-level control over workspace files, PTY execution, and pipeline triggers. There are no read-only or staged-only scopes.
- Mitigation: Deployments should supply credentials through environment variables or files with `0600` permissions rather than command line arguments.

### 2.3 Reverse Proxy Header Spoofing (X-Forwarded-For)
- Threat: The rate limiter checks `X-Forwarded-For` before checking `r.RemoteAddr`.
- Measured Vulnerability: If an external attacker sends `X-Forwarded-For: 127.0.0.1`, the daemon misidentifies the client as a local loopback connection and exempts the client from rate limiting.
- Mandatory Mitigation: Reverse proxies (such as Nginx or Caddy) placed in front of `read-daemon` must explicitly strip incoming `X-Forwarded-For` headers from untrusted clients and replace them with the verified client IP.

### 2.4 Advisory Locking on State Files
- Threat: State transitions use POSIX `libc::flock` on `.read/STATE.json`.
- Analysis: POSIX file locks are advisory on Linux. Any process that opens `.read/STATE.json` and writes directly to it without calling `flock` can overwrite state data without error.
- Mitigation: The framework verifies the cryptographic state hash against the append-only `WAL.log` on every CAS transition. Tampering with `STATE.json` causes a state hash mismatch on the next operation.

### 2.5 Master Override Rationale Governance
- Threat: Developers or autonomous agents can bypass verification failures by issuing `read-engine override <OPERATOR> <RATIONALE>`.
- Analysis: The engine checks only that the operator identifier is non-empty and the rationale string contains at least 30 characters. This string length check does not verify technical correctness or genuine human authorization.
- Mitigation: The override writes an immutable entry into `WAL.log` and stamps the verification hash with the `OVR-` prefix. This provides cryptographic auditability in the WAL history. Full enterprise governance requires server-side CI enforcement and dual-key authorization rather than client-side length checks.

### 2.6 Pre-Commit Hook Bypass (git commit --no-verify)
- Threat: Git executes client-side hooks locally. Any user or script can bypass the verification gate by running `git commit --no-verify`.
- Mitigation: Client hooks are a developer convenience. True policy enforcement requires a server-side gate on the CI/CD pipeline (such as `.github/workflows/read-gate.yml`) that rejects pushes lacking a verified state and a valid WAL git tag anchor.
