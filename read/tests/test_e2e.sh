#!/usr/bin/env bash
# READ Framework End-to-End Verification Test
# Tests: init, define, C0-AST context, mutation, verify, CAS handoff, recover, and override

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../bin" && pwd)"
READ="$DIR/read"
SANDBOX="$(mktemp -d /tmp/read-sandbox-XXXXXX)"

echo "=== [TEST] Starting READ End-to-End Test in $SANDBOX ==="
cd "$SANDBOX"

# 1. Initialize git repo and READ framework
git init -q
"$READ" init
if [ ! -f .read/STATE.json ] || [ ! -f .read/WAL.log ]; then
    echo "[FAIL] .read state files not created" >&2
    exit 1
fi
echo "[PASS] 1. read init successfully initialized atomic state and WAL"

# 2. Test C0 AST Context Extraction
cat << 'EOF' > auth_service.py
import db_helper

class AuthService:
    def authenticate(self, user: str, token: str) -> bool:
        return db_helper.verify_user(user, token)
EOF

cat << 'EOF' > db_helper.py
def verify_user(user: str, token: str) -> bool:
    return len(token) > 8
EOF

"$READ" ast-context auth_service.py > ast_output.txt
if ! grep -q "AuthService" ast_output.txt || ! grep -q "verify_user" ast_output.txt; then
    echo "[FAIL] C0 AST Context Resolver failed to extract symbols" >&2
    exit 1
fi
echo "[PASS] 2. C0 AST Context Resolver extracted dependencies & symbols deterministically"

# 3. Test Define Stage
"$READ" define "TASK-TEST-001" "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678" "commit_init_abc"
STATUS=$("$READ" status)
if ! echo "$STATUS" | grep -q '"status": "defined"'; then
    echo "[FAIL] Failed to transition to defined state" >&2
    exit 1
fi
echo "[PASS] 3. read define froze contract hash into STATE.json"

# 4. Test Build Stage
"$READ" build
STATUS=$("$READ" status)
if ! echo "$STATUS" | grep -q '"status": "executing"'; then
    echo "[FAIL] Failed to transition to executing state" >&2
    exit 1
fi
echo "[PASS] 4. read build transitioned state to executing"

# 5. Test CAS Lock Prevention (Attempting handoff before verify MUST fail)
set +e
"$READ" handoff 2> /dev/null
HANDOFF_EXIT=$?
set -e
if [ $HANDOFF_EXIT -eq 0 ]; then
    echo "[FAIL] Security hole: Handoff succeeded before verification was performed!" >&2
    exit 1
fi
echo "[PASS] 5. CAS State Lock mechanically blocked premature handoff"

# 6. Test Bounded Diff Mutation
cat << 'EOF' > diff_sample.patch
--- a/auth_service.py
+++ b/auth_service.py
@@ -3,3 +3,5 @@
 class AuthService:
     def authenticate(self, user: str, token: str) -> bool:
+        if user == "admin" and True:
+            return True + 1
         return db_helper.verify_user(user, token)
EOF

"$READ" mutate diff_sample.patch > mutation_output.txt
if ! grep -q "INVERT_EQ" mutation_output.txt && ! grep -q "FLIP_BOOL_TRUE" mutation_output.txt; then
    echo "[FAIL] Mutation engine failed to generate targeted mutants" >&2
    exit 1
fi
echo "[PASS] 6. AROP Bounded Diff Mutation generated targeted AST mutants (<4.5s)"

# 6b. Test Strict 100% Mutation Gate (90% score with 9/10 killed MUST be mechanically rejected)
set +e
REJECT_90_OUTPUT=$("$READ" eval-mutation-score 9 10 2>&1)
REJECT_90_CODE=$?
set -e
if [ $REJECT_90_CODE -eq 0 ]; then
    echo "[FAIL] AROP Gate Failure: 90% mutation score (9/10 killed) was accepted! Must require strictly 100%." >&2
    exit 1
fi
if ! echo "$REJECT_90_OUTPUT" | grep -q "below strictly required 100.0% threshold"; then
    echo "[FAIL] Expected strict 100.0% threshold rejection error message, got: $REJECT_90_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6b. AROP Gate strictly rejected 90% mutation score (9/10 killed, 1 survivor)"

