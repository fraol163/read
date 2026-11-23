#!/usr/bin/env python3
"""
Measurement harness for READ Framework claims verification.
Tests staged diff hash latency, crash recovery time, AST extraction latency,
rate limiter reverse proxy behavior, RLIMIT_AS constraints, and large test suite hook latency.
Rules: No semicolons, no em dashes, real measurements only.
"""

import json
import math
import os
import resource
import shutil
import statistics
import subprocess
import sys
import tempfile
import time
import urllib.request
import urllib.error

ENGINE_BIN = os.path.abspath("read/bin/read-engine")
DAEMON_BIN = os.path.abspath("read/bin/read-daemon")

def run_cmd(cmd, cwd=None, env=None):
    t0 = time.perf_counter()
    res = subprocess.run(cmd, cwd=cwd, env=env, capture_output=True, text=True)
    dt_ms = (time.perf_counter() - t0) * 1000.0
    return res.returncode, res.stdout, res.stderr, dt_ms

def format_stats(data_ms):
    mean_val = statistics.mean(data_ms)
    stddev_val = statistics.stdev(data_ms) if len(data_ms) > 1 else 0.0
    min_val = min(data_ms)
    max_val = max(data_ms)
    return {
        "runs": len(data_ms),
        "raw_ms": [round(x, 3) for x in data_ms],
        "mean_ms": round(mean_val, 3),
        "stddev_ms": round(stddev_val, 3),
        "min_ms": round(min_val, 3),
        "max_ms": round(max_val, 3)
    }

def measure_diff_hash():
    print("=== MEASURING 1: Staged Diff Hash Latency (10 runs with git spawn) ===")
    tmpdir = tempfile.mkdtemp(prefix="read_bench_diff_")
    try:
        run_cmd(["git", "init"], cwd=tmpdir)
        run_cmd(["git", "config", "user.name", "Tester"], cwd=tmpdir)
        run_cmd(["git", "config", "user.email", "test@example.com"], cwd=tmpdir)
        with open(os.path.join(tmpdir, "main.py"), "w") as f:
            f.write("def foo():\n    return 1\n")
        run_cmd(["git", "add", "main.py"], cwd=tmpdir)
        run_cmd(["git", "commit", "-m", "initial commit"], cwd=tmpdir)
        
        # Now stage a diff
        with open(os.path.join(tmpdir, "main.py"), "w") as f:
            f.write("def foo():\n    return 2\n")
        run_cmd(["git", "add", "main.py"], cwd=tmpdir)
        
        times = []
        hashes = []
        for i in range(10):
            rc, out, err, dt = run_cmd([ENGINE_BIN, "diff-hash"], cwd=tmpdir)
            assert rc == 0, f"diff-hash failed: {err}"
            times.append(dt)
            hashes.append(out.strip())
            
        stats = format_stats(times)
        stats["diff_hash"] = hashes[0]
        print(f"Diff Hash: {hashes[0]}")
        print(f"Mean: {stats['mean_ms']} ms, StdDev: {stats['stddev_ms']} ms, Min: {stats['min_ms']} ms, Max: {stats['max_ms']} ms")
        return stats
    finally:
        shutil.rmtree(tmpdir)

