# Expected Outcome: Bugfix Task

The test suite in `test_paginate.py` must pass completely with exit code 0.

## Required Passing Tests

1. `test_first_page`: Verifies `paginate(items, page=1, page_size=3)` returns `items[0:3]`.
2. `test_second_page`: Verifies `paginate(items, page=2, page_size=3)` returns `items[3:6]`.
3. `test_empty_list`: Verifies paginating an empty list returns an empty list `[]`.
4. `test_out_of_bounds_page`: Verifies requesting a page past the end of the collection returns `[]`.
5. `test_invalid_inputs`: Verifies requesting `page <= 0` or `page_size <= 0` raises `ValueError`.
