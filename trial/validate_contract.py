#!/usr/bin/env python3
import sys
import os
import re

def validate_markdown_contract(path):
    if not os.path.exists(path):
        return False, f"File not found: {path}"
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()

    errors = []
    if not re.search(r"^#\s+Task:\s*.+", content, re.MULTILINE):
        errors.append("Missing title header '# Task: <title>'")
    if not re.search(r"^##\s+Goal\s*$", content, re.MULTILINE):
        errors.append("Missing '## Goal' section")
    if not re.search(r"^##\s+Running Tests\s*$", content, re.MULTILINE):
        errors.append("Missing '## Running Tests' section")
    if not re.search(r"```(?:bash|sh)?\s*\n\s*(?:python3|pytest)[^\n]+", content):
        errors.append("Missing runnable test command block")

    if errors:
        return False, ", ".join(errors)
    return True, "Valid Markdown task contract"

def parse_simple_yaml(content):
    data = {}
    current_key = None
    multiline_val = []
    in_multiline = False

    lines = content.splitlines()
    for line in lines:
        stripped = line.strip()
        if not stripped or stripped.startswith("#"):
            continue

        if in_multiline:
            if line.startswith("  ") and not re.match(r"^[a-zA-Z0-9_]+:\s*", stripped):
                multiline_val.append(stripped)
                continue
            else:
                data[current_key] = " ".join(multiline_val)
                in_multiline = False
                multiline_val = []

        if re.match(r"^[a-zA-Z0-9_]+:\s*>", stripped):
            current_key = stripped.split(":", 1)[0].strip()
            in_multiline = True
            multiline_val = []
        elif ":" in stripped:
            k, v = stripped.split(":", 1)
            k = k.strip()
            v = v.strip()
            if not v:
                current_key = k
                data[k] = []
            else:
                current_key = k
                data[k] = v
        elif stripped.startswith("- ") and current_key:
            item = stripped[2:].strip()
            if isinstance(data.get(current_key), list):
                data[current_key].append(item)
            else:
                data[current_key] = [item]

    if in_multiline and current_key:
        data[current_key] = " ".join(multiline_val)

    return data

def validate_yaml_contract(path):
    if not os.path.exists(path):
        return False, f"File not found: {path}"
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()

    errors = []
    data = parse_simple_yaml(content)

    required_keys = ["id", "title", "objective", "affected_files", "test_command", "timeout_minutes"]
    for req in required_keys:
        if req not in data or not data[req]:
            errors.append(f"Missing required field: '{req}'")

    if "timeout_minutes" in data:
        try:
            val = int(data["timeout_minutes"])
            if val <= 0:
                errors.append("Field 'timeout_minutes' must be a positive integer")
        except ValueError:
            errors.append("Field 'timeout_minutes' must be an integer")

    if errors:
        return False, ", ".join(errors)
    return True, "Valid YAML task contract"

def validate_file(path):
    if path.endswith(".yaml") or path.endswith(".yml"):
        return validate_yaml_contract(path)
    elif path.endswith(".md"):
        return validate_markdown_contract(path)
    else:
        return False, f"Unsupported contract format: {path}"

def main():
    if len(sys.argv) < 2:
        print("Usage: validate_contract.py <contract_file_or_dir>...")
        sys.exit(1)

    all_passed = True
    for target in sys.argv[1:]:
        if os.path.isdir(target):
            files = [os.path.join(target, f) for f in os.listdir(target) if f.endswith((".md", ".yaml", ".yml"))]
        else:
            files = [target]

        for fpath in files:
            ok, msg = validate_file(fpath)
            status = "PASS" if ok else "FAIL"
            print(f"[{status}] {fpath}: {msg}")
            if not ok:
                all_passed = False

    sys.exit(0 if all_passed else 1)

if __name__ == "__main__":
    main()
