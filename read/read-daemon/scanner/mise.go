package scanner

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// MiseToolVersionInfo represents a version entry in mise ls --json.
type MiseToolVersionInfo struct {
	Version          string `json:"version"`
	RequestedVersion string `json:"requested_version"`
	InstallPath      string `json:"install_path"`
	Installed        bool   `json:"installed"`
	Active           bool   `json:"active"`
	Source           struct {
		Type string `json:"type"`
		Path string `json:"path"`
	} `json:"source"`
}

// QueryMiseTools queries the mise runtime using 'mise ls --json'.
// It returns a map of tool identifier to active/installed tool version info.
func QueryMiseTools() map[string]MiseToolVersionInfo {
	tools := make(map[string]MiseToolVersionInfo)

	// Check if mise exists
	miseBin, err := exec.LookPath("mise")
	if err != nil {
		miseBin = "/usr/bin/mise"
		if !isExecutable(miseBin) {
			return tools
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, miseBin, "ls", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), "MISE_AUTO_UPDATE=0", "MISE_QUIET=1")

	if err := cmd.Run(); err != nil {
		return tools
	}

	var rawMap map[string][]MiseToolVersionInfo
	if err := json.Unmarshal(stdout.Bytes(), &rawMap); err != nil {
		return tools
	}

	for toolName, versions := range rawMap {
		if len(versions) == 0 {
			continue
		}
		// Prefer active version, then installed version, then latest defined
		selected := versions[0]
		for _, v := range versions {
			if v.Active {
				selected = v
				break
			} else if v.Installed && !selected.Installed {
				selected = v
			}
		}
		tools[toolName] = selected
	}

	return tools
}

// IsAIAgentTool evaluates whether a mise tool is an AI agent or coding assistant
// using dynamic semantic detection across package metadata, configuration files, and naming.
// It is fully portable across any OS, distribution, or user environment without host-specific assumptions.
func IsAIAgentTool(toolKey string, info MiseToolVersionInfo) bool {
	lowerKey := strings.ToLower(toolKey)

	// 1. Non-agent utility and compiler tools to exclude across all environments
	excludeList := []string{
		"node", "uv", "playwright", "gh", "go", "python", "php",
		"java", "dotnet", "bun", "ruby", "rust", "deno",
	}
	for _, exc := range excludeList {
		if lowerKey == exc {
			return false
		}
	}

	// 2. Check if tool is explicitly declared in an agent account wrapper config
	if strings.Contains(strings.ToLower(info.Source.Path), "agent") {
		return true
	}

	// 3. Semantic tokens in tool identifier or source URL
	aiTokens := []string{
		"agent", "ai", "copilot", "assistant", "llm",
		"deepseek", "anthropic", "claude", "openai", "codex",
		"moonshot", "minimax", "kilocode", "google", "antigravity",
		"grok", "xai", "qwen", "pi", "devin", "kiro", "crush",
		"freebuff", "hunk", "muse", "ori", "hey", "omp", "hermes",
		"ghui", "kilo", "kimi", "mcode", "dsh", "opencode",
	}

	for _, token := range aiTokens {
		if strings.Contains(lowerKey, token) {
			return true
		}
	}

	// 4. Inspect package.json in install directory if available
	if info.InstallPath != "" {
		pkgJsonPath := filepath.Join(info.InstallPath, "package.json")
		if data, err := os.ReadFile(pkgJsonPath); err == nil {
			lowerData := strings.ToLower(string(data))
			for _, token := range aiTokens {
				if strings.Contains(lowerData, token) {
					return true
				}
			}
		}
	}

	return false
}

