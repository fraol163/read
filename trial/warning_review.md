# Reviewer Checklist for Pre-Commit Gate Warnings

## Purpose

This document provides guidelines and an evaluation checklist for determining whether pre-commit gate warnings identify genuine gaps in test coverage or represent noise such as equivalent mutants.

## Diagnostic Output Format

The pre-commit gate in warning mode outputs diagnostics in the following standardized format:

1. Test Suite Diagnostics (Failing Baseline):
```
----------------------------------------------------------------
[READ GATE WARNING] Test suite failed with exit code <EXIT_CODE> (Warning Mode):
<RAW_TEST_OUTPUT>
----------------------------------------------------------------
MUTATION SKIPPED: baseline tests fail
================================================================
[READ GATE WARNING] Commit allowed in warning mode.
Review test failures before release.
================================================================
```

2. Mutation Evaluation Diagnostics (Passing Baseline):
```
[READ AROP] Evaluating <TOTAL> targeted mutants on production code diff...
  [<OPERATOR> at L<LINE> in <FILE>] (mutated: <MUTATED_LINE>) -> <KILLED | SURVIVED>
----------------------------------------------------------------
[READ GATE WARNING] Mutation score: <SCORE>% (Killed <KILLED>/<TOTAL>)
Surviving mutants:
  - [SURVIVOR] <OPERATOR> at <FILE>:<LINE> (mutated: <MUTATED_LINE>)
Commit allowed in warning mode. Review test gaps for remediation.
----------------------------------------------------------------
```

## Review Rules and Survivor Judging Protocol

Reviewers judge only SURVIVED mutants, and only after validation shows real survivors on a passing baseline. Mutants are not judged when baseline tests fail.

Each SURVIVED mutant gets one label: "real test gap" or "equivalent mutant", with a one-line reason.
If a reviewer cannot label a survivor in under two minutes, it goes into an "unclear" bucket, counted separately, and not counted as a gap.

### 1. Label Definitions
- Real Test Gap: The mutant causes observable functional divergence in behavior that a comprehensive unit test suite should detect and fail.
- Equivalent Mutant: The mutant produces behavior semantically identical to the original code or alters non-functional string descriptions without changing control flow or error classes.
- Unclear Bucket: Assigned when classification exceeds two minutes of inspection. Counted separately and explicitly excluded from test gap counts.

### 2. Time-Boxed Evaluation (Two-Minute Rule)
Reviewers must complete the evaluation of a single mutant survivor in under two minutes. If ambiguity or complex code paths prevent reaching a definitive label within two minutes, the survivor is immediately routed to the "unclear" bucket.

## Blinded Evaluation Methodology

To prevent reviewer bias based on developer identity or experience level, warning reviews must follow a blinded protocol:

1. Identification Masking:
   - Each warning review package is assigned a randomized evaluation token.
   - The participant identifier (`participant_id`) and git commit author metadata are stripped before the package reaches reviewers.
   - Reviewers receive only:
     - The staged git diff
     - The exact mutant line diagnostic: `[<OPERATOR> at L<LINE> in <FILE>] (mutated: <MUTATED_LINE>) -> <STATUS>`
     - The test suite execution output and test file source code
2. Independent Scoring:
   - Two reviewers evaluate each blinded package independently.
   - Reviewers score each item before discussing discrepancies.
   - If scores diverge, a third reviewer acts as tie-breaker.

## Reviewer Checklist

For each warning, the reviewer evaluates the following four questions:

### 1. Reachability and Semantic Relevance
- Is the mutated line executable during normal operation of the component?
- Does the mutant alter meaningful business logic or state mutations?
- Score options:
  - Yes: The mutant changes observable logic that callers rely upon.
  - No: The mutant affects dead code or defensive fallback logic that cannot be reached.

### 2. Equivalence Check
- Is the mutant semantically equivalent to the original code?
- Examples of equivalent mutants:
  - Replacing `<` with `<=` when the boundary value is impossible by invariant.
  - Inverting boolean logic on a parameter that is already constrained earlier.
- Score options:
  - Real Gap: The mutant produces observable divergence in behavior that should fail a test.
  - Equivalent Mutant: The mutant cannot produce a different outcome in any valid program state.

### 3. Missing Assertion vs Missing Test Case
- If the warning points to a real gap, what was omitted by the developer?
  - Missing Assertion: A test already exercises the code path, but lacks an `assertEqual` or similar check.
  - Missing Test Case: The input condition or boundary is completely absent from the test suite.

### 4. Actionability
- Could a junior engineer resolve this warning within 5 minutes without architectural redesign?
  - Highly Actionable: Adding a specific test case or assertion immediately kills the mutant.
  - Complex Action: Requires refactoring component interfaces to make the state observable.

## Scoring Template

For each evaluated warning, record:

```
Review Token: <random_uuid>
Task ID: <bugfix | endpoint | refactor>
Mutant Diagnostic: [<OPERATOR> at L<LINE> in <FILE>] (mutated: <MUTATED_LINE>) -> <STATUS>
Mutant Location: <filepath>:<linenumber>
Mutant Operator: <operator_name>
Mutated Code: <mutated_line_text>
Outcome: SURVIVED
Classification: [real test gap | equivalent mutant | unclear]
Reason: <one-line reason>
Gap Type: [Missing Assertion | Missing Case | None]
Resolution Actionable in 5 Min: [Yes | No]
Notes: <observations without participant attribution>
```
