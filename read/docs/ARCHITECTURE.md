# READ System Architecture Specification

## 1. Executive Summary

READ (Resource-conscious, Evidence-driven AI Development) is an executable software engineering framework and process supervision system. It eliminates non-deterministic conversational loops in AI-assisted software construction, replacing probabilistic human-in-the-loop chat with mechanical, evidence-driven verification gates.

The system topology is organized around a dual-engine architecture:
1. `read-core`: A high-performance Rust engine providing deterministic AST parsing, atomic Compare-And-Swap (CAS) state management, cryptographic write-ahead logging (WAL), staged diff hashing, and bounded mutation testing (AROP).
2. `read-daemon`: A Go daemon providing dynamic agent/socket discovery, pseudo-terminal (PTY) multiplexing over WebSockets, process group sandboxing, and 3-way storage synchronization across workstation, mobile, and VPS nodes.

```
+---------------------------------------------------------------------------------------+
|                                    READ ARCHITECTURE                                  |
|                                                                                       |
|   +------------------------------------+    +-------------------------------------+   |
|   | External AI Agents / Harnesses     |    | Local Inference Servers             |   |
|   | - DeepSeek Harness (dsh)           |    | - Ollama (127.0.0.1:11434)          |   |
|   | - Claude Code CLI (claude)         |    | - llama.cpp (127.0.0.1:8080)        |   |
|   | - Aider (aider)                    |    | - vLLM (127.0.0.1:8000)             |   |
|   | - Open Interpreter / Goose / Cursor|    | - Custom HTTP Sockets               |   |
|   +-----------------+------------------+    +------------------+------------------+   |
|                     |                                          |                      |
|                     +--------------------+---------------------+                      |
|                                          |                                            |
|                                          v                                            |
|   +-------------------------------------------------------------------------------+   |
|   | read-daemon (Go 1.22+) Supervisor Layer                                       |   |
|   |  - Process Group Isolation (Setpgid, recursive tree kill, /proc orphan sweep) |   |
|   |  - Token Redaction Engine (10 regex filters, Bearer/sk-* redaction)           |   |
|   |  - Symlink-Safe Sandbox Resolver (realpath validation, URL decoding defense)  |   |
|   |  - WebSocket Interactive PTY Terminal (creack/pty, resize, signal forwarding) |   |
|   |  - 3-Way Storage Engine (pc_only, phone_only, both_sync, Merkle delta plans)  |   |
|   |  - HTTP Gateway API (/api/v1/*)                                               |   |
|   +--------------------------------------+----------------------------------------+   |
|                                          |                                            |
|                        POSIX CLI & Direct Process Invocation                          |
|                                          |                                            |
|                                          v                                            |
|   +-------------------------------------------------------------------------------+   |
|   | read-core (Rust 1.75+) Computational Layer                                    |   |
|   |  - C0 Deterministic AST Context Resolver ($0.00 cost, <5ms, no model calls)   |   |
|   |  - Compare-And-Swap (CAS) State Engine (.method/STATE.json, POSIX flock)      |   |
|   |  - Append-Only Write-Ahead Log (.method/WAL.log, SHA-256 hash-chained entries)|   |
|   |  - Staged Diff Hasher (SHA-256 over git diff --staged, <0.1ms)                |   |
|   |  - Asymmetric Bounded Mutation Verifier (AROP, <=10 mutants, <4.5s ceiling)   |   |
|   |  - Master Override Audit Ledger ([OVERRIDE] with SHA-256 proof)               |   |
|   |  - Zero-Token Crash Recovery System (<10ms recovery time)                     |   |
|   +-------------------------------------------------------------------------------+   |
|                                          |                                            |
|                                          v                                            |
|                          Filesystem & Git Repository Root                             |
|              .git/ | .method/ (STATE.json, WAL.log) | Source Worktree                 |
+---------------------------------------------------------------------------------------+
```

---

## 2. Dual-Engine Topology

### 2.1 Separation of Responsibilities

