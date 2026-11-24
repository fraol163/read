#!/usr/bin/env python3
"""READ Framework Universal MCP Server Module."""
import os
import sys

def main():
    if len(sys.argv) > 1 and sys.argv[1] in ["setup", "install", "--setup", "--install", "detect", "--detect"]:
        try:
            import harness_setup
            harness_setup.main()
            return
        except ImportError:
            script_dir = os.path.dirname(os.path.realpath(__file__))
            harness_path = os.path.join(script_dir, "harness_setup.py")
            if os.path.exists(harness_path):
                import importlib.util
                spec = importlib.util.spec_from_file_location("harness_setup", harness_path)
                mod = importlib.util.module_from_spec(spec)
                spec.loader.exec_module(mod)
                mod.main()
                return

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
