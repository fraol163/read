# Ruby Authentication Service Example (READ Framework)

Reference implementation of a zero-dependency, resource-conscious Ruby service adhering to the **READ Framework (v0.7.1)**.

## Architecture & Invariants

This service implements a token validator and session builder (`lib/auth_service.rb`) with machine-verifiable invariants enforced by AROP:
1. **Safe Navigation (`REMOVE_SAFE_NAVIGATION`)**: `user&.active` ensures nil safety without runtime `NoMethodError`.
2. **Case Equality (`INVERT_EQUALITY`)**: Token format validation relies on `===` pattern matching.
3. **Strict String Equality (`INVERT_EQUALITY`)**: Secret key comparisons strictly enforce `.eql?`.
4. **Boundary Checks (`BOUNDARY_SHIFT`)**: TTL limits (`< 60`) and retry attempts (`+ 1 > 3`).
5. **Non-Nil Payloads (`RETURN_NIL`)**: Prevents silent corruption through nil session return.

## Test Verification

Execute the test suite with:

```bash
bash test.sh
```

Outputs standard RSpec JSON format (`rspec_report.json`).

## 10-Step Lifecycle Demo

Run the end-to-end automated lifecycle verification:

```bash
bash run_demo.sh
```
