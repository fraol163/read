# Project Engineering Contract (PROJECT.md)

## 1. Project Identity & Purpose
- Repository Name: [REPLACE_WITH_PROJECT_NAME]
- Purpose: [REPLACE_WITH_CONCISE_MISSION]
- Target Runtime / Language: [e.g., Python 3.11, Rust 1.75, Go 1.22, Node 20]

## 2. Invariants & Non-Negotiables
- Architecture Invariant 1: [e.g., All state modifications must be atomic and reversible]
- Architecture Invariant 2: [e.g., Public API endpoints must be authenticated and rate-limited]
- Resource Boundary: Maximum memory footprint <= 1.2 GB RAM
- External Network Policy: Deny external network calls during unit test execution

## 3. Explicit Non-Goals
- Do NOT introduce heavyweight ORM frameworks or unnecessary microservice layers.
- Do NOT rewrite working existing modules unless explicitly specified in an approved task contract.
- Do NOT introduce unvetted third-party dependencies.

## 4. Test & Verification Commands
- Unit Tests: `[e.g., pytest tests/ -v | cargo test | go test ./...]`
- Linting / Format: `[e.g., ruff check . | cargo clippy | golangci-lint run]`
- Target Mutation Score: 100% mutants killed (strictly zero survivors)
