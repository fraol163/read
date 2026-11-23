import unittest
from invoice import calculate_total

class TestInvoice(unittest.TestCase):
    def setUp(self):
        self.items = [
            {"name": "Widget A", "price": 20.0, "quantity": 2},
            {"name": "Widget B", "price": 60.0, "quantity": 1},
        ]
        # subtotal = 20*2 + 60*1 = 100.0

    def test_standard_customer_no_coupon(self):
        total = calculate_total(self.items, customer_tier="standard")
        self.assertEqual(total, 100.0)

    def test_vip_customer_discount(self):
        total = calculate_total(self.items, customer_tier="vip")
        self.assertEqual(total, 85.0)

    def test_coupon_discount(self):
        total = calculate_total(self.items, customer_tier="standard", coupon_code="SAVE10")
        self.assertEqual(total, 90.0)

    def test_combined_vip_and_coupon(self):
        total = calculate_total(self.items, customer_tier="vip", coupon_code="SAVE10")
        self.assertEqual(total, 75.0)

    def test_unknown_coupon_raises_error(self):
        with self.assertRaises(ValueError):
            calculate_total(self.items, customer_tier="standard", coupon_code="INVALID99")

    def test_empty_items_returns_zero(self):
        total = calculate_total([], customer_tier="vip", coupon_code="SAVE10")
        self.assertEqual(total, 0.0)

if __name__ == "__main__":
    unittest.main()
