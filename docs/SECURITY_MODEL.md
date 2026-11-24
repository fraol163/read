# READ Security Architecture and Threat Model

## 1. Threat Model for Autonomous AI Agents

Autonomous AI agents (such as Claude Code, DeepSeek Harness, and Aider) execute tools, generate shell commands, and modify local file systems. When running with minimal human supervision, several critical security threats emerge:

1. **Filesystem Escape**: An agent generates malicious or malformed relative paths (`../../../../etc/shadow` or `../../.ssh/id_rsa`), reading or overwriting sensitive host files.
2. **Symlink Traversal**: An agent creates a symlink inside the workspace pointing to an external directory (e.g. `ln -s /etc ./config_link`), attempting to bypass directory boundaries.
3. **Secret Token Exfiltration**: An agent prints runtime environment variables, auth headers, or configuration files containing sensitive API keys into stdout, terminal logs, or git commits.
4. **Runaway Zombie Processes**: An agent spawns background tasks (e.g. web servers, infinite compilation loops, unkillable test runners) that linger after the agent exits, consuming host RAM and CPU.
5. **State Tampering**: A compromised agent or external script attempts to forge verification proofs by editing local state files to bypass testing gates.

READ treats external agents as untrusted execution entities and enforces four mandatory security invariants.

```
+-----------------------------------------------------------------------------------------+
|                                READ SECURITY PERIMETER                                  |
|                                                                                         |
|   Untrusted Agent / External Process                                                    |
|        |                                                                                |
|        +---> Attempt Path Traversal ("../../.ssh") ---> Blocked: ErrPathForbidden      |
|        +---> Attempt External Symlink Target      ---> Blocked: Symlink Boundary Check  |
|        +---> Output Emits API Key / Token         ---> Masked: ***REDACTED***           |
|        +---> Spawns Detached Subprocesses         ---> Trapped: Dedicated PGID Group   |
|        +---> Forges verify_hash in STATE.json     ---> Rejected: WAL Cryptographic Mismatch|
|                                                                                         |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Invariant 1: Process Group Isolation and Clean Cleanup

### 2.1 The Problem of Zombie Agent Processes
When an AI agent executes shell commands, it frequently spawns subprocesses:
```bash
python3 -m http.server 8000 &
cargo watch -x check
```
If the parent agent crashes, is killed by an OOM event, or receives `SIGTERM`, standard process termination only kills the top-level PID. The spawned child processes become orphaned, adopted by PID 1 (`init` / `systemd`), and continue consuming resources and listening on network ports.

### 2.2 Process Tree Containment (`read-daemon/security/process.go`)
READ neutralizes this vector through strict POSIX process group boundaries:

1. **New Process Group**: Every process spawned by the daemon is configured with:
   ```go
   cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
   ```
   This guarantees the command and all its descendants belong to a distinct process group ID (`PGID`).
2. **Descendant Tree Construction**:
   When stopping a harness (`KillProcessTree`), the daemon scans `/proc` to construct an in-memory graph of parent-child relationships, discovering all descendant PIDs even if they detached from the parent.
3. **Signal Escalation Protocol**:
   - First, `SIGTERM` is sent to the negative process group (`-pgid`) and all enumerated descendants.
   - A 50-millisecond grace period allows processes to flush write buffers and close file descriptors.
   - Second, `SIGKILL` is sent unconditionally to `-pgid` and all descendant PIDs.
4. **Post-Termination Audit**:
   The daemon inspects `/proc` again to confirm that zero processes remain with that PGID. If any stubborn processes exist, they are re-signaled with `SIGKILL`.

---

## 3. Invariant 2: Filesystem Boundaries and Symlink Traversal Prevention

### 3.1 Strict Workspace Boundary Enforcement (`read-daemon/security/path.go`)
Every filesystem operation in the daemon passes through `ResolveWorkspacePath(workspaceRoot, requestedPath)`:

```go
cleanPath, err := security.ResolveWorkspacePath(workspaceRoot, requestedPath)
if err != nil {
    // Access denied: ErrPathForbidden
}
```

### 3.2 Multi-Layered Validation Steps
1. **Realpath Normalization**:
   The workspace root is canonicalized using `filepath.EvalSymlinks` and `filepath.Abs` to resolve any existing symlinks in the host path.
2. **Multi-Round URL Decoding**:
   Attackers often use URL percent-encoding (e.g. `%2e%2e%2f` for `../`) or double encoding (`%252e%252e%252f`). The resolver runs up to three iterative decoding rounds to expose hidden traversal attempts.
3. **Null-Byte and Control Character Rejection**:
   Any path containing null bytes (`\x00`) or control characters is rejected immediately.
4. **Lexical Boundary Check**:
   The candidate path is verified using `filepath.Rel(realRoot, candidateFullPath)`. If the relative path starts with `..` or escapes the root, it is rejected.
5. **Symlink Target Inspection**:
   If the target or any parent directory is a symbolic link:
   - The link target is resolved via `os.Readlink` or `filepath.EvalSymlinks`.
   - The resolved target's canonical location is evaluated against `realRoot`.
   - If the symlink points outside the workspace root, the request is blocked with `ErrPathForbidden`.

---

## 4. Invariant 3: Real-Time Token and Secret Redaction

### 4.1 Token Scrubbing Engine (`read-daemon/security/sanitize.go`)
External agents routinely display environment variables or print curl commands that include authorization credentials.

READ intercepts all text streams, WebSocket terminal buffers, and API responses, passing them through 10 deterministic regex scrubbers:

```go
s := security.SanitizeText(rawOutput)
```

### 4.2 Covered Secret Patterns

| Category | Targeted Pattern | Replacement |
| :--- | :--- | :--- |
| **Anthropic Keys** | `\bsk-ant-[a-zA-Z0-9_-]{20,}\b` | `***REDACTED***` |
| **OpenAI Keys** | `\bsk-(?:proj-\|svcacct-\|none-)?[a-zA-Z0-9_-]{20,}\b` | `***REDACTED***` |
| **DeepSeek Keys** | `\bsk-[a-f0-9]{32,}\b` | `***REDACTED***` |
| **GitHub Tokens** | `\bgh[pousr]-[a-zA-Z0-9]{20,}\b` | `***REDACTED***` |
| **HuggingFace** | `\bhf_[a-zA-Z0-9]{20,}\b` | `***REDACTED***` |
| **Bearer Tokens** | `\b(?i:bearer\s+)([a-zA-Z0-9_\-\.]{15,})\b` | `bearer ***REDACTED***` |
| **Env Assignments**| `\b([A-Z0-9_]*(KEY\|TOKEN\|SECRET\|PASSWORD)[A-Z0-9_]*\s*=\s*)([^\s"';&]+)` | `${1}***REDACTED***` |
| **JSON/YAML Keys** | `(?i)(["']?(api_key\|token\|secret\|password)["']?\s*[:=]\s*["']?)([^"'\s,]+)` | `${1}***REDACTED***` |
| **CLI Flags** | `(?i)(--(api-key\|token\|secret\|password)(?:=\|\s+))([^\s"']+)` | `${1}***REDACTED***` |

### 4.3 Argument and Environment Masking
When commands are launched via `read-daemon`, CLI arguments are scrubbed using `SanitizeArgs`:
- If an argument is `--api-key`, `--token`, or `-k`, the subsequent argument value is replaced with `***REDACTED***`.
- Environment variable arrays (`KEY=VALUE`) are sanitized via `SanitizeEnv`.

---

## 5. Invariant 4: CAS Integrity and Tamper Resistance

### 5.1 Defense Against State Forgery
A common flaw in file-based workflow engines is that an agent can simply edit the state JSON file (e.g., setting `"status": "verified"`) to bypass tests.

READ prevents this mechanically:
1. **Cryptographic Proof Chaining**:
   `.read/STATE.json` cannot transition without a valid `verify_hash`.
2. **WAL Cross-Verification**:
   During `read-engine handoff`, the engine does not merely trust `STATE.json`. It scans `.read/WAL.log` from genesis to verify:
   - For regular verifications: an authentic `VERIFIED` event with the exact `verify_hash` exists in the log.
   - For Master Overrides: an authentic `MASTER_OVERRIDE` event with the exact `OVR-...` hash exists.
3. **Diff Drift Lock**:
   The engine computes `git diff --staged` at the exact moment of handoff. If a single character was altered in staged files after verification ran, the handoff fails with:
   `CasLockViolation: Staged diff hash drift detected! Re-verification required.`
4. **POSIX Lock Protection**:
   All state updates are guarded by `libc::flock(LOCK_EX)`. Manual or concurrent write attempts to `.read/STATE.json` fail or block cleanly.
