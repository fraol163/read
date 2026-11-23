# TypeScript Service Example: Role-Based Authorization and Token Validation

This project demonstrates the complete READ framework lifecycle applied to a production-grade TypeScript microservice component.

## Contents

- `src/auth_service.ts`: TypeScript RBAC authorization and constant-time token verification engine.
- `tests/auth_service.test.ts`: Test suite verifying boundary conditions, optional chaining resilience, role rejection, and equality checks.
- `package.json`: NPM package manifest declaring Express, Jest devDependencies, and test scripts.
- `tsconfig.json`: TypeScript compiler configuration.
- `contract.json`: Formal task contract with machine-verifiable invariants.
- `run_demo.sh`: Automated script executing the 4-stage gated READ lifecycle.

## Running the Tests Directly

To run the test suite directly:
```bash
npm test
# or: node --experimental-strip-types --test tests/auth_service.test.ts
```

## Running the Complete READ Lifecycle Verification

To run the complete automated READ lifecycle (init -> define -> build -> ast-context -> verify -> handoff -> recover):
```bash
bash run_demo.sh
```
