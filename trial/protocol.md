# READ Framework Student Trial Protocol

## Trial Parameters

1. 20 minutes per task
2. 80% completion rate
3. 3 retries per task
4. 12 junior engineers, 1 to 3 years experience, no prior READ tooling experience
5. Warning mode

## Objectives

This trial evaluates the usability, friction, and effectiveness of the READ methodology when applied by engineers with no prior exposure to READ tooling. The trial measures task completion velocity, retry rates, and warning resolution without blocking developer workflow.

## Structure and Timing

- Three independent tasks: bugfix, endpoint, and refactor.
- Fixed time allocation of 20 minutes per task.
- Participants execute tasks sequentially with short rest intervals between tasks.
- If a participant reaches 20 minutes on a task without passing all tests, the task is marked as timed out.

## Task Success Definition

A task is defined as successfully finished when all unit tests in the assigned task test suite pass with exit code 0 on the participant's final committed code within the 20-minute time limit.

Justification: Defining completion strictly by passing the task test suite allows the trial to evaluate baseline functional correctness independently from measuring how participants interact with and remediate mutation warnings.

## Cohort Completion Threshold (80% Rule)

The 80% completion rule (recorded in [record_template.csv](file:///home/renan/Downloads/me/methodology_session/trial/record_template.csv) and [measurements.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/measurements.log)) counts PARTICIPANTS who pass all three tasks. At least 10 of 12 participants must pass all three tasks. Per-task pass rates are reported as a secondary table.

Justification: Counting participants who pass all three tasks ensures that the cohort demonstrates proficiency across bug fixing, endpoint implementation, and refactoring rather than compensating for a failed domain with multiple simple passes.

## Stop-or-Continue Rule

1. Task-Level Progression:
   - If a participant exhausts their 3-retry cap or reaches the 20-minute limit on any task without passing all tests, they must stop work on that task, record the task as failed in `record_template.csv`, and continue immediately to the next assigned task.
   - A failure or timeout on one task does not disqualify a participant from continuing to the next task.
2. Cohort-Level Evaluation:
   - If overall cohort completion (recorded in [record_template.csv](file:///home/renan/Downloads/me/methodology_session/trial/record_template.csv) and [measurements.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/measurements.log)) falls below the 80% threshold (fewer than 10 of 12 participants passing all three tasks), the trial is stopped for protocol and guidance review before running subsequent cohorts.
   - If an unrecoverable platform or environment defect occurs during a session, the trial session is stopped immediately for remediation.

## Cohort Profile

- 12 junior engineers.
- Experience level: 1 to 3 years of professional software development experience.
- Prerequisite: Familiarity with Git, Python, and unit testing concepts.
- Tooling background: Zero prior exposure to READ verification commands or hooks.

## Enforcement Policy and Mode Tamper Controls

- Mode: Warning mode.
- In warning mode, pre-commit verification gates run tests and mutation checks.
- If test failures or surviving mutants occur, the gate prints detailed diagnostics and exits with return code 0.
- Commits are permitted to proceed so participants are never locked out of saving work.
- Participants are encouraged to review warning diagnostics and resolve test gaps within their 3 allowed remediation retries.

### Controls Preventing Participant Mode Modification

To prevent participants from tampering with enforcement mode during the trial:
1. Filesystem Permission Control:
   - The `.read/` directory and `.read/config.json` file in each participant workspace are set to read-only permissions (`chmod 444 .read/config.json`) and owned by the supervisor account.
2. Automated Submission Audit Control:
   - During automated evaluation, every participant submission is checked via `trial/audit_config.sh <repo> <baseline>`.
   - The audit covers committed changes only.
   - The hook enforces the working-tree config during active development.
   - Residual Risk: A participant who owns the directory can delete and recreate `.read/config.json` in their local working tree during task work without committing the altered file.
   - Supervisor-Side Check: To mitigate this residual risk, the supervisor-side grading pipeline executes `trial/supervisor_check.sh <repo> <canonical_config>`. The script performs an automated clean clone check, verifies that `.read/config.json` matches the canonical baseline hash, and runs the test suite against the canonical configuration in an isolated sandbox. Any submission exhibiting divergent behavior or tampered configuration is rejected and flagged for manual review.

## Reviewer Protocol for Surviving Mutants

Reviewers judge only SURVIVED mutants, and only after step 3 shows real survivors on a passing baseline.
- Each SURVIVED mutant gets one label: "real test gap" or "equivalent mutant", with a one-line reason.
- If a reviewer cannot label a survivor in under two minutes, it goes into an "unclear" bucket, counted separately, and not counted as a gap.

### Task Control and Reviewer Evaluation Scope

Bugfix is a control task (all mutants killed). Reviewer labeling runs on whichever of endpoint or refactor produce survivors in step 2 or subsequent participant evaluation.

## Data Collection

- Every task attempt is logged in `record_template.csv`.
- Tracked metrics:
  - Participant identifier
  - Task identifier
  - Start timestamp and end timestamp
  - Elapsed duration in minutes
  - Number of remediation retries
  - Gate warnings emitted
  - Warnings resolved
  - Final test pass status
  - Escalation status
  - Qualitative notes

## Evaluation Criteria

- Target cohort completion rate: At least 10 of 12 participants passing all three tasks within the 20-minute limit per task. Per-task pass rates are reported as a secondary table.
- Warning review: Warnings emitted during commits are reviewed using the blinded review protocol in `warning_review.md`.

## Commit Note: Changes in fd913cb

Splitting commit fd913cb is not possible without rewriting Git history. The file breakdown for each component change in commit fd913cb is documented below:
- (a) Mutator: `read/read-core/src/mutation/mod.rs`
- (b) Binary: `read/bin/read-engine`
- (c) Supervisor check: `trial/supervisor_check.sh`
- (d) Protocol: `trial/protocol.md`
