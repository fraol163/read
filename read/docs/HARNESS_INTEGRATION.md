# READ Harness and AI Agent Integration Guide

## 1. Architectural Role of Harnesses in READ

READ does not replace external AI agent CLIs or inference engines. Instead, it provides a supervisory control plane that:
- Automatically detects installed agents and local inference servers.
- Sandboxes external processes in dedicated POSIX process groups.
- Streams terminal I/O over secure WebSockets with ANSI escape code fidelity.
- Scrubs sensitive API keys and tokens from all standard output in real time.
- Restricts agent file modifications to the current task boundary using CAS state gates.

```
+-----------------------------------------------------------------------------------------+
|                                SUPERVISION TOPOLOGY                                     |
|                                                                                         |
|   +------------------------------------+    +-------------------------------------+     |
|   | External AI Agent CLIs             |    | Local Inference Engines             |     |
|   | - DeepSeek Harness (dsh)           |    | - Ollama (127.0.0.1:11434)          |     |
|   | - Claude Code CLI (claude)         |    | - llama.cpp (127.0.0.1:8080)        |     |
|   | - Aider (aider)                    |    | - vLLM (127.0.0.1:8000)             |     |
|   +-----------------+------------------+    +------------------+------------------+     |
|                     |                                          |                        |
|                     +--------------------+---------------------+                        |
|                                          |                                              |
|                                          v                                              |
|   +---------------------------------------------------------------------------------+   |
|   | read-daemon Scanner & Process Supervisor                                        |   |
|   |  1. Discovers binaries in PATH, mise, npm, cargo, pip                           |   |
|   |  2. Probes local TCP sockets (11434, 8080, 8000) with 200ms health-checks      |   |
|   |  3. Assigns independent process groups (Setpgid)                                |   |
|   |  4. Scrubs stdout/stderr via 10 token regex engines (SanitizeText)              |   |
|   |  5. Exposes interactive WebSocket PTY (/ws/pty) & HTTP RPC (/api/v1/*)          |   |
|   +--------------------------------------+------------------------------------------+   |
|                                          |                                              |
|                                          v                                              |
|   +---------------------------------------------------------------------------------+   |
|   | Gated Execution Worktree (.read/STATE.json + git diff --staged)               |   |
|   +---------------------------------------------------------------------------------+   |
+-----------------------------------------------------------------------------------------+
```

---

## 2. Dynamic Discovery Subsystem

### 2.1 Scanner Mechanics & CLI Inspection
Inspect all discovered harnesses directly with the CLI:
```bash
read scan           # Human-readable formatted discovery summary
read scan -json     # Machine-readable JSON output for tooling
```

On daemon startup or via `GET /api/v1/harnesses`, the discovery scanner (`read-daemon/scanner/scanner.go` and `read-daemon/scanner/mise.go`) inspects the host environment without requiring manual configuration:

1. **Path & Package Manager Search**:
   Resolves binaries across system `PATH` and tool manager directories (`~/.local/share/mise`, `~/.nvm`, `~/.cargo/bin`, `~/.local/bin`, `~/.bun/bin`, `~/.asdf`, `/usr/local/bin`).
2. **Dynamic Mise Registry Discovery**:
   Directly parses `mise ls --json` and system tool tables (`/etc/mise/conf.d/*.toml`), extracting exact installed/configured versions and paths with sub-100ms latency.
3. **Comprehensive Agent Catalog (49+ Supported Harnesses)**:
   - **Google DeepMind & Anthropic**: `antigravity` (`agy`), `claude` (`claude-code`), `gemini`
   - **OpenAI & xAI**: `codex`, `grok`
   - **Developer & Code Agents**: `dsh` (DeepSeek), `copilot`, `cursor-agent`, `devin`, `pi`, `qwen`, `crush`, `opencode`, `kiro-cli`, `kilo`, `kimi`, `mcode`, `amp`, `basecamp`, `cf`, `commandcode`, `freebuff`, `hunk`, `muse`, `ori`, `hey`, `omp` (Oh-My-Pi), `hermes`, `ghui`
   - **Open-Source Tools**: `aider`, `goose`, `open-interpreter`, `sgpt`, `plandex`, `mentat`, `cody`, `continue`, `cline` / `roo`, `autogpt`, `gpt-engineer`, `llm`, `fabric`, `mods`, `tgpt`
   - **Local Inference Engines**: `ollama`, `vllm`, `tabby`, `localai`
4. **Local Inference Socket Probing**:
   Scans local loopback addresses with short TCP connect timeouts (200ms):
   - Ollama: `http://127.0.0.1:11434/api/tags`
   - llama.cpp server: `http://127.0.0.1:8080/health`
   - vLLM / LocalAI: `http://127.0.0.1:8000/v1/models`
5. **Containerized Harnesses**:
   If Docker is present on the host, the scanner queries running containers for matching image labels or ports.

### 2.2 Custom Harness Configuration
You can define custom or proprietary agents by creating `.read/harnesses.json` in your workspace or globally in `~/.config/read/harnesses.json`:

```json
{
  "harnesses": [
    {
      "id": "deepseek-local",
      "name": "DeepSeek R1 Local vLLM Runner",
      "type": "custom",
      "command": "/usr/local/bin/dsh",
      "args": [
        "--model", "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B",
        "--endpoint", "http://127.0.0.1:8000/v1"
      ],
      "env": {
        "TEMPERATURE": "0.2"
      },
      "status": "available",
      "enabled": true
    },
    {
      "id": "custom-agent",
      "name": "Internal Automated Fixer",
      "type": "cli",
      "command": "python3",
      "args": ["-m", "company_agent.cli", "--autonomous"],
      "status": "available",
      "enabled": true
    }
  ]
}
```

