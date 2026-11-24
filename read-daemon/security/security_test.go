package security

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPathTraversalRejection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sec_test_path_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create real files and symlinks inside workspace
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	testFile := filepath.Join(tmpDir, "valid.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Symlink pointing to /etc (outside workspace)
	symlinkEtc := filepath.Join(tmpDir, "symlink_etc")
	_ = os.Symlink("/etc", symlinkEtc)

	// Symlink pointing to /etc/passwd (outside workspace)
	symlinkPasswd := filepath.Join(tmpDir, "symlink_passwd")
	_ = os.Symlink("/etc/passwd", symlinkPasswd)

	// Symlink pointing to non-existent file outside workspace
	symlinkBrokenOutside := filepath.Join(tmpDir, "broken_outside")
	_ = os.Symlink("/etc/nonexistent_file_9999", symlinkBrokenOutside)

	// Valid symlink pointing inside workspace
	symlinkValid := filepath.Join(tmpDir, "symlink_valid")
	_ = os.Symlink(testFile, symlinkValid)

	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{"valid file", "valid.txt", false},
		{"valid empty root", "", false},
		{"valid slash root", "/", false},
		{"valid dot root", ".", false},
		{"valid subdir file", "subdir/file.txt", false},
		{"valid symlink inside", "symlink_valid", false},

		// Traversal payloads
		{"simple traversal", "../etc/passwd", true},
		{"slash traversal", "/../etc/passwd", true},
		{"deep traversal", "../../../../etc/shadow", true},
		{"dot slash traversal", "./../../etc/shadow", true},
		{"middle traversal", "subdir/../../../../etc/passwd", true},

		// URL encoded payloads
		{"url encoded dot dot slash", "%2e%2e%2fetc/passwd", true},
		{"url encoded both parts", "%2e%2e/%2e%2e/etc/passwd", true},
		{"url encoded full traversal", "%2e%2e%2f%2e%2e%2fetc%2fshadow", true},
		{"double url encoded", "%252e%252e%252fetc/passwd", true},

		// Symlink escapes
		{"symlink to etc dir file", "symlink_etc/passwd", true},
		{"symlink to etc dir nested", "symlink_etc/pam.d/other", true},
		{"symlink directly to passwd", "symlink_passwd", true},
		{"broken symlink outside", "broken_outside", true},
		{"symlink new file outside", "symlink_etc/new_file.txt", true},

		// Null bytes and separator injection
		{"null byte injection", "valid.txt\x00/../../etc/passwd", true},
		{"null byte simple", "valid.txt\x00extra", true},
		{"windows backslash traversal", "..\\..\\etc\\passwd", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := ResolveWorkspacePath(tmpDir, tc.path)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for path %q, got resolved: %q", tc.path, resolved)
				}
				if !errors.Is(err, ErrPathForbidden) {
					t.Fatalf("expected ErrPathForbidden for path %q, got: %v", tc.path, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error for path %q: %v", tc.path, err)
				}
				if resolved == "" {
					t.Fatalf("expected non-empty resolved path for %q", tc.path)
				}
			}
		})
	}
}

