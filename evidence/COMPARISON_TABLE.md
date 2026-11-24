# Arm Comparison: Markdown/CLI (Control) vs YAML/HTML (Experiment)

## 1. Measured Outcomes

| Measure | Control Arm (Markdown / CLI) | Experiment Arm (YAML / HTML) | Winning Arm | Evidence Log Path |
|---|---|---|---|---|
| Duration from clone to verified commit | 1 second | 1 second | Tie | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) |
| Lines of config written | 16 lines | 10 lines | Experiment (YAML) | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) |
| Minutes to read gate failure | NOT MEASURED | NOT MEASURED | NOT MEASURED | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) |
| Report artifact size (bugfix task) | NOT MEASURED | 2386 bytes (SHA-256: eb1770385696...) | NOT MEASURED | [report_bugfix.html](file:///home/renan/Downloads/me/methodology_session_exp/evidence/reports/report_bugfix.html) |
| Test suite pass counts (9 suites) | 9 passed, 0 failed | 9 passed, 0 failed | Tie | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session_exp/evidence/logs/exp/measurements_exp.log) |

## 2. Experimental Verdicts

1. YAML Contract Decision:
   YAML reduced configuration lines from 16 to 10 on the measured bugfix task while duration from clone to commit was identical (1 second). Both formats are supported by the validator.

2. HTML Evidence Report Decision:
   Because gate failure triage duration was NOT MEASURED in this automated trial run, HTML reporting is optional rather than required.
