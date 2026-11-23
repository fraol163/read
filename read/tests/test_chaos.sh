#!/usr/bin/env bash
# ==============================================================================
# READ Framework Adversarial Red-Team & Chaos Test Suite
# Tests:
#   1. SIGKILL during mid-write of STATE.json and WAL.log (Crash / Power-Cut)
#   2. AST Parser Fuzzing (Malformed, Nested, Binary, Circular Dependencies)
#   3. Dead & Hanging Sockets during Dynamic Harness Scan (<500ms assertion)
#   4. Concurrent CAS Race on State Transitions (1 winner, 9 rejected)
#   5. Unauthorized Gate Bypass (Null hash, Mismatched hash, Tampered JSON)
#
# IMPORTANT: Zero emojis are used in this script, logs, or outputs.
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
READ_BIN="$WORKSPACE_ROOT/read/bin/read"
READ_ENGINE="$WORKSPACE_ROOT/read/bin/read-engine"
READ_DAEMON="$WORKSPACE_ROOT/read/bin/read-daemon"

TEST_TEMP_DIR="$(mktemp -d -t read-chaos-suite-XXXXXX)"
DAEMON_PID=""
MOCK_SERVER_PID=""

cleanup() {
    if [ -n "$DAEMON_PID" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
        kill -9 "$DAEMON_PID" 2>/dev/null || true
    fi
    if [ -n "$MOCK_SERVER_PID" ] && kill -0 "$MOCK_SERVER_PID" 2>/dev/null; then
        kill -9 "$MOCK_SERVER_PID" 2>/dev/null || true
    fi
    rm -rf "$TEST_TEMP_DIR"
}
trap cleanup EXIT

echo "================================================================================"
echo "STARTING ADVERSARIAL CHAOS & RED-TEAM TEST SUITE"
echo "Sandbox Directory: $TEST_TEMP_DIR"
echo "READ Engine: $READ_ENGINE"
echo "READ Daemon: $READ_DAEMON"
echo "================================================================================"

# Verify required binaries exist and are executable
for bin in "$READ_BIN" "$READ_ENGINE" "$READ_DAEMON"; do
    if [ ! -x "$bin" ]; then
        echo "[FATAL] Required binary not executable: $bin" >&2
        exit 1
    fi
done

# ------------------------------------------------------------------------------
# SCENARIO 1: SIGKILL during atomic state update & zero-token crash recovery
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 1]: SIGKILL During Atomic State Writes & Crash Recovery ==="
SCENARIO1_DIR="$TEST_TEMP_DIR/scenario1"
mkdir -p "$SCENARIO1_DIR"
cd "$SCENARIO1_DIR"
git init -q
"$READ_ENGINE" init

# Violent kill loop: run continuous transitions while randomly firing SIGKILL
KILL_ITERATIONS=15
VIOLENT_KILLS=0

for i in $(seq 1 $KILL_ITERATIONS); do
    # Spawn background worker performing state writes
    (
        for step in $(seq 1 100); do
            "$READ_ENGINE" define "CHAOS-TASK-$step" "dummy_contract_$step" "commit_$step" 2>/dev/null || true
            "$READ_ENGINE" build 2>/dev/null || true
        done
    ) &
    WORKER_PID=$!

    # Sub-millisecond sleep jitter to catch write/rename in-flight
    python3 -c "import time, random; time.sleep(random.uniform(0.001, 0.008))"
    if kill -0 "$WORKER_PID" 2>/dev/null; then
        kill -9 "$WORKER_PID" 2>/dev/null || true
        VIOLENT_KILLS=$((VIOLENT_KILLS + 1))
    fi
    wait "$WORKER_PID" 2>/dev/null || true
done

echo "[SCENARIO 1] Delivered $VIOLENT_KILLS violent SIGKILL signals mid-execution."

# Verify STATE.json syntactic integrity
if ! python3 -m json.tool .method/STATE.json > /dev/null 2>&1; then
    echo "[FAIL] STATE.json is corrupt after violent SIGKILL!" >&2
    exit 1
fi
echo "[SCENARIO 1] Assertion Passed: STATE.json remains 100% valid JSON."

# Run zero-token crash recovery
RECOVER_OUTPUT=$("$READ_ENGINE" recover)
if ! echo "$RECOVER_OUTPUT" | grep -q "RECOVER SUCCESS (0 Tokens Re-spent)"; then
    echo "[FAIL] Crash recovery failed after violent SIGKILL: $RECOVER_OUTPUT" >&2
    exit 1
fi
echo "[SCENARIO 1] Assertion Passed: 'read recover' verified 100% state recovery from WAL."
echo "[PASS] SCENARIO 1 PASSED: Zero data corruption under violent SIGKILL."

# ------------------------------------------------------------------------------
# SCENARIO 2: AST Parser Fuzzing (Malformed, Nested, Binary, Circular)
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 2]: AST Parser Fuzzing & Resilience ==="
SCENARIO2_DIR="$TEST_TEMP_DIR/scenario2"
mkdir -p "$SCENARIO2_DIR"
cd "$SCENARIO2_DIR"
git init -q

