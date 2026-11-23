package harness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"read-daemon/scanner"
	"read-daemon/security"
)

type ExecutionStatus string

const (
	StatusIdle      ExecutionStatus = "idle"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusAborted   ExecutionStatus = "aborted"
)

type HarnessRunRequest struct {
	HarnessID      string   `json:"harness_id"`
	Prompt         string   `json:"prompt"`
	Args           []string `json:"args,omitempty"`
	Workspace      string   `json:"workspace,omitempty"`
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"` // 0 = default (15m)
	MaxMemoryBytes uint64   `json:"max_memory_bytes,omitempty"` // 0 = default (1.2GB)
}

type HarnessEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "stdout", "stderr", "status", "artifact"
	Content   string    `json:"content"`
}

type ActiveTask struct {
	ID        string          `json:"id"`
	HarnessID string          `json:"harness_id"`
	Status    ExecutionStatus `json:"status"`
	Prompt    string          `json:"prompt"`
	StartTime time.Time       `json:"start_time"`
	EndTime   *time.Time      `json:"end_time,omitempty"`
	Events    []HarnessEvent  `json:"events"`
	Timeout   time.Duration   `json:"timeout,omitempty"`
	MaxMemory uint64          `json:"max_memory,omitempty"`
	cmd       *exec.Cmd
	pgid      int
	mutex     sync.RWMutex
}

type Runner struct {
	tasks map[string]*ActiveTask
	mutex sync.RWMutex
}

var DefaultRunner = &Runner{
	tasks: make(map[string]*ActiveTask),
}

// StartTask launches a harness with given prompt and args
func (r *Runner) StartTask(req HarnessRunRequest, registry *scanner.HarnessRegistry) (*ActiveTask, error) {
	harness, found := registry.GetHarness(req.HarnessID)
	if !found {
		return nil, fmt.Errorf("harness '%s' not registered", req.HarnessID)
	}

	if !harness.Enabled {
		return nil, fmt.Errorf("harness '%s' is currently disabled", req.HarnessID)
	}

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}

	maxMem := req.MaxMemoryBytes
	if maxMem == 0 {
		// Dynamically calibrated memory limit (30% available RAM ceiling)
		maxMem = security.GetDynamicMemoryLimit()
	}

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &ActiveTask{
		ID:        taskID,
		HarnessID: req.HarnessID,
		Status:    StatusRunning,
		Prompt:    security.SanitizeText(req.Prompt),
		StartTime: time.Now(),
		Events:    make([]HarnessEvent, 0),
		Timeout:   timeout,
		MaxMemory: maxMem,
	}

	r.mutex.Lock()
	r.tasks[taskID] = task
	r.mutex.Unlock()

	// Build command based on harness ID
	var cmd *exec.Cmd
	switch harness.ID {
	case "deepseek_harness": // dsh
		args := append([]string{}, req.Args...)
		if req.Prompt != "" {
			args = append(args, req.Prompt)
		}
		cmd = exec.Command(harness.Path, args...)
	case "claude_code": // claude
		args := []string{"-p", req.Prompt}
		args = append(args, req.Args...)
		cmd = exec.Command(harness.Path, args...)
	case "aider": // aider
		args := []string{"--message", req.Prompt}
		args = append(args, req.Args...)
		cmd = exec.Command(harness.Path, args...)
	default: // custom or generic binary
		args := append([]string{}, req.Args...)
		if req.Prompt != "" {
			args = append(args, req.Prompt)
		}
		cmd = exec.Command(harness.Path, args...)
	}

	if req.Workspace != "" {
		cmd.Dir = req.Workspace
	}

	// Process group isolation: place child process and all its subprocesses in its own PGID
	security.SetProcessGroup(cmd)

	task.cmd = cmd

	// Launch async execution
	go r.monitorExecution(task, cmd)

	return task, nil
}

