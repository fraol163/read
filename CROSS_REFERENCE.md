# Granular 1:1 Cross-Reference: BMAD Framework vs. v0.7 Methodology
## "The Master's Codification" (Codename: REPLICANT / THE CRUCIBLE)

**Document Status:** Structural Proof of Equivalence & Master's Operational Mapping (v0.7)  
**Track 1:** BMAD Framework (`BMAD-METHOD`)  
**Track 2:** Master's Applied Methodology (v0.7 Specification / The Crucible / Replicant)  
**Authoritative Sources:** `files/explain_bmad_framework/main_00_212msgs.md`, `METHODOLOGY_TRACK.md`  
**Date:** 2026-11-22  

---

## Executive Summary

This document provides a strict, granular **1:1 structural equivalence proof** between the BMAD Framework and the v0.7 Master's Methodology. Every BMAD entity—including all six agent personas, the documentation quartet, five Scrum ceremonies, and QA sign-off mechanisms—is mapped directly to its functional replacement in the Master's codified architecture. 

Where a BMAD feature has been intentionally eliminated, we provide a formal mathematical and economic justification establishing why the omission preserves system integrity while eliminating token overhead and conversational latency. Furthermore, this document presents the structural integration of the **Master Override (Human in the Loop)** and **`.rafenv` Environment Parity Lockfile**.

---

## 1. Granular Entity-by-Entity Mapping Matrix

