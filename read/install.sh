#!/bin/sh
# ==============================================================================
# READ Framework Standalone POSIX Installer (v0.7.0)
# Resource-conscious, Evidence-driven AI Development
#
# Supported Systems: Arch Linux, Debian/Ubuntu, Fedora/RHEL, macOS, Termux, POSIX
# ==============================================================================

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="0.7.0"

# Defaults
DEFAULT_PREFIX=""
DEFAULT_CONFIG_DIR=""
if [ "$(id -u)" -eq 0 ]; then
    DEFAULT_PREFIX="/usr/local"
    DEFAULT_CONFIG_DIR="/etc/read"
elif [ -n "${TERMUX_VERSION:-}" ] && [ -n "${PREFIX:-}" ]; then
    DEFAULT_PREFIX="$PREFIX"
    DEFAULT_CONFIG_DIR="$PREFIX/etc/read"
else
    DEFAULT_PREFIX="${HOME}/.local"
    if [ -n "${XDG_CONFIG_HOME:-}" ]; then
        DEFAULT_CONFIG_DIR="${XDG_CONFIG_HOME}/read"
    else
        DEFAULT_CONFIG_DIR="${HOME}/.config/read"
    fi
fi

PREFIX="${PREFIX:-$DEFAULT_PREFIX}"
BINDIR=""
CONFIG_DIR="${CONFIG_DIR:-$DEFAULT_CONFIG_DIR}"
NO_BUILD=0
FORCE_CONFIG=0
DRY_RUN=0
CHECK_ONLY=0
DO_SETUP=""

# Helper logging
log_info() {
    printf "[INFO] %s\n" "$*"
}

log_warn() {
    printf "[WARN] %s\n" "$*" >&2
}

log_error() {
    printf "[ERROR] %s\n" "$*" >&2
}

print_help() {
    cat << EOF
READ Framework Standalone Installer (v${VERSION})
Resource-conscious, Evidence-driven AI Development

USAGE:
    ./install.sh [OPTIONS]

OPTIONS:
    --prefix=DIR        Base installation prefix (default: ${DEFAULT_PREFIX})
    --bin-dir=DIR       Binary target directory (default: PREFIX/bin)
    --config-dir=DIR    Configuration directory (default: ~/.config/read)
    --no-build          Skip binary compilation (use existing binaries in bin/)
    --check-only        Verify toolchain prerequisites and exit
    -f, --force         Overwrite existing configuration file
    -n, --dry-run       Print installation actions without modifying system
    -h, --help          Show this help message

ENVIRONMENT VARIABLES:
    PREFIX              Installation prefix
    BINDIR              Binary target directory
    XDG_CONFIG_HOME     Config base directory
EOF
}

# Parse command line arguments
for arg in "$@"; do
    case "$arg" in
        --prefix=*)
            PREFIX="${arg#*=}"
            ;;
        --bin-dir=*)
            BINDIR="${arg#*=}"
            ;;
        --config-dir=*)
            CONFIG_DIR="${arg#*=}"
            ;;
        --no-build)
            NO_BUILD=1
            ;;
        --check-only)
            CHECK_ONLY=1
            ;;
        -f|--force)
            FORCE_CONFIG=1
            ;;
        -n|--dry-run)
            DRY_RUN=1
            ;;
        -s|--setup)
            DO_SETUP=1
            ;;
        --no-setup|--batch|-y)
            DO_SETUP=0
            ;;
        -h|--help)
            print_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $arg"
            print_help
            exit 1
            ;;
    esac
done

if [ -z "$BINDIR" ]; then
    BINDIR="${PREFIX}/bin"
fi

if [ -z "$CONFIG_DIR" ]; then
    if [ "$PREFIX" = "/usr" ] || [ "$PREFIX" = "/usr/local" ]; then
        CONFIG_DIR="/etc/read"
    elif [ -n "${XDG_CONFIG_HOME:-}" ]; then
        CONFIG_DIR="${XDG_CONFIG_HOME}/read"
    else
        CONFIG_DIR="${HOME}/.config/read"
    fi
fi