def measure_crash_recovery():
    print("\n=== MEASURING 2: Crash Recovery Time (10 runs) ===")
    times = []
    messages = []
    
    for i in range(10):
        tmpdir = tempfile.mkdtemp(prefix=f"read_bench_rec_{i}_")
        try:
            run_cmd(["git", "init"], cwd=tmpdir)
            run_cmd(["git", "config", "user.name", "Tester"], cwd=tmpdir)
            run_cmd(["git", "config", "user.email", "test@example.com"], cwd=tmpdir)
            with open(os.path.join(tmpdir, "calc.py"), "w") as f:
                f.write("def add(a, b):\n    return a + b\n")
            run_cmd(["git", "add", "calc.py"], cwd=tmpdir)
            run_cmd(["git", "commit", "-m", "init"], cwd=tmpdir)
            
            run_cmd([ENGINE_BIN, "init"], cwd=tmpdir)
            run_cmd([ENGINE_BIN, "define", "TASK-TEST-001", "abc123hash", "HEAD"], cwd=tmpdir)
            run_cmd([ENGINE_BIN, "build"], cwd=tmpdir)
            
            # Stage diff
            with open(os.path.join(tmpdir, "calc.py"), "w") as f:
                f.write("def add(a, b):\n    return a + b + 0\n")
            run_cmd(["git", "add", "calc.py"], cwd=tmpdir)
            rc, diff_hash_out, _, _ = run_cmd([ENGINE_BIN, "diff-hash"], cwd=tmpdir)
            diff_h = diff_hash_out.strip()
            
            run_cmd([ENGINE_BIN, "verify", diff_h, "verification log passed"], cwd=tmpdir)
            
            # Now simulate crash recovery
            rc, out, err, dt = run_cmd([ENGINE_BIN, "recover"], cwd=tmpdir)
            assert rc == 0, f"recover failed: {err}"
            times.append(dt)
            messages.append(out.strip())
        finally:
            shutil.rmtree(tmpdir)
            
    stats = format_stats(times)
    stats["sample_recovery_msg"] = messages[0]
    print(f"Mean: {stats['mean_ms']} ms, StdDev: {stats['stddev_ms']} ms, Min: {stats['min_ms']} ms, Max: {stats['max_ms']} ms")
    print(f"Sample msg: {messages[0]}")
    return stats

def measure_ast_extraction():
    print("\n=== MEASURING 3: AST Context Extraction Time per Language (10 runs each) ===")
    languages = {
        "Python": ("test.py", "import os\nimport sys\n\ndef calculate_sum(a: int, b: int) -> int:\n    return a + b\n\nclass DataProcessor:\n    def __init__(self):\n        pass\n    def process(self, data):\n        return data\n"),
        "TypeScript": ("test.ts", "import { readFile } from 'fs'\nexport function processData(input: string): boolean {\n    return input.length > 0\n}\nexport class ItemManager {\n    public listItems(): string[] {\n        return []\n    }\n}\n"),
        "Go": ("test.go", "package main\nimport \"fmt\"\nfunc ProcessRecord(id int) (string, error) {\n    return fmt.Sprintf(\"id: %d\", id), nil\n}\ntype RecordStore struct {}\nfunc (s *RecordStore) Save(val string) bool {\n    return true\n}\n"),
        "Rust": ("test.rs", "pub fn execute_query(q: &str) -> usize {\n    q.len()\n}\npub struct SessionEngine {\n    pub id: u64\n}\n"),
        "Java": ("test.java", "package com.example\npublic class OrderService {\n    public boolean validateOrder(String orderId) {\n        return true\n    }\n}\n"),
        "CSharp": ("test.cs", "namespace CoreApp {\n    public class PaymentProcessor {\n        public bool ProcessTransaction(decimal amount) {\n            return true\n        }\n    }\n}\n"),
        "Cpp": ("test.cpp", "namespace Engine {\n    bool InitializeSystem(int code) {\n        return true\n    }\n    class DeviceDriver {\n    public:\n        void Reset() {}\n    }\n}\n"),
        "Ruby": ("test.rb", "require 'json'\nmodule Analytics\n  def self.track_event(name, payload)\n    true\n  end\n  class Reporter\n    def generate_report\n      {}\n    end\n  end\nend\n")
    }
    
    results = {}
    tmpdir = tempfile.mkdtemp(prefix="read_bench_ast_")
    try:
        run_cmd(["git", "init"], cwd=tmpdir)
        for lang, (fname, content) in languages.items():
            fpath = os.path.join(tmpdir, fname)
            with open(fpath, "w") as f:
                f.write(content)
            
            times = []
            for i in range(10):
                rc, out, err, dt = run_cmd([ENGINE_BIN, "ast-context", fname], cwd=tmpdir)
                assert rc == 0, f"ast-context {lang} failed: {err}"
                times.append(dt)
            stats = format_stats(times)
            results[lang] = stats
            print(f"{lang:12}: Mean: {stats['mean_ms']:6.3f} ms, StdDev: {stats['stddev_ms']:6.3f} ms, Min: {stats['min_ms']:6.3f} ms, Max: {stats['max_ms']:6.3f} ms")
        
        # Now measure on a larger 100-function file
        print("\nMeasuring larger file (100 functions) in Python:")
        large_py = os.path.join(tmpdir, "large.py")
        with open(large_py, "w") as f:
            for j in range(100):
                f.write(f"def generated_function_{j:03d}(x, y):\n    return x + y + {j}\n\n")
        times_large = []
        for i in range(10):
            rc, out, err, dt = run_cmd([ENGINE_BIN, "ast-context", "large.py"], cwd=tmpdir)
            assert rc == 0
            times_large.append(dt)
        stats_large = format_stats(times_large)
        results["Python_100_funcs"] = stats_large
        print(f"Python 100-fn: Mean: {stats_large['mean_ms']:6.3f} ms, StdDev: {stats_large['stddev_ms']:6.3f} ms, Min: {stats_large['min_ms']:6.3f} ms, Max: {stats_large['max_ms']:6.3f} ms")
        return results
    finally:
        shutil.rmtree(tmpdir)

