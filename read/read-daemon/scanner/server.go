package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ServerProbeResult holds the outcome of a server probe.
type ServerProbeResult struct {
	ID          string
	Name        string
	Type        string
	Address     string
	Port        int
	Version     string
	Models      []string
	Description string
	Engine      string
}

// DiscoverListeningTCPPorts inspects /proc/net/tcp and /proc/net/tcp6 to dynamically discover
// all active listening TCP ports bound to localhost without any hardcoded port lists.
func DiscoverListeningTCPPorts() []int {
	ports := make([]int, 0)
	seen := make(map[int]bool)

	parseFile := func(path string) {
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == 0 || strings.TrimSpace(line) == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			// fields[1] = local_address, fields[3] = st (state)
			state := fields[3]
			if state != "0A" { // 0A = TCP_LISTEN
				continue
			}
			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}
			ipHex := parts[0]
			portHex := parts[1]

			// Check if loopback or all interfaces:
			// 0100007F = 127.0.0.1 (little endian)
			// 00000000 = 0.0.0.0
			isLocal := ipHex == "0100007F" || ipHex == "00000000" ||
				strings.HasPrefix(ipHex, "0000000000000000") ||
				ipHex == "00000000000000000000000001000000"

			if isLocal {
				port64, err := strconv.ParseInt(portHex, 16, 32)
				if err == nil && port64 > 0 && port64 < 65536 {
					p := int(port64)
					// Probe non-privileged application ports
					if p >= 1024 && !seen[p] {
						seen[p] = true
						ports = append(ports, p)
					}
				}
			}
		}
	}

	parseFile("/proc/net/tcp")
	parseFile("/proc/net/tcp6")

	if len(ports) == 0 {
		// Non-Linux fallback (macOS, BSD) using lsof
		if out, err := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P").Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				idx := strings.LastIndex(line, ":")
				if idx != -1 {
					sub := line[idx+1:]
					fields := strings.Fields(sub)
					if len(fields) > 0 {
						if p, err := strconv.Atoi(fields[0]); err == nil && p >= 1024 && !seen[p] {
							seen[p] = true
							ports = append(ports, p)
						}
					}
				}
			}
		}
	}

	return ports
}

// ProbeLocalServers dynamically discovers and inspects LLM inference ports concurrently.
func ProbeLocalServers(registry *HarnessRegistry, ports []int, hosts []string) {
	discovered := DiscoverListeningTCPPorts()

	// Combine explicitly requested ports with dynamically discovered listening ports
	allPorts := make([]int, 0, len(ports)+len(discovered)+16)
	seen := make(map[int]bool)

	for _, p := range ports {
		if !seen[p] {
			seen[p] = true
			allPorts = append(allPorts, p)
		}
	}

	for _, p := range discovered {
		if !seen[p] {
			seen[p] = true
			allPorts = append(allPorts, p)
		}
	}

	// Always ensure standard AI inference fallback ports are evaluated
	standardDefaults := []int{11434, 8080, 8000, 1234, 1337, 2242, 5000, 5001, 9997, 4000}
	for _, p := range standardDefaults {
		if !seen[p] {
			seen[p] = true
			allPorts = append(allPorts, p)
		}
	}

	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1"}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	scannedPorts := make(map[int]bool)

	for _, port := range allPorts {
		if scannedPorts[port] {
			continue
		}
		scannedPorts[port] = true

		wg.Add(1)
		go func(p int) {
			defer wg.Done()

			// Check TCP connectivity with 200ms timeout
			addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(p))
			conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
			if err != nil {
				return
			}
			conn.Close()

			// Port is open: execute HTTP fingerprinting
			result, found := fingerprintServer("127.0.0.1", p)
			if !found {
				return
			}

			harness := AgentHarness{
				ID:          result.ID,
				Name:        result.Name,
				Type:        "local_server",
				Path:        fmt.Sprintf("http://127.0.0.1:%d", p),
				Version:     result.Version,
				Status:      "running",
				Enabled:     true,
				Description: result.Description,
				Metadata: map[string]string{
					"endpoint": fmt.Sprintf("http://127.0.0.1:%d", p),
					"engine":   result.Engine,
					"port":     strconv.Itoa(p),
				},
			}

			if len(result.Models) > 0 {
				harness.Metadata["models"] = strings.Join(result.Models, ", ")
			}

			mu.Lock()
			registry.AddHarness(harness)
			mu.Unlock()
		}(port)
	}

	wg.Wait()
}

// FingerprintServer probes a specific host and port for known LLM server APIs.
func FingerprintServer(host string, port int) (*ServerProbeResult, bool) {
	return fingerprintServer(host, port)
}

