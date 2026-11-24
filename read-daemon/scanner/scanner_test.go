package scanner

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHarnessRegistry(t *testing.T) {
	reg := NewHarnessRegistry()

	h1 := AgentHarness{
		ID:          "test_agent",
		Name:        "Test Agent",
		Type:        "cli",
		Path:        "/bin/test",
		Version:     "1.0.0",
		Status:      "available",
		Enabled:     true,
		Description: "A test harness",
		Metadata: map[string]string{
			"source": "unit_test",
		},
	}

	reg.AddHarness(h1)

	got, found := reg.GetHarness("test_agent")
	if !found {
		t.Fatalf("expected harness test_agent to be found")
	}
	if got.Name != "Test Agent" || got.Version != "1.0.0" {
		t.Errorf("unexpected harness values: %+v", got)
	}

	// Update existing harness
	h1Update := AgentHarness{
		ID:      "test_agent",
		Version: "1.1.0",
		Metadata: map[string]string{
			"updated": "true",
		},
	}
	reg.AddHarness(h1Update)

	gotUpdated, _ := reg.GetHarness("test_agent")
	if gotUpdated.Version != "1.1.0" {
		t.Errorf("expected version 1.1.0, got %s", gotUpdated.Version)
	}
	if gotUpdated.Metadata["updated"] != "true" || gotUpdated.Metadata["source"] != "unit_test" {
		t.Errorf("expected merged metadata, got %+v", gotUpdated.Metadata)
	}

	// Test by type
	cliList := reg.HarnessesByType("cli")
	if len(cliList) != 1 {
		t.Errorf("expected 1 cli harness, got %d", len(cliList))
	}
	serverList := reg.HarnessesByType("local_server")
	if len(serverList) != 0 {
		t.Errorf("expected 0 server harnesses, got %d", len(serverList))
	}
}

func TestCLIAgentDiscovery(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock executables
	mockBinaries := map[string]string{
		"claude": "#!/bin/sh\necho 'Claude CLI version 1.2.3'\n",
		"aider":  "#!/bin/sh\necho 'aider 0.45.0'\n",
		"goose":  "#!/bin/sh\necho 'goose version 0.8.0'\n",
		"dsh":    "#!/bin/sh\necho 'dsh 0.2.2'\n",
	}

	for bin, content := range mockBinaries {
		binPath := filepath.Join(tmpDir, bin)
		if err := os.WriteFile(binPath, []byte(content), 0755); err != nil {
			t.Fatalf("failed to write mock binary %s: %v", bin, err)
		}
	}

	// Create a non-executable file that matches an agent name to verify zero false positives
	nonExecPath := filepath.Join(tmpDir, "cursor")
	if err := os.WriteFile(nonExecPath, []byte("not executable"), 0644); err != nil {
		t.Fatalf("failed to write non-executable file: %v", err)
	}

	reg := NewHarnessRegistry()
	ScanCLIAgents(reg, []string{tmpDir})

	// Verify mock binaries were detected
	expectedIDs := []string{"claude_code", "aider", "goose", "deepseek_harness"}
	for _, id := range expectedIDs {
		h, found := reg.GetHarness(id)
		if !found {
			t.Errorf("expected %s to be detected", id)
			continue
		}
		if h.Status != "available" {
			t.Errorf("expected %s status available, got %s", id, h.Status)
		}
		if h.Version == "unknown" || h.Version == "" {
			t.Errorf("expected valid version for %s, got %s", id, h.Version)
		}
	}

	// Verify non-executable was not picked up from tmpDir unless found elsewhere on system
	if cursorHarness, found := reg.GetHarness("cursor_agent"); found {
		if cursorHarness.Path == nonExecPath {
			t.Errorf("non-executable file at %s was erroneously detected", nonExecPath)
		}
	}
}

