# Expected Outcome: Tiered Rebate Calculation

## Tests Required

The test suite in `test_rebate.py` verifies:
1. `test_gold`: Order with tier "gold" adds 20 rebate bonus.
2. `test_zero`: Order with 0 amount adds 20 rebate bonus.
3. `test_negative`: Order with negative amount raises ValueError.

All 3 unit tests must pass with exit code 0.

## Mutation Gate Survivor Analysis

- Survivor: `INVERT_EQ at rebate.py:7 (mutated: elif tier != "silver":)`
- Label: agent-labeled, pending reviewer
- Reason: The test suite only exercises gold tier and negative amount branches leaving the silver tier rebate unexercised.
