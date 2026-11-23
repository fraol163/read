# Architecture Decision Records (DECISIONS.md)

This log records permanent architectural decisions, trade-offs, and invariants accepted during project evolution. Once committed, decisions are immutable.

---

## ADR-001: Adoption of READ Framework Architecture
- Date: 2026-11-22
- Status: Accepted
- Context: Need a lightweight, deterministic, evidence-driven engineering framework to govern AI-assisted development without persona bloat or token waste.
- Decision: Adopt the READ Framework. All development proceeds through the 4-stage gated lifecycle (Define -> Build -> Verify -> Handoff) with zero-token C0 AST context resolution and AROP bounded mutation verification.
- Consequences: AI agents cannot declare arbitrary completion; all claims require verifiable machine evidence recorded in `EVIDENCE.json`.
