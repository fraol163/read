# Arm Comparison: Markdown/CLI (Control) vs YAML/HTML (Experiment)

## 1. Measured Outcomes

| Measure | Control Value | Experiment Value | Evidence Log Path | Status |
|---|---|---|---|---|
| Minutes from clone to first verified commit | NOT MEASURED | NOT MEASURED | [authoring_timing.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/authoring_timing.log) | NOT MEASURED |
| Lines of config written (bugfix task) | 16 lines | 10 lines | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) | MEASURED |
| Minutes to read gate failure triage | NOT MEASURED | NOT MEASURED | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) | NOT MEASURED |
| Report artifact size and hash (bugfix) | NOT MEASURED | 2386 bytes (SHA-256: eb1770385696...) | [report_bugfix.html](file:///home/renan/Downloads/me/methodology_session_exp/evidence/reports/report_bugfix.html) | MEASURED |
| Test suite pass counts (9 suites) | 9 passed, 0 failed | 9 passed, 0 failed | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) | MEASURED |

## 2. Experimental Verdicts

1. Configuration Format Evaluation:
   YAML contracts required 10 lines compared to 16 lines for Markdown on the evaluated bugfix task. Fewer lines of configuration alone does not prove reduced authoring friction or lower error rates without participant trial data. Therefore no winner is declared on lines of config alone. Both Markdown and YAML formats remain supported by the contract validator.

2. Evidence Report Evaluation:
   Because gate failure triage duration was NOT MEASURED, HTML evidence report generation is optional rather than mandatory. Both arms achieve parity across all 9 test suites with 0 failures.
