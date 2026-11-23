# ==============================================================================
# READ Framework Unified Makefile (v0.7.0)
# Resource-conscious, Evidence-driven AI Development
# ==============================================================================

SHELL := /bin/sh
VERSION := 0.7.0

# Prefix detection: /usr/local for root, ~/.local for standard user
PREFIX ?= $(shell if [ `id -u` -eq 0 ]; then echo /usr/local; else echo $(HOME)/.local; fi)
BINDIR ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share/read

ifeq ($(PREFIX),/usr)
    CONFIG_DIR ?= /etc/read
else ifeq ($(PREFIX),/usr/local)
    CONFIG_DIR ?= /etc/read
else
    CONFIG_DIR ?= $(if $(XDG_CONFIG_HOME),$(XDG_CONFIG_HOME)/read,$(HOME)/.config/read)
endif

CARGO ?= cargo
GO ?= go
INSTALL ?= install
MKDIR ?= mkdir -p
TAR ?= tar

OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)

.PHONY: all build test install clean verify package uninstall help

all: build

help:
	@echo "READ Framework Build System (v$(VERSION))"
	@echo ""
	@echo "Targets:"
	@echo "  build     Build Rust read-engine and Go read-daemon release binaries"
	@echo "  test      Run full test suite (cargo test, go test, e2e, system, chaos)"
	@echo "  install   Install read binaries to $(DESTDIR)$(BINDIR)"
	@echo "  verify    Verify toolchain prerequisites and binary execution"
	@echo "  package   Assemble release tarball in dist/"
	@echo "  clean     Remove build artifacts and temporary test state"
	@echo "  uninstall Remove installed binaries from $(DESTDIR)$(BINDIR)"

build:
	@echo "[READ] Building native Rust engine (read-core)..."
	cd read-core && $(CARGO) build --release
	@TARGET_DIR=$$($(CARGO) metadata --format-version 1 --manifest-path read-core/Cargo.toml --no-deps | sed -n 's/.*"target_directory":"\([^"]*\)".*/\1/p'); \
	if [ -f "$$TARGET_DIR/release/read-engine" ]; then \
		cp "$$TARGET_DIR/release/read-engine" bin/read-engine; \
	elif [ -f "read-core/target/release/read-engine" ]; then \
		cp read-core/target/release/read-engine bin/read-engine; \
	else \
		echo "[ERROR] Could not find compiled read-engine binary" >&2; exit 1; \
	fi
	@echo "[READ] Building Go host daemon (read-daemon)..."
	cd read-daemon && $(GO) build -o ../bin/read-daemon ./cmd/read-daemon
	chmod +x bin/read bin/read-engine bin/read-daemon
	@echo "[READ] Build complete: bin/read, bin/read-engine, bin/read-daemon"

test: build
	@echo "[READ] Running Rust unit & property tests..."
	cd read-core && $(CARGO) test
	@echo "[READ] Running Go unit & subsystem tests..."
	cd read-daemon && $(GO) test ./...
	@echo "[READ] Running End-to-End lifecycle verification..."
	tests/test_e2e.sh
	@echo "[READ] Running Full System & Gateway integration tests..."
	tests/test_full_system.sh
	@echo "[READ] Running Adversarial Red-Team & Chaos test suite..."
	tests/test_chaos.sh
	@echo "[READ] Running Packaging & Installation verification..."
	tests/test_packaging.sh
	@echo "[READ] Running Autonomous Pipeline & Orchestrator tests..."
	tests/test_pipeline.sh
	@echo "[READ] Running BMAD Zero-Command Declarative Workflow test suite..."
	tests/test_bmad_zero_command.sh
	@echo "[READ] Running Live Real-Time Execution Audit test suite..."
	tests/test_live_realtime.sh
	@echo "[READ] All test suites completed with 100% pass rate."

install: build
	@echo "[READ] Installing binaries into $(DESTDIR)$(BINDIR)..."
	$(MKDIR) "$(DESTDIR)$(BINDIR)"
	$(INSTALL) -m 0755 bin/read "$(DESTDIR)$(BINDIR)/read"
	$(INSTALL) -m 0755 bin/read-engine "$(DESTDIR)$(BINDIR)/read-engine"
	$(INSTALL) -m 0755 bin/read-daemon "$(DESTDIR)$(BINDIR)/read-daemon"
	ln -sf read "$(DESTDIR)$(BINDIR)/read-cli"
	@echo "[READ] Installing declarative template bundle into $(DESTDIR)$(DATADIR)/template..."
	$(MKDIR) "$(DESTDIR)$(DATADIR)/template"
	cp -r template/.read "$(DESTDIR)$(DATADIR)/template/"
	@echo "[READ] Initializing default configuration into $(DESTDIR)$(CONFIG_DIR)..."
	$(MKDIR) "$(DESTDIR)$(CONFIG_DIR)"
	@if [ ! -f "$(DESTDIR)$(CONFIG_DIR)/config.json" ]; then \
		$(INSTALL) -m 0644 config.default.json "$(DESTDIR)$(CONFIG_DIR)/config.json"; \
		echo "[READ] Created default config: $(DESTDIR)$(CONFIG_DIR)/config.json"; \
	else \
		echo "[READ] Preserving existing config: $(DESTDIR)$(CONFIG_DIR)/config.json"; \
	fi
	@echo "[READ] Installation successful."
	@echo "[READ] Verified CLI: $$("$(DESTDIR)$(BINDIR)/read" --version | head -n 1)"

verify:
	@echo "=== [READ TOOLCHAIN VERIFICATION] ==="
	@printf "Checking rustc:  " && rustc --version || { echo "MISSING"; exit 1; }
	@printf "Checking cargo:  " && cargo --version || { echo "MISSING"; exit 1; }
	@printf "Checking go:     " && go version || { echo "MISSING"; exit 1; }
	@printf "Checking git:    " && git --version || { echo "MISSING"; exit 1; }
	@printf "Checking python: " && python3 --version || { echo "MISSING"; exit 1; }
	@echo "=== [BINARY VERIFICATION] ==="
	@bin/read --version
	@bin/read-engine --version
	@bin/read-daemon -version
	@echo "=== [TOOLCHAIN & BINARIES VERIFIED] ==="

package: build
	@echo "[READ] Assembling distribution release tarball..."
	$(MKDIR) dist
	$(MKDIR) dist/read-$(VERSION)
	cp -r bin config.default.json config.schema.json install.sh README.md Makefile template docs dist/read-$(VERSION)/
	cd dist && $(TAR) -czf read-v$(VERSION)-$(OS)-$(ARCH).tar.gz read-$(VERSION)
	rm -rf dist/read-$(VERSION)
	@echo "[READ] Package created: dist/read-v$(VERSION)-$(OS)-$(ARCH).tar.gz"

clean:
	@echo "[READ] Cleaning build artifacts..."
	cd read-core && $(CARGO) clean
	cd read-daemon && $(GO) clean
	rm -rf dist
	rm -f /tmp/read-sandbox-* /tmp/read-system-sandbox-* /tmp/read-chaos-suite-* 2>/dev/null || true
	@echo "[READ] Workspace cleaned."

uninstall:
	@echo "[READ] Uninstalling binaries from $(DESTDIR)$(BINDIR)..."
	rm -f "$(DESTDIR)$(BINDIR)/read"
	rm -f "$(DESTDIR)$(BINDIR)/read-cli"
	rm -f "$(DESTDIR)$(BINDIR)/read-engine"
	rm -f "$(DESTDIR)$(BINDIR)/read-daemon"
	@echo "[READ] Binaries removed. Config in $(DESTDIR)$(CONFIG_DIR) preserved."