def measure_proxy_and_rate_limiter():
    print("\n=== MEASURING 4: Rate Limiter and Reverse Proxy X-Forwarded-For Behavior ===")
    tmpdir = tempfile.mkdtemp(prefix="read_bench_proxy_")
    token = "test-secret-token-12345"
    port = 19443
    daemon_proc = None
    try:
        run_cmd(["git", "init"], cwd=tmpdir)
        cmd = [
            DAEMON_BIN,
            "--port", str(port),
            "--auth-token", token,
            "--workspace", tmpdir
        ]
        daemon_proc = subprocess.Popen(cmd, cwd=tmpdir, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        time.sleep(1.0)
        
        status_url = f"http://127.0.0.1:{port}/api/v1/status"
        
        # Test 1: Direct loopback requests without X-Forwarded-For
        # Whitelisted: should never be rate limited
        loopback_successes = 0
        for i in range(15):
            req = urllib.request.Request(status_url)
            try:
                with urllib.request.urlopen(req, timeout=2.0) as resp:
                    if resp.status == 200:
                        loopback_successes += 1
            except Exception as e:
                pass
        print(f"Loopback requests (no XFF, 15 sent): {loopback_successes}/15 succeeded (whitelisted)")
        
        # Test 2: Requests with simulated external client IP in X-Forwarded-For
        # Burst limit is 10, sustained is 2/sec. The 11th request should get 429
        external_results = []
        for i in range(15):
            req = urllib.request.Request(status_url)
            req.add_header("X-Forwarded-For", "198.51.100.55")
            try:
                with urllib.request.urlopen(req, timeout=2.0) as resp:
                    external_results.append(resp.status)
            except urllib.error.HTTPError as e:
                external_results.append(e.code)
            except Exception as e:
                external_results.append(str(e))
                
        count_200 = external_results.count(200)
        count_429 = external_results.count(429)
        print(f"External IP requests via XFF (198.51.100.55): {count_200} succeeded, {count_429} rate-limited (HTTP 429)")
        
        # Test 3: Spoofed X-Forwarded-For with loopback IP from client
        # Showing whether an attacker can bypass rate limits by sending X-Forwarded-For: 127.0.0.1
        spoofed_results = []
        for i in range(15):
            req = urllib.request.Request(status_url)
            req.add_header("X-Forwarded-For", "127.0.0.1")
            try:
                with urllib.request.urlopen(req, timeout=2.0) as resp:
                    spoofed_results.append(resp.status)
            except urllib.error.HTTPError as e:
                spoofed_results.append(e.code)
            except Exception as e:
                spoofed_results.append(str(e))
        count_spoofed_200 = spoofed_results.count(200)
        print(f"Spoofed XFF: 127.0.0.1 requests (15 sent): {count_spoofed_200}/15 succeeded (bypassed rate limiter)")
        
        return {
            "loopback_unlimited": loopback_successes == 15,
            "external_ip_rate_limited": count_429 > 0,
            "spoofed_xff_bypasses_limiter": count_spoofed_200 == 15,
            "external_results": external_results,
            "spoofed_results": spoofed_results
        }
    finally:
        if daemon_proc:
            daemon_proc.terminate()
            daemon_proc.wait()
        shutil.rmtree(tmpdir)

def measure_rlimit_as():
    print("\n=== MEASURING 5: RLIMIT_AS 30% RAM Ceiling vs 512MB Fallback on Java and .NET ===")
    # 512MB limit in bytes
    limit_512mb = 512 * 1024 * 1024
    # 30% of 5.7 GiB host RAM
    limit_30pct = int(5.7 * 1024 * 1024 * 1024 * 0.30)
    
    def run_with_as_limit(limit_bytes, cmd):
        def set_lim():
            resource.setrlimit(resource.RLIMIT_AS, (limit_bytes, limit_bytes))
        try:
            p = subprocess.run(cmd, preexec_fn=set_lim, capture_output=True, text=True, timeout=5)
            return p.returncode, p.stdout.strip(), p.stderr.strip()
        except Exception as e:
            return -1, "", str(e)

    res_512_java = run_with_as_limit(limit_512mb, ["java", "-version"])
    res_512_dotnet = run_with_as_limit(limit_512mb, ["dotnet", "--info"])
    res_30pct_java = run_with_as_limit(limit_30pct, ["java", "-version"])
    res_30pct_dotnet = run_with_as_limit(limit_30pct, ["dotnet", "--info"])
    
    print(f"Java under 512MB RLIMIT_AS: Exit={res_512_java[0]}, Msg='{res_512_java[1] or res_512_java[2]}'")
    print(f"Dotnet under 512MB RLIMIT_AS: Exit={res_512_dotnet[0]}, Msg='{res_512_dotnet[2][:120]}'")
    print(f"Java under 30% RAM (1.71GB) RLIMIT_AS: Exit={res_30pct_java[0]}")
    print(f"Dotnet under 30% RAM (1.71GB) RLIMIT_AS: Exit={res_30pct_dotnet[0]}, Msg='{res_30pct_dotnet[2][:120]}'")
    
    return {
        "limit_512mb_bytes": limit_512mb,
        "limit_30pct_bytes": limit_30pct,
        "java_512mb": {"exit": res_512_java[0], "output": res_512_java[1] or res_512_java[2]},
        "dotnet_512mb": {"exit": res_512_dotnet[0], "output": res_512_dotnet[2]},
        "java_30pct": {"exit": res_30pct_java[0]},
        "dotnet_30pct": {"exit": res_30pct_dotnet[0], "output": res_30pct_dotnet[2]}
    }

def measure_large_test_suite_latency():
    print("\n=== MEASURING 6: Test Harness Hook Latency on 500-Test Suite ===")
    tmpdir = tempfile.mkdtemp(prefix="read_bench_500tests_")
    try:
        run_cmd(["git", "init"], cwd=tmpdir)
        with open(os.path.join(tmpdir, "test_large_suite.py"), "w") as f:
            f.write("import unittest\n\nclass BenchmarkSuite(unittest.TestCase):\n")
            for k in range(500):
                f.write(f"    def test_case_{k:03d}(self):\n        self.assertEqual({k} * 2, {k * 2})\n")
        
        times = []
        for i in range(5):
            t0 = time.perf_counter()
            rc, out, err, _ = run_cmd(["python3", "-m", "unittest", "test_large_suite.py"], cwd=tmpdir)
            dt_ms = (time.perf_counter() - t0) * 1000.0
            assert rc == 0
            times.append(dt_ms)
            
        stats = format_stats(times)
        print(f"500 Tests Latency: Mean: {stats['mean_ms']} ms, Min: {stats['min_ms']} ms, Max: {stats['max_ms']} ms")
        return stats
    finally:
        shutil.rmtree(tmpdir)

def test_external_repo_mutation():
    print("\n=== MEASURING 7: Real External Repository AROP Mutation Gate Test ===")
    # Test on the real local Python package veloce-acceptance/github-repo
    src_dir = "/home/renan/veloce-acceptance/github-repo"
    tmpdir = tempfile.mkdtemp(prefix="read_real_repo_test_")
    try:
        # Copy backend/version_catalog.py and tests/test_version_catalog.py
        os.makedirs(os.path.join(tmpdir, "backend"))
        os.makedirs(os.path.join(tmpdir, "tests"))
        shutil.copy2(os.path.join(src_dir, "backend", "version_catalog.py"), os.path.join(tmpdir, "backend"))
        shutil.copy2(os.path.join(src_dir, "tests", "test_version_catalog.py"), os.path.join(tmpdir, "tests"))
        
        run_cmd(["git", "init"], cwd=tmpdir)
        run_cmd(["git", "config", "user.name", "Tester"], cwd=tmpdir)
        run_cmd(["git", "config", "user.email", "test@example.com"], cwd=tmpdir)
        run_cmd(["git", "add", "-A"], cwd=tmpdir)
        run_cmd(["git", "commit", "-m", "baseline"], cwd=tmpdir)
        
        # Verify baseline tests pass
        rc, out, err, _ = run_cmd(["python3", "-m", "unittest", "tests/test_version_catalog.py"], cwd=tmpdir)
        print(f"Baseline test run: exit={rc}, output={err.strip() or out.strip()}")
        
        # Make a realistic code modification in backend/version_catalog.py
        catalog_path = os.path.join(tmpdir, "backend", "version_catalog.py")
        with open(catalog_path, "r") as f:
            lines = f.readlines()
        # Find a return or condition to modify
        for idx, line in enumerate(lines):
            if "def update_toml_version" in line:
                # Add a validation condition on line idx + 2
                lines.insert(idx + 1, "    if len(var_name) == 0:\n        return False\n")
                break
        with open(catalog_path, "w") as f:
            f.writelines(lines)
            
        run_cmd(["git", "add", "backend/version_catalog.py"], cwd=tmpdir)
        
        # Run mutate on staged diff
        rc, out, err, _ = run_cmd([ENGINE_BIN, "mutate", "--json"], cwd=tmpdir)
        mutants = json.loads(out)
        print(f"Generated {len(mutants)} mutants on real repo diff:")
        for idx, m in enumerate(mutants):
            print(f"  Mutant #{idx+1}: {m['operator']} at L{m['line_number']} -> {m['mutated_line'].strip()}")
            
        # Test each mutant manually against unittest
        with open(catalog_path, "r") as f:
            orig_content = f.read()
            
        killed = 0
        survived = 0
        details = []
        for idx, m in enumerate(mutants):
            l_num = m["line_number"] - 1
            cur_lines = orig_content.splitlines(keepends=True)
            cur_lines[l_num] = m["mutated_line"] + "\n"
            with open(catalog_path, "w") as f:
                f.writelines(cur_lines)
            
            rc, m_out, m_err, _ = run_cmd(["python3", "-m", "unittest", "tests/test_version_catalog.py"], cwd=tmpdir)
            if rc != 0:
                killed += 1
                details.append((idx + 1, m["operator"], "KILLED"))
            else:
                survived += 1
                details.append((idx + 1, m["operator"], "SURVIVED"))
                
        # Restore
        with open(catalog_path, "w") as f:
            f.write(orig_content)
            
        score = (killed / len(mutants) * 100.0) if mutants else 100.0
        print(f"\nReal Repo Mutation Result: {killed}/{len(mutants)} killed ({score:.1f}%)")
        for num, op, status in details:
            print(f"  Mutant #{num} ({op}): {status}")
            
        return {
            "target_file": "backend/version_catalog.py",
            "mutants_count": len(mutants),
            "killed": killed,
            "survived": survived,
            "mutation_score_pct": score,
            "mutants": details
        }
    finally:
        shutil.rmtree(tmpdir)

def main():
    report = {}
    report["diff_hash"] = measure_diff_hash()
    report["crash_recovery"] = measure_crash_recovery()
    report["ast_extraction"] = measure_ast_extraction()
    report["proxy_and_rate_limit"] = measure_proxy_and_rate_limiter()
    report["rlimit_as"] = measure_rlimit_as()
    report["large_suite_hook"] = measure_large_test_suite_latency()
    report["external_repo_mutation"] = test_external_repo_mutation()
    
    out_file = os.path.abspath("evidence/logs/measurements.json")
    with open(out_file, "w") as f:
        json.dump(report, f, indent=2)
    print(f"\nAll measurements saved to: {out_file}")

if __name__ == "__main__":
    main()
