# Go Package Example: Concurrent In-Memory Cache

This project demonstrates the complete READ framework lifecycle applied to a production Go package.

## Contents

- `cache/cache.go`: Thread-safe in-memory key-value store with TTL expiration, lazy eviction, and stats.
- `cache/cache_test.go`: Complete Go test suite verifying concurrency, expiration, and error handling.
- `contract.json`: Task contract specifying invariants and baseline commit.
- `run_demo.sh`: Executable demonstration of the 4-stage gated lifecycle.

## Running Tests Directly

```bash
go test -v ./cache
```

## Running READ Lifecycle Verification

```bash
bash run_demo.sh
```
