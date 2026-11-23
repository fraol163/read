#!/bin/sh
# ==============================================================================
# READ Framework Packaging & Installation Verification Test
# Tests:
#   1. Standalone install.sh in clean isolated sandbox directory
#   2. Binary installation: read, read-engine, read-daemon, read-cli
#   3. Configuration setup: ~/.config/read/config.json
#   4. Immediate CLI execution: read --version, read --help, read config
#   5. PATH availability & fresh user workflow
#   6. Makefile targets: build, test, install, verify, package, uninstall
#
# IMPORTANT: Zero emojis are used in this script, logs, or outputs.
# ==============================================================================

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

TEST_SANDBOX="$(mktemp -d /tmp/read-pkg-test-XXXXXX)"
cleanup() {
    chmod -R +w "$TEST_SANDBOX" 2>/dev/null || true
    rm -rf "$TEST_SANDBOX"
}
trap cleanup EXIT

echo "================================================================================"
echo "STARTING PACKAGING & INSTALLATION VERIFICATION TEST"
echo "Test Sandbox: $TEST_SANDBOX"
echo "Framework Root: $REPO_ROOT"
echo "================================================================================"

# 1. Setup isolated user environment
ORIGINAL_HOME="$HOME"
MOCK_HOME="$TEST_SANDBOX/mock_home"
MOCK_PREFIX="$MOCK_HOME/.local"
MOCK_BIN="$MOCK_PREFIX/bin"
MOCK_CONFIG="$MOCK_HOME/.config/read"
mkdir -p "$MOCK_HOME"

export RUSTUP_HOME="${RUSTUP_HOME:-$ORIGINAL_HOME/.rustup}"
export CARGO_HOME="${CARGO_HOME:-$ORIGINAL_HOME/.cargo}"
export GOPATH="${GOPATH:-$ORIGINAL_HOME/go}"
export HOME="$MOCK_HOME"
export PATH="$MOCK_BIN:$PATH"
export XDG_CONFIG_HOME="$MOCK_HOME/.config"

# 2. Test install.sh prerequisites check
echo "[TEST 1] Running installer prerequisite check..."
"$REPO_ROOT/install.sh" --check-only > "$TEST_SANDBOX/check_only.log"
if ! grep -q "All toolchain prerequisites satisfied" "$TEST_SANDBOX/check_only.log"; then
    echo "[FAIL] Prerequisite check failed" >&2
    exit 1
fi
echo "[PASS] 1. Prerequisites check detected all toolchains"

# 3. Test install.sh dry-run
echo "[TEST 2] Running installer dry-run..."
"$REPO_ROOT/install.sh" --dry-run --prefix="$MOCK_PREFIX" > "$TEST_SANDBOX/dry_run.log"
if ! grep -q "\[DRY-RUN\] Would install binaries" "$TEST_SANDBOX/dry_run.log"; then
    echo "[FAIL] Dry-run failed" >&2
    exit 1
fi
if [ -d "$MOCK_BIN" ]; then
    echo "[FAIL] Dry-run created directories unexpectedly" >&2
    exit 1
fi
echo "[PASS] 2. Dry-run executed safely without side effects"

# 4. Test full installation via install.sh
echo "[TEST 3] Running full install.sh into isolated prefix..."
"$REPO_ROOT/install.sh" --batch --prefix="$MOCK_PREFIX" --config-dir="$MOCK_CONFIG" > "$TEST_SANDBOX/install.log"

# Assert binaries exist and are executable
for bin in read read-engine read-daemon read-cli; do
    if [ ! -x "$MOCK_BIN/$bin" ]; then
        echo "[FAIL] Installed binary missing or not executable: $MOCK_BIN/$bin" >&2
        exit 1
    fi
done
echo "[PASS] 3. All binaries (read, read-engine, read-daemon, read-cli) installed and executable"

# Assert default configuration was created
if [ ! -f "$MOCK_CONFIG/config.json" ]; then
    echo "[FAIL] Configuration file not initialized: $MOCK_CONFIG/config.json" >&2
    exit 1
fi
if ! grep -q '"strict_cas": true' "$MOCK_CONFIG/config.json"; then
    echo "[FAIL] Configuration missing required strict_cas key" >&2
    exit 1
