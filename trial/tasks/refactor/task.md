# Task: Refactor and Complete Invoice Pricing Calculator

## Goal

Refactor and complete `calculate_total` in `invoice.py`.
The current starting implementation has bugs in discount calculation:
- Standard customers receive no discount (0%).
- VIP customers should receive a 15% discount (0.15), but starting code applies 0%.
- Coupon `SAVE10` provides an additional 10% discount (0.10).
- Discounts are additive (for example, a VIP customer with `SAVE10` receives 0.15 + 0.10 = 0.25 discount).
- An unrecognized coupon code must raise `ValueError`.
- Empty item lists must return 0.0 without calculation errors.
- Results must be rounded to two decimal places.

## Running Tests

Run the test suite with a single command from inside the repository:

```bash
python3 -m unittest test_invoice.py
```