Traditional agent architectures combine process execution, state tracking, and LLM communication in a single monolithic runtime. This leads to severe architectural flaws:
- A crash or out-of-memory (OOM) kill in the agent terminates state tracking, causing process amnesia.
- State mutation occurs concurrently without transactional locking, leading to race conditions.
- Context is assembled dynamically by querying the LLM repeatedly, driving up token costs.

READ separates execution into two decoupled domains:

| Attribute | `read-core` (Rust) | `read-daemon` (Go) |
| :--- | :--- | :--- |
| **Primary Domain** | Deterministic Computation & Invariant Enforcement | Process Supervision, Network I/O, Storage Sync |
| **Execution Mode** | Synchronous CLI binary, sub-millisecond execution | Long-running asynchronous daemon service |
| **State Storage** | On-disk atomic JSON, POSIX file lock, chained WAL | In-memory registry, ephemeral session handles |
| **Security Scope** | State transitions, Diff hashing, AST parsing | Process tree sandboxing, token scrubbing, path checks |
| **Memory Footprint** | Peak < 15 MB during full AST graph traversal | Stable ~20-30 MB resident set size (RSS) |
| **Failure Domain** | Pure functional operations; fail-closed on error | Supervision tree isolates client process crashes |

### 2.2 Inter-Engine Communication
Communication between `read-daemon` and `read-core` follows strict boundaries:
1. **Direct Invocation**: The Go daemon executes `read-core` (`read-engine`) via standard exec pipes with process-group boundaries.
2. **Atomic Disk State**: State handoffs occur exclusively through the locked `.method/STATE.json` and `.method/WAL.log` files.
3. **Exit Code Semantics**: `read-core` communicates failures via standardized POSIX exit codes and structured JSON errors on stderr.

---

## 3. Zero-Token Deterministic AST Context Resolver

### 3.1 Problem Definition: Token Bloat in Conversational Agents
Conversational tools inject entire source files into LLM prompts. For a service consisting of 15 files totaling 4,000 lines, a single agent prompt consumes 20,000 to 30,000 tokens ($0.06 to $0.30 per interaction). Over dozens of iterations, developers spend tens of dollars re-sending unchanged function bodies and documentation comments.

### 3.2 C0 Resolver Engine Design
`read-core` implements the C0 Context Resolver (`read_core::ast::AstContextResolver`). It analyzes code locally on the host CPU without querying external models:
- **Zero Token Cost**: Token consumption is exactly $0.00.
- **Latency**: Graph traversal and signature extraction complete in under 5 milliseconds.
- **Transitive Dependency Tracing**: Starting from a target file, it extracts local import declarations up to a configurable depth (default: 3 levels).
- **Public Signature Extraction**: It discards function implementations, private local variables, and internal loops, extracting only public interfaces (classes, structs, functions, interfaces, methods, constants).

```
Target File: src/service/order_handler.py
        |
        +---> Resolves Imports:
        |       - from ..models.order import Order, OrderStatus
        |       - from ..db.connection import get_db_session
        |
        +---> Traverses Dependency Graph (Depth <= 3)
        |
        +---> Strips Method Bodies & Internal Logic
        |
        v
Output C0 Manifest:
  - Symbols: Order, OrderStatus, get_db_session, handle_incoming_order
  - Signatures: def handle_incoming_order(order_id: str, db: Session) -> OrderResult
  - Estimated Context Tokens: ~180 tokens (vs 4,200 tokens for full source tree)
```

### 3.3 Multi-Language Support
The resolver contains tailored grammar extractors for four primary ecosystems:
- **Python (`.py`, `.pyi`)**: Identifies `def`, `async def`, `class`, type aliases, decorators, and local `import` / `from . import` statements.
- **Rust (`.rs`)**: Identifies `pub fn`, `pub struct`, `pub enum`, `pub trait`, `pub type`, and module declarations (`mod`, `use crate::*`).
- **Go (`.go`)**: Identifies exported identifiers (capitalized `func`, `type Struct`, `type Interface`, `const`), and internal package imports.
- **TypeScript & JavaScript (`.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`)**: Identifies `export function`, `export class`, `export interface`, `export type`, `export const`, and relative ES module imports.