func TestServerProbing_Ollama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[{"name":"llama3:8b","model":"llama3:8b"},{"name":"qwen2.5-coder:7b","model":"qwen2.5-coder:7b"}]}`))
		case "/api/version":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"0.1.32"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected Ollama server to be fingerprinted on port %d", port)
	}

	if result.ID != "ollama" {
		t.Errorf("expected ID ollama, got %s", result.ID)
	}
	if result.Version != "0.1.32" {
		t.Errorf("expected version 0.1.32, got %s", result.Version)
	}
	if len(result.Models) != 2 || result.Models[0] != "llama3:8b" {
		t.Errorf("unexpected models: %+v", result.Models)
	}
}

func TestServerProbing_VLLM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/version":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"0.4.2"}`))
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"meta-llama/Meta-Llama-3-8B-Instruct"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected vLLM server to be fingerprinted on port %d", port)
	}

	if result.ID != "vllm_server" {
		t.Errorf("expected ID vllm_server, got %s", result.ID)
	}
	if result.Version != "0.4.2" {
		t.Errorf("expected version 0.4.2, got %s", result.Version)
	}
	if len(result.Models) != 1 || result.Models[0] != "meta-llama/Meta-Llama-3-8B-Instruct" {
		t.Errorf("unexpected models: %+v", result.Models)
	}
}

func TestServerProbing_LlamaCpp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/props":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"default_generation_settings":{"n_predict":-1}}`))
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"qwen-2.5-7b-instruct.Q4_K_M.gguf"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected llama.cpp server to be fingerprinted on port %d", port)
	}

	if result.ID != "llama_cpp_server" {
		t.Errorf("expected ID llama_cpp_server, got %s", result.ID)
	}
	if len(result.Models) != 1 || result.Models[0] != "qwen-2.5-7b-instruct.Q4_K_M.gguf" {
		t.Errorf("unexpected models: %+v", result.Models)
	}
}

func TestServerProbing_LocalAI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/readyz":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("readyz: ok"))
		case "/v1/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected LocalAI server to be fingerprinted on port %d", port)
	}

	if result.ID != "localai_server" {
		t.Errorf("expected ID localai_server, got %s", result.ID)
	}
}

func TestServerProbing_TGI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/info":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"model_id":"mistralai/Mistral-7B-v0.1","version":"1.4.0"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected TGI server to be fingerprinted on port %d", port)
	}

	if result.ID != "tgi_server" {
		t.Errorf("expected ID tgi_server, got %s", result.ID)
	}
	if result.Version != "1.4.0" {
		t.Errorf("expected version 1.4.0, got %s", result.Version)
	}
	if len(result.Models) != 1 || result.Models[0] != "mistralai/Mistral-7B-v0.1" {
		t.Errorf("unexpected models: %+v", result.Models)
	}
}

func TestServerProbing_Tabby(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/health":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"model":"StarCoder-1B","version":{"git_commit":"abcdef"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected Tabby server to be fingerprinted on port %d", port)
	}

	if result.ID != "tabby_server" {
		t.Errorf("expected ID tabby_server, got %s", result.ID)
	}
	if len(result.Models) != 1 || result.Models[0] != "StarCoder-1B" {
		t.Errorf("unexpected models: %+v", result.Models)
	}
}

func TestServerProbing_Triton(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/health/ready":
			w.WriteHeader(http.StatusOK)
		case "/v2/models":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"models":[{"name":"tensorrt_llm_ensemble"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	result, found := FingerprintServer("127.0.0.1", port)
	if !found {
		t.Fatalf("expected Triton server to be fingerprinted on port %d", port)
	}

	if result.ID != "triton_inference_server" {
		t.Errorf("expected ID triton_inference_server, got %s", result.ID)
	}
}

func TestServerProbing_ZeroFalsePositivesOnGenericHTTP(t *testing.T) {
	// A generic HTTP server (e.g. static website or random service) returning 404 or non-LLM HTML
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body>Hello World</body></html>"))
	}))
	defer server.Close()

	port := extractPort(t, server.URL)
	_, found := FingerprintServer("127.0.0.1", port)
	if found {
		t.Errorf("generic web server on port %d was erroneously identified as LLM server", port)
	}
}

