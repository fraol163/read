# Experiment Plan: Contract Format and Evidence Representation

## 1. Overview and Hypothesis

This experiment evaluates whether substituting YAML task contracts for Markdown task contracts and generating structured HTML evidence reports improves developer authoring efficiency, reduces configuration syntax errors, or speeds up reviewer verification triage compared to the baseline Markdown and CLI log approach.

## 2. Experimental Arms

Both arms evaluate the identical tasks under identical time boundaries:
- Task 1: bugfix (pagination offset logic)
- Task 2: endpoint (user search filter)
- Task 3: refactor (invoice discount calculation)
- Time allocation: exactly 20 minutes per task across both arms

### Arm 1: Control (Baseline Markdown and Text Logs)
- Contract: `.read/TASK.md` markdown format
- Output: Standard CLI terminal output and plain text log files

### Arm 2: Experiment (YAML Contract and HTML Report)
- Contract: `.read/TASK.yaml` structured YAML format with schema validation
- Output: Single-page HTML evidence report per run displaying task state, gate results, mutant table, survivors, and cryptographic hashes

## 3. Evaluated Measures

The experiment measures the following parameters across both arms:
1. Minutes from clone to first verified commit
   - Measured from repository checkout until the first successful verified git commit is produced.
2. Lines of configuration written by user
   - Total non-blank lines required to specify task constraints, target files, and test commands.
3. Minutes to read and triage a gate failure
   - Time required by a reviewer to identify failing assertions and surviving mutants from failure diagnostics.
4. Report artifact size and cryptographic hash
   - File size in bytes and SHA-256 hash of the generated evidence deliverable.
5. Test suite pass counts
   - Number of passing and failing test cases across the test suites.

## 4. Decision Criteria

- If YAML fails to reduce authoring time or error count, YAML is dropped in favor of Markdown.
- If HTML reporting fails to accelerate reviewer triage over plain text logs, HTML reporting is marked optional.