### 3.4 Fault-Tolerant Degradation
Code undergoing active editing frequently contains incomplete syntax. Standard parsers crash or throw unrecoverable parse errors on broken ASTs. The READ resolver utilizes a layered parsing strategy:
1. Structured lexical scan for high-confidence symbol tokens.
2. Graceful fallback to regex-guided boundary extraction if brackets or semicolons are unbalanced.
3. Sets `parse_degraded = true` in the output manifest without aborting the lifecycle.

---

## 4. Compare-And-Swap (CAS) State Engine

### 4.1 State File Schema (`.method/STATE.json`)
All task execution is governed by a persistent state machine stored in `.method/STATE.json`:

```json
{
  "state_revision": 4,
  "status": "executing",
  "active_task_id": "TASK-PAYMENT-001",
  "contract_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "base_commit": "7a3f81e642bc9a9b8f2d5e1f0e4b2a8d1c6e3b5a",
  "staged_diff_hash": "a1b2c3d4e5f60718293a4b5c6d7e8f90123456789abcdef0123456789abcdef0",
  "verify_hash": null,
  "cas_lock": {
    "expected_revision": 4,
    "locked_by": "worker-session-78b1",
    "lease_expires": "2026-11-22T14:30:00Z"
  }
}
```

### 4.2 State Machine Transition Rules
State transitions are strictly linear and monotonic. Any attempt to skip stages or bypass checks results in a `CasLockViolation` error.

```
       +--------+
       |  Idle  |
       +---+----+
           |  read define <TASK_ID> <CONTRACT_HASH> <BASE_COMMIT>
           v
       +--------+
       |Defined |
       +---+----+
           |  read build
           v
      +-----------+
      | Executing |
      +-----+-----+
            |  read verify <DIFF_HASH> <VERIFY_LOG> (or read override)
            v
       +----------+
       | Verified |
       +----+-----+
            |  read handoff (Verifies verify_hash and diff stability)
            v
   +------------------+
   | HandoffComplete  |
   +------------------+
```

### 4.3 Atomic File Swaps and POSIX Locking
To guarantee zero corruption under concurrency:
1. **Advisory Locking**: The engine acquires an exclusive lock on `.method/.state.lock` using `libc::flock(fd, libc::LOCK_EX)`. The lock is automatically released upon Drop.
2. **Lease Checking**: If an active lease is held by another worker and has not expired, modifications are rejected.
3. **Atomic Replace**: Changes are written to `.method/STATE.json.tmp`, flushed to physical storage with `fsync`, and atomically moved into place using `rename(2)`.

---

## 5. Append-Only Write-Ahead Log (WAL)

### 5.1 Format and Cryptographic Chaining
Every mutation is written to `.method/WAL.log` before committing state to disk. Each entry forms a cryptographic hash chain:

```
<timestamp> | prev_hash=<64-char hex> | <EVENT_TYPE> | <payload> | entry_hash=<64-char hex>
```

Hash calculation formula:
$$\text{entry\_hash} = \text{SHA-256}(\text{timestamp} \parallel \text{"|"} \parallel \text{prev\_hash} \parallel \text{"|"} \parallel \text{event\_type} \parallel \text{"|"} \parallel \text{payload})$$

The genesis entry uses 64 zeros:
`0000000000000000000000000000000000000000000000000000000000000000`.

### 5.2 Automatic Log Compaction
To ensure minimal resource usage on constrained systems:
- The engine checks WAL size prior to appends.
- If the log exceeds 10,000 entries or 5 MB in size, compaction triggers automatically.
- A `CHECKPOINT` entry is generated containing the serialized snapshot of current state.
- Older entries are pruned, and the checkpoint becomes the new root of the hash chain.

### 5.3 Crash Recovery Protocol (`read recover`)
When an agent dies due to SIGKILL, OOM, or system reboot:
1. The operator or daemon executes `read recover`.
2. The engine verifies the unbroken cryptographic chain in `WAL.log` from genesis to the final entry.
3. It loads `.method/STATE.json` and verifies that `state_revision` matches the sequence of logged events.
4. It computes `git diff --staged` and ensures no silent drift has occurred.
5. The working context is restored in under 10 milliseconds without spending a single LLM token.

