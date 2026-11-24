#!/bin/sh
# ──────────────────────────────────────────────────────────────────────────────
# READ Framework Standalone POSIX Installer (v0.1.0)
# Resource-conscious, Evidence-driven AI Development
#
# Supported Systems: Arch Linux, Debian/Ubuntu, Fedora/RHEL, macOS, Windows (WSL2), Termux, POSIX
# ──────────────────────────────────────────────────────────────────────────────

set -eu

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="0.1.0"

# Defaults
DEFAULT_PREFIX=""
DEFAULT_CONFIG_DIR=""
if [ "$(id -u)" -eq 0 ]
then
    DEFAULT_PREFIX="/usr/local"
    DEFAULT_CONFIG_DIR="/etc/read"
elif [ -n "${TERMUX_VERSION:-}" ] && [ -n "${PREFIX:-}" ]
then
    DEFAULT_PREFIX="$PREFIX"
    DEFAULT_CONFIG_DIR="$PREFIX/etc/read"
else
    DEFAULT_PREFIX="${HOME}/.local"
    if [ -n "${XDG_CONFIG_HOME:-}" ]
    then
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
PREBUILT=0

# Formatting helpers (respect non-interactive / non-tty environments)
if [ -t 1 ]
then
    C_BOLD="\033[1m"
    C_CYAN="\033[36m"
    C_GREEN="\033[32m"
    C_YELLOW="\033[33m"
    C_RED="\033[31m"
    C_RESET="\033[0m"
else
    C_BOLD=""
    C_CYAN=""
    C_GREEN=""
    C_YELLOW=""
    C_RED=""
    C_RESET=""
fi

log_banner() {
    printf "%b" "$C_CYAN"
    cat << "EOF"
    ____  _________    ____ 
   / __ \/ ____/   |  / __ \
  / /_/ / __/ / /| | / / / /
 / _, _/ /___/ ___ |/ /_/ / 
/_/ |_/_____/_/  |_/_____/  
EOF
    printf "%b" "$C_RESET"
    printf "%bREAD Framework Standalone Installer (v%s)%b\n" "$C_BOLD" "$VERSION" "$C_RESET"
    printf "Resource-conscious, Evidence-driven AI Development\n"
    printf "────────────────────────────────────────────────────────────────\n"
}

log_step() {
    printf "\n%b[%s]%b %b%s%b\n" "$C_GREEN" "$1" "$C_RESET" "$C_BOLD" "$2" "$C_RESET"
}

log_info() {
    printf "  [INFO] %s\n" "$*"
}

log_found() {
    printf "  [FOUND] %-8s %s\n" "$1:" "$2"
}

log_missing() {
    printf "  [MISSING] %s\n" "$*"
}

log_ok() {
    printf "  [OK] %s\n" "$*"
}

log_warn() {
    printf "  %b[WARN]%b %s\n" "$C_YELLOW" "$C_RESET" "$*" >&2
}

log_error() {
    printf "  %b[ERROR]%b %s\n" "$C_RED" "$C_RESET" "$*" >&2
}

print_help() {
    cat << EOF
READ Framework Standalone Installer (v${VERSION})
Resource-conscious, Evidence-driven AI Development

USAGE:
    ./install.sh [OPTIONS]
    curl -fsSL https://raw.githubusercontent.com/fraol163/read/main/install.sh | bash

OPTIONS:
    --prefix=DIR        Base installation prefix (default: ${DEFAULT_PREFIX})
    --bin-dir=DIR       Binary target directory (default: PREFIX/bin)
    --config-dir=DIR    Configuration directory (default: ~/.config/read)
    --prebuilt          Prefer downloading pre-compiled binaries if available
    --no-build          Skip compilation (use existing binaries in bin/)
    --check-only        Verify toolchain prerequisites and exit
    -f, --force         Overwrite existing configuration file
    -n, --dry-run       Print installation actions without modifying system
    -s, --setup         Initialize READ methodology in current workspace
    -y, --batch         Non-interactive mode (skip interactive prompts)
    -h, --help          Show this help message

PLATFORMS:
    Linux (x86_64, aarch64), macOS (Darwin), Windows (WSL2)
EOF
}