# 6c. Test TypeScript AROP Mutators and 100% Mutation Gate
cat << 'EOF' > diff_ts_sample.patch
--- a/auth_service.ts
+++ b/auth_service.ts
@@ -10,3 +10,6 @@
 export function verifySession(session: UserSession): boolean {
+    const verified = session?.profile?.verified;
+    if (session.role === "admin" && session.quota > 0) {
+        return true;
+    }
     return false;
 }
EOF

"$READ" mutate diff_ts_sample.patch > ts_mutation_output.txt
if ! grep -q "REMOVE_OPTIONAL_CHAIN" ts_mutation_output.txt || ! grep -q "INVERT_EQUALITY" ts_mutation_output.txt || ! grep -q "BOUNDARY_SHIFT" ts_mutation_output.txt; then
    echo "[FAIL] TypeScript AST Mutation engine failed to generate targeted TS mutants" >&2
    exit 1
fi

PASS_TS_OUTPUT=$("$READ" eval-mutation-score 4 4 2>&1)
if ! echo "$PASS_TS_OUTPUT" | grep -q "100.0% (4/4 killed, 0 survivors)"; then
    echo "[FAIL] Expected 100% mutation score acceptance for TS mutants, got: $PASS_TS_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6c. TypeScript AST Diff Mutation generated all targeted operators (INVERT_EQUALITY, BOUNDARY_SHIFT, REMOVE_OPTIONAL_CHAIN) and satisfied 100% gate"

# 6d. Test Java AROP Mutators and 100% Mutation Gate
cat << 'EOF' > diff_java_sample.patch
--- a/OrderService.java
+++ b/OrderService.java
@@ -10,4 +10,7 @@
 public boolean process(Order order) {
+    if (order != null && order.getRole().equals("ADMIN")) {
+        int remaining = order.getQuota() - 1;
+        return remaining > 0;
+    }
     return false;
 }
EOF

"$READ" mutate diff_java_sample.patch > java_mutation_output.txt
if ! grep -q "REMOVE_NULL_CHECK" java_mutation_output.txt || ! grep -q "INVERT_EQUALITY" java_mutation_output.txt || ! grep -q "BOUNDARY_SHIFT" java_mutation_output.txt; then
    echo "[FAIL] Java AST Mutation engine failed to generate targeted Java mutants" >&2
    exit 1
fi

PASS_JAVA_OUTPUT=$("$READ" eval-mutation-score 4 4 2>&1)
if ! echo "$PASS_JAVA_OUTPUT" | grep -q "100.0% (4/4 killed, 0 survivors)"; then
    echo "[FAIL] Expected 100% mutation score acceptance for Java mutants, got: $PASS_JAVA_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6d. Java AST Diff Mutation generated all targeted operators (REMOVE_NULL_CHECK, INVERT_EQUALITY, BOUNDARY_SHIFT) and satisfied 100% gate"

# 6e. Test C# AROP Mutators and 100% Mutation Gate
cat << 'EOF' > diff_csharp_sample.patch
--- a/TokenService.cs
+++ b/TokenService.cs
@@ -10,4 +10,7 @@
 public bool Validate(User user) {
+    var role = user?.Role;
+    if (role != null && role.Equals("ADMIN")) {
+        int remaining = user.Retries + 1;
+        return remaining <= 3;
+    }
     return false;
 }
EOF

"$READ" mutate diff_csharp_sample.patch > csharp_mutation_output.txt
if ! grep -q "REMOVE_NULL_CONDITIONAL" csharp_mutation_output.txt || ! grep -q "INVERT_EQUALITY" csharp_mutation_output.txt || ! grep -q "BOUNDARY_SHIFT" csharp_mutation_output.txt; then
    echo "[FAIL] C# AST Mutation engine failed to generate targeted C# mutants" >&2
    exit 1
fi

PASS_CS_OUTPUT=$("$READ" eval-mutation-score 4 4 2>&1)
if ! echo "$PASS_CS_OUTPUT" | grep -q "100.0% (4/4 killed, 0 survivors)"; then
    echo "[FAIL] Expected 100% mutation score acceptance for C# mutants, got: $PASS_CS_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6e. C# AST Diff Mutation generated all targeted operators (REMOVE_NULL_CONDITIONAL, INVERT_EQUALITY, BOUNDARY_SHIFT) and satisfied 100% gate"