func fingerprintServer(host string, port int) (*ServerProbeResult, bool) {
	type probeOutcome struct {
		result *ServerProbeResult
		found  bool
	}
	done := make(chan probeOutcome, 1)

	go func() {
		client := &http.Client{
			Timeout: 200 * time.Millisecond,
		}
		baseURL := fmt.Sprintf("http://%s:%d", host, port)

		// 1. Check Ollama API (/api/tags and /api/version)
		if res, ok := probeOllama(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 2. Check Triton Inference Server (/v2/health/ready and /v2/models)
		if res, ok := probeTriton(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 3. Check TGI (/info)
		if res, ok := probeTGI(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 4. Check Tabby (/v1/health or /v1beta/health)
		if res, ok := probeTabby(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 5. Check LocalAI (/readyz)
		if res, ok := probeLocalAI(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 6. Check llama.cpp (/props or /slots)
		if res, ok := probeLlamaCpp(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 7. Check vLLM (/version or /health)
		if res, ok := probeVLLM(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		// 8. General OpenAI Compatible endpoint check (/v1/models)
		if res, ok := probeOpenAICompatible(client, baseURL, port); ok {
			done <- probeOutcome{res, true}
			return
		}

		done <- probeOutcome{nil, false}
	}()

	select {
	case outcome := <-done:
		return outcome.result, outcome.found
	case <-time.After(400 * time.Millisecond):
		return nil, false
	}
}

func probeOllama(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, false
	}

	var data struct {
		Models []struct {
			Name  string `json:"name"`
			Model string `json:"model"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &data); err != nil || len(data.Models) == 0 {
		return nil, false
	}

	models := make([]string, 0, len(data.Models))
	for _, m := range data.Models {
		if m.Name != "" {
			models = append(models, m.Name)
		} else if m.Model != "" {
			models = append(models, m.Model)
		}
	}

	version := "unknown"
	if vResp, err := client.Get(baseURL + "/api/version"); err == nil && vResp.StatusCode == http.StatusOK {
		defer vResp.Body.Close()
		var vData struct {
			Version string `json:"version"`
		}
		if vBody, err := io.ReadAll(io.LimitReader(vResp.Body, 4096)); err == nil {
			if json.Unmarshal(vBody, &vData) == nil && vData.Version != "" {
				version = vData.Version
			}
		}
	}

	return &ServerProbeResult{
		ID:          "ollama",
		Name:        "Ollama Local LLM Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: "Local quantized LLM server running models via Ollama API",
		Engine:      "ollama",
	}, true
}

func probeTriton(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/v2/health/ready")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer resp.Body.Close()

	mResp, err := client.Get(baseURL + "/v2/models")
	if err != nil || mResp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer mResp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(mResp.Body, 65536))
	if err != nil {
		return nil, false
	}

	var mData struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &mData); err != nil || len(mData.Models) == 0 {
		return nil, false
	}

	models := make([]string, 0, len(mData.Models))
	for _, m := range mData.Models {
		if m.Name != "" {
			models = append(models, m.Name)
		}
	}

	return &ServerProbeResult{
		ID:          "triton_inference_server",
		Name:        "Triton Inference Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     "v2",
		Models:      models,
		Description: "NVIDIA Triton high-performance multi-framework inference server",
		Engine:      "triton",
	}, true
}

func probeTGI(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/info")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, false
	}

	var data struct {
		ModelID string `json:"model_id"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &data); err != nil || data.ModelID == "" {
		return nil, false
	}

	models := []string{data.ModelID}
	version := data.Version
	if version == "" {
		version = "unknown"
	}

	return &ServerProbeResult{
		ID:          "tgi_server",
		Name:        "Text Generation Inference (TGI)",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: "Hugging Face Text Generation Inference production server",
		Engine:      "tgi",
	}, true
}

func probeTabby(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	endpoint := baseURL + "/v1/health"
	resp, err := client.Get(endpoint)
	if err != nil || resp.StatusCode != http.StatusOK {
		endpoint = baseURL + "/v1beta/health"
		resp, err = client.Get(endpoint)
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, false
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, false
	}

	var data struct {
		Model   string `json:"model"`
		Version struct {
			BuildDate string `json:"build_date"`
			GitCommit string `json:"git_commit"`
		} `json:"version"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, false
	}
	if data.Model == "" && data.Version.GitCommit == "" {
		return nil, false
	}

	models := make([]string, 0)
	if data.Model != "" {
		models = append(models, data.Model)
	}

	version := "unknown"
	if data.Version.GitCommit != "" {
		version = data.Version.GitCommit
	}

	return &ServerProbeResult{
		ID:          "tabby_server",
		Name:        "Tabby Code Completion Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: "Self-hosted AI coding assistant server",
		Engine:      "tabby",
	}, true
}

func probeLocalAI(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/readyz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil || !strings.Contains(strings.ToLower(string(body)), "ready") {
		return nil, false
	}

	models, version, valid := fetchOpenAIModels(client, baseURL)
	if !valid {
		return nil, false
	}

	return &ServerProbeResult{
		ID:          "localai_server",
		Name:        "LocalAI Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: "LocalAI drop-in replacement REST API for OpenAI",
		Engine:      "localai",
	}, true
}

func probeLlamaCpp(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/props")
	if err != nil || resp.StatusCode != http.StatusOK {
		resp, err = client.Get(baseURL + "/slots")
		if err != nil || resp.StatusCode != http.StatusOK {
			return nil, false
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, false
	}

	var data struct {
		DefaultGenerationSettings map[string]interface{} `json:"default_generation_settings"`
		TotalSlots                int                    `json:"total_slots"`
	}
	if err := json.Unmarshal(body, &data); err != nil || (data.DefaultGenerationSettings == nil && data.TotalSlots == 0) {
		return nil, false
	}

	models, version, _ := fetchOpenAIModels(client, baseURL)

	return &ServerProbeResult{
		ID:          "llama_cpp_server",
		Name:        "llama.cpp HTTP Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: "Lightweight native C++ GGUF inference engine",
		Engine:      "llama_cpp",
	}, true
}

func probeVLLM(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	resp, err := client.Get(baseURL + "/version")
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, false
	}

	var data struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &data); err != nil || data.Version == "" {
		return nil, false
	}

	models, _, _ := fetchOpenAIModels(client, baseURL)

	return &ServerProbeResult{
		ID:          "vllm_server",
		Name:        "vLLM Inference Server",
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     data.Version,
		Models:      models,
		Description: "High-throughput, memory-efficient LLM serving engine",
		Engine:      "vllm",
	}, true
}

func probeOpenAICompatible(client *http.Client, baseURL string, port int) (*ServerProbeResult, bool) {
	models, version, valid := fetchOpenAIModels(client, baseURL)
	if !valid {
		return nil, false
	}

	id := fmt.Sprintf("openai_compatible_%d", port)
	name := fmt.Sprintf("OpenAI-Compatible LLM Server (:%d)", port)
	desc := "Local OpenAI-compatible inference server exposing /v1/models"
	engine := "openai_compatible"

	switch port {
	case 1234:
		id = "lm_studio"
		name = "LM Studio Local Server"
		desc = "LM Studio local developer inference server"
		engine = "lm_studio"
	case 1337:
		id = "jan_ai"
		name = "Jan Local AI Server"
		desc = "Jan open-source desktop AI local inference server"
		engine = "jan"
	case 2242:
		id = "aphrodite_engine"
		name = "Aphrodite Engine Server"
		desc = "Aphrodite high-throughput inference engine"
		engine = "aphrodite"
	case 8000:
		id = "fastchat_vllm_server"
		name = "FastChat / vLLM Inference Server"
		desc = "High-performance LLM serving server running on port 8000"
		engine = "fastchat"
	case 8080:
		id = "llama_cpp_server"
		name = "llama.cpp / Compatible HTTP Server"
		desc = "Native GGUF / local inference engine running on port 8080"
		engine = "llama_cpp"
	}

	return &ServerProbeResult{
		ID:          id,
		Name:        name,
		Type:        "local_server",
		Address:     baseURL,
		Port:        port,
		Version:     version,
		Models:      models,
		Description: desc,
		Engine:      engine,
	}, true
}

// fetchOpenAIModels attempts to query /v1/models and returns models, version, and whether it was valid.
func fetchOpenAIModels(client *http.Client, baseURL string) ([]string, string, bool) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return nil, "unknown", false
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, "unknown", false
	}
	defer resp.Body.Close()

	serverHeader := resp.Header.Get("Server")
	version := "unknown"
	if serverHeader != "" {
		version = serverHeader
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, version, false
	}

	var data struct {
		Object string `json:"object"`
		Data   []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, version, false
	}

	// OpenAI specification guarantees object == "list" or non-empty data array
	if data.Object != "list" && len(data.Data) == 0 {
		return nil, version, false
	}

	models := make([]string, 0, len(data.Data))
	for _, item := range data.Data {
		if item.ID != "" {
			models = append(models, item.ID)
		}
	}

	return models, version, true
}