# Parse command line arguments
for arg in "$@"
do
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
        --prebuilt)
            PREBUILT=1
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

if [ -z "$BINDIR" ]
then
    BINDIR="${PREFIX}/bin"
fi

if [ -z "$CONFIG_DIR" ]
then
    if [ "$PREFIX" = "/usr" ] || [ "$PREFIX" = "/usr/local" ]
    then
        CONFIG_DIR="/etc/read"
    elif [ -n "${XDG_CONFIG_HOME:-}" ]
    then
        CONFIG_DIR="${XDG_CONFIG_HOME}/read"
    else
        CONFIG_DIR="${HOME}/.config/read"
    fi
fi

# Detect OS and distribution
detect_platform() {
    OS_NAME="$(uname -s)"
    DISTRO="unknown"

    if [ -n "${TERMUX_VERSION:-}" ]
    then
        DISTRO="termux"
    elif [ "$OS_NAME" = "Darwin" ]
    then
        DISTRO="macos"
    elif echo "$OS_NAME" | grep -qE "MINGW|MSYS|CYGWIN"
    then
        DISTRO="windows_posix"
    elif [ "$OS_NAME" = "Linux" ]
    then
        if [ -f /etc/os-release ]
        then
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
        elif [ -f /etc/arch-release ]
        then
            DISTRO="arch"
        elif [ -f /etc/debian_version ]
        then
            DISTRO="debian"
        elif [ -f /etc/redhat-release ]
        then
            DISTRO="fedora"
        else
            DISTRO="generic_linux"
        fi
    else
        DISTRO="generic_posix"
    fi

    echo "$DISTRO"
}

# Resolve source directory (works locally and via curl | bash pipe)
SRC_DIR="$SCRIPT_DIR"
TEMP_DIR=""

cleanup_temp() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]
    then
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup_temp EXIT INT TERM

resolve_source() {
    if [ -d "${SRC_DIR}/read-core" ] && [ -d "${SRC_DIR}/read-daemon" ]
    then
        return 0
    fi

    log_info "Installer executed outside source repository. Retrieving framework source..."
    TEMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'read-install')"

    if command -v git >/dev/null 2>&1
    then
        log_info "Cloning shallow repository from https://github.com/fraol163/read.git..."
        if git clone --depth 1 -q https://github.com/fraol163/read.git "$TEMP_DIR"
        then
            SRC_DIR="$TEMP_DIR"
            log_ok "Framework source fetched successfully into temporary workspace."
            return 0
        fi
    fi

    if command -v curl >/dev/null 2>&1
    then
        log_info "Downloading source tarball from GitHub..."
        if curl -fsSL https://github.com/fraol163/read/archive/refs/heads/main.tar.gz | tar -xz -C "$TEMP_DIR" --strip-components=1 2>/dev/null
        then
            SRC_DIR="$TEMP_DIR"
            log_ok "Framework archive downloaded and unpacked successfully."
            return 0
        fi
    fi

    if command -v wget >/dev/null 2>&1
    then
        log_info "Downloading source tarball via wget..."
        if wget -qO- https://github.com/fraol163/read/archive/refs/heads/main.tar.gz | tar -xz -C "$TEMP_DIR" --strip-components=1 2>/dev/null
        then
            SRC_DIR="$TEMP_DIR"
            log_ok "Framework archive downloaded and unpacked successfully."
            return 0
        fi
    fi

    log_error "Failed to retrieve READ framework source automatically."
    log_error "Please clone the repository manually:"
    log_error "  git clone https://github.com/fraol163/read.git && cd read && ./install.sh"
    exit 1
}

