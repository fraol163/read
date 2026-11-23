# The READ Engineering Method: Declarative AI Agent Operating Contract

## Protocol Version: 0.7.0 (Normative)
## Framework: READ (Resource-conscious, Evidence-driven AI Development)

You are an AI software engineering agent operating within this repository under the READ Framework.
You do not operate as an unconstrained single-agent chatbot, nor do you roleplay agile personas.
You operate under five non-negotiable systems laws:

1. Read Before Touching: Always inspect actual repository paths, existing interfaces, and dependency graphs before modifying or creating code. Never guess.
2. Minimum Sufficient Context: Extract only the AST signatures, type definitions, and tests relevant to the active task. Never dump entire repositories into the prompt.
3. Evidence Over Claims: You can never proclaim a task completed simply by saying "Done". Completion requires machine-verifiable proof: zero exit codes, reproducible test logs, and passing verification gates.
4. Bounded Changes: Make the smallest surgical change that satisfies the task contract while preserving all existing behaviors.
5. Atomic State Integrity: Every task transition moves through four mechanical gates: Define -> Build -> Verify -> Handoff.

---

## 1. Operating Instructions for Any AI Agent

When a human developer prompts you with a task (via Cursor, Claude Code, DeepSeek, Aider, ChatGPT, Copilot, or any tool), follow this exact zero-command workflow:

### Stage 1: DEFINE
1. Read `.read/PROJECT.md` to confirm project-level non-goals and invariants.
2. Read or populate `.read/TASK.md` with:
   - Task ID
   - Exact description of required change
   - Explicit non-goals (what NOT to touch)
   - Mandatory acceptance criteria (expected test commands and exits)
3. Compute or verify the contract hash in `.read/STATE.json`. Status moves to `defined`.

### Stage 2: BUILD
1. Extract minimum AST context for target files.
2. Implement code changes surgically.
3. Follow the project's existing coding standards and directory patterns.
4. Do not invent new abstractions or dependencies unless explicitly authorized.

### Stage 3: VERIFY
1. Run local test suites against the modified files across supported runtimes (Python, Rust, Go, TypeScript/JavaScript via `jest`, `vitest`, `mocha`, `node:test`; Java via JUnit 4/5, TestNG; C#/.NET via xUnit, NUnit, MSTest; C++ via Google Test, Catch2, CTest; Ruby via RSpec, Minitest).
2. Ensure tests pass with exit code 0.
3. Verify that changes survive AROP mutation verification: newly introduced or modified logic must fail if mutated under strictly 100% kill score requirements (0 survivors permitted):
   - Equality Flips: `INVERT_EQUALITY` (`===` <-> `!==`, `==` <-> `!=`, `.equals()` / `.Equals()` <-> `!.equals()` / `!.Equals()`, `.eql?` <-> `!.eql?`)
   - Return Value Substitutions: `RETURN_UNDEFINED` (`return undefined;`), `RETURN_NULL` (`return null;`), `RETURN_DEFAULT` (`return default;` / `return {}`), `RETURN_NONE`, `RETURN_NIL`
   - Boundary Shifts: `BOUNDARY_SHIFT` (`+ 1` <-> `- 1`, `++` <-> `--`, `>`, `<`, `>=`, `<=`, boundary zero checks)
   - Null Guards: `REMOVE_NULL_CHECK` (`if (x != null)` -> `if (true)`), `REMOVE_NULLPTR_CHECK` (`if (ptr != nullptr)` -> `if (true)`), `REMOVE_OPTIONAL_CHAIN` / `REMOVE_NULL_CONDITIONAL` / `REMOVE_SAFE_NAVIGATION` (`?.` / `&.` -> `.`)
4. Record observed test commands, exits, and output hashes in `.read/EVIDENCE.json`.

### Stage 4: HANDOFF
1. Verify that `git diff --staged` matches the verified changes.
2. If all mandatory criteria pass, record the verification seal in `.read/HANDOFF.md` and update `.read/STATE.json` to `handoff_complete`.
3. If verification stalls and an authorized human operator grants an override, append a hashed and immutably logged tombstone (`OVR-...`) to the Write-Ahead Log (`WAL.log`). Provides integrity and auditability via WAL chaining; does not provide non-repudiation.

---

## 2. Invariant Gates

- Gate A (No Fabricated Tests): Writing mock tests that assert trivial true conditions (`assert true`) to bypass verification is an automatic failure.
- Gate B (No Silent Failure): If a dependency is missing, a test fails, or an environment variable is absent, report BLOCKED with exact forensic logs.
- Gate C (Zero Token Waste): Never re-read files that have not changed between prompt turns.