func (r *Runner) monitorExecution(task *ActiveTask, cmd *exec.Cmd) {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		task.appendEvent("stderr", fmt.Sprintf("Failed to acquire stdout pipe: %v", err))
		task.setStatus(StatusFailed)
		return
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		task.appendEvent("stderr", fmt.Sprintf("Failed to acquire stderr pipe: %v", err))
		task.setStatus(StatusFailed)
		return
	}

	if err := cmd.Start(); err != nil {
		task.appendEvent("stderr", fmt.Sprintf("Failed to start process: %v", err))
		task.setStatus(StatusFailed)
		return
	}

	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil || pgid <= 1 {
		pgid = pid
	}
	task.mutex.Lock()
	task.pgid = pgid
	task.mutex.Unlock()

	// Apply dynamic RLIMIT_AS to child process immediately upon launch
	_ = security.SetProcessRlimitAS(pid, task.MaxMemory)

	// Supervisor for process timeout and memory limits
	stopSupervisor := make(chan struct{})
	var stopOnce sync.Once
	stopSup := func() {
		stopOnce.Do(func() {
			close(stopSupervisor)
		})
	}
	defer stopSup()

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		startTime := task.StartTime

		for {
			select {
			case <-stopSupervisor:
				return
			case now := <-ticker.C:
				// Process timeout check
				if task.Timeout > 0 && now.Sub(startTime) > task.Timeout {
					task.appendEvent("stderr", fmt.Sprintf("Task exceeded timeout limit of %v; terminating process group %d", task.Timeout, task.pgid))
					_ = security.KillProcessGroup(task.pgid)
					task.setStatus(StatusFailed)
					return
				}

				// Memory guard check
				if task.MaxMemory > 0 && task.pgid > 1 {
					rss, err := security.GetProcessGroupMemory(task.pgid)
					if err == nil && rss > task.MaxMemory {
						task.appendEvent("stderr", fmt.Sprintf("Task exceeded memory limit (%d MB > %d MB); terminating process group %d", rss/(1024*1024), task.MaxMemory/(1024*1024), task.pgid))
						_ = security.KillProcessGroup(task.pgid)
						task.setStatus(StatusFailed)
						return
					}
				}
			}
		}
	}()

	// Stream stdout with token sanitization
	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			task.appendEvent("stdout", scanner.Text())
		}
	}()

	// Stream stderr with token sanitization
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			task.appendEvent("stderr", scanner.Text())
		}
	}()

	err = cmd.Wait()
	stopSup()

	// Clean up any remaining child processes in process group
	if cmd.Process != nil {
		_ = security.KillProcessTree(cmd.Process.Pid, task.pgid)
	}

	now := time.Now()
	task.mutex.Lock()
	task.EndTime = &now
	if task.Status == StatusRunning {
		if err != nil {
			task.Status = StatusFailed
			task.Events = append(task.Events, HarnessEvent{
				Timestamp: now,
				Type:      "status",
				Content:   security.SanitizeText(fmt.Sprintf("Process exited with error: %v", err)),
			})
		} else {
			task.Status = StatusCompleted
			task.Events = append(task.Events, HarnessEvent{
				Timestamp: now,
				Type:      "status",
				Content:   "Process finished successfully with exit code 0",
			})
		}
	}
	task.mutex.Unlock()
}

func (t *ActiveTask) appendEvent(eventType, content string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Events = append(t.Events, HarnessEvent{
		Timestamp: time.Now(),
		Type:      eventType,
		Content:   security.SanitizeText(content),
	})
}

func (t *ActiveTask) setStatus(status ExecutionStatus) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Status = status
	now := time.Now()
	t.EndTime = &now
}

func (r *Runner) StopTask(taskID string) error {
	r.mutex.RLock()
	task, exists := r.tasks[taskID]
	r.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' not found", taskID)
	}

	task.mutex.Lock()
	defer task.mutex.Unlock()

	if task.Status != StatusRunning {
		return fmt.Errorf("task '%s' is not running", taskID)
	}

	var killErr error
	if task.cmd != nil && task.cmd.Process != nil {
		killErr = security.KillProcessTree(task.cmd.Process.Pid, task.pgid)
	} else if task.pgid > 1 {
		killErr = security.KillProcessGroup(task.pgid)
	}

	task.Status = StatusAborted
	now := time.Now()
	task.EndTime = &now
	task.Events = append(task.Events, HarnessEvent{
		Timestamp: now,
		Type:      "status",
		Content:   "Task aborted by user request",
	})

	return killErr
}

func (r *Runner) GetTask(taskID string) (*ActiveTask, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	t, found := r.tasks[taskID]
	return t, found
}

// MarshalJSON safely serializes ActiveTask
func (t *ActiveTask) MarshalJSON() ([]byte, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	type Alias ActiveTask
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	})
}