func TestKeyRedaction(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCheck func(string) bool
	}{
		{
			name:  "OpenAI standard key",
			input: "export OPENAI_API_KEY=sk-proj-1234567890abcdef1234567890",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "OPENAI_API_KEY=***REDACTED***") && !strings.Contains(out, "sk-proj-")
			},
		},
		{
			name:  "Anthropic API key",
			input: "ANTHROPIC_API_KEY=sk-ant-api03-abcdef1234567890abcdef123456",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "ANTHROPIC_API_KEY=***REDACTED***") && !strings.Contains(out, "sk-ant-api03-")
			},
		},
		{
			name:  "DeepSeek API key",
			input: "DEEPSEEK_API_KEY=sk-" + "0123456789abcdef0123456789abcdef",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "DEEPSEEK_API_KEY=***REDACTED***") && !strings.Contains(out, "0123456789abcdef")
			},
		},
		{
			name:  "CLI flag --api-key",
			input: "read-daemon --api-key sk-test12345678901234567890",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "--api-key ***REDACTED***")
			},
		},
		{
			name:  "CLI flag --api-key with equals",
			input: "run --api-key=secret1234567890123456",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "--api-key=***REDACTED***")
			},
		},
		{
			name:  "HTTP Authorization Bearer token",
			input: "Authorization: Bearer ya29.a0AfH6SMC1234567890abcdef",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "Bearer ***REDACTED***") && !strings.Contains(out, "ya29")
			},
		},
		{
			name:  "GitHub personal access token",
			input: "GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuv",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "GITHUB_TOKEN=***REDACTED***") && !strings.Contains(out, "ghp_")
			},
		},
		{
			name:  "HuggingFace token",
			input: "HF_TOKEN=hf_abcdef1234567890abcdef1234567890",
			expectedCheck: func(out string) bool {
				return strings.Contains(out, "HF_TOKEN=***REDACTED***") && !strings.Contains(out, "hf_")
			},
		},
		{
			name:  "JSON secret key",
			input: `{"api_key": "my-secret-key-12345"}`,
			expectedCheck: func(out string) bool {
				return strings.Contains(out, `"api_key": "***REDACTED***"`)
			},
		},
		{
			name:  "Harmless output unchanged",
			input: "Normal log output: 25 tests passed, 0 failures, exit code 0",
			expectedCheck: func(out string) bool {
				return out == "Normal log output: 25 tests passed, 0 failures, exit code 0"
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sanitized := SanitizeText(tc.input)
			if !tc.expectedCheck(sanitized) {
				t.Fatalf("sanitization failed for %q, got: %q", tc.input, sanitized)
			}
		})
	}
}

func TestSanitizeArgsAndEnv(t *testing.T) {
	args := []string{"--model", "gpt-4", "--api-key", "sk-secret12345678901234567890", "-v"}
	sanitizedArgs := SanitizeArgs(args)
	if sanitizedArgs[3] != RedactedPlaceholder {
		t.Fatalf("expected arg[3] to be redacted, got: %q", sanitizedArgs[3])
	}

	env := []string{
		"PATH=/usr/bin:/bin",
		"OPENAI_API_KEY=sk-test12345678901234567890",
		"DEBUG=true",
		"DATABASE_PASSWORD=supersecretpassword",
	}
	sanitizedEnv := SanitizeEnv(env)
	if !strings.Contains(sanitizedEnv[1], "OPENAI_API_KEY=***REDACTED***") {
		t.Fatalf("expected env[1] to be redacted, got: %q", sanitizedEnv[1])
	}
	if !strings.Contains(sanitizedEnv[3], "DATABASE_PASSWORD=***REDACTED***") {
		t.Fatalf("expected env[3] to be redacted, got: %q", sanitizedEnv[3])
	}
	if sanitizedEnv[0] != "PATH=/usr/bin:/bin" {
		t.Fatalf("expected PATH to be preserved, got: %q", sanitizedEnv[0])
	}
}

func TestProcessGroupCleanup(t *testing.T) {
	// Spawns a shell script with multiple child background processes
	// and verifies that KillProcessGroup terminates all children.
	pidFile, err := os.CreateTemp("", "sec_test_pids_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	pidFilePath := pidFile.Name()
	pidFile.Close()
	defer os.Remove(pidFilePath)

	// Bash command that launches two background sleep processes and writes their PIDs
	script := fmt.Sprintf(`
sleep 100 &
PID1=$!
sleep 100 &
PID2=$!
echo "$PID1 $PID2" > %s
wait
`, pidFilePath)

	cmd := exec.Command("bash", "-c", script)
	SetProcessGroup(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		t.Fatalf("failed to get pgid: %v", err)
	}
	if pgid != pid {
		t.Fatalf("expected pgid (%d) to equal pid (%d)", pgid, pid)
	}

	// Wait for child PIDs to be written to file
	var childPids []int
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		content, err := os.ReadFile(pidFilePath)
		if err == nil && len(strings.TrimSpace(string(content))) > 0 {
			fields := strings.Fields(string(content))
			if len(fields) == 2 {
				p1, _ := strconv.Atoi(fields[0])
				p2, _ := strconv.Atoi(fields[1])
				if p1 > 0 && p2 > 0 {
					childPids = []int{p1, p2}
					break
				}
			}
		}
	}

	if len(childPids) != 2 {
		_ = cmd.Process.Kill()
		t.Fatalf("failed to get child PIDs within timeout")
	}

	// Verify child processes are currently running
	for _, cpid := range childPids {
		if err := syscall.Kill(cpid, 0); err != nil {
			t.Fatalf("child process %d is not running: %v", cpid, err)
		}
	}

	// Kill entire process group
	if err := KillProcessGroup(pgid); err != nil {
		t.Fatalf("KillProcessGroup returned error: %v", err)
	}
	_ = cmd.Wait()

	// Wait brief moment to confirm OS has reaped/terminated them
	time.Sleep(100 * time.Millisecond)

	// Verify all child processes are dead (no orphans!)
	for _, cpid := range childPids {
		err := syscall.Kill(cpid, 0)
		if err == nil {
			t.Fatalf("orphan process %d is still alive after KillProcessGroup!", cpid)
		}
	}
}

