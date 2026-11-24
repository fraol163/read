package scanner

import (
	"sync"
	"time"
)

// AgentHarness represents a detected AI agent harness or local inference server on the host system.
type AgentHarness struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"` // cli, local_server, docker, custom
	Path        string            `json:"path"`
	Version     string            `json:"version,omitempty"`
	Status      string            `json:"status"` // available, running, offline
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// HarnessRegistry maintains the list of detected and user-configured harnesses.
type HarnessRegistry struct {
	mu        sync.RWMutex
	Harnesses []AgentHarness `json:"harnesses"`
}

// NewHarnessRegistry initializes an empty HarnessRegistry.
func NewHarnessRegistry() *HarnessRegistry {
	return &HarnessRegistry{
		Harnesses: make([]AgentHarness, 0),
	}
}

// GetHarness retrieves a harness by ID.
func (r *HarnessRegistry) GetHarness(id string) (*AgentHarness, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.Harnesses {
		if r.Harnesses[i].ID == id {
			return &r.Harnesses[i], true
		}
	}
	return nil, false
}

// AddHarness appends or updates a harness in the registry.
func (r *HarnessRegistry) AddHarness(h AgentHarness) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.Harnesses {
		if r.Harnesses[i].ID == h.ID {
			// Update existing record if new record provides more information
			if h.Version != "" && h.Version != "unknown" {
				r.Harnesses[i].Version = h.Version
			}
			if h.Status != "" {
				r.Harnesses[i].Status = h.Status
			}
			if h.Path != "" {
				r.Harnesses[i].Path = h.Path
			}
			if h.Metadata != nil {
				if r.Harnesses[i].Metadata == nil {
					r.Harnesses[i].Metadata = make(map[string]string)
				}
				for k, v := range h.Metadata {
					r.Harnesses[i].Metadata[k] = v
				}
			}
			return
		}
	}

	r.Harnesses = append(r.Harnesses, h)
}

// HarnessesByType returns all harnesses matching a given type (cli, local_server, docker, custom).
func (r *HarnessRegistry) HarnessesByType(harnessType string) []AgentHarness {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]AgentHarness, 0)
	for _, h := range r.Harnesses {
		if h.Type == harnessType {
			result = append(result, h)
		}
	}
	return result
}

// ScanOptions configures host scanning behavior.
type ScanOptions struct {
	CustomConfigFile string
	ScanCLI          bool
	ScanServers      bool
	ScanContainers   bool
	ScanConfigs      bool
	Timeout          time.Duration
	ExtraSearchPaths []string
	ServerPorts      []int
	ServerHosts      []string
	WorkspaceDir     string
}

// DefaultScanOptions returns optimized default scan options.
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		ScanCLI:        true,
		ScanServers:    true,
		ScanContainers: true,
		ScanConfigs:    true,
		Timeout:        950 * time.Millisecond,
		ServerPorts:    []int{11434, 8080, 8000, 1234, 1337, 2242, 5000, 5001, 9997, 4000},
		ServerHosts:    []string{"127.0.0.1", "localhost"},
	}
}
