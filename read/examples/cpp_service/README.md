# READ Reference Implementation: C++ Service

This example demonstrates the READ framework verifying a high-performance C++ memory management service using Google Test.

## Architecture

- **Project Specification**: `CMakeLists.txt` (Targeting C++17, Google Test, CTest)
- **Contract Specification**: `contract.json` (SHA-256 sealed task definition)
- **Service Header & Implementation**: `include/buffer_pool.hpp` & `src/buffer_pool.cpp`
- **Unit & Boundary Tests**: `tests/test_buffer_pool.cpp`
- **Test Runner**: `test.sh` (Compiles with `g++` and runs Google Test with XML output)
- **Lifecycle Demo**: `run_demo.sh`

## Targeted AST Mutation Operators

READ's AROP engine targets C++ AST nodes with 4 deterministic operators:
1. `REMOVE_NULLPTR_CHECK`: `if (ptr != nullptr)` -> `if (true)`
2. `INVERT_EQUALITY`: `==` <-> `!=`, `<` <-> `<=`, `>` <-> `>=`
3. `RETURN_DEFAULT`: `return buffer;` -> `return nullptr;` / `return {};`
4. `BOUNDARY_SHIFT`: `allocated_count_++` -> `allocated_count_--`

All generated mutants are bounded ($\le 10$), diff-targeted, and must be 100% killed before commit handoff is permitted.

## Running the Demo

```bash
./run_demo.sh
```
