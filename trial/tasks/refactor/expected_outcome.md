# Expected Outcome: Refactor Task

The test suite in `test_invoice.py` must pass completely with exit code 0.

## Required Passing Tests

1. `test_standard_customer_no_coupon`: Subtotal equals total with no discount applied.
2. `test_vip_customer_discount`: 15% discount applied for VIP tier.
3. `test_coupon_discount`: 10% discount applied for valid coupon `SAVE10`.
4. `test_combined_vip_and_coupon`: 25% combined discount applied.
5. `test_unknown_coupon_raises_error`: Unrecognized coupon code raises `ValueError`.
6. `test_empty_items_returns_zero`: Empty cart produces total of 0.0.
