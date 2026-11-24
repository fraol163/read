#!/usr/bin/env python3
"""
READ Framework: AI Agent Harness & Skill Setup
skills.sh Compatible Multi-Agent Discovery & Auto-Configuration
"""

import sys
import os
import json
import shutil
import platform

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

def detect_harnesses():
    home = get_home()
    claude_desktop_path = get_claude_desktop_config_path()

    harnesses = [
        {
            "id": "claude-code",
            "name": "Claude Code",
            "type": "cli",
            "config_path": os.path.join(home, ".claude.json"),
            "skills_dir": os.path.join(home, ".agents", "skills", "read"),
            "detected": bool(shutil.which("claude") or os.path.exists(os.path.join(home, ".claude.json")) or os.path.exists(os.path.join(home, ".claude")))
        },
        {
            "id": "cursor",
            "name": "Cursor",
            "type": "ide",
            "config_path": os.path.join(home, ".cursor", "mcp.json"),
            "rules_dir": os.path.join(home, ".cursor", "rules"),
            "detected": bool(shutil.which("cursor") or os.path.exists(os.path.join(home, ".cursor")) or os.path.exists(".cursor"))
        },
        {
            "id": "antigravity",
            "name": "Google Antigravity / Gemini CLI",
            "type": "cli",
            "config_path": os.path.join(home, ".gemini", "antigravity-cli", "mcp", "read.json"),
            "skills_dir": os.path.join(home, ".gemini", "config", "skills", "read"),
            "detected": bool(shutil.which("agy") or shutil.which("gemini") or os.path.exists(os.path.join(home, ".gemini")))
        },
        {
            "id": "claude-desktop",
            "name": "Claude Desktop",
            "type": "app",
            "config_path": claude_desktop_path,
            "detected": os.path.exists(os.path.dirname(claude_desktop_path))
        },
        {
            "id": "windsurf",
            "name": "Windsurf (Codeium)",
            "type": "ide",
            "config_path": os.path.join(home, ".codeium", "windsurf", "mcp_config.json"),
            "detected": os.path.exists(os.path.join(home, ".codeium", "windsurf"))
        },
        {
            "id": "continue",
            "name": "Continue.dev",
            "type": "ide",
            "config_path": os.path.join(home, ".continue", "config.json"),
            "detected": os.path.exists(os.path.join(home, ".continue"))
        },
        {
            "id": "zed",
            "name": "Zed Editor",
            "type": "ide",
            "config_path": os.path.join(home, ".config", "zed", "settings.json"),
            "detected": bool(shutil.which("zed") or os.path.exists(os.path.join(home, ".config", "zed")))
        },
        {
            "id": "goose",
            "name": "Goose (Block)",
            "type": "cli",
            "config_path": os.path.join(home, ".config", "goose", "config.yaml"),
            "detected": bool(shutil.which("goose") or os.path.exists(os.path.join(home, ".config", "goose")))
        }
    ]
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

def install_skill_file(target_dir):
    os.makedirs(target_dir, exist_ok=True)
    target_file = os.path.join(target_dir, "SKILL.md")
    with open(target_file, "w", encoding="utf-8") as f:
        f.write(SKILL_CONTENT)

def install_cursor_rule(rules_dir):
    os.makedirs(rules_dir, exist_ok=True)
    rule_file = os.path.join(rules_dir, "read.mdc")
    with open(rule_file, "w", encoding="utf-8") as f:
        f.write(SKILL_CONTENT)

def configure_harness(harness):
    h_id = harness["id"]
    cfg_path = harness.get("config_path")

    # MCP server registration
    if h_id == "antigravity":
        install_antigravity_config(cfg_path)
    elif h_id in ("claude-code", "cursor", "claude-desktop", "windsurf", "continue"):
        update_json_mcp(cfg_path)
    elif h_id == "zed":
        # Zed context_servers format
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
        # check if already added
        existing = [cs for cs in data.get("context_servers", []) if isinstance(cs, dict) and cs.get("id") == "read"]
        if not existing:
            data["context_servers"].append({
                "id": "read",
                "executable": "read-mcp",
                "args": []
            })
            with open(cfg_path, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=2)

    # Agent skill registration
    if "skills_dir" in harness:
        install_skill_file(harness["skills_dir"])
    if "rules_dir" in harness:
        install_cursor_rule(harness["rules_dir"])

def run_interactive_setup(auto_all=False, auto_detected=False):
    print("────────────────────────────────────────────────────────────────")
    print("   READ Framework: AI Agent Harness & Skill Setup")
    print("   skills.sh Compatible Multi-Agent Discovery & Configuration")
    print("────────────────────────────────────────────────────────────────")
    print("")

    harnesses = detect_harnesses()
    print("Scanning host for installed AI coding assistants & agent harnesses:")
    print("")

    detected_indices = []
    for i, h in enumerate(harnesses, 1):
        status = "[DETECTED]" if h["detected"] else "[NOT FOUND]"
        path_hint = h.get("config_path", "")
        if len(path_hint) > 36:
            path_hint = "..." + path_hint[-33:]
        print(f"  [{i}] {h['name']:<28} {path_hint:<36} {status}")
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
            # Parse comma or space separated numbers
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
    print(f"Configuring {len(selected_harnesses)} AI agent harness(es)...")
    for h in selected_harnesses:
        try:
            configure_harness(h)
            cfg_p = h.get("config_path", "")
            print(f"  [OK] {h['name']}: Registered MCP server 'read' in {cfg_p}")
            if "skills_dir" in h:
                print(f"       Installed agent skill in {h['skills_dir']}/SKILL.md")
            if "rules_dir" in h:
                print(f"       Installed agent rule in {h['rules_dir']}/read.mdc")
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
        print("Detected AI Agent Harnesses:")
        for h in harnesses:
            stat = "YES" if h["detected"] else "NO"
            print(f"  {h['name']:<28} detected={stat} config={h.get('config_path')}")
        sys.exit(0)

    auto_all = "--all" in args
    auto_detected = "--detected" in args or "--batch" in args or "-y" in args
    run_interactive_setup(auto_all=auto_all, auto_detected=auto_detected)

if __name__ == "__main__":
    main()
