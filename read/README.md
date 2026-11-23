# READ: Resource-conscious, Evidence-driven AI Development

## Architectural Specification and Framework Manual

READ is an executable software engineering framework and process supervision system designed to replace conversational LLM chat workflows with deterministic systems mechanics.

The framework enforces mechanical verification gates, deterministic context assembly, and cryptographic auditability across multi-agent AI development lifecycles.

READ operates on a **declarative, zero-command repository contract model (just like BMAD)**: developers and AI agents interact exclusively through repository markdown files (`.read/TASK.md`, `.read/STATE.json`, `.read/EVIDENCE.json`) and standard `git commit` workflows. Git hooks automatically freeze contracts, execute tests against AROP mutants, and seal verification without requiring manual CLI command execution.

READ is divided into two decoupled execution layers:
1. **read-core**: A native Rust computational engine (edition 2021) enforcing atomic state transitions, zero-token C0 AST context extraction, staged git diff hashing, append-only write-ahead logging (WAL), and asymmetric bounded mutation verification (AROP).
2. **read-daemon**: A Go service managing POSIX process supervision, WebSocket pseudo-terminal (PTY) multiplexing, dynamic agent socket discovery, real-time credential scrubbing, and 3-way storage synchronization across workstation, remote client, and VPS nodes.

---

## Table of Contents

