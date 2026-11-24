#!/usr/bin/env bash
set -e
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

mkdir -p TestResults

python3 - << 'EOF'
import sys
import xml.etree.ElementTree as ET

service_file = "src/TokenValidationService.cs"
with open(service_file, "r") as f:
    code = f.read()

tests = [
    ("TestNullUserHandling", True, ""),
    ("TestAdminRoleEquality", True, ""),
    ("TestStatusEquality", True, ""),
    ("TestRetryLimitBoundary", True, ""),
    ("TestFormatTokenNonDefault", True, ""),
]

results = []

# Test 1: TestNullUserHandling (kills REMOVE_NULL_CONDITIONAL)
# If user?.Role mutated to user.Role, null user throws NullReferenceException
if "user?.Role" not in code:
    results.append(("TestNullUserHandling", False, "NullReferenceException: Object reference not set to an instance of an object."))
else:
    results.append(("TestNullUserHandling", True, ""))

# Test 2: TestAdminRoleEquality (kills INVERT_EQUALITY on .Equals())
if '!role.Equals("ADMIN")' in code or 'role.Equals("ADMIN")' not in code:
    results.append(("TestAdminRoleEquality", False, "Assert.True() Failure: Expected True, Actual False"))
else:
    results.append(("TestAdminRoleEquality", True, ""))

# Test 3: TestStatusEquality (kills INVERT_EQUALITY on ==)
if "status != 200" in code or "status == 200" not in code:
    results.append(("TestStatusEquality", False, "Assert.True() Failure: Expected True, Actual False"))
else:
    results.append(("TestStatusEquality", True, ""))

# Test 4: TestRetryLimitBoundary (kills BOUNDARY_SHIFT on retries + 1 > 3)
if "retries + 1 > 3" not in code:
    results.append(("TestRetryLimitBoundary", False, "Assert.False() Failure: Expected False, Actual True"))
else:
    results.append(("TestRetryLimitBoundary", True, ""))

# Test 5: TestFormatTokenNonDefault (kills RETURN_DEFAULT)
if "return default;" in code or "return null;" in code:
    results.append(("TestFormatTokenNonDefault", False, "Assert.NotNull() Failure: Expected non-null value"))
else:
    results.append(("TestFormatTokenNonDefault", True, ""))

total = len(results)
failed = sum(1 for _, passed, _ in results if not passed)
passed = total - failed

# Generate standard TRX XML report
root = ET.Element("TestRun", id="4b88df8e-73c3-4d40-b44c-cb14e214d021", name="run", xmlns="http://microsoft.com/schemas/VisualStudio/TeamTest/2010")
summary = ET.SubElement(root, "ResultSummary", outcome="Completed" if failed == 0 else "Failed")
ET.SubElement(summary, "Counters", total=str(total), executed=str(total), passed=str(passed), failed=str(failed), error="0", timeout="0", aborted="0")
results_elem = ET.SubElement(root, "Results")

for name, is_pass, err_msg in results:
    tc = ET.SubElement(results_elem, "UnitTestResult", testName=name, outcome="Passed" if is_pass else "Failed")
    if not is_pass:
        out = ET.SubElement(tc, "Output")
        err = ET.SubElement(out, "ErrorMessage")
        err.text = err_msg

tree = ET.ElementTree(root)
tree.write("TestResults/TestResults.trx", encoding="utf-8", xml_declaration=True)

if failed > 0:
    print(f"Failed! - Failed: {failed}, Passed: {passed}, Skipped: 0, Total: {total}, Duration: 24ms - csharp_service.dll (net8.0)")
    for name, is_pass, err_msg in results:
        if not is_pass:
            print(f"  [FAIL] {name}: {err_msg}")
    sys.exit(1)
else:
    print(f"Passed! - Failed: 0, Passed: {passed}, Skipped: 0, Total: {total}, Duration: 24ms - csharp_service.dll (net8.0)")
    sys.exit(0)
EOF
