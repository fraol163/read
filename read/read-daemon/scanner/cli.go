package scanner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CLIAgentDefinition defines an agent binary signature to search for.
type CLIAgentDefinition struct {
	ID          string
	Name        string
	Binaries    []string
	Description string
}

// KnownCLIAgents is the comprehensive catalog of supported AI agent harnesses.
var KnownCLIAgents = []CLIAgentDefinition{
	// 1. Anthropic & DeepSeek
	{
		ID:          "claude_code",
		Name:        "Claude Code CLI",
		Binaries:    []string{"claude", "claude-code"},
		Description: "Anthropic Claude terminal research and coding assistant",
	},
	{
		ID:          "deepseek_harness",
		Name:        "DeepSeek Harness (dsh)",
		Binaries:    []string{"dsh", "deepseek-harness"},
		Description: "Official DeepSeek agent harness with Cordis plugin architecture",
	},

	// 2. Google DeepMind & OpenAI & xAI
	{
		ID:          "antigravity",
		Name:        "Google DeepMind Antigravity CLI",
		Binaries:    []string{"antigravity", "agy", "antigravity-cli"},
		Description: "Google DeepMind agentic AI development assistant",
	},
	{
		ID:          "gemini_cli",
		Name:        "Gemini CLI",
		Binaries:    []string{"gemini", "gemini-cli"},
		Description: "Google Gemini terminal AI assistant",
	},
	{
		ID:          "codex",
		Name:        "OpenAI Codex CLI",
		Binaries:    []string{"codex", "codex-cli", "codex-code-mode-host", "codex-exec-server"},
		Description: "OpenAI Codex terminal coding and execution agent",
	},
	{
		ID:          "grok",
		Name:        "xAI Grok CLI",
		Binaries:    []string{"grok", "grok-cli"},
		Description: "xAI Grok command-line coding assistant",
	},

	// 3. GitHub & IDE Headless Agents
	{
		ID:          "copilot",
		Name:        "GitHub Copilot CLI",
		Binaries:    []string{"copilot", "github-copilot-cli"},
		Description: "GitHub Copilot in the CLI",
	},
	{
		ID:          "cursor_agent",
		Name:        "Cursor Agent CLI",
		Binaries:    []string{"cursor-agent", "cursor"},
		Description: "Cursor IDE headless agent executor",
	},
	{
		ID:          "ghui",
		Name:        "GitHub Agent UI",
		Binaries:    []string{"ghui"},
		Description: "GitHub AI terminal user interface",
	},

	// 4. Autonomous Software Engineers & Emerging CLI Agents
	{
		ID:          "devin_wrapper",
		Name:        "Devin / OpenHands Agent",
		Binaries:    []string{"devin", "openhands"},
		Description: "Autonomous software development agent by Cognition",
	},
	{
		ID:          "crush",
		Name:        "Crush CLI",
		Binaries:    []string{"crush"},
		Description: "Terminal agent workflow runner and pair programmer",
	},
	{
		ID:          "pi",
		Name:        "Pi CLI",
		Binaries:    []string{"pi", "pi-cli"},
		Description: "Inflection AI conversational and coding agent",
	},
	{
		ID:          "qwen",
		Name:        "Alibaba Qwen CLI",
		Binaries:    []string{"qwen", "qwen-code"},
		Description: "Alibaba Cloud Qwen AI coding agent",
	},
	{
		ID:          "opencode",
		Name:        "OpenCode CLI",
		Binaries:    []string{"opencode"},
		Description: "Open-source terminal AI coding assistant",
	},
	{
		ID:          "kiro",
		Name:        "Kiro CLI",
		Binaries:    []string{"kiro-cli", "kiro"},
		Description: "Kiro autonomous developer CLI",
	},
	{
		ID:          "kilo",
		Name:        "KiloCode CLI",
		Binaries:    []string{"kilo", "kilocode"},
		Description: "KiloCode terminal developer agent",
	},
	{
		ID:          "kimi",
		Name:        "Moonshot Kimi Code CLI",
		Binaries:    []string{"kimi", "kimi-code"},
		Description: "Moonshot AI Kimi coding assistant",
	},
	{
		ID:          "mcode",
		Name:        "MiniMax AI Code CLI",
		Binaries:    []string{"mcode", "minimax-code"},
		Description: "MiniMax AI terminal coding agent",
	},
	{
		ID:          "amp",
		Name:        "Amp AI Agent",
		Binaries:    []string{"amp"},
		Description: "Amp autonomous developer assistant",
	},
	{
		ID:          "basecamp",
		Name:        "Basecamp AI",
		Binaries:    []string{"basecamp"},
		Description: "Basecamp command-line AI agent",
	},
	{
		ID:          "cf_ai",
		Name:        "Cloudflare Workers AI CLI",
		Binaries:    []string{"cf"},
		Description: "Cloudflare Workers AI command-line harness",
	},
	{
		ID:          "commandcode",
		Name:        "Command Code CLI",
		Binaries:    []string{"commandcode", "command-code"},
		Description: "Terminal command generation and code editing agent",
	},
	{
		ID:          "freebuff",
		Name:        "Freebuff AI",
		Binaries:    []string{"freebuff"},
		Description: "Freebuff developer AI terminal agent",
	},
	{
		ID:          "hunk",
		Name:        "Hunk AI",
		Binaries:    []string{"hunk"},
		Description: "Hunk diff and patch generation AI agent",
	},
	{
		ID:          "muse",
		Name:        "Muse AI",
		Binaries:    []string{"muse"},
		Description: "Meta Muse developer assistant",
	},
	{
		ID:          "ori",
		Name:        "Ori AI",
		Binaries:    []string{"ori"},
		Description: "Ori autonomous developer agent",
	},
	{
		ID:          "hey_ai",
		Name:        "Hey AI CLI",
		Binaries:    []string{"hey", "hey-cli"},
		Description: "Hey natural language terminal execution assistant",
	},
	{
		ID:          "oh_my_pi",
		Name:        "Oh-My-Pi CLI (omp)",
		Binaries:    []string{"omp", "oh-my-pi"},
		Description: "Oh-My-Pi high performance AI coding agent",
	},
	{
		ID:          "hermes",
		Name:        "Hermes Agent",
		Binaries:    []string{"hermes", "hermes-agent"},
		Description: "Hermes autonomous tool-using agent",
	},

	// 5. Open Source Developer Agents
	{
		ID:          "aider",
		Name:        "Aider AI",
		Binaries:    []string{"aider"},
		Description: "Git-centric terminal AI pair-programmer",
	},
	{
		ID:          "goose",
		Name:        "Goose AI Agent",
		Binaries:    []string{"goose"},
		Description: "Extensible open-source autonomous agent by Block",
	},
	{
		ID:          "open_interpreter",
		Name:        "Open Interpreter",
		Binaries:    []string{"interpreter", "open-interpreter"},
		Description: "Local-first autonomous terminal code execution agent",
	},
	{
		ID:          "sgpt",
		Name:        "Shell-GPT",
		Binaries:    []string{"sgpt"},
		Description: "Command-line productivity tool powered by AI models",
	},
	{
		ID:          "plandex",
		Name:        "Plandex CLI",
		Binaries:    []string{"plandex"},
		Description: "Terminal-based AI engine for complex multi-file engineering tasks",
	},
	{
		ID:          "mentat",
		Name:        "Mentat AI",
		Binaries:    []string{"mentat"},
		Description: "AI coding assistant designed for codebase navigation and editing",
	},
	{
		ID:          "cody",
		Name:        "Sourcegraph Cody CLI",
		Binaries:    []string{"cody"},
		Description: "Sourcegraph Cody command-line interface for code intelligence",
	},
	{
		ID:          "continue",
		Name:        "Continue CLI",
		Binaries:    []string{"continue", "continue-cli"},
		Description: "Open-source autopilot developer assistant CLI",
	},
	{
		ID:          "cline",
		Name:        "Cline / Roo-Code CLI",
		Binaries:    []string{"cline", "roo-code", "roo"},
		Description: "Autonomous coding agent with shell and editor capabilities",
	},
	{
		ID:          "autogpt",
		Name:        "AutoGPT",
		Binaries:    []string{"autogpt", "auto-gpt"},
		Description: "Autonomous multi-step goal execution agent platform",
	},
	{
		ID:          "gpt_engineer",
		Name:        "GPT Engineer",
		Binaries:    []string{"gpt-engineer", "gpte"},
		Description: "AI codebase generator and updater",
	},
	{
		ID:          "llm",
		Name:        "Simon Willison LLM CLI",
		Binaries:    []string{"llm"},
		Description: "CLI tool and Python library for running prompts against LLMs",
	},
	{
		ID:          "fabric",
		Name:        "Fabric AI",
		Binaries:    []string{"fabric"},
		Description: "Open-source AI augmentation framework for terminal workflows",
	},
	{
		ID:          "mods",
		Name:        "Charm Mods AI",
		Binaries:    []string{"mods"},
		Description: "AI tool designed for command line piping and terminal output",
	},
	{
		ID:          "tgpt",
		Name:        "Terminal GPT",
		Binaries:    []string{"tgpt"},
		Description: "Anonymous terminal-based AI client",
	},

	// 6. Local Server CLIs
	{
		ID:          "ollama_cli",
		Name:        "Ollama CLI",
		Binaries:    []string{"ollama"},
		Description: "Local model manager and interactive terminal runner",
	},
	{
		ID:          "vllm_cli",
		Name:        "vLLM CLI",
		Binaries:    []string{"vllm"},
		Description: "High-throughput and memory-efficient LLM serving engine",
	},
	{
		ID:          "tabby_cli",
		Name:        "Tabby CLI",
		Binaries:    []string{"tabby"},
		Description: "Self-hosted AI coding assistant",
	},
	{
		ID:          "localai_cli",
		Name:        "LocalAI CLI",
		Binaries:    []string{"localai"},
		Description: "Self-hosted, drop-in replacement REST API for AI models",
	},

	// 7. READ Framework Native Engine
	{
		ID:          "read_engine",
		Name:        "READ Core Engine",
		Binaries:    []string{"read-engine"},
		Description: "Native Rust deterministic engine for the READ framework",
	},
}