1. [Core Technical Problem Space](#1-core-technical-problem-space)
2. [Dual-Engine Topology](#2-dual-engine-topology)
3. [The 4-Stage Gated Lifecycle](#3-the-4-stage-gated-lifecycle)
4. [Engine Subsystems (read-core)](#4-engine-subsystems-read-core)
   - [4.1 C0 Deterministic AST Context Resolver](#41-c0-deterministic-ast-context-resolver)
   - [4.2 Compare-And-Swap (CAS) State Engine](#42-compare-and-swap-cas-state-engine)
   - [4.3 Cryptographic Write-Ahead Log (WAL)](#43-cryptographic-write-ahead-log-wal)
   - [4.4 AROP Bounded Mutation Testing](#44-arop-bounded-mutation-testing)
   - [4.5 Master Override Audit Trail](#45-master-override-audit-trail)
5. [Daemon Subsystems (read-daemon)](#5-daemon-subsystems-read-daemon)
   - [5.1 Dynamic Agent and Socket Discovery](#51-dynamic-agent-and-socket-discovery)
   - [5.2 Process Supervision and PTY Multiplexing](#52-process-supervision-and-pty-multiplexing)
   - [5.3 3-Way Storage Engine](#53-3-way-storage-engine)
   - [5.4 Real-Time Credential Sanitization](#54-real-time-credential-sanitization)
6. [Command-Line Interface Reference](#6-command-line-interface-reference)
7. [HTTP & WebSocket Gateway API Reference](#7-http--websocket-gateway-api-reference)
8. [Documentation Suite](#8-documentation-suite)
9. [Verified Working Examples](#9-verified-working-examples)
10. [Automated Verification and Chaos Suites](#10-automated-verification-and-chaos-suites)
11. [Learning READ: Adversarial Induction Suite](#11-learning-read-adversarial-induction-suite)

---

## 1. Core Technical Problem Space

Conversational AI programming tools (such as raw chat interfaces, unconstrained agent loops, and prompt-engineered frameworks) suffer from four fundamental failure modes:

1. **Token Bloat and Cost Runaway**: Re-sending complete source trees, multi-page persona prompts, and verbose conversational histories consumes tens of thousands of tokens per iteration without guarantees of semantic accuracy.
2. **Silent Regressions and Tautological Tests**: Agents frequently report successful code generation when tests pass, yet the tests often assert nothing meaningful or fail to guard critical edge cases.
3. **Unenforced Lifecycle Invariants**: Without operating-system and filesystem barriers, an agent can declare a task complete despite introducing syntax errors, missing dependencies, or altering out-of-scope files.
4. **Process Amnesia**: When an execution environment crashes, runs out of memory (OOM), or experiences an interrupted session, conversational tools lose transient context, requiring a full re-spend of tokens to reconstruct working state.

READ eliminates these failure modes through mechanical systems enforcement rather than prompt engineering.

---

## 2. Dual-Engine Topology

```
+-----------------------------------------------------------------------------------+
| Host Systems (PC / VPS / CI Runner)                                               |
|                                                                                   |
|  +------------------------------------+    +-----------------------------------+  |
|  | Discovered AI Agents & Harnesses   |    | Dynamic Socket / Engine Discovery |  |
|  | - DeepSeek Harness (dsh)           |    | - Ollama (11434)                  |  |
|  | - Claude Code CLI (claude)         |    | - llama.cpp (8080)                |  |
|  | - Aider (aider)                    |    | - vLLM (8000)                     |  |
|  +-----------------+------------------+    +-----------------+-----------------+  |
|                    |                                         |                    |
|                    +--------------------+--------------------+                    |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  | read-daemon (Go 1.22+)                                                      |  |
|  | - Process Supervision & Output Multiplexing                                 |  |
|  | - Bidirectional Remote Terminal Streaming (PTY)                             |  |
|  | - 3-Way Storage Engine (pc_only, phone_only, both_sync)                       |  |
|  | - Real-Time Secret Redaction (sk-ant, sk-proj, Bearer, gh tokens)           |  |
|  | - Workspace File Sandbox & Hash Verification                                |  |
|  +--------------------------------------+--------------------------------------+  |
|                                         |                                         |
|                                         v                                         |
|  +-----------------------------------------------------------------------------+  |
|  | read-core (Rust 1.75+)                                                       |  |
|  | - C0 Deterministic AST Context Resolver ($0.00 token cost, <5ms)            |  |
|  | - Compare-And-Swap (CAS) State Engine (.method/STATE.json)                  |  |
|  | - Append-Only Write-Ahead Log (.method/WAL.log, SHA-256 hash chaining)      |  |
|  | - Asymmetric Bounded Mutation Verifier (AROP, <=10 mutants, <4.5s)          |  |
|  | - Staged Diff Hasher (SHA-256 over git diff --staged, <0.1ms)               |  |
|  | - Master Override Audit Trail ([OVERRIDE] with SHA-256 proof)               |  |
|  | - Zero-Token Crash Recovery System (<10ms recovery time)                    |  |
|  +-----------------------------------------------------------------------------+  |
+-----------------------------------------------------------------------------------+
```

---

## 3. The 4-Stage Gated Lifecycle

All software modifications within READ proceed through a strictly gated, 4-stage mechanical lifecycle:

```
+--------+            +--------+            +-----------+            +----------+            +------------------+
|  Idle  | --define-> |Defined | --build--> | Executing | --verify-> | Verified | --handoff->| HandoffComplete  |
+--------+            +--------+            +-----------+            +----------+            +------------------+
```

1. **Define Stage**:
   - Freezes the task contract (task ID, invariants, target files, base commit).
   - Computes SHA-256 contract digest and records it in `.method/STATE.json`.
   - Appends `DEFINE` event to `.method/WAL.log`.
2. **Build Stage**:
   - Acquires CAS worker lease.
   - Extracts C0 AST public interface context ($0.00 token cost).
   - Transitions state to `Executing`.
3. **Verify Stage**:
   - Stages modified code in Git worktree (`git add`).
   - Computes staged diff hash (`git diff --staged` streamed into SHA-256).
   - Runs AROP bounded mutation test suite.
   - Seals `verify_hash` into state and logs `VERIFIED` event in WAL.
   - Alternatively, records an audited human `MASTER_OVERRIDE` with rationale proof.
4. **Handoff Stage**:
   - Mechanically verifies that status is `Verified` and `verify_hash` is non-null.
   - Validates that `verify_hash` matches an authentic WAL cryptographic entry.
   - Checks staged diff stability: blocks handoff if code drifted since verification.
   - Transitions to `HandoffComplete`.

Detailed documentation: [LIFECYCLE.md](docs/LIFECYCLE.md).

---

## 4. Engine Subsystems (read-core)

### 4.1 C0 Deterministic AST Context Resolver
Instead of passing full source trees to LLMs:
- Traverses import declarations up to a configurable depth (default: 3).
- Extracts public function and class signatures, stripping function bodies and private implementation details.
- Supports Python, Rust, Go, TypeScript/JavaScript, Java, C#/.NET, C++, and Ruby.
- Completes in under 5 milliseconds on host CPU with zero external API calls.

### 4.2 Compare-And-Swap (CAS) State Engine
State is maintained in `.method/STATE.json` under atomic POSIX flock semantics:
- Revision Tracking: Every state transition increments `state_revision`.
- Atomic writes: Uses `.tmp` file creation followed by `fsync` and atomic `rename(2)`.
- Concurrent collision prevention: Worker leases expire automatically after duration unless renewed.

### 4.3 Cryptographic Write-Ahead Log (WAL)
Every mutation is written to `.method/WAL.log` before updating state:
```
<timestamp> | prev_hash=<64 hex> | <EVENT_TYPE> | <payload> | entry_hash=<64 hex>
```
- Each entry hashes the timestamp, previous hash, event type, and payload.
- In the event of an OOM or process kill, `read-engine recover` verifies the log and restores state in less than 10 milliseconds without spending any LLM tokens.
- Automatically compacts with a `CHECKPOINT` entry when exceeding 10,000 entries or 5 MB.

### 4.4 AROP Bounded Mutation Testing
Standard mutation testing takes dozens of minutes across a codebase. AROP (Asymmetric Bounded Mutation Testing) enforces strict execution bounds:
- Restricts mutations strictly to modified lines in `git diff --staged`.
- Bounds mutant population to a maximum of 10 targeted alterations ($M \le 10$).
- Full multi-language support: Python, Rust, Go, TypeScript/JavaScript, Java, C#/.NET, C++, and Ruby.
- Targeted AST mutation operators:
  - TypeScript / JavaScript: `INVERT_EQUALITY` (`===` <-> `!==`), `RETURN_UNDEFINED` (`return undefined;`), `BOUNDARY_SHIFT`, `REMOVE_OPTIONAL_CHAIN` (`?.` -> `.`).
  - Java: `INVERT_EQUALITY` (`==` <-> `!=`, `.equals()` <-> `!.equals()`), `RETURN_NULL` (`return null;`), `BOUNDARY_SHIFT`, `REMOVE_NULL_CHECK` (`if (x != null)` -> `if (true)`).
  - C# / .NET: `INVERT_EQUALITY` (`==` <-> `!=`, `.Equals()` <-> `!.Equals()`), `RETURN_DEFAULT` (`return default;`), `BOUNDARY_SHIFT`, `REMOVE_NULL_CONDITIONAL` (`?.` -> `.`).
  - C++: `INVERT_EQUALITY` (`==` <-> `!=`, `<` <-> `<=`, `>` <-> `>=`), `RETURN_DEFAULT` (`return x` <-> `return {}` / `return nullptr` / `return 0`), `BOUNDARY_SHIFT` (`n` <-> `n+1`/`n-1`, `++` <-> `--`), `REMOVE_NULLPTR_CHECK` (`if (ptr != nullptr)` -> `if (true)`).
  - Ruby: `REMOVE_SAFE_NAVIGATION` (`x&.y` <-> `x.y`), `INVERT_EQUALITY` (`==` <-> `!=`, `===` <-> `!==`, `.eql?` <-> `!.eql?`), `RETURN_NIL` (`return x` <-> `return nil`), `BOUNDARY_SHIFT` (`n` <-> `n+1`/`n-1`, `<` <-> `<=`, `>` <-> `>=`).
- Universal test harness auto-detection:
  - Node.js / TS: `jest`, `vitest`, `mocha`, `node:test` from `package.json`.
  - Java: JUnit 4, JUnit 5, TestNG from `pom.xml` / `build.gradle` (Surefire/Failsafe XML parsing).
  - C# / .NET: xUnit, NUnit, MSTest from `*.csproj` (TRX XML parsing).
  - C++: Google Test, Catch2, CTest from `CMakeLists.txt` / `Makefile` (JUnit XML / Catch2 XML parsing).
  - Ruby: RSpec, Minitest from `Gemfile` / `Gemfile.lock` (RSpec JSON / Minitest TAP parsing).
- Enforces a 4.5-second execution timeout ceiling ($T \le 4500\text{ms}$).
- Verification requires a strictly 100% kill score ($S = 0$ surviving mutants; any survivor immediately aborts commit).

### 4.5 Master Override Audit Trail
When legitimate changes produce surviving mutants or emergency fixes must bypass tests:
- An operator applies `read-engine override <OPERATOR> <RATIONALE>`.
- Rationale must be at least 30 characters.
- Generates cryptographic proof:
  $$\text{override\_hash} = \text{"OVR-"} \parallel \text{SHA-256}(\text{rationale} \parallel \text{operator} \parallel \text{timestamp})$$
- Seals state into `Verified` with full auditable trail in `WAL.log`.

---

## 5. Daemon Subsystems (read-daemon)

### 5.1 Dynamic Agent and Socket Discovery
The Go daemon scans host environments without hardcoded tool paths:
- Inspects `PATH`, `mise`, `npm`, `cargo`, and `pip` binaries (`dsh`, `claude`, `aider`, `interpreter`, `goose`, `cursor`, `gemini`).
- Probes local inference servers via 200ms socket timeouts: Ollama (`11434`), llama.cpp (`8080`), and vLLM (`8000`).
- Loads custom harness definitions from `.read/harnesses.json` or `~/.config/read/harnesses.json`.

### 5.2 Process Supervision and PTY Multiplexing
- Exposes `/ws/pty`, backed by `creack/pty`, for full interactive terminal streaming.
- Supports ANSI escape sequences, window resize events (`SIGWINCH`), and interrupt forwarding (`Ctrl+C`).
- Enforces process group isolation (`Setpgid: true`).
- Guarantees complete cleanup of child process trees (`KillProcessTree`): `SIGTERM` -> 50ms wait -> `SIGKILL` -> `/proc` orphan sweep.

### 5.3 3-Way Storage Engine
Synchronizes workspace files across three node roles (`RolePC`, `RolePhone`, `RoleVPS`):
- `pc_only`: Files kept exclusively on the developer workstation (e.g. build directories, local caches).
- `phone_only`: Ephemeral artifacts for mobile inspection excluded from repository history.
- `both_sync`: Fully synchronized files tracked by hierarchical Merkle tree digests.

### 5.4 Real-Time Credential Sanitization
All daemon outputs, WebSocket terminal streams, and CLI arguments pass through 10 regular expression filters masking sensitive keys with `***REDACTED***`:
- Anthropic keys (`sk-ant-...`)
- OpenAI keys (`sk-proj-...`, `sk-...`)
- DeepSeek keys (`sk-[a-f0-9]{32,}`)
- GitHub Personal Access Tokens (`ghp-...`, `gho-...`)
- HTTP Bearer tokens (`Authorization: Bearer ...`)

---

## 6. Command-Line Interface Reference

### Native Core CLI (`read-engine` or `read`)

```bash
# Initialize state directory and WAL
read init

# View current CAS state machine status
read status

# Freeze task contract into defined state
read define <TASK_ID> <CONTRACT_HASH> <BASE_COMMIT>

# Transition state to executing (Build stage)
read build

# Deterministically extract public AST signatures ($0.00 cost)
read ast-context <TARGET_FILE>

# Compute SHA-256 digest of git diff --staged (<0.1ms)
read diff-hash

# Generate targeted diff mutants (<4.5s)
read mutate --staged

# Record verification output and seal verify_hash
read verify <DIFF_HASH> "<VERIFY_LOG>"

# Apply Master Override with human operator rationale
read override "<OPERATOR>" "<RATIONALE (min 30 chars)>"

# Execute CAS-gated handoff to handoff_complete
read handoff

# Replay WAL and restore state after crash (0 tokens re-spent)
read recover
```

### Autonomous Pipeline & Host Discovery CLI (`read`)

```bash
# Discover and inspect all AI agent harnesses installed on the host (sub-150ms)
read scan

# Output discovered harnesses as structured JSON
read scan -json

# Execute autonomous 5-stage agent pipeline with verification gate
read run "Implement token verification logic"

# Execute autonomous pipeline targeting a specific agent harness
read run --harness=codex "Implement password hash verification"
```

### Daemon CLI (`read-daemon` or `read daemon`)

```bash
# Launch daemon on default port 8089
read daemon

# Launch daemon with custom port and workspace
read daemon --port=9000 --workspace=/path/to/repo --engine-bin=/path/to/read-engine
```

---

## 7. HTTP & WebSocket Gateway API Reference

When `read-daemon` is running, it exposes the following endpoints:

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/ws/pty` | WebSocket endpoint for interactive PTY terminal streaming. |
| `GET` | `/api/v1/status` | Returns system status, workspace root, harness count, and storage policy. |
| `GET` | `/api/v1/harnesses` | Returns list of all discovered and configured AI agent harnesses. |
| `POST`| `/api/v1/harnesses/toggle` | Enables or disables a specific harness. |
| `POST`| `/api/v1/harnesses/exec` | Spawns a harness within a dedicated isolated process group. |
| `POST`| `/api/v1/harnesses/stop` | Terminates a running harness and kills all child processes in the tree. |
| `GET` | `/api/v1/state` | Returns the current `.method/STATE.json` object. |
| `POST`| `/api/v1/state/override` | Submits a Master Override with rationale and operator ID (hashed tombstone in WAL). |
| `POST`| `/api/v1/task/command` | Proxies lifecycle commands (`define`, `build`, `verify`, `handoff`). |
| `GET` | `/api/v1/fs/list` | Lists files within the workspace root with SHA-256 hashes. |
| `GET` | `/api/v1/fs/read` | Safely reads a file within the workspace sandbox. |
| `POST`| `/api/v1/fs/write` | Safely writes a file within the workspace sandbox. |
| `GET` | `/api/v1/sync/manifest` | Generates a hierarchical Merkle manifest for delta synchronization. |
| `POST`| `/api/v1/sync/policy` | Sets the storage policy (`pc_only`, `phone_only`, `both_sync`) for paths. |

---

## 8. Documentation Suite

The complete documentation suite is available in the `docs/` directory:

- [ARCHITECTURE.md](docs/ARCHITECTURE.md): Deep systems architecture, dual-engine topology, zero-token AST context resolver, CAS state engine, append-only WAL, process supervision, PTY multiplexing, and 3-way storage sync.
- [LIFECYCLE.md](docs/LIFECYCLE.md): The 4-Stage Gated Lifecycle (Define -> Build -> Verify -> Handoff), contract hashing, AROP bounded mutation math, and cryptographic Master Override audit trail.
- [HARNESS_INTEGRATION.md](docs/HARNESS_INTEGRATION.md): Detailed integration guide for DeepSeek Harness (dsh), Claude Code, Aider, Ollama, vLLM, and custom harnesses.
- [CONFIGURATION.md](docs/CONFIGURATION.md): Specification of global config (`~/.config/read/config.json`), project config (`.read/config.json`), storage policies, and environment variables.
- [SECURITY_MODEL.md](docs/SECURITY_MODEL.md): Security architecture covering process group isolation, resource limits, symlink traversal prevention, and API key sanitization.

---

## 9. Verified Working Examples

Working demo projects with test suites and lifecycle automation scripts are located in `examples/`:

- [Python Ingress Rate Limiter Service](examples/python_service/): Real Python service with pytest, demonstrating `read init`, `read define`, `read build`, `read verify`, `read handoff`.
  Run demo: `bash examples/python_service/run_demo.sh`
- [Rust Bounded Ring Buffer Crate](examples/rust_crate/): Real Rust library crate with zero allocations on hot paths and cargo tests.
  Run demo: `bash examples/rust_crate/run_demo.sh`
- [Go Concurrent In-Memory Cache Module](examples/go_module/): Real Go package with TTL expiration, goroutine safety, and standard go tests.
  Run demo: `bash examples/go_module/run_demo.sh`
- [TypeScript Authorization Service](examples/typescript_service/): Production TypeScript microservice with Express and Jest, verifying RBAC permissions, optional chaining, and token validation under strict 100% mutation testing.
  Run demo: `bash examples/typescript_service/run_demo.sh`
- [Java Order Validation Service](examples/java_service/): Spring Boot / JUnit 5 microservice verifying role validation, null checks, and boundary logic under strict 100% mutation testing.
  Run demo: `bash examples/java_service/run_demo.sh`
- [C# Token Validation Service](examples/csharp_service/): ASP.NET Core / xUnit microservice verifying null conditionals (`?.`), `.Equals()` checks, and default returns under strict 100% mutation testing.
  Run demo: `bash examples/csharp_service/run_demo.sh`
- [C++ Buffer Pool Service](examples/cpp_service/): High-performance C++17 memory pool with CMake and Google Test, verifying pointer null checks, capacity boundaries, and return values under strict 100% mutation testing.
  Run demo: `bash examples/cpp_service/run_demo.sh`
- [Ruby Authentication Service](examples/ruby_service/): Zero-dependency Ruby service with RSpec format verification, asserting safe navigation (`&.`), case equality (`===`), strict string equality (`eql?`), and TTL boundaries under strict 100% mutation testing.
  Run demo: `bash examples/ruby_service/run_demo.sh`

---

## 10. Automated Verification and Chaos Suites

The repository contains end-to-end automated verification suites:

```bash
# 1. Native Rust Engine Verification (9 checks)
bash read/tests/test_e2e.sh

# 2. Host Daemon and Integration Verification (9 checks)
bash read/tests/test_full_system.sh

# 3. Chaos and Resiliency Suite (CAS collision, tamper resistance, recovery)
bash read/tests/test_chaos.sh
```

---

## 11. Learning READ: Adversarial Induction Suite

READ rejects passive tutorials, roleplay guidelines, and superficial walkthroughs. Adopters internalize framework mechanics through **adversarial failure simulation** under the `read learn` command suite:

```bash
# Initialize isolated induction workspace under .read-learn/
read learn init

# Load specific induction scenario (1, 2, or 3)
read learn scenario <1|2|3>

# Run automated adversarial diagnosis on current scenario
read learn diagnose

# Inspect onboarding progress and mastery score in STATE.json
read learn progress
```

### The Three Adversarial Failure Scenarios

1. **Scenario 01: "Brittle Test Trap" (AROP 100% Mutation Gate Enforcement)**
   - **The Trap**: Code is paired with an apparently "passing" unit test suite that yields only an 80% mutation kill score (2 mutants survive boundary checks).
   - **The Failure**: Commit is blocked mechanically. Developer must strengthen assertions until 100% of mutants are killed.
   - **Master's Rationale**: In Veloce Stage 4 / VIREN, conversational QA and superficial test suites passed 100% of tests, but regex parsers silently accepted corrupted outputs and malformed state. Test suites that merely run without asserting boundary behavior are an illusion. AROP Pass 2 gate demands strictly 100% mutation kill score—if a mutant survives, your tests are blind.

2. **Scenario 02: "Context Hallucination Burn" (C0-AST Resolution vs Context Bloat)**
   - **The Trap**: Sprawling 12,000-token legacy service with bloated metadata.
   - **The Contrast**: In conversational systems, dumping full files triggers quadratic prefill costs and token bleed. In READ, the C0-AST context resolver deterministically extracts exact dependency closures in <5ms for $0.00 (98.5% token reduction).
   - **Master's Rationale**: In Hermes / VIREN, transcripts and full-file dumps consumed 277k tokens of context prefill, triggering quadratic billing, context rot, and model hallucinations. The C0-AST context resolver deterministically extracts exact dependency closures in <5ms for $0.00. Never spend tokens on code structure that deterministic AST analysis resolves for free.

3. **Scenario 03: "CAS Race Condition" (Atomic Locks & WAL Crash Recovery)**
   - **The Trap**: Simulated mid-build process crash leaves in-memory state desynchronized and CAS lease expired.
   - **The Recovery**: The developer triggers `read recover` to replay verified Write-Ahead Log entries, restoring 100% state revision without re-executing agents or spending any tokens.
   - **Master's Rationale**: In Veloce Termux / Fraol, unmetered in-memory agents crashed mid-stream, losing execution state and forcing complete task re-runs. BMAD assumes infinite uptime and unmetered cloud compute. READ assumes computational finitude and process death: Compare-And-Swap (CAS) state locks mechanically prevent concurrent corruption, while the cryptographic Write-Ahead Log (WAL) recovers 100% of state with zero tokens re-spent.
