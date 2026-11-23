import unittest
from paginate import paginate

class TestPaginate(unittest.TestCase):
    def setUp(self):
        self.sample = [10, 20, 30, 40, 50, 60, 70, 80]

    def test_first_page(self):
        result = paginate(self.sample, page=1, page_size=3)
        self.assertEqual(result, [10, 20, 30])

    def test_second_page(self):
        result = paginate(self.sample, page=2, page_size=3)
        self.assertEqual(result, [40, 50, 60])

    def test_empty_list(self):
        result = paginate([], page=1, page_size=5)
        self.assertEqual(result, [])

    def test_out_of_bounds_page(self):
        result = paginate(self.sample, page=10, page_size=3)
        self.assertEqual(result, [])

    def test_invalid_inputs(self):
        with self.assertRaises(ValueError):
            paginate(self.sample, page=0, page_size=3)
        with self.assertRaises(ValueError):
            paginate(self.sample, page=1, page_size=0)

if __name__ == "__main__":
    unittest.main()