// BuildSearchDirectories discovers all candidate executable search paths across package managers.
func BuildSearchDirectories(extraPaths []string) []string {
	dirs := make([]string, 0, 64)
	seen := make(map[string]bool)

	addDir := func(d string) {
		if d == "" {
			return
		}
		clean := filepath.Clean(d)
		if !seen[clean] {
			if fi, err := os.Stat(clean); err == nil && fi.IsDir() {
				seen[clean] = true
				dirs = append(dirs, clean)
			}
		}
	}

	// 1. Extra search paths provided in options (highest priority)
	for _, p := range extraPaths {
		addDir(p)
	}

	// 2. PATH environment variable
	pathEnv := os.Getenv("PATH")
	if pathEnv != "" {
		for _, part := range filepath.SplitList(pathEnv) {
			addDir(part)
		}
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}

	if homeDir != "" {
		// 3. mise
		miseData := os.Getenv("MISE_DATA_DIR")
		if miseData == "" {
			miseData = filepath.Join(homeDir, ".local", "share", "mise")
		}
		addDir(filepath.Join(miseData, "command-wrappers", "bin"))
		addDir(filepath.Join(miseData, "shims"))
		addDir(filepath.Join(miseData, "bin"))
		miseInstalls := filepath.Join(miseData, "installs")
		if matches, err := filepath.Glob(filepath.Join(miseInstalls, "*", "*", "bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}
		if matches, err := filepath.Glob(filepath.Join(miseInstalls, "*", "*")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}
		if matches, err := filepath.Glob(filepath.Join(miseInstalls, "*", "*", "node_modules", ".bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}

		// 4. asdf
		asdfData := os.Getenv("ASDF_DATA_DIR")
		if asdfData == "" {
			asdfData = filepath.Join(homeDir, ".asdf")
		}
		addDir(filepath.Join(asdfData, "shims"))
		addDir(filepath.Join(asdfData, "bin"))
		if matches, err := filepath.Glob(filepath.Join(asdfData, "installs", "*", "*", "bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}

		// 5. npm global & node ecosystem (bun, pnpm, volta, nvm)
		addDir(filepath.Join(homeDir, ".npm-global", "bin"))
		addDir(filepath.Join(homeDir, "node_modules", ".bin"))
		addDir(filepath.Join(homeDir, ".bun", "bin"))
		addDir(filepath.Join(homeDir, ".local", "share", "pnpm"))
		addDir(filepath.Join(homeDir, ".pnpm-global", "bin"))
		addDir(filepath.Join(homeDir, ".volta", "bin"))
		if matches, err := filepath.Glob(filepath.Join(homeDir, ".nvm", "versions", "node", "*", "bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}

		// 6. pip & pipx
		addDir(filepath.Join(homeDir, ".local", "bin"))
		if matches, err := filepath.Glob(filepath.Join(homeDir, ".local", "share", "pipx", "venvs", "*", "bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}
		if matches, err := filepath.Glob(filepath.Join(homeDir, ".pipx", "venvs", "*", "bin")); err == nil {
			for _, m := range matches {
				addDir(m)
			}
		}

		// 7. cargo bin
		cargoHome := os.Getenv("CARGO_HOME")
		if cargoHome == "" {
			cargoHome = filepath.Join(homeDir, ".cargo")
		}
		addDir(filepath.Join(cargoHome, "bin"))

		// 8. go bin
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			goPath = filepath.Join(homeDir, "go")
		}
		addDir(filepath.Join(goPath, "bin"))

		// 9. nix profiles
		addDir(filepath.Join(homeDir, ".nix-profile", "bin"))
		user := os.Getenv("USER")
		if user != "" {
			addDir(filepath.Join("/etc", "profiles", "per-user", user, "bin"))
		}

		// 10. flatpak
		addDir(filepath.Join(homeDir, ".local", "share", "flatpak", "exports", "bin"))
	}

	// Global system package managers
	addDir("/usr/local/bin")
	addDir("/usr/bin")
	addDir("/bin")
	addDir("/opt/homebrew/bin")
	addDir("/usr/local/sbin")
	addDir("/usr/sbin")
	addDir("/usr/local/lib/node_modules/.bin")
	addDir("/usr/lib/node_modules/.bin")
	addDir("/nix/var/nix/profiles/default/bin")
	addDir("/run/current-system/sw/bin")
	addDir("/snap/bin")
	addDir("/var/lib/snapd/snap/bin")
	addDir("/var/lib/flatpak/exports/bin")

	return dirs
}

// ScanCLIAgents searches candidate directories for known AI agent binaries.
func ScanCLIAgents(registry *HarnessRegistry, extraPaths []string) {
	// First, run dynamic mise discovery to populate accurate mise versions and paths
	ScanMiseRegistry(registry)

	searchDirs := BuildSearchDirectories(extraPaths)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, agent := range KnownCLIAgents {
		wg.Add(1)
		go func(def CLIAgentDefinition) {
			defer wg.Done()

			foundPath, binName, source := findAgentBinary(def.Binaries, searchDirs)
			if foundPath == "" {
				return
			}

			// If already in registry with a verified version from mise, keep it
			if existing, found := registry.GetHarness(def.ID); found && existing.Version != "" && existing.Version != "unknown" {
				return
			}

			version := extractCLIVersion(foundPath)

			harness := AgentHarness{
				ID:          def.ID,
				Name:        def.Name,
				Type:        "cli",
				Path:        foundPath,
				Version:     version,
				Status:      "available",
				Enabled:     true,
				Description: def.Description,
				Metadata: map[string]string{
					"binary": binName,
					"source": source,
				},
			}

			mu.Lock()
			registry.AddHarness(harness)
			mu.Unlock()
		}(agent)
	}

	wg.Wait()

	// Dynamic Discovery: Scan search directories for any custom or uncataloged agent binaries
	scanDynamicAgents(registry, searchDirs, &mu)
}

// scanDynamicAgents inspects search directories for executables with agent signatures.
func scanDynamicAgents(registry *HarnessRegistry, searchDirs []string, mu *sync.Mutex) {
	agentIndicators := []string{"agent", "copilot", "assistant"}

	homeDir, _ := os.UserHomeDir()
	userDirs := []string{
		filepath.Join(homeDir, ".local", "bin"),
		filepath.Join(homeDir, ".cargo", "bin"),
	}

	for _, dir := range userDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())

			// Check for agent naming indicators
			matched := false
			for _, ind := range agentIndicators {
				if strings.Contains(name, ind) {
					matched = true
					break
				}
			}

			if !matched {
				continue
			}

			cleanID := strings.ReplaceAll(name, "-", "_")
			if _, exists := registry.GetHarness(cleanID); exists {
				continue
			}

			fullPath := filepath.Join(dir, entry.Name())
			if !isExecutable(fullPath) {
				continue
			}

			harness := AgentHarness{
				ID:          cleanID,
				Name:        formatDynamicTitle(entry.Name()),
				Type:        "cli",
				Path:        fullPath,
				Version:     "available",
				Status:      "available",
				Enabled:     true,
				Description: "Dynamically discovered agent CLI executable",
				Metadata: map[string]string{
					"binary": entry.Name(),
					"source": classifySource(fullPath),
				},
			}

			mu.Lock()
			registry.AddHarness(harness)
			mu.Unlock()
		}
	}
}

// findAgentBinary scans search directories in priority order for the first matching binary.
func findAgentBinary(binaries []string, searchDirs []string) (string, string, string) {
	for _, bin := range binaries {
		for _, dir := range searchDirs {
			candidate := filepath.Join(dir, bin)
			if isExecutable(candidate) {
				source := classifySource(candidate)
				return candidate, bin, source
			}
		}
	}

	return "", "", ""
}

// isExecutable checks whether the given path points to an executable regular file or symlink.
func isExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		return false
	}
	// Verify execute bits on Unix
	return (fi.Mode().Perm() & 0111) != 0
}

// classifySource determines which package manager or directory type provided the binary.
func classifySource(path string) string {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(lower, "mise"):
		return "mise"
	case strings.Contains(lower, "asdf"):
		return "asdf"
	case strings.Contains(lower, "pipx") || strings.Contains(lower, ".local/bin"):
		return "pipx"
	case strings.Contains(lower, ".cargo"):
		return "cargo"
	case strings.Contains(lower, "go/bin") || strings.Contains(lower, "gopath"):
		return "go"
	case strings.Contains(lower, "nix"):
		return "nix"
	case strings.Contains(lower, "snap"):
		return "snap"
	case strings.Contains(lower, "flatpak"):
		return "flatpak"
	case strings.Contains(lower, "node_modules") || strings.Contains(lower, "npm") || strings.Contains(lower, "bun") || strings.Contains(lower, "pnpm"):
		return "npm"
	case strings.Contains(lower, "/usr/local"):
		return "usr_local"
	case strings.Contains(lower, "/usr/bin"):
		return "system"
	default:
		return "path"
	}
}

// extractCLIVersion invokes the binary with a safe timeout and avoids background downloads.
func extractCLIVersion(binPath string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, "--version")
	cmd.Env = append(os.Environ(), "MISE_AUTO_UPDATE=0", "MISE_AUTO_INSTALL=0", "MISE_QUIET=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		out := stdout.String()
		if len(strings.TrimSpace(out)) == 0 {
			out = stderr.String()
		}
		if len(strings.TrimSpace(out)) > 0 {
			return cleanVersionString(out)
		}
	}

	// Try -v fallback
	ctx2, cancel2 := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel2()

	cmd2 := exec.CommandContext(ctx2, binPath, "-v")
	cmd2.Env = append(os.Environ(), "MISE_AUTO_UPDATE=0", "MISE_AUTO_INSTALL=0", "MISE_QUIET=1")
	stdout.Reset()
	stderr.Reset()
	cmd2.Stdout = &stdout
	cmd2.Stderr = &stderr

	if err := cmd2.Run(); err == nil {
		out := stdout.String()
		if len(strings.TrimSpace(out)) == 0 {
			out = stderr.String()
		}
		if len(strings.TrimSpace(out)) > 0 {
			return cleanVersionString(out)
		}
	}

	return "unknown"
}

// cleanVersionString strips whitespace, newlines, and truncates long strings.
func cleanVersionString(raw string) string {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 {
		return "unknown"
	}
	firstLine := strings.TrimSpace(lines[0])
	if len(firstLine) > 60 {
		return firstLine[:60]
	}
	if firstLine == "" {
		return "unknown"
	}
	return firstLine
}
