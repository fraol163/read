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
- Participants complete tasks sequentially with short rest intervals between tasks.
- If a participant reaches 20 minutes on a task without passing all tests, the task is marked as timed out.

## Cohort Profile

- 12 junior engineers.
- Experience level: 1 to 3 years of professional software development experience.
- Prerequisite: Familiarity with Git, Python, and unit testing concepts.
- Tooling background: Zero prior exposure to READ verification commands or hooks.

## Enforcement Policy

- Mode: Warning mode.
- In warning mode, pre-commit verification gates run tests and mutation checks.
- If test failures or surviving mutants occur, the gate prints detailed diagnostics and exits with return code 0.
- Commits are permitted to proceed so participants are never locked out of saving work.
- Participants are encouraged to review warning diagnostics and resolve test gaps within their 3 allowed remediation retries.

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

- Target cohort completion rate: 80% of all assigned tasks passing final test suites within the 20-minute limit.
- Warning review: Warnings emitted during commits are reviewed using the blinded review protocol in `warning_review.md`.
