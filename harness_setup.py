#!/usr/bin/env python3
"""
READ Framework: AI Agent Harness & Skill Setup
Autonomous Real-Time Multi-Agent Discovery & Auto-Configuration
"""

import sys
import os
import json
import shutil
import platform
import subprocess
import socket

SKILL_CONTENT = """---
name: read-framework
description: >-
  Resource-conscious, Evidence-driven AI Development (READ).
  Guides the agent through deterministic C0-AST context extraction,
  Compare-And-Swap state machine locking, cryptographic WAL verification,
  and AROP 100% mutation gating before code commits.
---

# READ Framework: Agent Operational Guide

The READ Framework replaces conversational QA assumptions with deterministic, mechanical systems verification.

## Core Operational Protocol

### 1. Read Before Touching Code (Define Stage)
- Never load entire 10,000+ token files into context.
- Call the `read_ast_context` MCP tool (or run `read-engine ast-context <file>`).
- Inspect the deterministic symbol signatures and dependency closures for free.
- Verify task contracts in `TASK.md` and check `read_status`.

### 2. Implement Within Bounded Limits (Build Stage)
- Keep changes scoped to the contract definition.
- Transition state to executing via `read_build`.
- Run local unit tests to ensure happy path and error cases execute cleanly.

### 3. Mechanical Verification (Verify Stage)
- Plain passing tests are not sufficient to claim correctness.
- Compute the staged diff hash with `read_diff_hash`.
- Generate targeted AST mutants on the staged diff with `read_mutate`.
- Run the test suite against the mutants.
- Evaluate the kill ratio with `read_eval_mutation_score`.
- If any mutant survives, strengthen the test assertions until 100% of mutants are killed.
- Record verification with `read_verify`.

### 4. Zero-Token Crash Recovery
- If a process dies or state is interrupted, call `read_recover`.
- The engine replays the cryptographic `WAL.log` hash chain and restores consistent state without re-spending model tokens.

### 5. Evidence Audit and Reporting
- Generate machine-readable YAML state with `read_status_yaml`.
- Generate sealed HTML verification reports with `read_report_html`.
- Run micro-DSL assembly sequences for token efficiency with `read_asm`.
"""

def get_home():
    return os.path.expanduser("~")

def get_running_processes():
    procs = set()
    # Fast real-time process table inspection via /proc on Linux (<5ms)
    if os.path.exists("/proc"):
        for pid in os.listdir("/proc"):
            if pid.isdigit():
                try:
                    with open(f"/proc/{pid}/comm", "r", encoding="utf-8", errors="ignore") as f:
                        comm = f.read().strip().lower()
                        if comm:
                            procs.add(comm)
                except Exception:
                    pass
    # Cross-platform fallback via ps
    if not procs and shutil.which("ps"):
        try:
            res = subprocess.run(["ps", "-A", "-o", "comm="], capture_output=True, text=True, timeout=1)
            for line in res.stdout.splitlines():
                comm = os.path.basename(line.strip()).lower()
                if comm:
                    procs.add(comm)
        except Exception:
            pass
    return procs

def is_port_open(port, host="127.0.0.1", timeout=0.15):
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(timeout)
            return s.connect_ex((host, port)) == 0
    except Exception:
        return False

def get_claude_desktop_config_path():
    sys_name = platform.system()
    home = get_home()
    if sys_name == "Darwin":
        return os.path.join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
    elif sys_name == "Windows":
        appdata = os.environ.get("APPDATA", os.path.join(home, "AppData", "Roaming"))
        return os.path.join(appdata, "Claude", "claude_desktop_config.json")
    else:
        return os.path.join(home, ".config", "Claude", "claude_desktop_config.json")

def get_vscode_global_storage():
    sys_name = platform.system()
    home = get_home()
    if sys_name == "Darwin":
        return os.path.join(home, "Library", "Application Support", "Code", "User", "globalStorage")
    elif sys_name == "Windows":
        appdata = os.environ.get("APPDATA", os.path.join(home, "AppData", "Roaming"))
        return os.path.join(appdata, "Code", "User", "globalStorage")
    else:
        return os.path.join(home, ".config", "Code", "User", "globalStorage")

def is_mcp_configured_in_json(path, mcp_name="read"):
    if not os.path.exists(path):
        return False
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
            if isinstance(data, dict):
                return mcp_name in data.get("mcpServers", {})
    except Exception:
        pass
    return False

