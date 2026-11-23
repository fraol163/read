#!/usr/bin/env python3
"""
Timed execution of TUTORIAL.md in a fresh workspace.
Measures wall-clock time, step counts, machine specifications, and exit codes.
Rules: No semicolons, no em dashes, real measurements only.
"""

import datetime
import os
import platform
import shutil
import subprocess
import sys
import tempfile
import time

READ_BIN = os.path.abspath("read/bin/read")

def main():
    t_start = datetime.datetime.now(datetime.timezone.utc)
    perf_start = time.perf_counter()
    
    print("=== TIMED TUTORIAL VERIFICATION EXECUTION ===")
    print(f"Start Timestamp UTC: {t_start.isoformat()}")
    print(f"Platform: {platform.platform()}")
    print(f"Architecture: {platform.machine()}")
    print(f"Processor: {platform.processor()}")
    print(f"Python Version: {sys.version.split()[0]}")
    print(f"READ Binary: {READ_BIN}")
    print("")

    tmpdir = tempfile.mkdtemp(prefix="read_tutorial_timed_")
    steps = []

    def log_step(step_num, step_name, cmd_fn):
        s_t0 = time.perf_counter()
        rc, out = cmd_fn()
        s_dt = (time.perf_counter() - s_t0) * 1000.0
        steps.append((step_num, step_name, rc, s_dt))
        status_str = "PASS" if rc == 0 else "FAIL"
        print(f"[Step {step_num}] {step_name:40} -> {status_str} (exit: {rc}, latency: {s_dt:6.2f} ms)")
        if rc != 0:
            print(f"Error output:\n{out}")
            sys.exit(1)

    try:
        # Step 1: Workspace init
        def step1():
            p1 = subprocess.run(["git", "init", "-q"], cwd=tmpdir, capture_output=True, text=True)
            p2 = subprocess.run(["git", "config", "user.name", "Tutorial User"], cwd=tmpdir, capture_output=True, text=True)
            p3 = subprocess.run(["git", "config", "user.email", "tutorial@user.org"], cwd=tmpdir, capture_output=True, text=True)
            p4 = subprocess.run([READ_BIN, "init"], cwd=tmpdir, capture_output=True, text=True)
            return p4.returncode, p4.stdout + p4.stderr
        log_step(1, "Workspace Init (git init & read init)", step1)

        # Step 2: Contract definition
        def step2():
            task_path = os.path.join(tmpdir, ".read", "TASK.md")
            with open(task_path, "w") as f:
                f.write("# TASK CONTRACT: Math Service\nTaskID: TASK-001\nPreconditions:\n- Python 3\nPostconditions:\n- is_positive(n)\n")
            p_hash = subprocess.run(["sha256sum", ".read/TASK.md"], cwd=tmpdir, capture_output=True, text=True)
            chash = p_hash.stdout.split()[0]
            p = subprocess.run([READ_BIN, "define", "TASK-001", chash, "HEAD"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(2, "Task Contract Freeze (read define)", step2)

        # Step 3: Transition to build
        def step3():
            p = subprocess.run([READ_BIN, "build"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(3, "Transition to Build (read build)", step3)

        # Step 4: Implement code and tests
        def step4():
            with open(os.path.join(tmpdir, "math_service.py"), "w") as f:
                f.write("def is_positive(n: int) -> bool:\n    return n > 0\n")
            with open(os.path.join(tmpdir, "test_math.py"), "w") as f:
                f.write("import unittest\nfrom math_service import is_positive\nclass TestMath(unittest.TestCase):\n    def test_pos(self):\n        self.assertTrue(is_positive(5))\n    def test_neg(self):\n        self.assertFalse(is_positive(-5))\n    def test_zero(self):\n        self.assertFalse(is_positive(0))\nif __name__ == '__main__':\n    unittest.main()\n")
            p = subprocess.run(["python3", "-m", "unittest", "test_math.py"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(4, "Implement Code & Test Suite", step4)

        # Step 5: Stage changes and diff-hash
        def step5():
            subprocess.run(["git", "add", "math_service.py", "test_math.py"], cwd=tmpdir, check=True)
            p = subprocess.run([READ_BIN, "diff-hash"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(5, "Stage Changes & Compute Diff Hash", step5)

        # Step 6: Verify and record test log
        def step6():
            p_dh = subprocess.run([READ_BIN, "diff-hash"], cwd=tmpdir, capture_output=True, text=True)
            dh = p_dh.stdout.strip()
            p = subprocess.run([READ_BIN, "verify", dh, "All 3 unit tests passed"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(6, "Record Verification (read verify)", step6)

        # Step 7: CAS handoff transition
        def step7():
            p = subprocess.run([READ_BIN, "handoff"], cwd=tmpdir, capture_output=True, text=True)
            return p.returncode, p.stdout + p.stderr
        log_step(7, "Execute CAS Handoff (read handoff)", step7)

        # Step 8: Gate rejection on weak tests
        def step8():
            with open(os.path.join(tmpdir, "math_service.py"), "w") as f:
                f.write("def is_positive(n: int) -> bool:\n    return n >= 0\n")
            subprocess.run(["git", "add", "math_service.py"], cwd=tmpdir, check=True)
            p_mut = subprocess.run([READ_BIN, "mutate", "--json"], cwd=tmpdir, capture_output=True, text=True)
            return p_mut.returncode, p_mut.stdout + p_mut.stderr
        log_step(8, "Mutation Gate Verification Check", step8)

    finally:
        shutil.rmtree(tmpdir)

    perf_end = time.perf_counter()
    t_end = datetime.datetime.now(datetime.timezone.utc)
    total_elapsed_ms = (perf_end - perf_start) * 1000.0

    print("")
    print("=== SUMMARY METRICS ===")
    print(f"End Timestamp UTC: {t_end.isoformat()}")
    print(f"Total Step Count: {len(steps)}")
    print(f"Total Wall-Clock Time: {total_elapsed_ms:.2f} ms ({total_elapsed_ms / 1000.0:.3f} seconds)")
    print(f"All {len(steps)} steps completed with exit code 0.")

if __name__ == "__main__":
    main()