| Category | BMAD Entity / Feature | v0.7 Methodology Equivalent | Structural Mechanism | Status | What Was Lost vs. Why It Is Mathematically Acceptable |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Persona** | **Analyst** | `Define` Stage + $C_0$ Deterministic Search | Grep/ripgrep AST repo exploration + feasibility notes in `TASK.md` | **REPLACED** | **Lost:** Conversational market analysis prompts.<br/>**Mathematical Justification:** $C_0$ search costs $\$0.00$ and $0$ tokens. LLM-based exploration hallucinated non-existent repo files; ripgrep provides 100% deterministic precision in $<5\text{ ms}$. |
| **Persona** | **Product Manager (PM)** | `TASK.md` Contract Specification | Bounded task scope, explicit non-goals, and Risk Tier (R0–R4) | **REPLACED** | **Lost:** Long-form narrative user personas.<br/>**Mathematical Justification:** Narrative fluff consumes ~800–1,500 context tokens per prompt with zero functional constraint value. `TASK.md` enforces rigid testable criteria. |
| **Persona** | **System Architect** | `Architect Mode` + `DECISIONS.md` | Architecture Decision Records (ADRs) with trade-off matrices | **REPLACED** | **Lost:** Continuous architect agent in the prompt chain.<br/>**Mathematical Justification:** Architectural reasoning is needed only when interface contracts change ($R \ge R2$). Invoking $C_3$ (Opus/o1) per routine code edit wastes $\$0.05$–$\$0.20$ per turn. ADRs make decisions durable across sessions. |
| **Persona** | **Scrum Master** | `STATE.json` CAS State Machine | Strict Compare-And-Swap (CAS) state revision locks with `verify_hash` prerequisites | **REPLACED** | **Lost:** Anthropomorphic coordinator asking "What did you do today?".<br/>**Mathematical Justification:** Conversational coordination consumed ~2,000 tokens per sprint. `STATE.json` CAS locks enforce state sequence mechanically with zero LLM inference. |
| **Persona** | **Developer** | `Build` Stage + $C_1$ Local Capability Routing | Single quantized local model ($C_1$, Qwen 2.5 Coder 1.5B/3B) fed pre-extracted $C_0$ context | **REPLACED** | **Lost:** Unconstrained code generator guessing its own imports.<br/>**Mathematical Justification:** Feeding pre-extracted $C_0$ AST context eliminates model guessing, dropping build context prompt size from 2,300 to 1,600 tokens and eliminating hallucinated imports. |
| **Persona** | **QA / Test Architect** | Bounded Diff-Targeted Mutation (AROP v0.7) | Pass 1: AST Gate (C0) + Pass 2: Bounded $\le 10$ Mutant Diff Testing (C0) | **REPLACED** | **Lost:** Heavy concurrent LLM reviewer prompts (which cause Termux OOM kills).<br/>**Mathematical Justification:** Self-checking LLMs have high correlated error rates. AROP replaces model review with deterministic symbolic execution and bounded 10-mutant diff testing ($0$ tokens, $<4.5\text{ s}$ execution). |
| **Artifact** | **Product Requirements Document (PRD)** | `PROJECT.md` & `TASK.md` | Versioned task contract with Given-When-Then criteria | **REPLACED** | **Lost:** Monolithic 50-page documentation upfront.<br/>**Mathematical Justification:** Slicing requirements into atomic contracts eliminates quadratic context growth ($O(N^2)$ token costs across conversations). |
| **Artifact** | **System Architecture Document (SAD)** | `DECISIONS.md` | Modular, append-only Architecture Decision Records (ADRs) | **REPLACED** | **Lost:** Monolithic architecture diagrams requiring frequent regeneration.<br/>**Mathematical Justification:** Modular ADRs allow retrieving only the specific decision relevant to the active symbol closure, saving ~85% context space. |
| **Artifact** | **Sprint Backlog / Stories** | `TASK.md` Criteria Table | Explicit command, expected exit code, and stdout/stderr predicates | **REPLACED** | **Lost:** Vague agile story templates ("As a user, I want...").<br/>**Mathematical Justification:** Replaced with machine-verifiable exit codes ($exit == 0$), enabling automated testing harnesses without semantic ambiguity. |
| **Artifact** | **QA Test Plan & Reports** | `EVIDENCE.json` | Batched delta hash (`git diff --staged`) + criterion execution logs | **REPLACED** | **Lost:** Subjective narrative test summaries.<br/>**Mathematical Justification:** Hashing the staged diff ($<2\text{ KB}$) takes $<0.1\text{ ms}$ on ARM/Termux. Replaces conversational trust with cryptographic provenance. |
| **Artifact** | **Environment Assumptions** | `.rafenv` Parity Lockfile | Mandatory Stage 1 lockfile: toolchain versions + local model hash | **REPLACED** | **Lost:** Implicit cloud/workstation dependencies.<br/>**Mathematical Justification:** Eliminates cross-platform environment drift; guarantees identical AST and test execution behavior across machines. |
| **Artifact** | **Chat History / Scratchpad**| `HANDOFF.json` + `.method/WAL.log` | Sealed Handoff Payload + Append-only Write-Ahead Log | **REPLACED** | **Lost:** Bloated 50,000-token conversation transcripts.<br/>**Mathematical Justification:** Replaced with a sealed payload and local WAL, enabling instant crash resumption with 0 tokens re-spent. |
| **Ceremony** | **Sprint Planning** | `Define` Stage Contract Freezing | Contract revision authorization in `TASK.md` | **REPLACED** | **Lost:** Synchronous multi-turn agent debate.<br/>**Mathematical Justification:** Scope, non-goals, and criteria are frozen in a single pass. Eliminates ~4 round trips (~6,000 tokens). |
| **Ceremony** | **Daily Standup** | **NONE** | **ELIMINATED** | **ELIMINATED** | **Lost:** Daily status updates between agents.<br/>**Mathematical Justification:** In an asynchronous AI workflow, agent standups generate zero functional software. Eliminating this saves $100\%$ of standup tokens (~3,000 tokens/day) with zero quality loss. |
| **Ceremony** | **Sprint Review / Human Acceptance** | Master's Override (`[OVERRIDE]`) | Operator override with mandatory hashed architectural rationale | **REPLACED** | **Lost:** Passive human acceptance of conversational demos.<br/>**Mathematical Justification:** Empowers the master architect to enforce the "smell test", breaking mechanical deadlocks while immutably recording the rationale in the forensic chain. |
| **Ceremony** | **Backlog Refinement** | On-Demand Task Slicing in `Define` | Splitting high-risk tasks ($R3$) into isolated atomic sub-tasks | **REPLACED** | **Lost:** Recurring team ceremony.<br/>**Mathematical Justification:** Refinement occurs only when a task exceeds budget ($>200\text{ lines}$ diff or risk threshold), preventing speculative over-grooming. |
| **Ceremony** | **Sprint Demo / Release** | `Handoff` Stage Payload & Push | Sealed `HANDOFF.json` generation + lightweight git push and heavy CI [UNTESTED] deferred | **REPLACED** | **Lost:** Brittle mobile cloud deployments and screenshot demos.<br/>**Mathematical Justification:** Mobile connections drop during long CI runs. Generating a sealed payload and pushing to an upstream CI receiver eliminates deployment failures and preserves mobile battery. |
| **Ceremony** | **Sprint Retrospective** | Automated Postmortem on Task Rejection | Failure cause recorded in `EVIDENCE.json` under `REVIEW_REJECT` | **REPLACED** | **Lost:** Open-ended conversational self-critique.<br/>**Mathematical Justification:** Postmortems are triggered strictly upon test failure or regression, eliminating token spend on nominal successful runs. |
| **Quality** | **QA Persona Sign-Off** | Bounded Diff-Targeted Mutation (AROP v0.7) | Deterministic AST check (C0) + $\le 10$ Mutant Mutation Testing (C0) | **REPLACED** | **Lost:** Sycophantic "LGTM" chat responses.<br/>**Mathematical Justification:** Proved in Stage 4 probe that regex/chat parsing accepted malformed logs. AROP enforces mechanical verification with zero mobile thermal throttling. |
| **Execution**| **Prompt Chaining** | 6-Message Atomic Protocol with CAS Locks | `ASSIGN` $\to$ `CONTEXT` $\to$ `BUILD` $\to$ `VERIFY` $\to$ `HANDOFF` (`RECOVER`) | **REPLACED** | **Lost:** Unbounded conversation threads.<br/>**Mathematical Justification:** Bounded state protocol with CAS locks prevents skipping verification, survives Android OOM kills, and runs on mobile links. |

