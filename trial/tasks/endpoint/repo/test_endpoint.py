import unittest
from endpoint import handle_search

class TestEndpoint(unittest.TestCase):
    def setUp(self):
        self.users = [
            {"id": 1, "name": "Alice Smith", "role": "admin"},
            {"id": 2, "name": "Bob Jones", "role": "member"},
            {"id": 3, "name": "Kalil Ali", "role": "member"},
            {"id": 4, "name": "Alicia Keys", "role": "guest"},
            {"id": 5, "name": "Charlie Brown", "role": "member"},
        ]

    def test_search_by_name_substring(self):
        res = handle_search(self.users, query="ali")
        names = [u["name"] for u in res]
        self.assertEqual(names, ["Alice Smith", "Kalil Ali", "Alicia Keys"])

    def test_search_with_role_filter(self):
        res = handle_search(self.users, query="ali", role="member")
        names = [u["name"] for u in res]
        self.assertEqual(names, ["Kalil Ali"])

    def test_search_case_insensitive(self):
        res = handle_search(self.users, query="BOB")
        self.assertEqual(len(res), 1)
        self.assertEqual(res[0]["name"], "Bob Jones")

    def test_search_limit_enforced(self):
        res = handle_search(self.users, query="", limit=2)
        self.assertEqual(len(res), 2)

    def test_search_invalid_limit_raises_error(self):
        with self.assertRaises(ValueError):
            handle_search(self.users, query="test", limit=0)
        with self.assertRaises(ValueError):
            handle_search(self.users, query="test", limit=-5)

if __name__ == "__main__":
    unittest.main()
