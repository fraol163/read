# Task: Tiered Rebate Calculation

## Goal

Implement the `apply_rebate` function in `rebate.py` to calculate order rebates based on customer tier:
- If amount is negative, raise a `ValueError("negative amount")`
- Base bonus is 0
- Tier "gold" receives 20 bonus
- Tier "silver" receives 10 bonus
- Return the sum of amount and bonus

## Running Tests

Run the test suite with a single command from inside the repository:

```bash
python3 -m unittest test_rebate.py
```