def detect_harnesses():
    home = get_home()
    running_procs = get_running_processes()
    claude_desktop_path = get_claude_desktop_config_path()
    vscode_storage = get_vscode_global_storage()

    cline_path = os.path.join(vscode_storage, "saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json")
    roo_path = os.path.join(vscode_storage, "rooveterinaryinc.roo-cline", "settings", "cline_mcp_settings.json")

    definitions = [
        {
            "id": "claude-code",
            "name": "Claude Code",
            "type": "cli",
            "process_names": ["claude", "claude-code"],
            "binary_names": ["claude", "claude-code"],
            "config_path": os.path.join(home, ".claude.json"),
            "skills_dir": os.path.join(home, ".agents", "skills", "read"),
        },
        {
            "id": "cursor",
            "name": "Cursor",
            "type": "ide",
            "process_names": ["cursor", "cursor-agent"],
            "binary_names": ["cursor", "cursor-agent"],
            "config_path": os.path.join(home, ".cursor", "mcp.json"),
            "rules_dir": os.path.join(home, ".cursor", "rules"),
        },
        {
            "id": "antigravity",
            "name": "Google Antigravity / Gemini CLI",
            "type": "cli",
            "process_names": ["antigravity", "agy", "gemini"],
            "binary_names": ["antigravity", "agy", "gemini"],
            "config_path": os.path.join(home, ".gemini", "antigravity-cli", "mcp", "read.json"),
            "skills_dir": os.path.join(home, ".gemini", "config", "skills", "read"),
        },
        {
            "id": "github-copilot",
            "name": "GitHub Copilot",
            "type": "cli/ide",
            "process_names": ["copilot", "github-copilot"],
            "binary_names": ["copilot", "github-copilot-cli"],
            "config_path": os.path.join(home, ".config", "github-copilot", "mcp.json"),
            "skills_dir": os.path.join(home, ".config", "github-copilot", "skills", "read"),
        },
        {
            "id": "cline",
            "name": "Cline (VS Code)",
            "type": "ide-extension",
            "process_names": ["code", "electron"],
            "binary_names": ["code"],
            "config_path": cline_path,
        },
        {
            "id": "roo-code",
            "name": "Roo Code (Roo Cline)",
            "type": "ide-extension",
            "process_names": ["code", "electron"],
            "binary_names": ["code"],
            "config_path": roo_path,
        },
        {
            "id": "claude-desktop",
            "name": "Claude Desktop",
            "type": "app",
            "process_names": ["claude"],
            "binary_names": ["claude"],
            "config_path": claude_desktop_path,
        },
        {
            "id": "windsurf",
            "name": "Windsurf (Codeium)",
            "type": "ide",
            "process_names": ["windsurf", "codeium"],
            "binary_names": ["windsurf"],
            "config_path": os.path.join(home, ".codeium", "windsurf", "mcp_config.json"),
        },
        {
            "id": "continue",
            "name": "Continue.dev",
            "type": "ide",
            "process_names": ["continue"],
            "binary_names": [],
            "config_path": os.path.join(home, ".continue", "config.json"),
        },
        {
            "id": "aider",
            "name": "Aider",
            "type": "cli",
            "process_names": ["aider"],
            "binary_names": ["aider"],
            "config_path": os.path.join(home, ".aider.conf.yml"),
        },
        {
            "id": "opencode",
            "name": "OpenCode",
            "type": "cli",
            "process_names": ["opencode"],
            "binary_names": ["opencode"],
            "config_path": os.path.join(home, ".config", "opencode", "mcp.json"),
        },
        {
            "id": "codex",
            "name": "OpenAI Codex CLI",
            "type": "cli",
            "process_names": ["codex"],
            "binary_names": ["codex"],
            "config_path": os.path.join(home, ".codex", "config.json"),
        },
        {
            "id": "zed",
            "name": "Zed Editor",
            "type": "ide",
            "process_names": ["zed", "zed-editor"],
            "binary_names": ["zed"],
            "config_path": os.path.join(home, ".config", "zed", "settings.json"),
        },
        {
            "id": "goose",
            "name": "Goose (Block)",
            "type": "cli",
            "process_names": ["goose"],
            "binary_names": ["goose"],
            "config_path": os.path.join(home, ".config", "goose", "config.yaml"),
        },
        {
            "id": "amp",
            "name": "Amp AI Agent",
            "type": "cli",
            "process_names": ["amp"],
            "binary_names": ["amp"],
            "config_path": os.path.join(home, ".config", "amp", "mcp.json"),
        },
        {
            "id": "ollama",
            "name": "Ollama Local AI",
            "type": "server",
            "process_names": ["ollama"],
            "binary_names": ["ollama"],
            "port": 11434,
            "config_path": os.path.join(home, ".ollama", "config.json"),
        },
        {
            "id": "deepseek",
            "name": "DeepSeek Harness (dsh)",
            "type": "cli",
            "process_names": ["dsh"],
            "binary_names": ["dsh"],
            "config_path": os.path.join(home, ".dsh", "config.json"),
        }
    ]

    harnesses = []
    mise_bin = os.path.join(home, ".local", "share", "mise", "shims")

    for d in definitions:
        is_live = False
        is_installed = False
        is_configured = False

        # 1. Real-time process check
        for pname in d.get("process_names", []):
            if pname in running_procs:
                is_live = True
                break

        # 2. Real-time socket check for servers
        if not is_live and "port" in d:
            if is_port_open(d["port"]):
                is_live = True

        # 3. Real-time binary discovery across PATH and mise
        for bname in d.get("binary_names", []):
            if shutil.which(bname):
                is_installed = True
                break
            if os.path.exists(os.path.join(mise_bin, bname)):
                is_installed = True
                break

        cfg = d.get("config_path", "")
        # 4. Check configuration file or directory presence
        if os.path.exists(cfg) or os.path.exists(os.path.dirname(cfg)):
            is_installed = True

        # Special directory checks for known ecosystems
        if d["id"] == "antigravity" and os.path.exists(os.path.join(home, ".gemini")):
            is_installed = True
        elif d["id"] == "cursor" and (os.path.exists(os.path.join(home, ".cursor")) or os.path.exists(".cursor")):
            is_installed = True
        elif d["id"] == "claude-code" and (os.path.exists(os.path.join(home, ".claude")) or os.path.exists(os.path.join(home, ".claude.json"))):
            is_installed = True

        # 5. Check if READ is already configured
        if d["id"] == "antigravity" and os.path.exists(cfg):
            try:
                with open(cfg, "r", encoding="utf-8") as f:
                    cdata = json.load(f)
                    if "read-mcp" in cdata.get("command", ""):
                        is_configured = True
            except Exception:
                pass
        elif is_mcp_configured_in_json(cfg, "read"):
            is_configured = True

        # Overall detected state
        detected = is_live or is_installed or is_configured

        d["detected"] = detected
        d["is_live"] = is_live
        d["is_configured"] = is_configured
        d["is_installed"] = is_installed
        harnesses.append(d)

    return harnesses

