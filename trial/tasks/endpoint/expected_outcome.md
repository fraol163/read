# Expected Outcome: Endpoint Task

The test suite in `test_endpoint.py` must pass completely with exit code 0.

## Required Passing Tests

1. `test_search_by_name_substring`: Verifies searching for substring "ali" finds "Alice" and "Kalil".
2. `test_search_with_role_filter`: Verifies role filter strictly isolates matching roles.
3. `test_search_case_insensitive`: Verifies uppercase queries match lowercase names.
4. `test_search_limit_enforced`: Verifies returning at most `limit` matches even when more candidates qualify.
5. `test_search_invalid_limit_raises_error`: Verifies `limit < 1` raises `ValueError`.
