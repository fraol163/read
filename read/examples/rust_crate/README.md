# Rust Library Crate Example: Bounded FIFO Ring Buffer

This project demonstrates the complete READ framework lifecycle applied to a high-performance native Rust library crate.

## Contents

- `src/lib.rs`: Library crate root exporting public structures and traits.
- `src/ring_buffer.rs`: Deterministic circular buffer implementation with configurable overflow policies (`Reject`, `Overwrite`).
- `tests/test_integration.rs`: Multi-element integration tests verifying stream buffering and overwrite mechanics.
- `contract.json`: Machine-readable task contract freezing task invariants and baseline commits.
- `run_demo.sh`: Executable demonstration of the 4-stage gated lifecycle.

## Running Tests Directly

```bash
cargo test
```

## Running READ Lifecycle Verification

```bash
bash run_demo.sh
```
