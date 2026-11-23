# READ Reference Implementation: C# Service

This example demonstrates the READ framework verifying an ASP.NET Core / C# service with xUnit.

## Architecture

- **Project Specification**: `csharp_service.csproj` (Targeting `net8.0`, xUnit 2.6.2, xunit.runner.visualstudio)
- **Contract Specification**: `contract.json` (SHA-256 sealed task definition)
- **Service Implementation**: `src/TokenValidationService.cs`
- **Unit & Property Tests**: `tests/TokenValidationServiceTest.cs`
- **Test Runner**: `test.sh` (Generates standard TRX reports in `TestResults/TestResults.trx`)
- **Lifecycle Demo**: `run_demo.sh`

## Targeted AST Mutation Operators

READ's AROP engine targets C# AST nodes with 4 deterministic operators:
1. `REMOVE_NULL_CONDITIONAL`: `user?.Role` -> `user.Role`
2. `INVERT_EQUALITY`: `role.Equals("ADMIN")` -> `!role.Equals("ADMIN")` and `==` -> `!=`
3. `RETURN_DEFAULT`: `return token;` -> `return default;`
4. `BOUNDARY_SHIFT`: `retries + 1 > 3` -> `retries - 1 > 3` and `>` -> `>=`

All generated mutants are bounded ($\le 10$), diff-targeted, and must be 100% killed before commit handoff is permitted.

## Running the Demo

```bash
./run_demo.sh
```
