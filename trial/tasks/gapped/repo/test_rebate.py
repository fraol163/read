import unittest
from rebate import apply_rebate

class TestRebate(unittest.TestCase):
    def test_gold(self):
        self.assertEqual(apply_rebate(100, "gold"), 120)

    def test_zero(self):
        self.assertEqual(apply_rebate(0, "gold"), 20)

    def test_negative(self):
        with self.assertRaises(ValueError):
            apply_rebate(-5, "gold")

if __name__ == "__main__":
    unittest.main()