# Detect OS and distribution
detect_platform() {
    OS_NAME="$(uname -s)"
    DISTRO="unknown"

    if [ -n "${TERMUX_VERSION:-}" ]; then
        DISTRO="termux"
    elif [ "$OS_NAME" = "Darwin" ]; then
        DISTRO="macos"
    elif [ "$OS_NAME" = "Linux" ]; then
        if [ -f /etc/os-release ]; then
            # Parse ID from os-release without sourcing
            DISTRO_ID="$(grep '^ID=' /etc/os-release | head -n1 | cut -d= -f2 | tr -d '"' | tr -d "'")"
            DISTRO_LIKE="$(grep '^ID_LIKE=' /etc/os-release | head -n1 | cut -d= -f2 | tr -d '"' | tr -d "'")"
            case "$DISTRO_ID" in
                arch|manjaro|endeavouros) DISTRO="arch" ;;
                ubuntu|debian|pop|mint)   DISTRO="debian" ;;
                fedora|rhel|centos|rocky|alma) DISTRO="fedora" ;;
                *)
                    case "$DISTRO_LIKE" in
                        *arch*)   DISTRO="arch" ;;
                        *debian*) DISTRO="debian" ;;
                        *fedora*) DISTRO="fedora" ;;
                        *) DISTRO="generic_linux" ;;
                    esac
                    ;;
            esac
        elif [ -f /etc/arch-release ]; then
            DISTRO="arch"
        elif [ -f /etc/debian_version ]; then
            DISTRO="debian"
        elif [ -f /etc/redhat-release ]; then
            DISTRO="fedora"
        else
            DISTRO="generic_linux"
        fi
    else
        DISTRO="generic_posix"
    fi

    echo "$DISTRO"
}

# Check prerequisite toolchains
check_toolchain() {
    MISSING_COUNT=0
    DISTRO="$(detect_platform)"

    log_info "Detecting system toolchain prerequisites..."

    # Check rustc
    if command -v rustc >/dev/null 2>&1; then
        RUSTC_VER="$(rustc --version 2>/dev/null | head -n1)"
        printf "  [FOUND] rustc:   %s\n" "$RUSTC_VER"
    else
        printf "  [MISSING] rustc\n"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check cargo
    if command -v cargo >/dev/null 2>&1; then
        CARGO_VER="$(cargo --version 2>/dev/null | head -n1)"
        printf "  [FOUND] cargo:   %s\n" "$CARGO_VER"
    else
        printf "  [MISSING] cargo\n"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check go
    if command -v go >/dev/null 2>&1; then
        GO_VER="$(go version 2>/dev/null | head -n1)"
        printf "  [FOUND] go:      %s\n" "$GO_VER"
    else
        printf "  [MISSING] go\n"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check git
    if command -v git >/dev/null 2>&1; then
        GIT_VER="$(git --version 2>/dev/null | head -n1)"
        printf "  [FOUND] git:     %s\n" "$GIT_VER"
    else
        printf "  [MISSING] git\n"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check python3
    if command -v python3 >/dev/null 2>&1; then
        PYTHON_VER="$(python3 --version 2>/dev/null | head -n1)"
        printf "  [FOUND] python3: %s\n" "$PYTHON_VER"
    else
        printf "  [MISSING] python3\n"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    if [ "$MISSING_COUNT" -gt 0 ]; then
        echo ""
        log_error "Missing $MISSING_COUNT required toolchain component(s) for building READ framework."
        log_error "Detected platform: $DISTRO"
        echo ""
        log_info "Install missing tools with the appropriate package command:"
        case "$DISTRO" in
            arch)
                printf "    sudo pacman -S --needed rust go git python\n"
                ;;
            debian)
                printf "    sudo apt-get update && sudo apt-get install -y cargo rustc golang git python3\n"
                ;;
            fedora)
                printf "    sudo dnf install -y rust cargo golang git python3\n"
                ;;
            macos)
                printf "    brew install rust go git python3\n"
                ;;
            termux)
                printf "    pkg update && pkg install -y rust golang git python\n"
                ;;
            *)
                printf "    Install Rust:   curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh\n"
                printf "    Install Go:     https://go.dev/dl/\n"
                printf "    Install Git:    https://git-scm.com/\n"
                printf "    Install Python: https://www.python.org/\n"
                ;;
        esac
        echo ""
        return 1
    fi

    log_info "All toolchain prerequisites satisfied."
    return 0
}

