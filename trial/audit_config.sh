#!/usr/bin/env bash
# trial/audit_config.sh: Audits .read/config.json against baseline commit
set -euo pipefail

REPO_DIR="${1:-.}"
BASELINE_COMMIT="${2:-HEAD~1}"

cd "$REPO_DIR"

if [ ! -d ".git" ]; then
    echo "Error: $REPO_DIR is not a git repository"
    exit 2
fi

# Fallback to root commit if specified baseline does not resolve
if ! git rev-parse --verify "$BASELINE_COMMIT" >/dev/null 2>&1; then
    BASELINE_COMMIT=$(git rev-list --max-parents=0 HEAD | head -n 1)
fi

echo "Auditing .read/config.json between baseline $BASELINE_COMMIT and HEAD in $REPO_DIR..."

CONFIG_DIFF=$(git diff "$BASELINE_COMMIT" HEAD -- .read/config.json 2>/dev/null || true)

if [ -n "$CONFIG_DIFF" ]; then
    echo "[AUDIT FAILED] Unauthorized modification detected in .read/config.json:"
    echo "$CONFIG_DIFF"
    exit 1
fi

echo "[AUDIT PASSED] .read/config.json is unchanged from baseline."
exit 0
