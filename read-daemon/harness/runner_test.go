package harness

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"read-daemon/scanner"
)

func createTestRegistry(binPath string) *scanner.HarnessRegistry {
	return &scanner.HarnessRegistry{
		Harnesses: []scanner.AgentHarness{
			{
				ID:      "test_harness",
				Name:    "Test Harness",
				Type:    "cli",
				Path:    binPath,
				Enabled: true,
			},
		},
	}
}

func TestRunnerProcessGroupCleanup(t *testing.T) {
	pidFile, err := os.CreateTemp("", "runner_pids_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	pidFilePath := pidFile.Name()
	pidFile.Close()
	defer os.Remove(pidFilePath)

	// A script that starts a child background process
	scriptFile, err := os.CreateTemp("", "runner_script_*.sh")
	if err != nil {
		t.Fatalf("failed to create temp script: %v", err)
	}
	scriptPath := scriptFile.Name()
	scriptContent := fmt.Sprintf(`#!/bin/bash
sleep 100 &
CHILD_PID=$!
echo "$CHILD_PID" > %s
wait
`, pidFilePath)
	_ = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	scriptFile.Close()
	defer os.Remove(scriptPath)

	runner := &Runner{tasks: make(map[string]*ActiveTask)}
	registry := createTestRegistry("/bin/bash")

	req := HarnessRunRequest{
		HarnessID: "test_harness",
		Args:      []string{scriptPath},
	}

	task, err := runner.StartTask(req, registry)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Wait for child PID to be written
	var childPid int
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		content, err := os.ReadFile(pidFilePath)
		if err == nil {
			trimmed := strings.TrimSpace(string(content))
			if pid, err := strconv.Atoi(trimmed); err == nil && pid > 0 {
				childPid = pid
				break
			}
		}
	}

	if childPid <= 0 {
		_ = runner.StopTask(task.ID)
		t.Fatalf("failed to read child PID within timeout")
	}

	// Confirm child is running
	if err := syscall.Kill(childPid, 0); err != nil {
		t.Fatalf("child process %d is not running: %v", childPid, err)
	}

	// Stop task via runner (must clean up process group)
	if err := runner.StopTask(task.ID); err != nil {
		t.Fatalf("StopTask failed: %v", err)
	}

	time.Sleep(150 * time.Millisecond)

	// Confirm child is terminated (no orphan)
	if err := syscall.Kill(childPid, 0); err == nil {
		t.Fatalf("child process %d is still alive after runner.StopTask!", childPid)
	}
}

func TestRunnerTimeout(t *testing.T) {
	runner := &Runner{tasks: make(map[string]*ActiveTask)}
	registry := createTestRegistry("/bin/sleep")

	req := HarnessRunRequest{
		HarnessID:      "test_harness",
		Args:           []string{"10"},
		TimeoutSeconds: 1, // 1 second timeout
	}

	task, err := runner.StartTask(req, registry)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Wait for timeout to trigger (within 2 seconds)
	deadline := time.Now().Add(3 * time.Second)
	timedOut := false
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		task.mutex.RLock()
		status := task.Status
		events := append([]HarnessEvent{}, task.Events...)
		task.mutex.RUnlock()

		if status == StatusFailed {
			for _, ev := range events {
				if strings.Contains(ev.Content, "exceeded timeout limit") {
					timedOut = true
					break
				}
			}
			if timedOut {
				break
			}
		}
	}

	if !timedOut {
		t.Fatalf("expected task to fail due to timeout, task status: %v", task.Status)
	}
}

func TestRunnerMemoryGuard(t *testing.T) {
	// Python one-liner allocating ~50MB RSS
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available, skipping memory guard test")
	}

	runner := &Runner{tasks: make(map[string]*ActiveTask)}
	registry := createTestRegistry(pythonPath)

	// Set low memory limit of 15 MB
	req := HarnessRunRequest{
		HarnessID:      "test_harness",
		Args:           []string{"-c", "import time; x = bytearray(40 * 1024 * 1024); time.sleep(5)"},
		MaxMemoryBytes: 15 * 1024 * 1024, // 15 MB
		TimeoutSeconds: 10,
	}

	task, err := runner.StartTask(req, registry)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Wait for memory guard to trigger termination
	deadline := time.Now().Add(10 * time.Second)
	memoryKilled := false
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		task.mutex.RLock()
		status := task.Status
		events := append([]HarnessEvent{}, task.Events...)
		task.mutex.RUnlock()

		if status == StatusFailed {
			for _, ev := range events {
				if strings.Contains(ev.Content, "exceeded memory limit") || strings.Contains(ev.Content, "signal:") || strings.Contains(ev.Content, "MemoryError") || strings.Contains(ev.Content, "exit") {
					memoryKilled = true
					break
				}
			}
			if memoryKilled {
				break
			}
		}
	}

	if !memoryKilled {
		t.Fatalf("expected task to be terminated by memory guard, status: %v", task.Status)
	}
}

func TestRunnerKeyRedactionInEvents(t *testing.T) {
	runner := &Runner{tasks: make(map[string]*ActiveTask)}
	registry := createTestRegistry("/bin/bash")

	secretOutput := "OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890"
	req := HarnessRunRequest{
		HarnessID: "test_harness",
		Args:      []string{"-c", fmt.Sprintf("echo %s", secretOutput)},
	}

	task, err := runner.StartTask(req, registry)
	if err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	// Wait for completion
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		task.mutex.RLock()
		status := task.Status
		task.mutex.RUnlock()
		if status == StatusCompleted {
			break
		}
	}

	task.mutex.RLock()
	defer task.mutex.RUnlock()

	foundRedacted := false
	for _, ev := range task.Events {
		if strings.Contains(ev.Content, "OPENAI_API_KEY=***REDACTED***") {
			foundRedacted = true
		}
		if strings.Contains(ev.Content, "sk-proj-1234567890abcdef") {
			t.Fatalf("found unmasked API key in event: %s", ev.Content)
		}
	}

	if !foundRedacted {
		t.Fatalf("expected redacted event output in task events")
	}
}
