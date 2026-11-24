# Python Service Example: Ingress Rate Limiter

This project demonstrates the complete READ framework lifecycle applied to a production-grade Python microservice component.

## Contents

- `src/rate_limiter.py`: Clean Python implementation of token bucket and sliding window rate limiting.
- `tests/test_rate_limiter.py`: Pytest suite verifying boundary conditions, client isolation, and capacity exhaustion.
- `contract.json`: Formal task contract with invariants and target files.
- `run_demo.sh`: Automated script executing the 4-stage gated lifecycle.

## Running the Tests

To run the unit test suite directly:
```bash
pytest tests/ -v
```

## Running the READ Lifecycle Verification

To run the complete automated READ lifecycle (init -> define -> build -> ast-context -> verify -> handoff -> recover):
```bash
bash run_demo.sh
```