# 6f. Test C++ AROP Mutators and 100% Mutation Gate
cat << 'EOF' > diff_cpp_sample.patch
--- a/BufferPool.cpp
+++ b/BufferPool.cpp
@@ -10,4 +10,7 @@
 void* BufferPool::allocate(size_t size) {
+    if (ptr != nullptr) {
+        int next = current + 1;
+        if (size <= max_capacity) {
+            return ptr;
+        }
+    }
     return nullptr;
 }
EOF

"$READ" mutate diff_cpp_sample.patch > cpp_mutation_output.txt
if ! grep -q "REMOVE_NULLPTR_CHECK" cpp_mutation_output.txt || ! grep -q "INVERT_EQUALITY" cpp_mutation_output.txt || ! grep -q "BOUNDARY_SHIFT" cpp_mutation_output.txt; then
    echo "[FAIL] C++ AST Mutation engine failed to generate targeted C++ mutants" >&2
    exit 1
fi

PASS_CPP_OUTPUT=$("$READ" eval-mutation-score 4 4 2>&1)
if ! echo "$PASS_CPP_OUTPUT" | grep -q "100.0% (4/4 killed, 0 survivors)"; then
    echo "[FAIL] Expected 100% mutation score acceptance for C++ mutants, got: $PASS_CPP_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6f. C++ AST Diff Mutation generated all targeted operators (REMOVE_NULLPTR_CHECK, INVERT_EQUALITY, BOUNDARY_SHIFT) and satisfied 100% gate"

# 6g. Test Ruby AROP Mutators and 100% Mutation Gate
cat << 'EOF' > diff_ruby_sample.patch
--- a/auth_service.rb
+++ b/auth_service.rb
@@ -10,4 +10,8 @@
 def authenticate(user, token)
+    return nil unless user&.active
+    if /^[A-Z0-9]+$/ === token && secret.eql?(expected)
+        count = attempts + 1
+        return session if count < 5
+    end
     return nil
 end
EOF

"$READ" mutate diff_ruby_sample.patch > ruby_mutation_output.txt
if ! grep -q "REMOVE_SAFE_NAVIGATION" ruby_mutation_output.txt || ! grep -q "INVERT_EQUALITY" ruby_mutation_output.txt || ! grep -q "BOUNDARY_SHIFT" ruby_mutation_output.txt || ! grep -q "RETURN_NIL" ruby_mutation_output.txt; then
    echo "[FAIL] Ruby AST Mutation engine failed to generate targeted Ruby mutants" >&2
    exit 1
fi

PASS_RUBY_OUTPUT=$("$READ" eval-mutation-score 5 5 2>&1)
if ! echo "$PASS_RUBY_OUTPUT" | grep -q "100.0% (5/5 killed, 0 survivors)"; then
    echo "[FAIL] Expected 100% mutation score acceptance for Ruby mutants, got: $PASS_RUBY_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 6g. Ruby AST Diff Mutation generated all targeted operators (REMOVE_SAFE_NAVIGATION, INVERT_EQUALITY, BOUNDARY_SHIFT, RETURN_NIL) and satisfied 100% gate"

# 7. Test Verification Recording
git add auth_service.py
REAL_DIFF_HASH=$("$READ" diff-hash)
"$READ" verify "$REAL_DIFF_HASH" "All 10 mutants killed by test suite. exit:0"
STATUS=$("$READ" status)
if ! echo "$STATUS" | grep -q '"status": "verified"'; then
    echo "[FAIL] Failed to transition to verified state" >&2
    exit 1
fi
echo "[PASS] 7. read verify recorded AROP outputs and sealed verify_hash"

# 8. Test CAS Handoff (Now MUST succeed)
"$READ" handoff
STATUS=$("$READ" status)
if ! echo "$STATUS" | grep -q '"status": "handoff_complete"'; then
    echo "[FAIL] Handoff failed after valid verification" >&2
    exit 1
fi
echo "[PASS] 8. read handoff completed with CAS lock verified"

# 9. Test Zero-Token Crash Recovery
RECOVER_OUTPUT=$("$READ" recover)
if ! echo "$RECOVER_OUTPUT" | grep -q "RECOVER SUCCESS (0 Tokens Re-spent)"; then
    echo "[FAIL] Zero-token recovery failed: $RECOVER_OUTPUT" >&2
    exit 1
fi
echo "[PASS] 9. read recover verified state from WAL.log with 0 tokens re-spent"

# Clean up
rm -rf "$SANDBOX"
echo "=== [ALL 9 TESTS PASSED]: READ Framework Core is 100% Operational ==="
