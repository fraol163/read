#!/usr/bin/env bash
set -e

echo "=== TEST: Outside Repository AROP Mutation Verification Gate ==="
echo "Target Codebase: /home/renan/gate8_reconstruction"
echo "Date: $(date --iso-8601=seconds)"

ENGINE="/home/renan/Downloads/me/methodology_session/read/bin/read-engine"
OUTSIDE_SRC="/home/renan/gate8_reconstruction"
SANDBOX=$(mktemp -d /tmp/read-outside-mutation-XXXXXX)

cleanup() {
    rm -rf "$SANDBOX"
}
trap cleanup EXIT

# Copy target outside files into sandbox
mkdir -p "$SANDBOX/backend" "$SANDBOX/tests"
cp "$OUTSIDE_SRC/backend/version_catalog.py" "$SANDBOX/backend/"
cp "$OUTSIDE_SRC/tests/test_version_catalog.py" "$SANDBOX/tests/"

cd "$SANDBOX"
git init -q
git config user.name "Outside Auditor"
git config user.email "auditor@outside.org"
git add -A
git commit -qm "Baseline commit from outside repository"

echo ""
echo "[STEP 1] Running baseline test suite on unmodified outside code..."
python3 -m unittest tests/test_version_catalog.py
echo "[PASS] Baseline tests pass on outside repository."

echo ""
echo "[STEP 2] Introducing realistic feature addition to backend/version_catalog.py..."
# Add input validation condition to update_toml_version
python3 -c "
with open('backend/version_catalog.py', 'r') as f:
    lines = f.readlines()
for i, line in enumerate(lines):
    if 'def update_toml_version' in line:
        lines.insert(i + 1, '    if len(var_name) == 0:\n        return False\n')
        break
with open('backend/version_catalog.py', 'w') as f:
    f.writelines(lines)
"

git add backend/version_catalog.py

echo ""
echo "[STEP 3] Generating AROP diff mutants via read-engine mutate..."
"$ENGINE" mutate --json > mutants.json
MUTANT_COUNT=$(python3 -c "
import json
m = json.load(open('mutants.json'))
print(len(m))
")
echo "Generated $MUTANT_COUNT targeted mutants on outside diff:"
python3 -c "
import json
m = json.load(open('mutants.json'))
for idx, mut in enumerate(m):
    print(f'  Mutant #{idx+1}: {mut[\"operator\"]} at L{mut[\"line_number\"]} -> {mut[\"mutated_line\"].strip()}')
"

echo ""
echo "[STEP 4] Evaluating each mutant against tests (Expect 100% kill on good suite)..."
python3 -c "
import json, subprocess, sys

with open('mutants.json') as f:
    mutants = json.load(f)
with open('backend/version_catalog.py') as f:
    orig_code = f.read()

killed = 0
for idx, m in enumerate(mutants):
    lines = orig_code.splitlines(keepends=True)
    l_idx = m['line_number'] - 1
    lines[l_idx] = m['mutated_line'] + '\n'
    with open('backend/version_catalog.py', 'w') as f:
        f.writelines(lines)
    
    res = subprocess.run(['python3', '-m', 'unittest', 'tests/test_version_catalog.py'], capture_output=True)
    if res.returncode != 0:
        killed += 1
        print(f'  Mutant #{idx+1} ({m[\"operator\"]}): KILLED (test suite failed as expected)')
    else:
        print(f'  Mutant #{idx+1} ({m[\"operator\"]}): SURVIVED (test suite failed to detect bug)')

with open('backend/version_catalog.py', 'w') as f:
    f.write(orig_code)

score = (killed / len(mutants)) * 100.0 if mutants else 100.0
print(f'Final Mutation Score: {score:.1f}% ({killed}/{len(mutants)} killed)')
if score < 100.0:
    sys.exit(1)
"
echo "[PASS] Outside repository mutation verification gate passed at strictly 100% kill rate."

echo ""
echo "=== OUTSIDE REPOSITORY MUTATION TEST COMPLETED SUCCESSFULLY ==="
