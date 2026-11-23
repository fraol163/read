package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DaemonConfig holds settings for the host daemon gateway.
type DaemonConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	WorkspaceRoot string `json:"workspace_root"`
	EngineBin     string `json:"engine_bin"`
	CustomHarness string `json:"custom_harness"`
}

// StorageConfig holds settings for the 3-way storage engine.
type StorageConfig struct {
	DefaultPolicy       string   `json:"default_policy"`
	SyncIntervalSeconds int      `json:"sync_interval_seconds"`
	ChunkSizeBytes      int      `json:"chunk_size_bytes"`
	ExcludePatterns     []string `json:"exclude_patterns"`
}

// CoreConfig holds settings for state and WAL persistence.
type CoreConfig struct {
	StateDir               string `json:"state_dir"`
	StateFile              string `json:"state_file"`
	WalFile                string `json:"wal_file"`
	StrictCas              bool   `json:"strict_cas"`
	WalCompactionThreshold int    `json:"wal_compaction_threshold"`
}

// AstConfig holds settings for the C0 AST resolver.
type AstConfig struct {
	MaxDepth           int      `json:"max_depth"`
	TimeoutMs          int      `json:"timeout_ms"`
	SupportedLanguages []string `json:"supported_languages"`
}

// MutationConfig holds settings for AROP bounded mutation testing.
type MutationConfig struct {
	MaxMutants int      `json:"max_mutants"`
	TimeoutMs  int      `json:"timeout_ms"`
	Operators  []string `json:"operators"`
}

// HarnessConfig holds settings for AI agent discovery and local inference servers.
type HarnessConfig struct {
	DiscoveryEnabled   bool              `json:"discovery_enabled"`
	ScanTimeoutMs      int               `json:"scan_timeout_ms"`
	LocalEndpoints     map[string]string `json:"local_endpoints"`
	CustomConfigPaths  []string          `json:"custom_config_paths"`
}

// SecurityConfig holds execution security and path sandboxing options.
type SecurityConfig struct {
	SandboxEnabled    bool     `json:"sandbox_enabled"`
	AllowProcessSpawn bool     `json:"allow_process_spawn"`
	AllowedPaths      []string `json:"allowed_paths"`
	BlockedCommands   []string `json:"blocked_commands"`
}

// LoggingConfig holds log level and output formatting options.
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	File   string `json:"file"`
}

// Config represents the complete READ configuration schema.
type Config struct {
	Version  string         `json:"version"`
	Daemon   DaemonConfig   `json:"daemon"`
	Storage  StorageConfig  `json:"storage"`
	Core     CoreConfig     `json:"core"`
	Ast      AstConfig      `json:"ast"`
	Mutation MutationConfig `json:"mutation"`
	Harness  HarnessConfig  `json:"harness"`
	Security SecurityConfig `json:"security"`
	Logging  LoggingConfig  `json:"logging"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version: "0.7.0",
		Daemon: DaemonConfig{
			Host:          "127.0.0.1",
			Port:          8089,
			WorkspaceRoot: ".",
			EngineBin:     "",
			CustomHarness: "",
		},
		Storage: StorageConfig{
			DefaultPolicy:       "both_sync",
			SyncIntervalSeconds: 30,
			ChunkSizeBytes:      1048576,
			ExcludePatterns: []string{
				".git",
				".method",
				"target",
				"node_modules",
				"__pycache__",
				".venv",
			},
		},
		Core: CoreConfig{
			StateDir:               ".read",
			StateFile:              ".read/STATE.json",
			WalFile:                ".read/WAL.log",
			StrictCas:              true,
			WalCompactionThreshold: 1000,
		},
		Ast: AstConfig{
			MaxDepth:  3,
			TimeoutMs: 100,
			SupportedLanguages: []string{
				"python",
				"typescript",
				"rust",
				"go",
			},
		},
		Mutation: MutationConfig{
			MaxMutants: 10,
			TimeoutMs:  4500,
			Operators: []string{
				"INVERT_EQ",
				"FLIP_BOOL_TRUE",
				"FLIP_BOOL_FALSE",
				"BOUNDARY_SHIFT",
				"RETURN_INVERT",
			},
		},
		Harness: HarnessConfig{
			DiscoveryEnabled: true,
			ScanTimeoutMs:    500,
			LocalEndpoints: map[string]string{
				"ollama":   "http://127.0.0.1:11434",
				"llamacpp": "http://127.0.0.1:8080",
				"vllm":     "http://127.0.0.1:8000",
			},
			CustomConfigPaths: []string{
				".read/harnesses.json",
				"~/.config/read/harnesses.json",
			},
		},
		Security: SecurityConfig{
			SandboxEnabled:    true,
			AllowProcessSpawn: true,
			AllowedPaths:      []string{"."},
			BlockedCommands: []string{
				"rm -rf /",
				":(){ :|:& };:",
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			File:   "",
		},
	}
}

// GlobalConfigPath returns the path to ~/.config/read/config.json.
func GlobalConfigPath() string {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "read", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv("HOME")
	}
	if home != "" {
		return filepath.Join(home, ".config", "read", "config.json")
	}
	return ""
}

// LocalConfigPath returns the path to <workspace>/.read/config.json.
func LocalConfigPath(workspace string) string {
	if workspace == "" {
		workspace = "."
	}
	return filepath.Join(workspace, ".read", "config.json")
}

// LoadConfig loads the active configuration by merging defaults, system config, global config, and workspace local config.
func LoadConfig(workspace string) *Config {
	cfg := DefaultConfig()

	// 1. Merge system config (/etc/read/config.json) if it exists
	if _, err := os.Stat("/etc/read/config.json"); err == nil {
		mergeFromFile(cfg, "/etc/read/config.json")
	}

	// 2. Merge user global config (~/.config/read/config.json) if it exists
	globalPath := GlobalConfigPath()
	if globalPath != "" {
		mergeFromFile(cfg, globalPath)
	}

	// 3. Merge local config (.read/config.json) if it exists
	localPath := LocalConfigPath(workspace)
	if localPath != "" {
		mergeFromFile(cfg, localPath)
	}

	return cfg
}

func mergeFromFile(cfg *Config, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, cfg)
}
