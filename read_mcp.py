#!/usr/bin/env python3
"""READ Framework Universal MCP Server Module."""
import os
import sys

def main():
    script_dir = os.path.dirname(os.path.realpath(__file__))
    target = os.path.join(script_dir, "bin", "read-mcp")
    if not os.path.exists(target):
        target = os.path.join(script_dir, "read", "bin", "read-mcp")

    with open(target, "r", encoding="utf-8") as f:
        code = f.read()

    globs = {"__file__": target, "__name__": "__main__"}
    exec(compile(code, target, "exec"), globs)

if __name__ == "__main__":
    main()