---

## 6. Process Supervision and PTY Multiplexing

### 6.1 Interactive Terminal Gateway (`/ws/pty`)
`read-daemon` provides direct pseudo-terminal control over WebSockets backed by `creack/pty`:
- Spawns target processes in dedicated PTY master/slave pairs.
- Bidirectionally streams stdin, stdout, and stderr with full terminal emulation support (ANSI colors, cursor movements, vim/nano sessions).
- Listens for terminal resize payloads from remote mobile or desktop web clients and issues `ioctl(TIOCSWINSZ)` to resize the terminal window on the fly.
- Handles forwarding of interrupt signals (`Ctrl+C` -> SIGINT, `Ctrl+\` -> SIGQUIT).

### 6.2 Process Tree Sandboxing and Orphan Reclamation
When an external harness launches child processes (e.g., node servers, test runners, python scripts), traditional process management often leaves zombie processes running if the parent crashes.

`read-daemon` enforces complete process lifecycle control:
1. **Process Group Assignment**: Commands are launched with `SysProcAttr: &syscall.SysProcAttr{Setpgid: true}`.
2. **Descendant Enumeration**: Before termination, the daemon scans `/proc` to build the full tree of child PIDs and process group IDs (PGIDs).
3. **Two-Stage Signal Escalation**:
   - Sends `SIGTERM` to the entire negative process group (`-pgid`) and all enumerated descendants.
   - Waits a 50-millisecond grace period for graceful cleanup.
   - Sends `SIGKILL` (`-pgid` and individual child PIDs) to eliminate lingering processes.
4. **Post-Termination Sweep**: Scans `/proc` again to verify no orphaned processes remain in the PGID.

---

## 7. 3-Way Storage Engine

### 7.1 Multi-Node Architectural Roles
READ supports distributed workflows across three operational node types:
- **Workstation (`RolePC`)**: The primary developer workstation with full source checkouts and desktop tools.
- **Mobile (`RolePhone`)**: A lightweight mobile controller running terminal interfaces or remote monitoring.
- **Remote Runner (`RoleVPS`)**: A cloud-hosted execution environment running high-throughput builds, continuous integration, or heavy mutation testing.

### 7.2 Storage Policies
Files and directories within the workspace are governed by explicit storage policies:
- `pc_only`: Retained strictly on the workstation (e.g., local IDE configurations, binary caches).
- `phone_only`: Ephemeral artifacts generated for mobile inspection and excluded from long-term repository history.
- `both_sync`: Fully synchronized across nodes using cryptographic digests.

### 7.3 Merkle Tree Delta Synchronization
To eliminate redundant file transfers over mobile data connections or high-latency SSH tunnels:
1. The daemon builds a hierarchical `MerkleNode` directory tree for the workspace.
2. Each node computes a deterministic SHA-256 hash over its contents and child node hashes.
3. When two nodes synchronize, they exchange root Merkle hashes.
4. If root hashes diverge, nodes traverse down the branch to isolate only the changed files.
5. A `DeltaPlan` is produced detailing actions: `upload`, `download`, `delete`, `skip_policy`, or `conflict_resolved`.

---

## 8. Performance and Resource Bounds

The framework is engineered to adhere to strict operational ceilings:

| Metric | Target Specification | Enforcement Mechanism |
| :--- | :--- | :--- |
| **AST Context Extraction** | < 5.0 milliseconds | Zero model calls, single-pass regex/lexer scan |
| **Staged Diff Hashing** | < 0.1 milliseconds | Direct pipe from `git diff --staged` into streaming SHA-256 |
| **Mutation Generation** | < 4.5 seconds hard timeout | Timer check in AST mutation loop (`timeout_duration`) |
| **Mutant Population** | <= 10 targeted mutants | Hard bound counter (`DEFAULT_MAX_MUTANTS = 10`) |
| **Memory Ceiling** | < 1.2 GB peak RAM | Diff size ceiling (`MAX_DIFF_BYTES = 50 MB`), bounded buffers |
| **Crash Recovery** | < 10.0 milliseconds | Direct sequential WAL replay without LLM queries |