# Compile engine and daemon
build_binaries() {
    log_info "Building READ native engine (read-core)..."
    (
        cd "${SCRIPT_DIR}/read-core"
        cargo build --release
    )

    TARGET_DIR="$(cargo metadata --format-version 1 --manifest-path "${SCRIPT_DIR}/read-core/Cargo.toml" --no-deps 2>/dev/null | sed -n 's/.*"target_directory":"\([^"]*\)".*/\1/p' || true)"
    ENGINE_SRC=""
    if [ -n "$TARGET_DIR" ] && [ -f "${TARGET_DIR}/release/read-engine" ]; then
        ENGINE_SRC="${TARGET_DIR}/release/read-engine"
    elif [ -f "${SCRIPT_DIR}/read-core/target/release/read-engine" ]; then
        ENGINE_SRC="${SCRIPT_DIR}/read-core/target/release/read-engine"
    else
        log_error "Compiled read-engine binary not found."
        exit 1
    fi

    mkdir -p "${SCRIPT_DIR}/bin"
    cp "$ENGINE_SRC" "${SCRIPT_DIR}/bin/read-engine"
    chmod +x "${SCRIPT_DIR}/bin/read-engine"

    log_info "Building READ host daemon (read-daemon)..."
    (
        cd "${SCRIPT_DIR}/read-daemon"
        go build -o "${SCRIPT_DIR}/bin/read-daemon" ./cmd/read-daemon
    )
    chmod +x "${SCRIPT_DIR}/bin/read-daemon"
    chmod +x "${SCRIPT_DIR}/bin/read"

    log_info "Build successful: bin/read, bin/read-engine, bin/read-daemon"
}

# Verify binary integrity
verify_installation() {
    TARGET_READ="${BINDIR}/read"
    log_info "Verifying installed CLI..."
    if [ ! -x "$TARGET_READ" ]; then
        log_error "Installed binary not found or not executable: $TARGET_READ"
        exit 1
    fi

    CLI_VER="$("$TARGET_READ" --version 2>/dev/null | head -n1)"
    log_info "Verification passed: $CLI_VER"

    # Check PATH
    PATH_FOUND=0
    OLD_IFS="$IFS"
    IFS=":"
    for p in $PATH; do
        if [ "$p" = "$BINDIR" ]; then
            PATH_FOUND=1
            break
        fi
    done
    IFS="$OLD_IFS"

    echo ""
    log_info "Shell integration notes:"
    log_info "1. Binary alias 'read-cli' is also symlinked to '$BINDIR/read-cli'."
    log_info "2. 'read' is a shell builtin in bash/zsh. To invoke 'read' directly as the CLI,"
    log_info "   add the builtin override to your shell profile:"

    USER_SHELL="$(basename "${SHELL:-sh}")"
    case "$USER_SHELL" in
        bash)
            printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.bashrc\n" "$BINDIR"
            printf "    echo 'enable -n read 2>/dev/null || true' >> ~/.bashrc\n"
            printf "    source ~/.bashrc\n"
            ;;
        zsh)
            printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.zshrc\n" "$BINDIR"
            printf "    echo 'disable -b read 2>/dev/null || true' >> ~/.zshrc\n"
            printf "    source ~/.zshrc\n"
            ;;
        fish)
            printf "    fish_add_path %s\n" "$BINDIR"
            ;;
        *)
            printf "    echo 'export PATH=\"%s:\$PATH\"' >> ~/.profile\n" "$BINDIR"
            printf "    . ~/.profile\n"
            ;;
    esac
    echo ""

    if [ "$PATH_FOUND" -eq 1 ]; then
        log_info "Binary directory '$BINDIR' is present in current PATH."
    else
        log_warn "Binary directory '$BINDIR' is NOT yet in current PATH."
    fi
}