// MapMiseToolToHarness maps a mise tool dynamically to an AgentHarness.
func MapMiseToolToHarness(toolKey string, info MiseToolVersionInfo, homeDir string) (AgentHarness, bool) {
	if !IsAIAgentTool(toolKey, info) {
		return AgentHarness{}, false
	}

	// Dynamic candidate binary discovery:
	// Search in:
	// 1. install_path/bin, install_path, install_path/node_modules/.bin
	// 2. ~/.local/share/mise/command-wrappers/bin
	// 3. ~/.local/share/mise/shims
	cleanKey := strings.TrimPrefix(toolKey, "npm:")
	cleanKey = strings.TrimPrefix(cleanKey, "http:")
	cleanKey = strings.TrimPrefix(cleanKey, "aqua:")
	cleanKey = strings.TrimPrefix(cleanKey, "@")

	var subPkg string
	if idx := strings.Index(cleanKey, "/"); idx != -1 {
		subPkg = cleanKey[idx+1:]
	}

	candidateBinNames := make([]string, 0, 8)
	if subPkg != "" {
		candidateBinNames = append(candidateBinNames, subPkg)
	}
	candidateBinNames = append(candidateBinNames, cleanKey)

	// Add well-known binary aliases dynamically
	switch {
	case strings.Contains(cleanKey, "dsh"):
		candidateBinNames = append(candidateBinNames, "dsh", "deepseek-harness")
	case strings.Contains(cleanKey, "minimax") || cleanKey == "code":
		candidateBinNames = append(candidateBinNames, "mcode", "minimax-code")
	case strings.Contains(cleanKey, "moonshot") || cleanKey == "kimi-code":
		candidateBinNames = append(candidateBinNames, "kimi", "kimi-code")
	case strings.Contains(cleanKey, "kilocode"):
		candidateBinNames = append(candidateBinNames, "kilo", "kilocode")
	case strings.Contains(cleanKey, "antigravity"):
		candidateBinNames = append(candidateBinNames, "antigravity", "agy", "antigravity-cli")
	case strings.Contains(cleanKey, "claude"):
		candidateBinNames = append(candidateBinNames, "claude", "claude-code")
	case strings.Contains(cleanKey, "codex"):
		candidateBinNames = append(candidateBinNames, "codex", "codex-cli")
	case cleanKey == "oh-my-pi":
		candidateBinNames = append(candidateBinNames, "omp", "oh-my-pi")
	case cleanKey == "hey-cli":
		candidateBinNames = append(candidateBinNames, "hey", "hey-cli")
	case cleanKey == "kiro-cli":
		candidateBinNames = append(candidateBinNames, "kiro-cli", "kiro")
	case cleanKey == "cf":
		candidateBinNames = append(candidateBinNames, "cf")
	case cleanKey == "command-code":
		candidateBinNames = append(candidateBinNames, "commandcode", "command-code")
	}

	binPath := ""

	// Check install directory if installed
	if info.InstallPath != "" {
		for _, b := range candidateBinNames {
			cand := filepath.Join(info.InstallPath, "bin", b)
			if isExecutable(cand) {
				binPath = cand
				break
			}
			cand = filepath.Join(info.InstallPath, b)
			if isExecutable(cand) {
				binPath = cand
				break
			}
			cand = filepath.Join(info.InstallPath, "node_modules", ".bin", b)
			if isExecutable(cand) {
				binPath = cand
				break
			}
		}
	}

	// Check command wrappers or shims
	if binPath == "" && homeDir != "" {
		for _, b := range candidateBinNames {
			wrapper := filepath.Join(homeDir, ".local", "share", "mise", "command-wrappers", "bin", b)
			if isExecutable(wrapper) {
				binPath = wrapper
				break
			}
			shim := filepath.Join(homeDir, ".local", "share", "mise", "shims", b)
			if isExecutable(shim) {
				binPath = shim
				break
			}
		}
	}

	if binPath == "" {
		return AgentHarness{}, false
	}

	// Determine normalized ID and display Name dynamically
	id := cleanKey
	if subPkg != "" {
		id = subPkg
	}
	if subPkg == "code" && strings.Contains(cleanKey, "minimax") {
		id = "mcode"
	}
	if subPkg == "cli" && strings.Contains(cleanKey, "kilo") {
		id = "kilo"
	}
	id = strings.ReplaceAll(id, "-", "_")

	// Check if known agent definition provides canonical name/description
	canonicalName := ""
	canonicalDesc := ""
	for _, def := range KnownCLIAgents {
		if def.ID == id || def.ID == cleanKey || (subPkg != "" && def.ID == subPkg) {
			canonicalName = def.Name
			canonicalDesc = def.Description
			id = def.ID
			break
		}
		for _, b := range def.Binaries {
			if b == cleanKey || (subPkg != "" && b == subPkg) {
				canonicalName = def.Name
				canonicalDesc = def.Description
				id = def.ID
				break
			}
		}
		if canonicalName != "" {
			break
		}
	}

	if canonicalName == "" {
		canonicalName = formatDynamicTitle(cleanKey)
	}
	if canonicalDesc == "" {
		canonicalDesc = "AI agent harness dynamically discovered via package manager"
	}

	status := "available"
	if !info.Installed {
		status = "configured"
	}

	version := info.Version
	if version == "" {
		version = info.RequestedVersion
	}

	return AgentHarness{
		ID:          id,
		Name:        canonicalName,
		Type:        "cli",
		Path:        binPath,
		Version:     version,
		Status:      status,
		Enabled:     true,
		Description: canonicalDesc,
		Metadata: map[string]string{
			"source":       "mise",
			"mise_tool":    toolKey,
			"installed":    boolToString(info.Installed),
			"install_path": info.InstallPath,
		},
	}, true
}

func formatDynamicTitle(key string) string {
	parts := strings.FieldsFunc(key, func(r rune) bool {
		return r == '-' || r == '_' || r == '/' || r == ':'
	})
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ") + " CLI"
}

// ScanMiseRegistry performs specialized scanning for mise-managed AI tools.
func ScanMiseRegistry(registry *HarnessRegistry) {
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}

	miseTools := QueryMiseTools()
	for toolKey, info := range miseTools {
		if harness, ok := MapMiseToolToHarness(toolKey, info, homeDir); ok {
			registry.AddHarness(harness)
		}
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
