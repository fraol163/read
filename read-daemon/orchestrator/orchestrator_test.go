package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func findEngineBin() string {
	if p, err := exec.LookPath("read-engine"); err == nil {
		return p
	}
	candidates := []string{
		"../../bin/read-engine",
		"../bin/read-engine",
		"./bin/read-engine",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if fi, err := os.Stat(abs); err == nil && !fi.IsDir() && (fi.Mode()&0111 != 0) {
				return abs
			}
		}
	}
	return ""
}

func setupTestWorkspace(t *testing.T) string {
	dir, err := os.MkdirTemp("", "read_orch_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s (%v)", string(out), err)
	}

	cmd = exec.Command("git", "config", "user.name", "READ Test")
	cmd.Dir = dir
	_ = cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@read.framework")
	cmd.Dir = dir
	_ = cmd.Run()

	_ = os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".read/\n.method/\n"), 0644)
	cmd = exec.Command("git", "add", ".gitignore")
	cmd.Dir = dir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "Ignore .read and .method")
	cmd.Dir = dir
	_ = cmd.Run()

	return dir
}

// createMockHarness writes a mock agent harness script that modifies code.
func createMockHarness(t *testing.T, dir string, scriptContent string) string {
	harnessDir, err := os.MkdirTemp("", "read_mock_harness_*")
	if err != nil {
		t.Fatalf("failed to create temp harness dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(harnessDir) })

	mockPath := filepath.Join(harnessDir, "mock_agent.sh")
	if err := os.WriteFile(mockPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("failed to write mock harness: %v", err)
	}

	harnessConfig := map[string]interface{}{
		"harnesses": []map[string]interface{}{
			{
				"id":          "mock_agent",
				"name":        "Mock Agent Harness",
				"type":        "cli",
				"path":        mockPath,
				"enabled":     true,
				"description": "Mock agent for automated testing",
			},
		},
	}
	cfgBytes, _ := json.Marshal(harnessConfig)
	cfgPath := filepath.Join(harnessDir, "harnesses.json")
	if err := os.WriteFile(cfgPath, cfgBytes, 0644); err != nil {
		t.Fatalf("failed to write harnesses.json: %v", err)
	}
	return cfgPath
}

func TestPipelineStage1AndStage2(t *testing.T) {
	engineBin := findEngineBin()
	if engineBin == "" {
		t.Skip("read-engine binary not found, skipping integration test")
	}

	ws := setupTestWorkspace(t)
	defer os.RemoveAll(ws)

	// Create a sample python source file for Stage 2 AST context
	pyFile := filepath.Join(ws, "math_service.py")
	pyCode := `def is_even(n: int) -> bool:
    pass
`
	if err := os.WriteFile(pyFile, []byte(pyCode), 0644); err != nil {
		t.Fatalf("failed to write python file: %v", err)
	}

	// Commit initial state
	cmd := exec.Command("git", "add", "math_service.py")
	cmd.Dir = ws
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "Initial commit")
	cmd.Dir = ws
	_ = cmd.Run()

	// Mock agent that modifies math_service.py
	mockScript := `#!/bin/bash
cat << 'EOF' > math_service.py
def is_even(n: int) -> bool:
    return n % 2 == 0
EOF
echo "[MOCK] Agent updated math_service.py successfully"
`
	cfgPath := createMockHarness(t, ws, mockScript)

	// Create test file that asserts is_even behavior
	testFile := filepath.Join(ws, "test_math.py")
	testCode := `import unittest
from math_service import is_even

class TestMath(unittest.TestCase):
    def test_even(self):
        self.assertTrue(is_even(4))
        self.assertTrue(is_even(0))
    def test_odd(self):
        self.assertFalse(is_even(5))
        self.assertFalse(is_even(-1))
    def test_boolean_return(self):
        self.assertIs(is_even(2), True)
        self.assertIs(is_even(3), False)

if __name__ == '__main__':
    unittest.main()
`
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cmd = exec.Command("git", "add", "test_math.py")
	cmd.Dir = ws
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "Add tests")
	cmd.Dir = ws
	_ = cmd.Run()

	orch, err := NewOrchestrator(PipelineConfig{
		Workspace:           ws,
		Prompt:              "Implement is_even function",
		HarnessID:           "mock_agent",
		EngineBin:           engineBin,
		CustomHarnessConfig: cfgPath,
		TestCommand:         "python3 -m unittest test_math.py",
		TimeoutSeconds:      10,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	res, err := orch.Run(context.Background())
	if err != nil {
		t.Fatalf("pipeline Run failed: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected pipeline to succeed, diagnostics: %s, error: %s", res.Diagnostics, res.Error)
	}

	if res.StageResults["stage1_contract_freeze"] != StageStatusPassed {
		t.Errorf("Stage 1 expected passed, got %s", res.StageResults["stage1_contract_freeze"])
	}
	if res.StageResults["stage2_deterministic_context"] != StageStatusPassed {
		t.Errorf("Stage 2 expected passed, got %s", res.StageResults["stage2_deterministic_context"])
	}
	if res.StageResults["stage3_harness_dispatch"] != StageStatusPassed {
		t.Errorf("Stage 3 expected passed, got %s", res.StageResults["stage3_harness_dispatch"])
	}
	if res.StageResults["stage4_mutation_verification"] != StageStatusPassed {
		t.Errorf("Stage 4 expected passed, got %s", res.StageResults["stage4_mutation_verification"])
	}
	if res.StageResults["stage5_sealing"] != StageStatusPassed {
		t.Errorf("Stage 5 expected passed, got %s", res.StageResults["stage5_sealing"])
	}

	if res.FinalState != "handoff_complete" {
		t.Errorf("expected final state handoff_complete, got %s", res.FinalState)
	}
	if res.DiffHash == "" {
		t.Errorf("expected non-empty diff hash")
	}
	if res.VerifyHash == "" {
		t.Errorf("expected non-empty verify hash")
	}

	// Verify STATE.json on disk strictly under .read/
	statePath := filepath.Join(ws, ".read", "STATE.json")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("failed to read STATE.json: %v", err)
	}
	var diskState struct {
		Status     string  `json:"status"`
		VerifyHash *string `json:"verify_hash"`
	}
	if err := json.Unmarshal(stateData, &diskState); err != nil {
		t.Fatalf("failed to parse STATE.json: %v", err)
	}
	if diskState.Status != "handoff_complete" {
		t.Errorf("STATE.json status is '%s', expected 'handoff_complete'", diskState.Status)
	}

	// Verify WAL.log contains DEFINE, BUILD_START, VERIFIED, HANDOFF strictly under .read/
	walPath := filepath.Join(ws, ".read", "WAL.log")
	walData, err := os.ReadFile(walPath)
	if err != nil {
		t.Fatalf("failed to read WAL.log: %v", err)
	}
	walStr := string(walData)
	for _, expectedEvent := range []string{"DEFINE", "BUILD_START", "VERIFIED", "HANDOFF"} {
		if !strings.Contains(walStr, expectedEvent) {
			t.Errorf("WAL.log missing event %s", expectedEvent)
		}
	}
}

func TestPipelineVerificationGateFailureAndOverride(t *testing.T) {
	engineBin := findEngineBin()
	if engineBin == "" {
		t.Skip("read-engine binary not found, skipping integration test")
	}

	ws := setupTestWorkspace(t)
	defer os.RemoveAll(ws)

	// File with logic
	targetFile := filepath.Join(ws, "logic.py")
	if err := os.WriteFile(targetFile, []byte("def check_threshold(x):\n    return x > 10\n"), 0644); err != nil {
		t.Fatalf("failed to write target: %v", err)
	}
	cmd := exec.Command("git", "add", "logic.py")
	cmd.Dir = ws
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "Initial")
	cmd.Dir = ws
	_ = cmd.Run()

	// Mock agent modifies file with lots of conditional mutations
	mockScript := `#!/bin/bash
cat << 'EOF' > logic.py
def check_threshold(x):
    if x == 100:
        return True
    if x > 10:
        return True
    return False
EOF
`
	cfgPath := createMockHarness(t, ws, mockScript)

	// Weak test suite: tests pass, but fails to assert x == 100, so mutant survives!
	testFile := filepath.Join(ws, "test_weak.py")
	testCode := `import unittest
from logic import check_threshold

class TestWeak(unittest.TestCase):
    def test_pass(self):
        self.assertTrue(check_threshold(15))
        self.assertFalse(check_threshold(5))

if __name__ == '__main__':
    unittest.main()
`
	if err := os.WriteFile(testFile, []byte(testCode), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	cmd = exec.Command("git", "add", "test_weak.py")
	cmd.Dir = ws
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-qm", "Add weak tests")
	cmd.Dir = ws
	_ = cmd.Run()

	// 1. First run WITHOUT auto-approve: verification gate MUST fail if mutants survive below 100%
	orch, err := NewOrchestrator(PipelineConfig{
		Workspace:           ws,
		Prompt:              "Add threshold check logic",
		HarnessID:           "mock_agent",
		EngineBin:           engineBin,
		CustomHarnessConfig: cfgPath,
		TestCommand:         "python3 -m unittest test_weak.py",
		AutoApprove:         false,
		TimeoutSeconds:      10,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	res, err := orch.Run(context.Background())
	// Depending on mutation count and killed count:
	// If score < 100%:
	if res.MutationScore < 100.0 {
		if res.Success {
			t.Errorf("pipeline should have failed verification gate with mutation score %.1f%%", res.MutationScore)
		}
		if res.FinalState == "handoff_complete" {
			t.Errorf("state should NOT be handoff_complete when gate fails without auto-approve")
		}
	}

	// 2. Now run WITH auto-approve: Master Override MUST be applied and handoff MUST succeed
	orchOverride, err := NewOrchestrator(PipelineConfig{
		Workspace:           ws,
		Prompt:              "Add threshold check logic with auto approve",
		HarnessID:           "mock_agent",
		EngineBin:           engineBin,
		CustomHarnessConfig: cfgPath,
		TestCommand:         "python3 -m unittest test_weak.py",
		AutoApprove:         true,
		TimeoutSeconds:      10,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	resOverride, err := orchOverride.Run(context.Background())
	if err != nil {
		t.Fatalf("auto-approved run failed: %v", err)
	}
	if !resOverride.Success {
		t.Errorf("expected auto-approved pipeline to succeed via Master Override, got: %s", resOverride.Diagnostics)
	}
	if resOverride.FinalState != "handoff_complete" {
		t.Errorf("expected final state handoff_complete, got %s", resOverride.FinalState)
	}
	if resOverride.VerifyHash == "" {
		t.Errorf("expected verify hash to be sealed")
	}
}

func TestPipelineHarnessTimeout(t *testing.T) {
	engineBin := findEngineBin()
	if engineBin == "" {
		t.Skip("read-engine binary not found, skipping integration test")
	}

	ws := setupTestWorkspace(t)
	defer os.RemoveAll(ws)

	// Mock agent that sleeps forever
	mockScript := `#!/bin/bash
sleep 60
`
	cfgPath := createMockHarness(t, ws, mockScript)

	orch, err := NewOrchestrator(PipelineConfig{
		Workspace:           ws,
		Prompt:              "Long running task",
		HarnessID:           "mock_agent",
		EngineBin:           engineBin,
		CustomHarnessConfig: cfgPath,
		TimeoutSeconds:      1, // 1 second timeout
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	startTime := time.Now()
	res, err := orch.Run(context.Background())
	duration := time.Since(startTime)

	if duration > 4*time.Second {
		t.Errorf("task should have timed out within ~1-2 seconds, took %v", duration)
	}
	if res.Success {
		t.Errorf("expected pipeline to fail due to harness timeout")
	}
	if res.StageResults["stage3_harness_dispatch"] != StageStatusFailed {
		t.Errorf("expected stage 3 to fail, got %s", res.StageResults["stage3_harness_dispatch"])
	}
}

func TestLegacyMethodDirectoryMigration(t *testing.T) {
	engineBin := findEngineBin()
	if engineBin == "" {
		t.Skip("read-engine binary not found, skipping integration test")
	}

	ws := setupTestWorkspace(t)
	defer os.RemoveAll(ws)

	legacyDir := filepath.Join(ws, ".method")
	if err := os.MkdirAll(legacyDir, 0755); err != nil {
		t.Fatalf("failed to create .method directory: %v", err)
	}
	legacyState := filepath.Join(legacyDir, "STATE.json")
	validJSON := `{"state_revision":0,"status":"idle","active_task_id":null,"contract_hash":null,"base_commit":null,"staged_diff_hash":null,"verify_hash":null,"cas_lock":{"expected_revision":0,"locked_by":"system","lease_expires":"2099-01-01T00:00:00Z"}}`
	if err := os.WriteFile(legacyState, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to write legacy state: %v", err)
	}
	legacyWal := filepath.Join(legacyDir, "WAL.log")
	if err := os.WriteFile(legacyWal, []byte("INIT\n"), 0644); err != nil {
		t.Fatalf("failed to write legacy wal: %v", err)
	}

	cmd := exec.Command(engineBin, "status")
	cmd.Dir = ws
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("read-engine status failed: %s (%v)", string(out), err)
	}

	if _, err := os.Stat(legacyDir); !os.IsNotExist(err) {
		t.Errorf("expected legacy .method to be migrated and removed")
	}
	newPath := filepath.Join(ws, ".read", "STATE.json")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("expected .read/STATE.json to exist after migration: %v", err)
	}
}

