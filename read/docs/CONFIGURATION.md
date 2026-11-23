# READ Configuration Specification

## 1. Configuration Hierarchy

READ uses a layered configuration system where higher-priority layers override lower ones:

```
+-------------------------------------------------------------------------------+
| Layer 1: CLI Flags (Highest Priority)                                         |
| e.g. --port=9000, --workspace=/path/to/repo, --custom-harness=/path/cfg.json   |
+---------------------------------------+---------------------------------------+
                                        | overrides
                                        v
+-------------------------------------------------------------------------------+
| Layer 2: Environment Variables                                                |
| e.g. READ_PORT, READ_WORKSPACE, READ_ENGINE_BIN, READ_STORAGE_POLICY          |
+---------------------------------------+---------------------------------------+
                                        | overrides
                                        v
+-------------------------------------------------------------------------------+
| Layer 3: Project Workspace Config (.read/config.json)                         |
| Workspace-specific storage policies, harness definitions, and mutation limits |
+---------------------------------------+---------------------------------------+
                                        | overrides
                                        v
+-------------------------------------------------------------------------------+
| Layer 4: User Global Config (~/.config/read/config.json)                      |
| User-wide agent definitions, default ports, and developer environment paths   |
+---------------------------------------+---------------------------------------+
                                        | overrides
                                        v
+-------------------------------------------------------------------------------+
| Layer 5: Built-in Defaults (Lowest Priority)                                  |
| Hardcoded safe defaults (Port 8089, Policy: both_sync, Max Mutants: 10)       |
+-------------------------------------------------------------------------------+
```

---

## 2. Global Configuration (`~/.config/read/config.json`)

Stored at `$HOME/.config/read/config.json` (or `%USERPROFILE%\.config\read\config.json` on Windows).

### Example Global Configuration:
```json
{
  "daemon": {
    "default_port": 8089,
    "bind_address": "0.0.0.0",
    "engine_bin": "/usr/local/bin/read-engine",
    "log_level": "info",
    "pty_shell": "/bin/bash"
  },
  "discovery": {
    "scan_paths": [
      "~/.local/share/mise/shims",
      "~/.cargo/bin",
      "~/.local/bin",
      "/usr/local/bin"
    ],
    "probe_sockets": [
      {"name": "ollama", "host": "127.0.0.1", "port": 11434},
      {"name": "vllm", "host": "127.0.0.1", "port": 8000},
      {"name": "llama.cpp", "host": "127.0.0.1", "port": 8080}
    ]
  },
  "storage": {
    "default_policy": "both_sync",
    "delta_chunk_size_bytes": 1048576
  }
}
```

---

## 3. Project Configuration (`.read/config.json`)

Stored at the root of the target repository: `.read/config.json`.

### Example Project Configuration:
```json
{
  "project_name": "payment-gateway-service",
  "version": "1.0.0",
  "engine": {
    "max_mutants": 10,
    "mutation_timeout_ms": 4500,
    "max_diff_bytes": 52428800,
    "ast_max_depth": 3
  },
  "storage_policies": {
    "default": "both_sync",
    "rules": [
      {"pattern": ".git/**", "policy": "pc_only"},
      {"pattern": "target/**", "policy": "pc_only"},
      {"pattern": "__pycache__/**", "policy": "pc_only"},
      {"pattern": ".venv/**", "policy": "pc_only"},
      {"pattern": "node_modules/**", "policy": "pc_only"},
      {"pattern": "docs/mobile/**", "policy": "phone_only"},
      {"pattern": "src/**", "policy": "both_sync"},
      {"pattern": "tests/**", "policy": "both_sync"}
    ]
  },
  "verification": {
    "test_command": "pytest tests/ -v",
    "coverage_threshold_percentage": 90.0,
    "fail_on_surviving_mutant": true
  }
}
```

### Field Definitions:

| Section | Key | Type | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `engine` | `max_mutants` | integer | `10` | Hard cap on mutants generated per verification cycle. |
| `engine` | `mutation_timeout_ms` | integer | `4500` | Max milliseconds allowed for mutation evaluation before timeout. |
| `engine` | `max_diff_bytes` | integer | `52428800` | 50 MB ceiling on unified diff parsing to prevent memory spikes. |
| `engine` | `ast_max_depth` | integer | `3` | Max recursion depth when following import statements. |
| `storage_policies`| `default` | string | `"both_sync"` | Fallback policy for files not matched by explicit pattern rules. |
| `storage_policies`| `rules` | array | `[]` | Glob patterns mapping paths to `pc_only`, `phone_only`, or `both_sync`. |

---

## 4. Harness Configuration (`.read/harnesses.json`)

Located at `.read/harnesses.json` or `~/.config/read/harnesses.json`.

```json
{
  "harnesses": [
    {
      "id": "dsh",
      "name": "DeepSeek Harness",
      "type": "cli",
      "command": "dsh",
      "args": ["--mode", "autonomous"],
      "env": {
        "DSH_TEMPERATURE": "0.1"
      },
      "status": "available",
      "enabled": true
    },
    {
      "id": "ollama-coder",
      "name": "Local Ollama Qwen2.5-Coder",
      "type": "socket",
      "port": 11434,
      "status": "available",
      "enabled": true
    }
  ]
}
```

---

## 5. Storage Policies Specification

READ partitions files across three operational roles (`pc`, `phone`, `vps`):

### 5.1 Storage Policy Types

| Policy | Behavior | Typical File Types |
| :--- | :--- | :--- |
| `pc_only` | File is stored exclusively on the host PC. When synchronizing with mobile or VPS, this file is excluded from delta manifests and never transmitted. | `.git/`, build artifacts (`target/`, `node_modules/`, `dist/`), virtualenvs (`.venv/`), compiler caches (`.cache/`). |
| `phone_only` | File is designated for mobile inspection only. It is excluded from the long-term repository history and stripped before production handoffs. | Ephemeral test screenshots, mobile debug captures, temporary audio/input notes. |
| `both_sync` | File is maintained in full cryptographic synchrony across all nodes. Any modification triggers a streaming SHA-256 Merkle update. | Source code (`.py`, `.rs`, `.go`, `.ts`), unit tests, documentation, `STATE.json`, `WAL.log`. |

### 5.2 Conflict Resolution Mechanics
When two nodes modify the same `both_sync` file concurrently:
1. The daemon compares the Merkle digest and modification timestamps (`modified_unix`).
2. If contents diverge and both nodes have changed since the last common ancestor revision, the node with the higher CAS revision wins.
3. The losing revision is archived locally as `<filename>.conflict.<timestamp>` to prevent data loss.

---

## 6. Environment Variables Reference

| Environment Variable | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `READ_WORKSPACE` | string | `.` | Absolute or relative path to the target project workspace. |
| `READ_PORT` | integer | `8089` | TCP port for daemon HTTP and WebSocket gateway. |
| `READ_ENGINE_BIN` | string | `./read/bin/read-engine` | Absolute path to the compiled native `read-core` binary. |
| `READ_STORAGE_POLICY`| string | `both_sync` | Default storage synchronization policy (`pc_only`, `phone_only`, `both_sync`). |
| `READ_CUSTOM_HARNESS`| string | `""` | Explicit path to a custom `harnesses.json` configuration file. |
| `READ_LOG_LEVEL` | string | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`). |
| `READ_TIMEOUT_MS` | integer | `4500` | Hard timeout ceiling in milliseconds for mutation test execution. |
| `READ_MAX_MUTANTS` | integer | `10` | Maximum mutant count generated by AROP. |
| `READ_REDACT_KEYS` | boolean | `true` | When true, scrubs API keys and secret tokens from all outputs. |