---

## 3. Harness Integration Walkthroughs

### 3.1 DeepSeek Harness (`dsh`)

The DeepSeek Harness is an autonomous coding agent optimized for DeepSeek-Coder and DeepSeek-R1 models.

#### Integration Pattern:
1. Ensure `dsh` is in your `PATH` or configured in `.read/harnesses.json`.
2. Launch the task lifecycle:
   ```bash
   read-engine define TASK-01 <CONTRACT_HASH> <BASE_COMMIT>
   read-engine build
   ```
3. Extract C0 AST context to provide to `dsh`:
   ```bash
   read-engine ast-context src/service.py > /tmp/c0_context.txt
   ```
4. Execute `dsh` under READ supervision:
   ```bash
   # Via Daemon HTTP API:
   curl -X POST http://127.0.0.1:8089/api/v1/harnesses/exec \
     -H "Content-Type: application/json" \
     -d '{
       "id": "dsh",
       "args": ["--task-file", "/tmp/c0_context.txt", "--auto-commit=false"]
     }'
   ```
5. `dsh` operates within the workspace. When it completes its modifications, stage the diff:
   ```bash
   git add src/service.py tests/test_service.py
   ```
6. Verify and handoff through `read-engine`.

---

### 3.2 Claude Code CLI (`claude`)

Anthropic's Claude Code CLI runs as an interactive terminal application.

#### Interactive Supervision via PTY:
Because `claude` uses terminal control sequences, interactive prompts, and ANSI formatting, standard pipe redirection breaks its UI. `read-daemon` solves this via `/ws/pty`:

1. Connect to the WebSocket terminal endpoint:
   ```
   ws://127.0.0.1:8089/ws/pty
   ```
2. The daemon spawns `claude` in an allocated pseudo-terminal.
3. Terminal resize messages (`SIGWINCH`) are sent via JSON over the WebSocket:
   ```json
   {
     "type": "resize",
     "rows": 40,
     "cols": 120
   }
   ```
4. All sensitive Anthropic tokens (`sk-ant-...`) emitted in the terminal stream are intercepted and replaced with `***REDACTED***` before leaving the host daemon.

---

### 3.3 Aider with Local Ollama / vLLM Models

Aider connects directly to Git repositories and local OpenAI-compatible endpoints.

#### Integration Recipe:
1. Ensure Ollama is running locally:
   ```bash
   ollama run qwen2.5-coder:14b
   ```
   `read-daemon` automatically detects the port `127.0.0.1:11434`.
2. Configure Aider to run non-interactively with bounded instructions:
   ```bash
   curl -X POST http://127.0.0.1:8089/api/v1/harnesses/exec \
     -H "Content-Type: application/json" \
     -d '{
       "id": "aider",
       "args": [
         "--model", "ollama/qwen2.5-coder:14b",
         "--openai-api-base", "http://127.0.0.1:11434/v1",
         "--yes",
         "--message", "Implement the order validation function defined in contract TASK-01"
       ]
     }'
   ```
3. Aider modifies the requested files.
4. The READ daemon ensures any spawned child processes are tracked and reaped when execution completes.

---

## 4. Supervision and Sandbox Guarantees

When any harness is executed through `read-daemon`:

### 4.1 Process Isolation
- Every process runs with `Setpgid: true`.
- If an agent attempts to background a process (e.g., launching a background server with `&` or `nohup`), the child process remains trapped in the allocated process group.
- Calling `/api/v1/harnesses/stop` sends `SIGTERM` to the entire negative PGID (`-pgid`), waits 50ms, and escalates to `SIGKILL`.

### 4.2 Credential Protection
The daemon actively scrubs stdout, stderr, and environment parameters using 10 specialized regex filters:
- Anthropic API keys (`sk-ant-...`)
- OpenAI API keys (`sk-...`, `sk-proj-...`)
- DeepSeek API keys (`sk-[a-f0-9]{32,}`)
- GitHub Personal Access Tokens (`ghp-...`, `gho-...`)
- HuggingFace tokens (`hf_...`)
- Bearer tokens (`Authorization: Bearer ...`)
- CLI key flags (`--api-key`, `--token`, `--secret`)

### 4.3 Automated End-to-End Task Orchestration Script
Below is an example Bash script demonstrating how a developer or CI job orchestrates an external agent using READ:

```bash
#!/usr/bin/env bash
set -e

TASK_ID="TASK-USER-AUTH"
CONTRACT_FILE="contract.json"
BASE_COMMIT=$(git rev-parse HEAD)

# 1. Compute deterministic contract hash
CONTRACT_HASH=$(sha256sum "$CONTRACT_FILE" | awk '{print $1}')

# 2. Define task in READ CAS state engine
read-engine define "$TASK_ID" "$CONTRACT_HASH" "$BASE_COMMIT"

# 3. Transition to Build stage
read-engine build

# 4. Deterministically extract C0 AST context
read-engine ast-context src/auth.py > /tmp/ast_context.txt

# 5. Run external agent with targeted context
dsh --prompt "Implement authenticate_user according to /tmp/ast_context.txt"

# 6. Stage changes and verify staged diff
git add src/auth.py tests/test_auth.py
DIFF_HASH=$(read-engine diff-hash)

# 7. Execute test suite and record verification
TEST_LOG=$(pytest tests/test_auth.py)
read-engine verify "$DIFF_HASH" "$TEST_LOG"

# 8. CAS-gated handoff
read-engine handoff

echo "Task $TASK_ID completed and verified."
```