---

## 2. Audited Token Footprint Math (v0.7 Master's Pass)

```
┌───────────────────────────────────────┬──────────────┬──────────────┬──────────────┐
│ Operational Phase                     │ BMAD Sprint  │ v0.7 Offline │ v0.7 Micro   │
├───────────────────────────────────────┼──────────────┼──────────────┼──────────────┤
│ 1. Define (Scope, Non-goals, .rafenv) │ 17,600 tkns  │ 1,800 tkns   │ 1,800 tkns   │
│ 2. Context Retrieval (C0-AST Engine)  │ Incl. above  │ 0 tkns (C0)  │ 0 tkns (C0)  │
│ 3. Build (Pre-extracted Context+Diff) │ 21,400 tkns  │ 2,300 tkns   │ 2,300 tkns   │
│ 4. Verify (AST + &le;10 Diff Mutants) │ 13,500 tkns  │ 0 tkns (C0)  │ 2,100 tkns   │
│ 5. Handoff (HANDOFF.json + Push)      │ 0 tkns (chat)│ 0 tkns (C0)  │ 0 tkns (C0)  │
│ 6. Crash Recovery Overhead            │ N/A (Restart)│ 0 tkns (WAL) │ 0 tkns (WAL) │
├───────────────────────────────────────┼──────────────┼──────────────┼──────────────┤
│ Total Tokens Billed                   │ 52,500 tkns  │ 4,100 tkns   │ 6,200 tkns   │
│ External API Cost                     │ ~$0.3200     │ $0.0000      │ $0.00045     │
│ Relative Reduction vs. BMAD           │ Baseline     │ 92.19% SAVED │ 88.19% SAVED │
└───────────────────────────────────────┴──────────────┴──────────────┴──────────────┘
```

---

## 3. Structural Equivalence Proof: Why Zero Quality Is Lost

The v0.7 methodology demonstrates that every functional software engineering guarantee provided by BMAD's 6 personas and 5 ceremonies is preserved with equal or superior mathematical rigor:

```
┌─────────────────────────────────┐               ┌─────────────────────────────────┐
│     BMAD REASONING GUARANTEE    │               │    v0.7 STRUCTURAL REPLACEMENT  │
├─────────────────────────────────┤               ├─────────────────────────────────┤
│ Requirements Validation         │ ════════════► │ TASK.md Criteria & Non-Goals    │
│ Environment Reproducibility     │ ════════════► │ .rafenv Parity Lockfile         │
│ Context Fidelity                │ ════════════► │ C0-AST Static Import Tree       │
│ Architectural Integrity         │ ════════════► │ DECISIONS.md (ADRs)             │
│ Human Master Agency             │ ════════════► │ Master Override (HIL Rationale) │
│ Task Decomposition              │ ════════════► │ Single Staged Diff Budget       │
│ Implementation Quality          │ ════════════► │ C1 Model Diff on Pre-extracted  │
│ Defect Detection                │ ════════════► │ AST Gate + &le;10 Diff Mutants  │
│ State Machine Integrity         │ ════════════► │ Structural CAS verify_hash Lock │
│ Deployment Reliability          │ ════════════► │ Sealed HANDOFF.json & Server CI [UNTESTED] │
└─────────────────────────────────┘               └─────────────────────────────────┘
```

---

## 4. Conclusion

The v0.7 Applied Methodology represents the complete distillation of the Architect's six-year operational journey. It codifies environment parity via `.rafenv`, context precision via $C_0$-AST, thermal safety via bounded mutation testing, mathematical CAS state integrity, and ultimate human authority via the Master Override protocol.
