package scanner

import (
	"sync"
)

// ScanHost dynamically detects all AI agent harnesses and local inference servers.
// It searches PATH, package managers (mise, asdf, npm, pipx, cargo, go, nix, snap, flatpak),
// probes local inference servers, inspects containers, and loads custom configs.
func ScanHost(customConfigFile string) (*HarnessRegistry, error) {
	opts := DefaultScanOptions()
	opts.CustomConfigFile = customConfigFile
	return ScanHostWithOptions(opts)
}

// ScanHostWithOptions performs concurrent discovery according to configured options.
func ScanHostWithOptions(opts ScanOptions) (*HarnessRegistry, error) {
	registry := NewHarnessRegistry()

	var wg sync.WaitGroup

	// 1. CLI Agent Discovery across package managers
	if opts.ScanCLI {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ScanCLIAgents(registry, opts.ExtraSearchPaths)
		}()
	}

	// 2. Local Inference Server Probing & HTTP Fingerprinting
	if opts.ScanServers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ProbeLocalServers(registry, opts.ServerPorts, opts.ServerHosts)
		}()
	}

	// 3. Container Runtime Inspection (Docker & Podman)
	if opts.ScanContainers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			InspectContainers(registry)
		}()
	}

	// 4. Custom Configuration Discovery (.read/harnesses.json, ~/.config/read/harnesses.json)
	if opts.ScanConfigs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			LoadCustomConfigs(registry, opts.WorkspaceDir, opts.CustomConfigFile)
		}()
	}

	wg.Wait()

	return registry, nil
}