func TestProcessGroupMemoryCheck(t *testing.T) {
	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		t.Fatalf("failed to get pgid: %v", err)
	}

	mem, err := GetProcessGroupMemory(pgid)
	if err != nil {
		t.Fatalf("GetProcessGroupMemory failed: %v", err)
	}
	if mem == 0 {
		t.Fatalf("expected non-zero RSS memory, got 0")
	}
}

func TestSignalParsing(t *testing.T) {
	if sig := ParseSignal("SIGINT"); sig != syscall.SIGINT {
		t.Fatalf("expected SIGINT, got %v", sig)
	}
	if sig := ParseSignal("term"); sig != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", sig)
	}
	if sig := ParseSignal("SIGHUP"); sig != syscall.SIGHUP {
		t.Fatalf("expected SIGHUP, got %v", sig)
	}
	if sig := ParseSignal("KILL"); sig != syscall.SIGKILL {
		t.Fatalf("expected SIGKILL, got %v", sig)
	}
	if sig := ParseSignal("invalid_sig"); sig != 0 {
		t.Fatalf("expected 0 for invalid sig, got %v", sig)
	}
}

func TestDynamicRLIMITCalibration(t *testing.T) {
	// 1. Live system calibration
	limit := CalibrateMemoryLimit()
	if limit == 0 {
		t.Fatalf("expected non-zero memory limit from CalibrateMemoryLimit")
	}

	// 2. Synthetic meminfo with known MemAvailable = 10,000,000 kB (~10 GB)
	tmpFile, err := os.CreateTemp("", "meminfo_mock_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	mockData := "MemTotal:       16384000 kB\nMemFree:         4000000 kB\nMemAvailable:   10000000 kB\n"
	if _, err := tmpFile.WriteString(mockData); err != nil {
		t.Fatalf("failed to write mock meminfo: %v", err)
	}
	tmpFile.Close()

	calibrated := CalibrateMemoryLimitFrom(tmpFile.Name())
	// 10,000,000 kB * 1024 = 10,240,000,000 bytes. 30% = 3,072,000,000 bytes.
	expected := uint64(10000000) * 1024 * 30 / 100
	if calibrated != expected {
		t.Fatalf("expected %d bytes, got %d", expected, calibrated)
	}

	// 3. Fallback when file doesn't exist -> strictly 512MB
	fallback := CalibrateMemoryLimitFrom("/nonexistent_path/fake_meminfo_9999")
	if fallback != DefaultFallbackMemoryLimit {
		t.Fatalf("expected fallback of 512MB (%d bytes), got %d", DefaultFallbackMemoryLimit, fallback)
	}

	// 4. Fallback when meminfo is corrupted / empty
	emptyTmp, _ := os.CreateTemp("", "meminfo_empty_*")
	defer os.Remove(emptyTmp.Name())
	emptyTmp.WriteString("Garbage data with no memory info\n")
	emptyTmp.Close()

	emptyFallback := CalibrateMemoryLimitFrom(emptyTmp.Name())
	if emptyFallback != DefaultFallbackMemoryLimit {
		t.Fatalf("expected fallback 512MB for corrupt meminfo, got %d", emptyFallback)
	}
}
