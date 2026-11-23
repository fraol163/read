package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LoadCustomConfigs reads custom agent harnesses from workspace and user config paths.
func LoadCustomConfigs(registry *HarnessRegistry, workspaceDir, customConfigFile string) {
	paths := make([]string, 0, 4)

	// 1. Workspace local config: .read/harnesses.json
	if workspaceDir != "" {
		paths = append(paths, filepath.Join(workspaceDir, ".read", "harnesses.json"))
	} else {
		paths = append(paths, filepath.Join(".read", "harnesses.json"))
	}

	// 2. User global config: ~/.config/read/harnesses.json
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}
	if homeDir != "" {
		paths = append(paths, filepath.Join(homeDir, ".config", "read", "harnesses.json"))
	}

	// 3. Explicit custom configuration file argument
	if customConfigFile != "" {
		paths = append(paths, customConfigFile)
	}

	seenPaths := make(map[string]bool)
	for _, p := range paths {
		clean := filepath.Clean(p)
		if seenPaths[clean] {
			continue
		}
		seenPaths[clean] = true

		loadConfigFile(registry, clean)
	}
}

// loadConfigFile loads a single JSON configuration file into the registry.
func loadConfigFile(registry *HarnessRegistry, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var harnesses []AgentHarness

	// Try parsing as []AgentHarness
	if err := json.Unmarshal(data, &harnesses); err != nil {
		// Try parsing as struct with "harnesses" field
		var container struct {
			Harnesses []AgentHarness `json:"harnesses"`
		}
		if err2 := json.Unmarshal(data, &container); err2 == nil {
			harnesses = container.Harnesses
		} else {
			return
		}
	}

	for _, h := range harnesses {
		if h.ID == "" {
			continue
		}
		if h.Name == "" {
			h.Name = h.ID
		}
		if h.Type == "" {
			h.Type = "custom"
		}
		if h.Status == "" {
			h.Status = "available"
		}
		h.Enabled = true

		registry.AddHarness(h)
	}
}
