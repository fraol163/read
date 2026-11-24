# Vision and Philosophical Foundation

## 1. Executive Summary

READ stands for Resource-conscious, Evidence-driven AI Development.
The framework provides a deterministic, resource-bounded runtime for AI agent development workflows.

Traditional AI engineering frameworks often rely on large language models executing multi-agent roleplay to make architectural decisions, critique code, and confirm task completion. READ replaces persona roleplay with local machine verification, static AST analysis, mutation testing, and cryptographic write-ahead logging.

## 2. Origin and Purpose

Current agentic development tools frequently generate substantial code changes that break subtle edge cases. When tests pass only because assertions are weak, human review becomes a bottleneck.

READ addresses three fundamental problems in agent workflows:
1. Token Burn: Repeated agent conversational rounds consume excessive API tokens without guaranteeing code correctness.
2. Unchecked Regressions: Unit test suites often fail to catch logic inversions or missing boundary checks introduced by generative models.
3. Lack of Auditability: Conversational chat logs do not provide forensic proof of what code changed or whether verification tests actually ran against the change.

READ establishes a closed-loop pipeline where agents must satisfy machine-verifiable contracts before code can transition to a committed state.

## 3. Contrast with BMAD

BMAD stands for Breakthrough Method for Agile AI-Driven Development (bmad-code-org/BMAD-METHOD, release v6.12.4).
READ differs fundamentally from BMAD across four core dimensions:

### Persona Theater vs Deterministic Routing
- BMAD Approach: Deploys multiple conversational personas (such as Architect, Product Owner, QA Specialist, and Scrum Master) discussing code in prompts.
- READ Rejection: Persona dialogue adds stochastic latency and consumes model context without providing verification proofs.
- READ Axiom: Route tasks through explicit state machine transitions governed by deterministic CAS logic in read-core.

### Narrative PRDs vs Machine-Verifiable Contracts
- BMAD Approach: Uses natural language markdown documents and user stories as acceptance criteria.
- READ Rejection: Natural language instructions contain inherent ambiguities that generative models interpret unpredictably.
- READ Axiom: Freeze machine-readable JSON contracts and explicit hash anchors before work begins. The hash of TASK.md anchors the initial state.

### Conversational QA vs Cryptographic Evidence
- BMAD Approach: Relies on agents reading test outputs and stating in conversation that the feature works.
- READ Rejection: Language models can hallucinate passing test runs or overlook subtle runtime failures.
- READ Axiom: All state transitions require cryptographically hashed test logs and diff hashes recorded into an append-only write-ahead log (WAL.log).

### Unmetered Abundance vs Computational Finitude
- BMAD Approach: Assumes abundant cloud compute and unmetered token context windows.
- READ Rejection: Unbounded context windows lead to degradation of attention and high infrastructure costs.
- READ Axiom: Enforce strict resource budgets. Bound mutation generation to ten targeted mutants. Set process memory ceilings via dynamic RLIMIT calibration to keep resource consumption bounded.