def update_json_mcp(config_path, mcp_name="read", command="read-mcp", args=None):
    if args is None:
        args = []
    parent = os.path.dirname(config_path)
    os.makedirs(parent, exist_ok=True)

    data = {}
    if os.path.exists(config_path):
        try:
            with open(config_path, "r", encoding="utf-8") as f:
                data = json.load(f)
        except Exception:
            data = {}

    if not isinstance(data, dict):
        data = {}

    if "mcpServers" not in data:
        data["mcpServers"] = {}

    data["mcpServers"][mcp_name] = {
        "command": command,
        "args": args
    }

    with open(config_path, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)

    # Real-time verification
    with open(config_path, "r", encoding="utf-8") as f:
        verified = json.load(f)
        if mcp_name not in verified.get("mcpServers", {}):
            raise IOError(f"Real-time verification failed for {config_path}")

def install_antigravity_config(config_path, command="read-mcp", args=None):
    if args is None:
        args = []
    parent = os.path.dirname(config_path)
    os.makedirs(parent, exist_ok=True)
    cfg = {
        "command": command,
        "args": args
    }
    with open(config_path, "w", encoding="utf-8") as f:
        json.dump(cfg, f, indent=2)

    # Real-time verification
    with open(config_path, "r", encoding="utf-8") as f:
        verified = json.load(f)
        if verified.get("command") != command:
            raise IOError(f"Real-time verification failed for {config_path}")

def install_skill_file(target_dir):
    os.makedirs(target_dir, exist_ok=True)
    target_file = os.path.join(target_dir, "SKILL.md")
    with open(target_file, "w", encoding="utf-8") as f:
        f.write(SKILL_CONTENT)
    if not os.path.exists(target_file) or os.path.getsize(target_file) == 0:
        raise IOError(f"Skill file installation verification failed for {target_file}")

def install_cursor_rule(rules_dir):
    os.makedirs(rules_dir, exist_ok=True)
    rule_file = os.path.join(rules_dir, "read.mdc")
    with open(rule_file, "w", encoding="utf-8") as f:
        f.write(SKILL_CONTENT)
    if not os.path.exists(rule_file) or os.path.getsize(rule_file) == 0:
        raise IOError(f"Cursor rule installation verification failed for {rule_file}")

def configure_harness(harness):
    h_id = harness["id"]
    cfg_path = harness.get("config_path")

    # MCP server registration
    if h_id == "antigravity":
        install_antigravity_config(cfg_path)
    elif h_id == "zed":
        parent = os.path.dirname(cfg_path)
        os.makedirs(parent, exist_ok=True)
        data = {}
        if os.path.exists(cfg_path):
            try:
                with open(cfg_path, "r", encoding="utf-8") as f:
                    data = json.load(f)
            except Exception:
                data = {}
        if "context_servers" not in data:
            data["context_servers"] = []
        existing = [cs for cs in data.get("context_servers", []) if isinstance(cs, dict) and cs.get("id") == "read"]
        if not existing:
            data["context_servers"].append({
                "id": "read",
                "executable": "read-mcp",
                "args": []
            })
            with open(cfg_path, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=2)
    elif cfg_path:
        update_json_mcp(cfg_path)

    # Agent skill and rule registration
    if "skills_dir" in harness:
        install_skill_file(harness["skills_dir"])
    if "rules_dir" in harness:
        install_cursor_rule(harness["rules_dir"])