# Check prerequisite toolchains
check_toolchain() {
    MISSING_COUNT=0
    DISTRO="$(detect_platform)"

    log_info "Detecting system toolchain prerequisites..."

    # Check rustc
    if command -v rustc >/dev/null 2>&1
    then
        RUSTC_VER="$(rustc --version 2>/dev/null | head -n1)"
        log_found "rustc" "$RUSTC_VER"
    else
        log_missing "rustc"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check cargo
    if command -v cargo >/dev/null 2>&1
    then
        CARGO_VER="$(cargo --version 2>/dev/null | head -n1)"
        log_found "cargo" "$CARGO_VER"
    else
        log_missing "cargo"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check go
    if command -v go >/dev/null 2>&1
    then
        GO_VER="$(go version 2>/dev/null | head -n1)"
        log_found "go" "$GO_VER"
    else
        log_missing "go"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check git
    if command -v git >/dev/null 2>&1
    then
        GIT_VER="$(git --version 2>/dev/null | head -n1)"
        log_found "git" "$GIT_VER"
    else
        log_missing "git"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    # Check python3
    if command -v python3 >/dev/null 2>&1
    then
        PYTHON_VER="$(python3 --version 2>/dev/null | head -n1)"
        log_found "python3" "$PYTHON_VER"
    else
        log_missing "python3"
        MISSING_COUNT=$((MISSING_COUNT + 1))
    fi

    if [ "$MISSING_COUNT" -gt 0 ]
    then
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
            windows_posix)
                printf "    winget install Rustlang.Rustup GoLang.Go Git.Git Python.Python.3\n"
                printf "    Or install in WSL2: wsl --install -d Ubuntu\n"
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
        cd "${SRC_DIR}/read-core"
        cargo build --release
    )

    TARGET_DIR="$(cargo metadata --format-version 1 --manifest-path "${SRC_DIR}/read-core/Cargo.toml" --no-deps 2>/dev/null | sed -n 's/.*"target_directory":"\([^"]*\)".*/\1/p' || true)"
    ENGINE_SRC=""
    if [ -n "$TARGET_DIR" ] && [ -f "${TARGET_DIR}/release/read-engine" ]
    then
        ENGINE_SRC="${TARGET_DIR}/release/read-engine"
    elif [ -f "${SRC_DIR}/read-core/target/release/read-engine" ]
    then
        ENGINE_SRC="${SRC_DIR}/read-core/target/release/read-engine"
    else
        log_error "Compiled read-engine binary not found."
        exit 1
    fi

    mkdir -p "${SRC_DIR}/bin"
    cp "$ENGINE_SRC" "${SRC_DIR}/bin/read-engine"
    chmod +x "${SRC_DIR}/bin/read-engine"

    log_info "Building READ host daemon (read-daemon)..."
    (
        cd "${SRC_DIR}/read-daemon"
        go build -trimpath -o "${SRC_DIR}/bin/read-daemon" cmd/read-daemon/main.go
    )
    chmod +x "${SRC_DIR}/bin/read-daemon"
    chmod +x "${SRC_DIR}/bin/read"
    if [ -f "${SRC_DIR}/bin/read-mcp" ]
    then
        chmod +x "${SRC_DIR}/bin/read-mcp"
    fi
    if [ -f "${SRC_DIR}/bin/read-report" ]
    then
        chmod +x "${SRC_DIR}/bin/read-report"
    fi

    log_info "Build successful: bin/read, bin/read-engine, bin/read-daemon"
}

# Verify binary integrity
verify_installation() {
    TARGET_READ="${BINDIR}/read"
    log_info "Verifying installed CLI..."
    if [ ! -x "$TARGET_READ" ]
    then
        log_error "Installed binary not found or not executable: $TARGET_READ"
        exit 1
    fi

    CLI_VER="$("$TARGET_READ" --version 2>/dev/null | head -n1)"
    log_ok "Verification passed: $CLI_VER"

    # Check PATH
    PATH_FOUND=0
    OLD_IFS="$IFS"
    IFS=":"
    for p in $PATH
    do
        if [ "$p" = "$BINDIR" ]
        then
            PATH_FOUND=1
            break
        fi
    done
    IFS="$OLD_IFS"

    echo ""
    log_info "Shell integration notes:"
    log_info "1. Binary alias 'read-cli' is symlinked to '$BINDIR/read-cli'."
    log_info "2. 'read' is a shell builtin in bash and zsh. To invoke 'read' directly as the CLI,"
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

    if [ "$PATH_FOUND" -eq 1 ]
    then
        log_ok "Binary directory '$BINDIR' is present in current PATH."
    else
        log_warn "Binary directory '$BINDIR' is NOT yet in current PATH."
    fi
}

# Main installer flow
main() {
    log_banner
    log_step "1/4" "Environment & Target Destinations"
    PLATFORM_NAME="$(uname -s) ($(uname -m))"
    log_info "Detected platform: ${PLATFORM_NAME}"
    log_info "Target binary directory: ${BINDIR}"
    log_info "Target config directory: ${CONFIG_DIR}"

    # Step 2: Source and Prerequisites
    log_step "2/4" "Source Resolution & Toolchain Discovery"
    resolve_source

    if ! check_toolchain
    then
        exit 1
    fi

    if [ "$CHECK_ONLY" -eq 1 ]
    then
        log_info "Prerequisites check complete (--check-only). Exiting."
        exit 0
    fi

    # Step 3: Compilation and installation
    log_step "3/4" "Compilation & Binary Assembly"
    if [ "$NO_BUILD" -eq 0 ]
    then
        if [ "$DRY_RUN" -eq 1 ]
        then
            log_info "[DRY-RUN] Would build read-engine and read-daemon release binaries"
        else
            build_binaries
        fi
    else
        log_info "Skipping build stage (--no-build). Verifying existing binaries in bin/..."
        for b in read read-engine read-daemon
        do
            if [ ! -f "${SRC_DIR}/bin/$b" ]
            then
                log_error "Binary '${SRC_DIR}/bin/$b' missing. Run without --no-build."
                exit 1
            fi
        done
    fi

    # Perform filesystem installation
    if [ "$DRY_RUN" -eq 1 ]
    then
        log_info "[DRY-RUN] Would create directory: $BINDIR"
        log_info "[DRY-RUN] Would install binaries: read, read-engine, read-daemon to $BINDIR"
        log_info "[DRY-RUN] Would create directory: $CONFIG_DIR"
        log_info "[DRY-RUN] Would install config to: $CONFIG_DIR/config.json"
        exit 0
    fi

    log_info "Installing binaries into ${BINDIR}..."
    mkdir -p "${BINDIR}"
    for b in read read-engine read-daemon read-mcp read-report
    do
        if [ -f "${SRC_DIR}/bin/$b" ]
        then
            cp "${SRC_DIR}/bin/$b" "${BINDIR}/$b"
            chmod 0755 "${BINDIR}/$b"
            log_ok "Installed ${BINDIR}/$b"
        fi
    done
    ln -sf read "${BINDIR}/read-cli"
    log_ok "Linked ${BINDIR}/read-cli -> read"

    DATADIR="${PREFIX}/share/read"
    log_info "Installing declarative template into ${DATADIR}/template..."
    mkdir -p "${DATADIR}/template"
    if [ -d "${SRC_DIR}/template/.read" ]
    then
        cp -r "${SRC_DIR}/template/.read" "${DATADIR}/template/"
        log_ok "Template installed to ${DATADIR}/template"
    fi

    if [ -f "${SRC_DIR}/harness_setup.py" ]
    then
        cp "${SRC_DIR}/harness_setup.py" "${BINDIR}/harness_setup.py"
        chmod 0755 "${BINDIR}/harness_setup.py"
        log_ok "Installed ${BINDIR}/harness_setup.py"
    fi

    log_info "Initializing configuration in ${CONFIG_DIR}..."
    mkdir -p "${CONFIG_DIR}"
    CONFIG_FILE="${CONFIG_DIR}/config.json"
    if [ ! -f "$CONFIG_FILE" ] || [ "$FORCE_CONFIG" -eq 1 ]
    then
        cp "${SRC_DIR}/config.default.json" "$CONFIG_FILE"
        chmod 0644 "$CONFIG_FILE"
        log_ok "Configuration initialized: $CONFIG_FILE"
    else
        log_ok "Configuration exists and preserved: $CONFIG_FILE"
    fi

    # Step 4: Verification and Setup
    log_step "4/4" "Verification & Shell Setup"
    verify_installation

    # Step 5: AI Agent Harness Discovery & Autonomous Configuration
    if [ "$CHECK_ONLY" -eq 0 ] && [ "$DRY_RUN" -eq 0 ]
    then
        log_step "5/5" "AI Agent Harness Discovery & Skill Setup"
        if [ "$DO_SETUP" = "1" ]
        then
            if [ -f "${BINDIR}/read-mcp" ]
            then
                "${BINDIR}/read-mcp" setup --all
            elif [ -f "${SRC_DIR}/harness_setup.py" ]
            then
                python3 "${SRC_DIR}/harness_setup.py" --all
            fi
        elif [ -r /dev/tty ] && [ -w /dev/tty ] && [ -t 1 ] && [ -z "${CI:-}" ] && [ "${READ_NON_INTERACTIVE:-0}" != "1" ]
        then
            printf "Would you like to auto-detect and configure READ in your AI coding assistants (Claude Code, Cursor, Antigravity)? [Y/n]: "
            read -r cfg_agents < /dev/tty || true
            case "$cfg_agents" in
                [nN]*)
                    log_info "Skipping AI assistant configuration. You can run 'read-mcp setup' anytime."
                    ;;
                *)
                    if [ -f "${BINDIR}/read-mcp" ]
                    then
                        "${BINDIR}/read-mcp" setup < /dev/tty
                    elif [ -f "${SRC_DIR}/harness_setup.py" ]
                    then
                        python3 "${SRC_DIR}/harness_setup.py" < /dev/tty
                    fi
                    ;;
            esac
        else
            log_info "Auto-configuring detected AI agent harnesses..."
            if [ -f "${BINDIR}/read-mcp" ]
            then
                "${BINDIR}/read-mcp" setup --detected
            elif [ -f "${SRC_DIR}/harness_setup.py" ]
            then
                python3 "${SRC_DIR}/harness_setup.py" --detected
            fi
        fi
    fi
    if [ "$CHECK_ONLY" -eq 0 ] && [ "$DRY_RUN" -eq 0 ]
    then
        if [ "$DO_SETUP" = "1" ]
        then
            "${BINDIR}/read" setup
        elif [ -r /dev/tty ] && [ -w /dev/tty ] && [ -t 1 ] && [ -z "${CI:-}" ] && [ "${READ_NON_INTERACTIVE:-0}" != "1" ]
        then
            echo ""
            printf "Would you like to initialize READ methodology in current workspace (%s)? [Y/n]: " "$(pwd)"
            read -r init_ws < /dev/tty || true
            case "$init_ws" in
                [nN]*)
                    ;;
                *)
                    "${BINDIR}/read" setup < /dev/tty
                    ;;
            esac
        fi
    fi

    echo ""
    printf "────────────────────────────────────────────────────────────────\n"
    printf "%bREAD Framework successfully installed! (v%s)%b\n" "$C_GREEN" "$VERSION" "$C_RESET"
    printf "Execute 'read --help' or 'read-mcp --check' to begin.\n"
    printf "────────────────────────────────────────────────────────────────\n"
}

main "$@"
