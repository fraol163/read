package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Version != "0.1.0" {
		t.Fatalf("expected version 0.1.0, got %s", cfg.Version)
	}
	if cfg.Daemon.Port != 8089 {
		t.Fatalf("expected port 8089, got %d", cfg.Daemon.Port)
	}
	if !cfg.Core.StrictCas {
		t.Fatalf("expected strict_cas true")
	}
	if cfg.Mutation.MaxMutants != 10 {
		t.Fatalf("expected max_mutants 10, got %d", cfg.Mutation.MaxMutants)
	}
}

func TestLocalConfigOverride(t *testing.T) {
	tmpDir := t.TempDir()
	dotRead := filepath.Join(tmpDir, ".read")
	if err := os.MkdirAll(dotRead, 0755); err != nil {
		t.Fatalf("failed to create .read dir: %v", err)
	}

	overrideJSON := `{
		"daemon": { "port": 9099 },
		"storage": { "default_policy": "phone_only" }
	}`
	if err := os.WriteFile(filepath.Join(dotRead, "config.json"), []byte(overrideJSON), 0644); err != nil {
		t.Fatalf("failed to write override config: %v", err)
	}

	cfg := LoadConfig(tmpDir)
	if cfg.Daemon.Port != 9099 {
		t.Fatalf("expected port 9099, got %d", cfg.Daemon.Port)
	}
	if cfg.Storage.DefaultPolicy != "phone_only" {
		t.Fatalf("expected default_policy phone_only, got %s", cfg.Storage.DefaultPolicy)
	}
	// Untouched default preserved
	if cfg.Core.StrictCas != true {
		t.Fatalf("expected strict_cas true")
	}
}