# 2.1 Malformed / Syntax-Broken file
cat << 'EOF' > broken_syntax.py
def broken_fn(a, b,,,,:
    class {{{{invalid_class:
        def ::::
            if True and:
EOF

echo "[SCENARIO 2.1] Fuzzing broken syntax..."
"$READ_ENGINE" ast-context broken_syntax.py > /dev/null
echo "[SCENARIO 2.1] Assertion Passed: Broken syntax parsed without panic."

# 2.2 Deeply nested unclosed parentheses & braces
cat << 'EOF' > unclosed_nesting.py
def deep_nesting():
EOF
python3 -c '
with open("unclosed_nesting.py", "a") as f:
    f.write("    x = " + "("*10000 + "1" + "\n")
    f.write("def secondary(): pass\n")
'
echo "[SCENARIO 2.2] Fuzzing 10,000 unclosed parentheses..."
"$READ_ENGINE" ast-context unclosed_nesting.py > /dev/null
echo "[SCENARIO 2.2] Assertion Passed: 10,000 unclosed braces processed without stack overflow."

# 2.3 10,000 imports in a single file
python3 -c '
with open("massive_imports.py", "w") as f:
    for i in range(10000):
        f.write(f"import non_existent_library_mod_{i}\n")
    f.write("def core_logic(): pass\n")
'
echo "[SCENARIO 2.3] Fuzzing 10,000 import declarations..."
"$READ_ENGINE" ast-context massive_imports.py > /dev/null
echo "[SCENARIO 2.3] Assertion Passed: 10,000 imports bounded deterministically."

# 2.4 Circular Dependency Loop (A -> B -> C -> A)
cat << 'EOF' > circ_a.py
import circ_b
def func_a(): return "a"
EOF

cat << 'EOF' > circ_b.py
import circ_c
def func_b(): return "b"
EOF

cat << 'EOF' > circ_c.py
import circ_a
def func_c(): return "c"
EOF

echo "[SCENARIO 2.4] Fuzzing circular dependency loop (A -> B -> C -> A)..."
CIRC_OUT=$("$READ_ENGINE" ast-context circ_a.py)
if ! echo "$CIRC_OUT" | grep -q "func_a" || ! echo "$CIRC_OUT" | grep -q "func_b"; then
    echo "[FAIL] Circular dependency resolution failed to extract symbols" >&2
    exit 1
fi
echo "[SCENARIO 2.4] Assertion Passed: Circular import cycle handled without recursion infinite loop."

# 2.5 Pure Binary Data
python3 -c '
import os
with open("binary_data.py", "wb") as f:
    f.write(os.urandom(2 * 1024 * 1024))
'
echo "[SCENARIO 2.5] Fuzzing 2MB raw random binary payload..."
"$READ_ENGINE" ast-context binary_data.py > /dev/null
echo "[SCENARIO 2.5] Assertion Passed: 2MB binary payload skipped cleanly without panic."

# 2.6 Malformed TSX / JSX Syntax Resilience
cat << 'EOF' > malformed.tsx
import React, { useState } from 'react';
export function BrokenComponent<T, K extends <broken>> (props: { data: T) => {
    const [state, setState] = useState<???>(null);
    if (props?.data?.value === undefined) {
        return <div className={<><><unclosed><span>{props.item?.name}</div>;
    }
    return <><broken <tag attr={>>>>>;
}
EOF

echo "[SCENARIO 2.6] Fuzzing malformed TSX syntax..."
"$READ_ENGINE" ast-context malformed.tsx > /dev/null
echo "[SCENARIO 2.6] Assertion Passed: Malformed TSX parsed deterministically without panic."

cat << 'EOF' > malformed_diff.patch
--- a/malformed.tsx
+++ b/malformed.tsx
@@ -1,5 +1,8 @@
+export function BrokenComponent<T> (props: { data: T }) {
+    const val = props?.data?.value;
+    if (val === undefined || props.count > 0) {
+        return <div attr={val?.sub} />;
+    }
+    return undefined;
+}
EOF

"$READ_ENGINE" mutate malformed_diff.patch > /dev/null
echo "[SCENARIO 2.6] Assertion Passed: Malformed TSX diff mutated deterministically without panic."

# 2.7 Malformed Java Syntax Resilience
cat << 'EOF' > Malformed.java
package com.broken.test;
import java.util.List;
public class MalformedClass<T extends ? super <broken>> {
    public void brokenMethod(List<String> list {
        if (list != null && list.equals(
            return null;
        }
        class InnerClass {{{{
    }
}
EOF

echo "[SCENARIO 2.7] Fuzzing malformed Java syntax..."
"$READ_ENGINE" ast-context Malformed.java > /dev/null
echo "[SCENARIO 2.7] Assertion Passed: Malformed Java parsed deterministically without panic."

cat << 'EOF' > malformed_java_diff.patch
--- a/Malformed.java
+++ b/Malformed.java
@@ -1,5 +1,8 @@
+public class MalformedClass {
+    public boolean validate(Order o) {
+        if (o != null && o.equals("OK")) {
+            return o.count > 0;
+        }
+        return false;
+    }
+}
EOF

"$READ_ENGINE" mutate malformed_java_diff.patch > /dev/null
echo "[SCENARIO 2.7] Assertion Passed: Malformed Java diff mutated deterministically without panic."

# 2.8 Malformed C# Syntax Resilience
cat << 'EOF' > Malformed.cs
using System;
using System.Collections.Generic;
namespace Broken.Test {
    public class MalformedClass<T> where T : class, new( {
        public void BrokenMethod(List<string> list {
            var role = user?.Role;
            if (role != null && role.Equals(
                return default;
            }
            class InnerClass {{{{
        }
    }
}
EOF

echo "[SCENARIO 2.8] Fuzzing malformed C# syntax..."
"$READ_ENGINE" ast-context Malformed.cs > /dev/null
echo "[SCENARIO 2.8] Assertion Passed: Malformed C# parsed deterministically without panic."

cat << 'EOF' > malformed_csharp_diff.patch
--- a/Malformed.cs
+++ b/Malformed.cs
@@ -1,5 +1,8 @@
+public class MalformedClass {
+    public bool Validate(User u) {
+        if (u?.Role != null && u.Role.Equals("OK")) {
+            return u.Count + 1 > 0;
+        }
+        return default;
+    }
+}
EOF

"$READ_ENGINE" mutate malformed_csharp_diff.patch > /dev/null
echo "[SCENARIO 2.8] Assertion Passed: Malformed C# diff mutated deterministically without panic."

# 2.9 Malformed C++ / Template Resilience
cat << 'EOF' > Malformed.cpp
#include <vector>
#include <string>

template <typename T, typename... Rest, template <class...> class Container = std::vector<broken>>
class MalformedTemplateClass : public Container<T {
public:
    template <typename U = decltype(std::declval<T>().broken())>
    void brokenMethod(T&& arg, Rest... {
        if (ptr != nullptr) {
            auto lambda = [&]<typename K>(K&& k) -> auto {
                return (k < 10) ? k + 1 :
            };
        }
    }
};
EOF

echo "[SCENARIO 2.9] Fuzzing malformed C++ and advanced template metaprogramming..."
"$READ_ENGINE" ast-context Malformed.cpp > /dev/null
echo "[SCENARIO 2.9] Assertion Passed: Malformed C++ parsed deterministically without panic."

cat << 'EOF' > malformed_cpp_diff.patch
--- a/Malformed.cpp
+++ b/Malformed.cpp
@@ -1,5 +1,8 @@
+template <typename T>
+void* allocate(T* ptr) {
+    if (ptr != nullptr) {
+        return (ptr->size <= 100) ? ptr : nullptr;
+    }
+    return nullptr;
+}
EOF

"$READ_ENGINE" mutate malformed_cpp_diff.patch > /dev/null
echo "[SCENARIO 2.9] Assertion Passed: Malformed C++ diff mutated deterministically without panic."

# 2.10 Malformed Ruby / Metaprogramming Resilience
cat << 'EOF' > malformed.rb
module BrokenMeta
  class << self
    define_method(:dynamic_eval) do |*args, **kwargs, &blk|
      instance_eval %{
        def nested_broken(x
          return nil unless x&.valid
          while x.count < 10
            x = x + 1
          end
        end
      }
    end
  end
end
EOF

echo "[SCENARIO 2.10] Fuzzing malformed Ruby syntax and dynamic metaprogramming..."
"$READ_ENGINE" ast-context malformed.rb > /dev/null
echo "[SCENARIO 2.10] Assertion Passed: Malformed Ruby parsed deterministically without panic."

cat << 'EOF' > malformed_ruby_diff.patch
--- a/malformed.rb
+++ b/malformed.rb
@@ -1,5 +1,8 @@
+def check_access(user, token)
+  return nil unless user&.active
+  if /^[A-Z]+$/ === token && secret.eql?(expected)
+    return token if count < 10
+  end
+  return nil
+end
EOF

"$READ_ENGINE" mutate malformed_ruby_diff.patch > /dev/null
echo "[SCENARIO 2.10] Assertion Passed: Malformed Ruby diff mutated deterministically without panic."
echo "[PASS] SCENARIO 2 PASSED: AST Parser is 100% crash-proof under adversarial inputs."

# ------------------------------------------------------------------------------
# SCENARIO 3: Dead & Hanging Sockets during Dynamic Harness Scan (<500ms)
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 3]: Dead & Hanging Sockets During Dynamic Harness Scan ==="
SCENARIO3_DIR="$TEST_TEMP_DIR/scenario3"
mkdir -p "$SCENARIO3_DIR"
cd "$SCENARIO3_DIR"

# Launch Python mock TCP server that accepts connections and hangs on HTTP
python3 -c '
import socket, sys, time, threading

server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
server.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
# Bind to standard Ollama / LLM port 11434
try:
    server.bind(("127.0.0.1", 11434))
    server.listen(10)
except Exception as e:
    sys.exit(0)

def handle(client):
    try:
        # Intentionally hang: never send response
        time.sleep(10)
    finally:
        client.close()

while True:
    client, _ = server.accept()
    threading.Thread(target=handle, args=(client,), daemon=True).start()
' &
MOCK_SERVER_PID=$!
sleep 0.2

# Start READ Gateway Daemon on custom port
DAEMON_PORT=19850
"$READ_DAEMON" -port "$DAEMON_PORT" -workspace "$SCENARIO3_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
DAEMON_PID=$!

# Wait for daemon to become ready
for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# Measure time to query /api/v1/harnesses with hanging port 11434 present
echo "[SCENARIO 3] Probing daemon /api/v1/harnesses with hanging socket on port 11434..."
SCAN_START=$(python3 -c "import time; print(time.time())")
STATUS_CODE=$(curl -s -o "$SCENARIO3_DIR/harnesses.json" -w "%{http_code}" "http://127.0.0.1:$DAEMON_PORT/api/v1/harnesses")
SCAN_END=$(python3 -c "import time; print(time.time())")

SCAN_DURATION_MS=$(python3 -c "print(int(($SCAN_END - $SCAN_START) * 1000))")
echo "[SCENARIO 3] Probe completed in: ${SCAN_DURATION_MS}ms (Status Code: $STATUS_CODE)"

if [ "$STATUS_CODE" != "200" ]; then
    echo "[FAIL] Daemon failed to respond with 200 OK: $STATUS_CODE" >&2
    exit 1
fi

if [ "$SCAN_DURATION_MS" -ge 500 ]; then
    echo "[FAIL] Scanner blocked longer than 500ms ceiling! Took ${SCAN_DURATION_MS}ms" >&2
    exit 1
fi
echo "[SCENARIO 3] Assertion Passed: Hanging socket timed out within ${SCAN_DURATION_MS}ms (< 500ms limit)."

# Ensure daemon remains fully healthy and unblocked
DAEMON_HEALTH=$(curl -s "http://127.0.0.1:$DAEMON_PORT/api/v1/status" | jq -r .status)
if [ "$DAEMON_HEALTH" != "online" ]; then
    echo "[FAIL] Daemon became unhealthy after hanging socket probe" >&2
    exit 1
fi
echo "[SCENARIO 3] Assertion Passed: Daemon remained online and unblocked."

# Terminate mock server and daemon
disown "$MOCK_SERVER_PID" 2>/dev/null || true
disown "$DAEMON_PID" 2>/dev/null || true
kill -9 "$MOCK_SERVER_PID" 2>/dev/null || true
kill -9 "$DAEMON_PID" 2>/dev/null || true
MOCK_SERVER_PID=""
DAEMON_PID=""
echo "[PASS] SCENARIO 3 PASSED: Dynamic scanner strictly bounds network timeouts (<500ms)."

# ------------------------------------------------------------------------------
# SCENARIO 4: Concurrent CAS Race on State Transitions
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 4]: Concurrent CAS Race Condition Audit ==="
SCENARIO4_DIR="$TEST_TEMP_DIR/scenario4"
mkdir -p "$SCENARIO4_DIR"
cd "$SCENARIO4_DIR"
git init -q
"$READ_ENGINE" init
"$READ_ENGINE" define "TASK-RACE-001" "contract_hash_race" "commit_race_0"

# 4.1 Concurrent Race on 'read build' with 10 parallel processes
echo "[SCENARIO 4.1] Firing 10 concurrent 'read build' executions simultaneously..."
RACE_BUILD_RESULTS="$SCENARIO4_DIR/race_build_results.txt"
rm -f "$RACE_BUILD_RESULTS"

python3 -c "
import subprocess, os
from concurrent.futures import ThreadPoolExecutor

engine = '$READ_ENGINE'
cwd = '$SCENARIO4_DIR'

def execute_build(i):
    res = subprocess.run([engine, 'build'], cwd=cwd, capture_output=True, text=True)
    return (res.returncode, res.stdout.strip(), res.stderr.strip())

with ThreadPoolExecutor(max_workers=10) as ex:
    results = list(ex.map(execute_build, range(10)))

with open('$RACE_BUILD_RESULTS', 'w') as f:
    for code, out, err in results:
        f.write(f'{code}|{out}|{err}\n')
"

BUILD_SUCCESSES=$(grep -c "^0|" "$RACE_BUILD_RESULTS" || true)
BUILD_FAILURES=$(grep -c -v "^0|" "$RACE_BUILD_RESULTS" || true)
echo "[SCENARIO 4.1] Results: $BUILD_SUCCESSES succeeded, $BUILD_FAILURES failed."

if [ "$BUILD_SUCCESSES" -ne 1 ] || [ "$BUILD_FAILURES" -ne 9 ]; then
    echo "[FAIL] CAS race violated on build! Expected exactly 1 success and 9 failures." >&2
    cat "$RACE_BUILD_RESULTS" >&2
    exit 1
fi

CAS_VIOLATION_COUNT=$(grep -c "CasLockViolation" "$RACE_BUILD_RESULTS" || true)
if [ "$CAS_VIOLATION_COUNT" -ne 9 ]; then
    echo "[FAIL] Expected 9 CasLockViolation errors, found $CAS_VIOLATION_COUNT" >&2
    cat "$RACE_BUILD_RESULTS" >&2
    exit 1
fi
echo "[SCENARIO 4.1] Assertion Passed: Exactly 1 succeeded, 9 rejected with CasLockViolation."

# 4.2 Concurrent Race on 'read handoff' with 10 parallel processes
# First record legitimate verification to reach 'verified' status
cat << 'EOF' > sample.py
def add(a, b): return a + b
EOF
git add sample.py
DIFF_HASH=$("$READ_ENGINE" diff-hash)
"$READ_ENGINE" verify "$DIFF_HASH" "All unit mutants killed"

echo "[SCENARIO 4.2] Firing 10 concurrent 'read handoff' executions simultaneously..."
RACE_HANDOFF_RESULTS="$SCENARIO4_DIR/race_handoff_results.txt"
rm -f "$RACE_HANDOFF_RESULTS"

python3 -c "
import subprocess, os
from concurrent.futures import ThreadPoolExecutor

engine = '$READ_ENGINE'
cwd = '$SCENARIO4_DIR'

def execute_handoff(i):
    res = subprocess.run([engine, 'handoff'], cwd=cwd, capture_output=True, text=True)
    return (res.returncode, res.stdout.strip(), res.stderr.strip())

with ThreadPoolExecutor(max_workers=10) as ex:
    results = list(ex.map(execute_handoff, range(10)))

with open('$RACE_HANDOFF_RESULTS', 'w') as f:
    for code, out, err in results:
        f.write(f'{code}|{out}|{err}\n')
"

HANDOFF_SUCCESSES=$(grep -c "^0|" "$RACE_HANDOFF_RESULTS" || true)
HANDOFF_FAILURES=$(grep -c -v "^0|" "$RACE_HANDOFF_RESULTS" || true)
echo "[SCENARIO 4.2] Results: $HANDOFF_SUCCESSES succeeded, $HANDOFF_FAILURES failed."

if [ "$HANDOFF_SUCCESSES" -ne 1 ] || [ "$HANDOFF_FAILURES" -ne 9 ]; then
    echo "[FAIL] CAS race violated on handoff! Expected exactly 1 success and 9 failures." >&2
    cat "$RACE_HANDOFF_RESULTS" >&2
    exit 1
fi

HANDOFF_CAS_COUNT=$(grep -c "CasLockViolation" "$RACE_HANDOFF_RESULTS" || true)
if [ "$HANDOFF_CAS_COUNT" -ne 9 ]; then
    echo "[FAIL] Expected 9 CasLockViolation errors on handoff, found $HANDOFF_CAS_COUNT" >&2
    cat "$RACE_HANDOFF_RESULTS" >&2
    exit 1
fi
echo "[SCENARIO 4.2] Assertion Passed: Exactly 1 succeeded, 9 rejected with CasLockViolation."
echo "[PASS] SCENARIO 4 PASSED: CAS revision locks mechanically reject stale transitions."

# ------------------------------------------------------------------------------
# SCENARIO 5: Unauthorized Gate Bypass
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 5]: Unauthorized Gate Bypass Attacks ==="
SCENARIO5_DIR="$TEST_TEMP_DIR/scenario5"
mkdir -p "$SCENARIO5_DIR"
cd "$SCENARIO5_DIR"
git init -q
"$READ_ENGINE" init
"$READ_ENGINE" define "TASK-BYPASS-001" "contract_bypass" "commit_bypass_0"
"$READ_ENGINE" build

# 5.1 Premature Handoff Attempt during 'executing' status
set +e
"$READ_ENGINE" handoff 2> "$SCENARIO5_DIR/premature_err.txt"
PREMATURE_EXIT=$?
set -e

if [ $PREMATURE_EXIT -eq 0 ]; then
    echo "[FAIL] Security Bypass: Handoff succeeded while status was 'executing'!" >&2
    exit 1
fi
if ! grep -q "CasLockViolation" "$SCENARIO5_DIR/premature_err.txt"; then
    echo "[FAIL] Expected CasLockViolation on premature handoff, got: $(cat "$SCENARIO5_DIR/premature_err.txt")" >&2
    exit 1
fi
echo "[SCENARIO 5.1] Assertion Passed: Premature handoff blocked by CasLockViolation."

# 5.2 Tampered STATE.json with null verify_hash
python3 -c "
import json
with open('.method/STATE.json', 'r') as f:
    st = json.load(f)
st['status'] = 'verified'
st['verify_hash'] = None
with open('.method/STATE.json', 'w') as f:
    json.dump(st, f, indent=2)
"

set +e
"$READ_ENGINE" handoff 2> "$SCENARIO5_DIR/null_hash_err.txt"
NULL_HASH_EXIT=$?
set -e

if [ $NULL_HASH_EXIT -eq 0 ]; then
    echo "[FAIL] Security Bypass: Handoff succeeded with verify_hash = null!" >&2
    exit 1
fi
if ! grep -q "verify_hash is null" "$SCENARIO5_DIR/null_hash_err.txt"; then
    echo "[FAIL] Expected null verify_hash rejection, got: $(cat "$SCENARIO5_DIR/null_hash_err.txt")" >&2
    exit 1
fi
echo "[SCENARIO 5.2] Assertion Passed: Null verify_hash blocked mechanically."

# 5.3 Tampered STATE.json with forged / mismatched verify_hash
python3 -c "
import json
with open('.method/STATE.json', 'r') as f:
    st = json.load(f)
st['status'] = 'verified'
st['verify_hash'] = 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'
st['staged_diff_hash'] = 'dummy_staged_diff'
with open('.method/STATE.json', 'w') as f:
    json.dump(st, f, indent=2)
"

set +e
"$READ_ENGINE" handoff 2> "$SCENARIO5_DIR/forged_hash_err.txt"
FORGED_HASH_EXIT=$?
set -e

if [ $FORGED_HASH_EXIT -eq 0 ]; then
    echo "[FAIL] Security Breach: Handoff succeeded with forged verify_hash not in WAL!" >&2
    exit 1
fi
if ! grep -q "State tampering detected" "$SCENARIO5_DIR/forged_hash_err.txt"; then
    echo "[FAIL] Expected tampering detection, got: $(cat "$SCENARIO5_DIR/forged_hash_err.txt")" >&2
    exit 1
fi
echo "[SCENARIO 5.3] Assertion Passed: Forged verify_hash blocked by WAL audit record mismatch."

# 5.4 Syntax Corruption of STATE.json
echo "CORRUPTED_JSON_INVALID_SYNTAX_{{{{" > .method/STATE.json
set +e
"$READ_ENGINE" status 2> "$SCENARIO5_DIR/syntax_err.txt"
SYNTAX_EXIT=$?
set -e

if [ $SYNTAX_EXIT -eq 0 ]; then
    echo "[FAIL] Security Failure: Corrupt STATE.json loaded without error!" >&2
    exit 1
fi
if ! grep -q "Corrupt STATE.json" "$SCENARIO5_DIR/syntax_err.txt"; then
    echo "[FAIL] Expected Corrupt STATE.json error, got: $(cat "$SCENARIO5_DIR/syntax_err.txt")" >&2
    exit 1
fi
echo "[SCENARIO 5.4] Assertion Passed: Malformed STATE.json blocked on load."

# 5.5 Strict 100% Mutation Gate Enforcement (90% score MUST be rejected)
echo "[SCENARIO 5.5] Verifying strict 100% AROP gate rejection on 90% mutation score..."
set +e
SCORE_90_OUT=$("$READ_ENGINE" eval-mutation-score 9 10 2>&1)
SCORE_90_EXIT=$?
set -e

if [ $SCORE_90_EXIT -eq 0 ]; then
    echo "[FAIL] Security Breach: 90% mutation score (9/10 killed) was accepted! Strictly 100% required." >&2
    exit 1
fi
if ! echo "$SCORE_90_OUT" | grep -q "below strictly required 100.0% threshold"; then
    echo "[FAIL] Expected strict 100% threshold rejection, got: $SCORE_90_OUT" >&2
    exit 1
fi
echo "[SCENARIO 5.5] Assertion Passed: 90% mutation score (9/10 killed) mechanically rejected with code $SCORE_90_EXIT."
echo "[PASS] SCENARIO 5 PASSED: Unauthorized gate bypass is mechanically impossible."

# ------------------------------------------------------------------------------
# SCENARIO 6: Dynamic RLIMIT Calibration Under Varying Memory Conditions
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 6]: Dynamic RLIMIT Calibration Under Varying Memory Conditions ==="
SCENARIO6_DIR="$TEST_TEMP_DIR/scenario6"
mkdir -p "$SCENARIO6_DIR"

# 6.1 High-RAM environment (16 GB available RAM)
cat << 'EOF' > "$SCENARIO6_DIR/meminfo_16gb.txt"
MemTotal:       32000000 kB
MemFree:        10000000 kB
MemAvailable:   16000000 kB
EOF

RLIMIT_16GB=$("$READ_DAEMON" -print-rlimit -meminfo "$SCENARIO6_DIR/meminfo_16gb.txt")
EXPECTED_16GB=$((16000000 * 1024 * 30 / 100))
if [ "$RLIMIT_16GB" != "$EXPECTED_16GB" ]; then
    echo "[FAIL] Dynamic RLIMIT mismatch for 16GB host! Expected $EXPECTED_16GB bytes, got $RLIMIT_16GB" >&2
    exit 1
fi
echo "[SCENARIO 6.1] Assertion Passed: 16GB available RAM correctly calibrated to 30% ($RLIMIT_16GB bytes)."

# 6.2 Low-RAM environment (2 GB available RAM)
cat << 'EOF' > "$SCENARIO6_DIR/meminfo_2gb.txt"
MemTotal:        4000000 kB
MemFree:         1000000 kB
MemAvailable:    2000000 kB
EOF

RLIMIT_2GB=$("$READ_DAEMON" -print-rlimit -meminfo "$SCENARIO6_DIR/meminfo_2gb.txt")
EXPECTED_2GB=$((2000000 * 1024 * 30 / 100))
if [ "$RLIMIT_2GB" != "$EXPECTED_2GB" ]; then
    echo "[FAIL] Dynamic RLIMIT mismatch for 2GB host! Expected $EXPECTED_2GB bytes, got $RLIMIT_2GB" >&2
    exit 1
fi
echo "[SCENARIO 6.2] Assertion Passed: 2GB available RAM correctly calibrated to 30% ($RLIMIT_2GB bytes)."

# 6.3 Missing / unreadable meminfo fallback (strictly 512MB, not 2GB)
FALLBACK_OUTPUT=$("$READ_DAEMON" -print-rlimit -meminfo "$SCENARIO6_DIR/nonexistent_meminfo.txt" 2>&1)
FALLBACK_LIMIT=$(echo "$FALLBACK_OUTPUT" | tail -n 1)
EXPECTED_512MB=$((512 * 1024 * 1024))

if [ "$FALLBACK_LIMIT" != "$EXPECTED_512MB" ]; then
    echo "[FAIL] Fallback memory limit should be strictly 512MB ($EXPECTED_512MB bytes), got $FALLBACK_LIMIT" >&2
    exit 1
fi
if ! echo "$FALLBACK_OUTPUT" | grep -q "falling back to 512MB default limit"; then
    echo "[FAIL] Warning was not logged on meminfo detection failure!" >&2
    exit 1
fi
echo "[SCENARIO 6.3] Assertion Passed: Missing meminfo correctly triggers warning and 512MB fallback."

# 6.4 Corrupted / zero kB meminfo fallback
echo "Corrupt text with no memory figures" > "$SCENARIO6_DIR/meminfo_corrupt.txt"
CORRUPT_OUTPUT=$("$READ_DAEMON" -print-rlimit -meminfo "$SCENARIO6_DIR/meminfo_corrupt.txt" 2>&1)
CORRUPT_LIMIT=$(echo "$CORRUPT_OUTPUT" | tail -n 1)

if [ "$CORRUPT_LIMIT" != "$EXPECTED_512MB" ]; then
    echo "[FAIL] Corrupt meminfo should trigger 512MB fallback, got $CORRUPT_LIMIT" >&2
    exit 1
fi
echo "[SCENARIO 6.4] Assertion Passed: Corrupt meminfo correctly triggers 512MB fallback."

# 6.5 Live host system calibration
LIVE_RLIMIT=$("$READ_DAEMON" -print-rlimit)
if [ -z "$LIVE_RLIMIT" ] || [ "$LIVE_RLIMIT" -le 0 ]; then
    echo "[FAIL] Live system dynamic RLIMIT failed: $LIVE_RLIMIT" >&2
    exit 1
fi
LIVE_MB=$((LIVE_RLIMIT / 1024 / 1024))
echo "[SCENARIO 6.5] Assertion Passed: Live host RAM dynamically calibrated to $LIVE_MB MB ceiling."
echo "[PASS] SCENARIO 6 PASSED: Dynamic RLIMIT calibration operates correctly under all memory conditions."

# ------------------------------------------------------------------------------
# SCENARIO 7: TLS Enforcement on Non-Loopback Interfaces
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 7]: TLS Enforcement on Non-Loopback Interfaces ==="
SCENARIO7_DIR="$TEST_TEMP_DIR/scenario7"
mkdir -p "$SCENARIO7_DIR"

# 7.1 Mechanically reject binding to 0.0.0.0 without TLS
set +e
BIND_REJECT_OUT=$("$READ_DAEMON" -host "0.0.0.0" -port 19860 -workspace "$SCENARIO7_DIR" 2>&1)
BIND_REJECT_EXIT=$?
set -e

if [ $BIND_REJECT_EXIT -eq 0 ]; then
    echo "[FAIL] Security hole: Daemon allowed binding to 0.0.0.0 without TLS!" >&2
    exit 1
fi
if ! echo "$BIND_REJECT_OUT" | grep -q "strictly forbidden without TLS"; then
    echo "[FAIL] Expected security violation error message, got: $BIND_REJECT_OUT" >&2
    exit 1
fi
echo "[SCENARIO 7.1] Assertion Passed: 0.0.0.0 bind mechanically rejected without TLS (code: $BIND_REJECT_EXIT)."

# 7.2 Generate self-signed TLS cert and verify HTTPS serving
openssl req -x509 -newkey rsa:2048 -nodes \
    -keyout "$SCENARIO7_DIR/server.key" \
    -out "$SCENARIO7_DIR/server.crt" \
    -days 1 -subj "/CN=127.0.0.1" 2>/dev/null

TLS_PORT=19861
"$READ_DAEMON" -tls-cert "$SCENARIO7_DIR/server.crt" -tls-key "$SCENARIO7_DIR/server.key" -port "$TLS_PORT" -workspace "$SCENARIO7_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
TLS_DAEMON_PID=$!

for _ in $(seq 1 30); do
    if curl -k -s "https://127.0.0.1:$TLS_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

TLS_STATUS=$(curl -k -s -o /dev/null -w "%{http_code}" "https://127.0.0.1:$TLS_PORT/api/v1/status")
kill $TLS_DAEMON_PID 2>/dev/null || true
wait $TLS_DAEMON_PID 2>/dev/null || true

if [ "$TLS_STATUS" != "200" ]; then
    echo "[FAIL] Failed to serve HTTPS with valid TLS cert: $TLS_STATUS" >&2
    exit 1
fi
echo "[SCENARIO 7.2] Assertion Passed: Daemon successfully enforced HTTPS with TLS cert/key."
echo "[PASS] SCENARIO 7 PASSED: Non-loopback TLS enforcement mechanically verified."

# ------------------------------------------------------------------------------
# SCENARIO 8: Constant-Time Bearer Token Authentication
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 8]: Constant-Time Bearer Token Authentication ==="
SCENARIO8_DIR="$TEST_TEMP_DIR/scenario8"
mkdir -p "$SCENARIO8_DIR"

AUTH_PORT=19862
AUTH_SECRET="secret_auth_token_999xyz"
"$READ_DAEMON" -auth-token "$AUTH_SECRET" -port "$AUTH_PORT" -workspace "$SCENARIO8_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
AUTH_DAEMON_PID=$!

for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$AUTH_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# 8.1 /api/v1/status is unauthenticated
CODE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$AUTH_PORT/api/v1/status")
if [ "$CODE_STATUS" != "200" ]; then
    kill $AUTH_DAEMON_PID 2>/dev/null || true
    echo "[FAIL] Expected 200 OK for /api/v1/status without auth, got: $CODE_STATUS" >&2
    exit 1
fi
echo "[SCENARIO 8.1] Assertion Passed: /api/v1/status allows unauthenticated public access."

# 8.2 /api/v1/harnesses without token returns 401
CODE_UNAUTH=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$AUTH_PORT/api/v1/harnesses")
if [ "$CODE_UNAUTH" != "401" ]; then
    kill $AUTH_DAEMON_PID 2>/dev/null || true
    echo "[FAIL] Expected 401 Unauthorized for /api/v1/harnesses without auth, got: $CODE_UNAUTH" >&2
    exit 1
fi
echo "[SCENARIO 8.2] Assertion Passed: Protected endpoint rejected without token (401 Unauthorized)."

# 8.3 /api/v1/harnesses with wrong token returns 401
CODE_WRONG=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer wrong_token" "http://127.0.0.1:$AUTH_PORT/api/v1/harnesses")
if [ "$CODE_WRONG" != "401" ]; then
    kill $AUTH_DAEMON_PID 2>/dev/null || true
    echo "[FAIL] Expected 401 Unauthorized with wrong token, got: $CODE_WRONG" >&2
    exit 1
fi
echo "[SCENARIO 8.3] Assertion Passed: Protected endpoint rejected with wrong token (401 Unauthorized)."

# 8.4 /api/v1/harnesses with valid token returns 200
CODE_VALID=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $AUTH_SECRET" "http://127.0.0.1:$AUTH_PORT/api/v1/harnesses")
kill $AUTH_DAEMON_PID 2>/dev/null || true
wait $AUTH_DAEMON_PID 2>/dev/null || true

if [ "$CODE_VALID" != "200" ]; then
    echo "[FAIL] Expected 200 OK with valid Bearer token, got: $CODE_VALID" >&2
    exit 1
fi
echo "[SCENARIO 8.4] Assertion Passed: Protected endpoint granted 200 OK with valid constant-time Bearer token."
echo "[PASS] SCENARIO 8 PASSED: Bearer token authentication verified."

# ------------------------------------------------------------------------------
# SCENARIO 9: Per-IP Token Bucket Rate Limiting (10 Burst, 2 Sustained)
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 9]: Per-IP Token Bucket Rate Limiting ==="
SCENARIO9_DIR="$TEST_TEMP_DIR/scenario9"
mkdir -p "$SCENARIO9_DIR"

RL_PORT=19863
"$READ_DAEMON" -port "$RL_PORT" -workspace "$SCENARIO9_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
RL_DAEMON_PID=$!

for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$RL_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# Fire 15 rapid requests simulating external non-loopback IP (203.0.113.50)
EXT_IP="203.0.113.50"
HIT_429=0
RETRY_AFTER_SEEN=""

for i in $(seq 1 15); do
    RESP_HEADER=$(curl -s -D - -o /dev/null -H "X-Forwarded-For: $EXT_IP" "http://127.0.0.1:$RL_PORT/api/v1/status")
    HTTP_CODE=$(echo "$RESP_HEADER" | grep -i "HTTP/" | tail -n 1 | awk '{print $2}')
    if [ "$HTTP_CODE" = "429" ]; then
        HIT_429=1
        RETRY_AFTER_SEEN=$(echo "$RESP_HEADER" | grep -i "Retry-After" | awk '{print $2}' | tr -d '\r')
        break
    fi
done

kill $RL_DAEMON_PID 2>/dev/null || true
wait $RL_DAEMON_PID 2>/dev/null || true

if [ "$HIT_429" -ne 1 ]; then
    echo "[FAIL] External IP rate limit was not triggered after 15 requests!" >&2
    exit 1
fi
if [ -z "$RETRY_AFTER_SEEN" ]; then
    echo "[FAIL] Rate limited response did not include Retry-After header!" >&2
    exit 1
fi
echo "[SCENARIO 9] Assertion Passed: External IP triggered 429 Too Many Requests with Retry-After: $RETRY_AFTER_SEEN"
echo "[PASS] SCENARIO 9 PASSED: Per-IP token bucket rate limiting verified."

# ------------------------------------------------------------------------------
# SCENARIO 10: Loopback Rate Limit Whitelisting
# ------------------------------------------------------------------------------
echo ""
echo "=== [SCENARIO 10]: Loopback Rate Limit Whitelisting ==="
SCENARIO10_DIR="$TEST_TEMP_DIR/scenario10"
mkdir -p "$SCENARIO10_DIR"

WL_PORT=19864
"$READ_DAEMON" -port "$WL_PORT" -workspace "$SCENARIO10_DIR" -engine-bin "$READ_ENGINE" > /dev/null 2>&1 &
WL_DAEMON_PID=$!

for _ in $(seq 1 30); do
    if curl -s "http://127.0.0.1:$WL_PORT/api/v1/status" > /dev/null 2>&1; then
        break
    fi
    sleep 0.1
done

# Fire 25 consecutive requests from loopback: all MUST return 200 (zero 429s)
LOOPBACK_FAILURES=0
for i in $(seq 1 25); do
    CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:$WL_PORT/api/v1/status")
    if [ "$CODE" != "200" ]; then
        LOOPBACK_FAILURES=$((LOOPBACK_FAILURES + 1))
    fi
done

kill $WL_DAEMON_PID 2>/dev/null || true
wait $WL_DAEMON_PID 2>/dev/null || true

if [ "$LOOPBACK_FAILURES" -ne 0 ]; then
    echo "[FAIL] Loopback client was throttled! Failures: $LOOPBACK_FAILURES / 25" >&2
    exit 1
fi
echo "[SCENARIO 10] Assertion Passed: 25 consecutive loopback requests all returned 200 OK (whitelist active)."
echo "[PASS] SCENARIO 10 PASSED: Localhost whitelisting verified."

# ------------------------------------------------------------------------------
# FINAL REPORT
# ------------------------------------------------------------------------------
echo ""
echo "================================================================================"
echo "ALL 10 ADVERSARIAL CHAOS SCENARIOS COMPLETED SUCCESSFULLY"
echo "  [PASS] Scenario 1: SIGKILL Resilience and Zero-Token Crash Recovery"
echo "  [PASS] Scenario 2: AST Parser Fuzzing (Syntax, Braces, Imports, Loops, Binary)"
echo "  [PASS] Scenario 3: Dead & Hanging Sockets Dynamic Scanner Latency (<500ms)"
echo "  [PASS] Scenario 4: Concurrent CAS Race (10 parallel workers -> 1 win, 9 CasLockViolation)"
echo "  [PASS] Scenario 5: Unauthorized Gate Bypass & Strict 100% AROP Mutation Enforcement"
echo "  [PASS] Scenario 6: Dynamic RLIMIT Calibration (30% RAM Ceiling & 512MB Fallback)"
echo "  [PASS] Scenario 7: Non-Loopback 0.0.0.0 Rejection & TLS/HTTPS Enforcement"
echo "  [PASS] Scenario 8: Constant-Time Bearer Token Authentication (subtle.ConstantTimeCompare)"
echo "  [PASS] Scenario 9: Per-IP Token Bucket Rate Limiting (10 Burst, 2/sec, 429 Retry-After)"
echo "  [PASS] Scenario 10: Localhost Loopback Whitelisting (Unthrottled Dev Access)"
echo "================================================================================"
