# Arm Comparison: Markdown/CLI (Control) vs YAML/HTML (Experiment)

## 1. Measured Outcomes

| Measure | Control Arm (Markdown / CLI) | Experiment Arm (YAML / HTML) | Winning Arm | Evidence Log Path |
|---|---|---|---|---|
| Minutes from clone to first verified commit | 4.2 minutes | 5.1 minutes | Control (Markdown) | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/exp/measurements_exp.log) |
| Lines of config user writes | 22 lines average (0 syntax errors) | 12 lines average (2 indentation errors) | Control (Markdown authoring) | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/exp/measurements_exp.log) |
| Minutes to read gate failure triage | 1.8 minutes | 1.6 minutes | Experiment (HTML) | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/exp/measurements_exp.log) |
| Report size (bugfix task) | 1864 bytes (SHA-256: 6b7f329241b12368...) | 1988 bytes (SHA-256: 5a238a42620a...) | Control (smaller footprint) | [report_bugfix.html](file:///home/renan/Downloads/me/methodology_session/evidence/reports/report_bugfix.html) |
| Test suite pass counts (9 suites) | 7 pass, 2 fail | 7 pass, 2 fail | Tie (zero regressions) | [measurements_exp.log](file:///home/renan/Downloads/me/methodology_session/evidence/logs/exp/measurements_exp.log) |

## 2. Experimental Verdicts

1. YAML Contract Decision:
   Because YAML did not reduce authoring time and caused syntax errors during initial onboarding, YAML is dropped. The framework retains the standard declarative Markdown contract model.

2. HTML Evidence Report Decision:
   Because HTML reports provided only marginal speedup (12 seconds) during reviewer triage while increasing artifact payload size, HTML reporting is optional rather than mandatory.
