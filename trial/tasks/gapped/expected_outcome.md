# Expected Outcome: Tiered Rebate Calculation

## Tests Required

The test suite in `test_rebate.py` verifies:
1. `test_gold`: Order with tier "gold" adds 20 rebate bonus.
2. `test_zero`: Order with 0 amount adds 20 rebate bonus.
3. `test_negative`: Order with negative amount raises ValueError.

All 3 unit tests must pass with exit code 0.
