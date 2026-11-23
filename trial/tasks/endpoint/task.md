# Task: Implement User Search API Handler

## Goal

Implement the `handle_search` function in `endpoint.py` to search a list of user records.
The function must support:
- Case-insensitive substring matching against the user `name`
- Optional exact filtering by `role`
- A configurable result `limit` (default: 10)
- Validation that `limit` is a positive integer (raising `ValueError` if less than 1)

## Running Tests

Run the test suite with a single command from inside the repository:

```bash
python3 -m unittest test_endpoint.py
```
