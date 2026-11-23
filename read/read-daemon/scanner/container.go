package scanner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ContainerKeywords lists substrings in image or container names indicating an AI harness or server.
var ContainerKeywords = []string{
	"ollama",
	"vllm",
	"localai",
	"local-ai",
	"llama.cpp",
	"llama-cpp",
	"llama_cpp",
	"text-generation-inference",
	"tgi",
	"tabby",
	"aphrodite",
	"open-webui",
	"openwebui",
	"open-interpreter",
	"interpreter",
	"auto-gpt",
	"autogpt",
	"gpt-engineer",
	"openhands",
	"devin",
	"fastchat",
	"tritonserver",
	"triton",
	"litellm",
	"jan",
	"deepseek",
	"aider",
	"qwen",
}

// InspectContainers checks Docker and Podman for running LLM and agent containers.
func InspectContainers(registry *HarnessRegistry) {
	var wg sync.WaitGroup

	runtimes := []string{"docker", "podman"}
	for _, rt := range runtimes {
		wg.Add(1)
		go func(runtime string) {
			defer wg.Done()

			if _, err := exec.LookPath(runtime); err != nil {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			cmd := exec.CommandContext(ctx, runtime, "ps", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}")
			var stdout bytes.Buffer
			cmd.Stdout = &stdout

			if err := cmd.Run(); err != nil {
				return
			}

			harnesses := ParseContainerOutput(runtime, stdout.String())
			for _, h := range harnesses {
				registry.AddHarness(h)
			}
		}(rt)
	}

	wg.Wait()
}

// ParseContainerOutput parses the tab-separated ps output from docker or podman.
func ParseContainerOutput(runtime, output string) []AgentHarness {
	results := make([]AgentHarness, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		id := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		image := strings.TrimSpace(parts[2])
		status := "running"
		ports := ""

		if len(parts) >= 4 {
			status = strings.TrimSpace(parts[3])
		}
		if len(parts) >= 5 {
			ports = strings.TrimSpace(parts[4])
		}

		if !isAIContainer(name, image) {
			continue
		}

		harnessID := fmt.Sprintf("%s_%s", runtime, sanitizeID(name))
		if harnessID == fmt.Sprintf("%s_", runtime) {
			harnessID = fmt.Sprintf("%s_%s", runtime, id)
		}

		harness := AgentHarness{
			ID:          harnessID,
			Name:        fmt.Sprintf("%s Container: %s", strings.Title(runtime), name),
			Type:        runtime,
			Path:        id,
			Status:      "running",
			Enabled:     true,
			Description: fmt.Sprintf("%s container running image %s (%s)", strings.Title(runtime), image, status),
			Metadata: map[string]string{
				"runtime":        runtime,
				"container_id":   id,
				"container_name": name,
				"image":          image,
				"ports":          ports,
				"status":         status,
			},
		}

		results = append(results, harness)
	}

	return results
}

func isAIContainer(name, image string) bool {
	combined := strings.ToLower(name + " " + image)
	for _, kw := range ContainerKeywords {
		if strings.Contains(combined, kw) {
			return true
		}
	}
	return false
}

func sanitizeID(name string) string {
	clean := strings.ToLower(name)
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "-", "_")
	clean = strings.ReplaceAll(clean, ":", "_")
	return clean
}