func TestContainerParsing(t *testing.T) {
	sampleOutput := `c1a2b3c4d5e6	ollama_prod	ollama/ollama:latest	Up 2 hours	0.0.0.0:11434->11434/tcp
f6e5d4c3b2a1	vllm_service	vllm/vllm-openai:v0.4.0	Up 5 hours	0.0.0.0:8000->8000/tcp
112233445566	openhands_agent	all-hands-ai/openhands:0.9	Up 1 hour	0.0.0.0:3000->3000/tcp
998877665544	database_pg	postgres:16-alpine	Up 2 days	0.0.0.0:5432->5432/tcp
aabbccddeeff	web_proxy	nginx:alpine	Up 3 days	0.0.0.0:80->80/tcp
`

	harnesses := ParseContainerOutput("docker", sampleOutput)

	// postgres and nginx must be filtered out
	if len(harnesses) != 3 {
		t.Fatalf("expected 3 AI containers, got %d", len(harnesses))
	}

	expectedIDs := map[string]string{
		"docker_ollama_prod":     "ollama/ollama:latest",
		"docker_vllm_service":    "vllm/vllm-openai:v0.4.0",
		"docker_openhands_agent": "all-hands-ai/openhands:0.9",
	}

	for _, h := range harnesses {
		expectedImage, exists := expectedIDs[h.ID]
		if !exists {
			t.Errorf("unexpected container harness ID: %s", h.ID)
			continue
		}
		if h.Type != "docker" {
			t.Errorf("expected type docker, got %s", h.Type)
		}
		if h.Metadata["image"] != expectedImage {
			t.Errorf("expected image %s, got %s", expectedImage, h.Metadata["image"])
		}
		if h.Status != "running" {
			t.Errorf("expected status running, got %s", h.Status)
		}
	}
}

func TestCustomConfigLoading(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Array format in custom config file
	customFile := filepath.Join(tmpDir, "custom.json")
	customData := []AgentHarness{
		{
			ID:          "custom_agent_1",
			Name:        "Custom Agent 1",
			Path:        "/usr/local/bin/agent1",
			Description: "Custom AI Agent",
		},
	}
	bytes1, _ := json.Marshal(customData)
	if err := os.WriteFile(customFile, bytes1, 0644); err != nil {
		t.Fatalf("failed to write custom file: %v", err)
	}

	// 2. Object format in .read/harnesses.json
	readDir := filepath.Join(tmpDir, ".read")
	if err := os.MkdirAll(readDir, 0755); err != nil {
		t.Fatalf("failed to create .read dir: %v", err)
	}
	workspaceFile := filepath.Join(readDir, "harnesses.json")
	workspaceData := map[string][]AgentHarness{
		"harnesses": {
			{
				ID:          "project_local_agent",
				Name:        "Project Local Agent",
				Path:        filepath.Join(tmpDir, "agent.sh"),
				Description: "Workspace local harness",
			},
		},
	}
	bytes2, _ := json.Marshal(workspaceData)
	if err := os.WriteFile(workspaceFile, bytes2, 0644); err != nil {
		t.Fatalf("failed to write workspace file: %v", err)
	}

	reg := NewHarnessRegistry()
	LoadCustomConfigs(reg, tmpDir, customFile)

	h1, found1 := reg.GetHarness("custom_agent_1")
	if !found1 {
		t.Fatalf("expected custom_agent_1 to be loaded")
	}
	if h1.Type != "custom" || h1.Status != "available" || !h1.Enabled {
		t.Errorf("unexpected defaults for custom_agent_1: %+v", h1)
	}

	h2, found2 := reg.GetHarness("project_local_agent")
	if !found2 {
		t.Fatalf("expected project_local_agent to be loaded")
	}
	if h2.Name != "Project Local Agent" {
		t.Errorf("unexpected name for project_local_agent: %s", h2.Name)
	}
}

func TestScanHost_Live(t *testing.T) {
	start := time.Now()
	reg, err := ScanHost("")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ScanHost failed with error: %v", err)
	}
	if reg == nil {
		t.Fatalf("expected non-nil registry")
	}

	// Verify scan speed constraint (< 2 seconds under high parallel load)
	if elapsed > 2*time.Second {
		t.Errorf("ScanHost took %v, which exceeds the speed constraint", elapsed)
	}

	t.Logf("ScanHost completed in %v. Detected %d harnesses:", elapsed, len(reg.Harnesses))
	for _, h := range reg.Harnesses {
		t.Logf("- [%s] %s (%s) at %s [status: %s, version: %s]", h.Type, h.Name, h.ID, h.Path, h.Status, h.Version)
	}
}

func BenchmarkScanHost(b *testing.B) {
	for i := 0; i < b.N; i++ {
		reg, err := ScanHost("")
		if err != nil {
			b.Fatalf("ScanHost error: %v", err)
		}
		if reg == nil {
			b.Fatalf("nil registry")
		}
	}
}

func extractPort(t *testing.T, rawURL string) int {
	parts := strings.Split(rawURL, ":")
	if len(parts) < 3 {
		t.Fatalf("invalid URL: %s", rawURL)
	}
	portStr := parts[len(parts)-1]
	p, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("invalid port in %s: %v", rawURL, err)
	}
	return p
}

// Ensure net import is used
var _ = net.JoinHostPort