fi
echo "[PASS] 4. Default configuration initialized: $MOCK_CONFIG/config.json"

# 5. Test CLI execution from installed binaries
echo "[TEST 5] Testing CLI flags (--version, --help, config)..."
"$MOCK_BIN/read" --version > "$TEST_SANDBOX/ver.log"
if ! grep -q "READ Framework Unified CLI" "$TEST_SANDBOX/ver.log"; then
    echo "[FAIL] read --version failed" >&2
    exit 1
fi

"$MOCK_BIN/read" --help > "$TEST_SANDBOX/help.log"
if ! grep -q "Resource-conscious, Evidence-driven AI Development" "$TEST_SANDBOX/help.log"; then
    echo "[FAIL] read --help failed" >&2
    exit 1
fi

"$MOCK_BIN/read-cli" --version > /dev/null
"$MOCK_BIN/read-cli" --help > /dev/null
echo "[PASS] 5. Installed CLI verified: --version and --help return exit code 0"

# 6. Test fresh user workspace workflow with installed CLI
echo "[TEST 6] Testing fresh project workflow using installed CLI..."
MOCK_PROJECT="$TEST_SANDBOX/fresh_project"
mkdir -p "$MOCK_PROJECT"
(
    cd "$MOCK_PROJECT"
    git init -q
    git config user.name "Fresh User"
    git config user.email "user@test.org"

    "$MOCK_BIN/read" init
    if [ ! -d ".read" ] || [ ! -f ".read/STATE.json" ]; then
        echo "[FAIL] Installed read CLI failed to initialize project" >&2
        exit 1
    fi
    if [ ! -d ".read" ] || [ ! -f ".read/TASK.md" ] || [ ! -x ".read/hooks/pre-commit" ]; then
        echo "[FAIL] Installed read init failed to deploy .read/ declarative contracts" >&2
        exit 1
    fi

    STATUS="$("$MOCK_BIN/read" status)"
    if ! echo "$STATUS" | grep -q '"status": "idle"'; then
        echo "[FAIL] Initial status not idle" >&2
        exit 1
    fi
)
echo "[PASS] 6. Fresh user successfully initialized and queried project with installed CLI"

# 7. Test configuration preservation (idempotent installation)
echo "[TEST 7] Testing configuration preservation on re-installation..."
echo '{"custom_marker": "preserve_test"}' > "$MOCK_CONFIG/config.json"
"$REPO_ROOT/install.sh" --no-build --prefix="$MOCK_PREFIX" --config-dir="$MOCK_CONFIG" > /dev/null
if ! grep -q "preserve_test" "$MOCK_CONFIG/config.json"; then
    echo "[FAIL] Existing user configuration was clobbered by installer" >&2
    exit 1
fi
echo "[PASS] 7. Existing user configuration strictly preserved on re-installation"

# 8. Test Makefile targets
echo "[TEST 8] Testing Makefile targets (verify, package, uninstall)..."
(
    cd "$REPO_ROOT"
    make verify > /dev/null
    make package > /dev/null
    if [ ! -f dist/read-v0.7.0-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m).tar.gz ]; then
        echo "[FAIL] make package did not create expected tarball" >&2
        exit 1
    fi

    # Test Makefile install into custom DESTDIR
    MAKE_DEST="$TEST_SANDBOX/make_dest"
    make install DESTDIR="$MAKE_DEST" PREFIX="/usr" > /dev/null
    if [ ! -x "$MAKE_DEST/usr/bin/read" ] || [ ! -f "$MAKE_DEST/etc/read/config.json" ]; then
        echo "[FAIL] make install with DESTDIR failed" >&2
        exit 1
    fi

    # Test Makefile uninstall
    make uninstall DESTDIR="$MAKE_DEST" PREFIX="/usr" > /dev/null
    if [ -f "$MAKE_DEST/usr/bin/read" ]; then
        echo "[FAIL] make uninstall failed to remove binary" >&2
        exit 1
    fi
)
echo "[PASS] 8. Makefile targets (verify, package, install with DESTDIR, uninstall) verified"

echo "================================================================================"
echo "ALL PACKAGING AND INSTALLATION VERIFICATION TESTS PASSED SUCCESSFULLY"
echo "================================================================================"