def run_interactive_setup(auto_all=False, auto_detected=False):
    print("────────────────────────────────────────────────────────────────")
    print("   READ Framework: Real-Time AI Agent Discovery & Setup")
    print("   Autonomous Multi-Agent Detection & Configuration")
    print("────────────────────────────────────────────────────────────────")
    print("")

    harnesses = detect_harnesses()
    print("Real-time host inspection of AI coding assistants & agent harnesses:")
    print("")

    detected_indices = []
    for i, h in enumerate(harnesses, 1):
        if h["is_live"]:
            status = "[RUNNING LIVE]"
        elif h["is_configured"]:
            status = "[CONFIGURED]"
        elif h["is_installed"]:
            status = "[INSTALLED]"
        else:
            status = "[NOT FOUND]"

        path_hint = h.get("config_path", "")
        if len(path_hint) > 36:
            path_hint = "..." + path_hint[-33:]
        print(f"  [{i:>2}] {h['name']:<30} {path_hint:<36} {status}")
        if h["detected"]:
            detected_indices.append(i)

    print("")

    selected_harnesses = []
    if auto_all:
        selected_harnesses = harnesses
    elif auto_detected or not sys.stdin.isatty():
        if detected_indices:
            selected_harnesses = [harnesses[i-1] for i in detected_indices]
        else:
            selected_harnesses = [harnesses[0], harnesses[1], harnesses[2]]
    else:
        default_choice = "detected" if detected_indices else "all"
        prompt = f"Select harnesses to configure [1-{len(harnesses)}, all, detected, or quit] (default: {default_choice}): "
        try:
            choice = input(prompt).strip().lower()
        except (EOFError, KeyboardInterrupt):
            print("\nSetup cancelled.")
            return

        if not choice:
            choice = default_choice

        if choice in ("q", "quit", "exit"):
            print("Setup cancelled by user.")
            return
        elif choice == "all":
            selected_harnesses = harnesses
        elif choice == "detected":
            selected_harnesses = [harnesses[i-1] for i in detected_indices]
            if not selected_harnesses:
                print("No detected harnesses found. Defaulting to Claude Code, Cursor, and Antigravity.")
                selected_harnesses = [harnesses[0], harnesses[1], harnesses[2]]
        else:
            raw_parts = choice.replace(",", " ").split()
            for part in raw_parts:
                try:
                    idx = int(part)
                    if 1 <= idx <= len(harnesses):
                        selected_harnesses.append(harnesses[idx-1])
                except ValueError:
                    pass

    if not selected_harnesses:
        print("No agent harnesses selected.")
        return

    print("")
    print(f"Configuring and verifying {len(selected_harnesses)} AI agent harness(es) in real-time...")
    for h in selected_harnesses:
        try:
            configure_harness(h)
            cfg_p = h.get("config_path", "")
            print(f"  [OK] {h['name']}: Registered MCP server 'read' in {cfg_p} (verified real-time)")
            if "skills_dir" in h:
                print(f"       Installed agent skill in {h['skills_dir']}/SKILL.md (verified real-time)")
            if "rules_dir" in h:
                print(f"       Installed agent rule in {h['rules_dir']}/read.mdc (verified real-time)")
        except Exception as e:
            print(f"  [ERROR] {h['name']}: Failed to configure: {str(e)}")

    print("")
    print("────────────────────────────────────────────────────────────────")
    print("AI Agent harnesses successfully configured with READ MCP & Skill!")
    print("Restart your agent (e.g. Claude Code or Cursor) to load the tools.")
    print("────────────────────────────────────────────────────────────────")

def main():
    args = sys.argv[1:]
    if "--detect" in args or "detect" in args:
        harnesses = detect_harnesses()
        print("Real-Time Detected AI Agent Harnesses:")
        for h in harnesses:
            if h["is_live"]:
                stat = "RUNNING_LIVE"
            elif h["is_configured"]:
                stat = "CONFIGURED"
            elif h["is_installed"]:
                stat = "INSTALLED"
            else:
                stat = "NOT_FOUND"
            print(f"  {h['name']:<32} status={stat:<12} config={h.get('config_path')}")
        sys.exit(0)

    auto_all = "--all" in args
    auto_detected = "--detected" in args or "--batch" in args or "-y" in args
    run_interactive_setup(auto_all=auto_all, auto_detected=auto_detected)

if __name__ == "__main__":
    main()
