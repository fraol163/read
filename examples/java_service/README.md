# Java Service Example: Order Validation & Allocation Service

This project demonstrates the complete READ framework lifecycle applied to a production-grade Java microservice component.

## Contents

- `src/main/java/com/example/service/OrderValidationService.java`: Order validation and allocation service enforcing null-checks, strict `.equals()` role comparison, and boundary limits.
- `src/test/java/com/example/service/OrderValidationServiceTest.java`: Verification suite testing boundary conditions, null rejection, and role authorization.
- `pom.xml`: Maven manifest declaring Spring Boot Starter Web and JUnit 5 Jupiter.
- `contract.json`: Formal task contract with machine-verifiable invariants.
- `test.sh`: Self-contained verification runner using OpenJDK javac and java.
- `run_demo.sh`: Automated script executing the 10-step gated READ lifecycle.

## Running the Tests Directly

To run the verification test suite directly:
```bash
bash test.sh
# or if maven is installed:
# mvn test
```

## Running the Complete READ Lifecycle Verification

To run the complete automated READ lifecycle (init -> define -> build -> ast-context -> verify -> handoff -> recover):
```bash
bash run_demo.sh
```
