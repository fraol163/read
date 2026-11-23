# Task: Fix Pagination Offset Bug

## Goal

The pagination helper function in `paginate.py` has an off-by-one bug in offset calculation for 1-based page indexing.
Calls with `page=1` return elements starting from index `page_size` instead of index `0`.

Fix the calculation in `paginate.py` so that 1-based indexing correctly returns slices from the beginning of the list without out-of-bounds or slice distortion errors.

## Running Tests

Run the test suite with a single command from inside the repository:

```bash
python3 -m unittest test_paginate.py
```