# Main installer flow
main() {
    log_info "================================================================"
    log_info "READ Framework Standalone Installer (v${VERSION})"
    log_info "Resource-conscious, Evidence-driven AI Development"
    log_info "================================================================"
    log_info "Target binary directory: ${BINDIR}"
    log_info "Target config directory: ${CONFIG_DIR}"

    # Check prerequisites
    if ! check_toolchain; then
        exit 1
    fi

    if [ "$CHECK_ONLY" -eq 1 ]; then
        log_info "Prerequisites check complete (--check-only). Exiting."
        exit 0
    fi

    # Compile binaries if requested
    if [ "$NO_BUILD" -eq 0 ]; then
        if [ "$DRY_RUN" -eq 1 ]; then
            log_info "[DRY-RUN] Would build read-engine and read-daemon release binaries"
        else
            build_binaries
        fi
    else
        log_info "Skipping build stage (--no-build). Verifying existing binaries in bin/..."
        for b in read read-engine read-daemon; do
            if [ ! -f "${SCRIPT_DIR}/bin/$b" ]; then
                log_error "Binary '${SCRIPT_DIR}/bin/$b' missing. Run without --no-build."
                exit 1
            fi
        done
    fi

    # Perform filesystem installation
    if [ "$DRY_RUN" -eq 1 ]; then
        log_info "[DRY-RUN] Would create directory: $BINDIR"
        log_info "[DRY-RUN] Would install binaries: read, read-engine, read-daemon to $BINDIR"
        log_info "[DRY-RUN] Would create directory: $CONFIG_DIR"
        log_info "[DRY-RUN] Would install config to: $CONFIG_DIR/config.json"
        exit 0
    fi

    log_info "Installing binaries into ${BINDIR}..."
    mkdir -p "${BINDIR}"
    cp "${SCRIPT_DIR}/bin/read" "${BINDIR}/read"
    cp "${SCRIPT_DIR}/bin/read-engine" "${BINDIR}/read-engine"
    cp "${SCRIPT_DIR}/bin/read-daemon" "${BINDIR}/read-daemon"
    chmod 0755 "${BINDIR}/read" "${BINDIR}/read-engine" "${BINDIR}/read-daemon"
    ln -sf read "${BINDIR}/read-cli"

    DATADIR="${PREFIX}/share/read"
    log_info "Installing declarative template into ${DATADIR}/template..."
    mkdir -p "${DATADIR}/template"
    cp -r "${SCRIPT_DIR}/template/.read" "${DATADIR}/template/"

    log_info "Initializing configuration in ${CONFIG_DIR}..."
    mkdir -p "${CONFIG_DIR}"
    CONFIG_FILE="${CONFIG_DIR}/config.json"
    if [ ! -f "$CONFIG_FILE" ] || [ "$FORCE_CONFIG" -eq 1 ]; then
        cp "${SCRIPT_DIR}/config.default.json" "$CONFIG_FILE"
        chmod 0644 "$CONFIG_FILE"
        log_info "Configuration initialized: $CONFIG_FILE"
    else
        log_info "Configuration exists and preserved: $CONFIG_FILE"
    fi

    # Verification
    verify_installation

    # Interactive BMAD-style workspace setup offer
    if [ "$DO_SETUP" = "1" ] || ( [ -z "$DO_SETUP" ] && [ -t 0 ] && [ -t 1 ] && [ -z "${CI:-}" ] && [ "${READ_NON_INTERACTIVE:-0}" != "1" ] && [ "$DRY_RUN" -eq 0 ] && [ "$CHECK_ONLY" -eq 0 ] ); then
        if [ "$DO_SETUP" = "1" ]; then
            "${BINDIR}/read" setup
        else
            echo ""
            printf "Would you like to initialize READ methodology in current workspace (%s)? [Y/n]: " "$(pwd)"
            read -r init_ws || true
            case "$init_ws" in
                [nN]*)
                    ;;
                *)
                    "${BINDIR}/read" setup
                    ;;
            esac
        fi
    fi

    log_info "================================================================"
    log_info "READ Framework successfully installed!"
    log_info "Execute 'read --help' to begin development."
    log_info "================================================================"
}

main "$@"
