package gateway

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"read-daemon/fs"
	"read-daemon/harness"
	"read-daemon/orchestrator"
	"read-daemon/pty"
	"read-daemon/scanner"
	"read-daemon/security"
	"read-daemon/sync"
)

// ServerConfig holds daemon gateway settings.
type ServerConfig struct {
	Host          string
	Port          int
	WorkspaceRoot string
	CustomHarness string
	ReadEngineBin string
	TLSCert       string
	TLSKey        string
	AuthToken     string
}

// GatewayServer handles HTTP, RPC, and WebSocket commands from remote controllers.
type GatewayServer struct {
	config      ServerConfig
	registry    *scanner.HarnessRegistry
	syncEngine  *sync.SyncEngine
	fsManager   *fs.FsManager
	runner      *harness.Runner
	rateLimiter *IPRateLimiter
}

func NewGatewayServer(cfg ServerConfig) (*GatewayServer, error) {
	hasTLS := cfg.TLSCert != "" && cfg.TLSKey != ""
	isLoopback := cfg.Host == "" || cfg.Host == "127.0.0.1" || cfg.Host == "localhost" || cfg.Host == "::1" || strings.HasPrefix(cfg.Host, "127.")
	if !hasTLS && !isLoopback {
		return nil, fmt.Errorf("[SECURITY VIOLATION] Refusing non-loopback bind to %s without TLS", cfg.Host)
	}

	reg, err := scanner.ScanHost(cfg.CustomHarness)
	if err != nil {
		return nil, err
	}

	syncEng := sync.NewSyncEngine(cfg.WorkspaceRoot, sync.PolicyBothSync)
	fsMgr := fs.NewFsManager(cfg.WorkspaceRoot, syncEng)

	return &GatewayServer{
		config:      cfg,
		registry:    reg,
		syncEngine:  syncEng,
		fsManager:   fsMgr,
		runner:      harness.DefaultRunner,
		rateLimiter: NewIPRateLimiter(),
	}, nil
}

// AuthMiddleware enforces constant-time Bearer token authentication.
// Supports Authorization header, Sec-WebSocket-Protocol (token.<token>), and query parameter (?token=).
// /api/v1/status is explicitly unauthenticated as a public health check.
func (s *GatewayServer) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/status" || s.config.AuthToken == "" {
			next.ServeHTTP(w, r)
			return
		}

		providedToken := ""
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			providedToken = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		} else if qToken := r.URL.Query().Get("token"); qToken != "" {
			providedToken = strings.TrimSpace(qToken)
		} else if secProto := r.Header.Get("Sec-WebSocket-Protocol"); strings.HasPrefix(secProto, "token.") {
			providedToken = strings.TrimSpace(strings.TrimPrefix(secProto, "token."))
		}

		if providedToken == "" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="READ Daemon"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized: Missing Bearer token, ?token= param, or Sec-WebSocket-Protocol",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(providedToken), []byte(s.config.AuthToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="READ Daemon"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized: Invalid token",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Handler returns the fully wrapped HTTP handler including rate limiting and auth.
func (s *GatewayServer) Handler() http.Handler {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)
	return s.rateLimiter.Middleware(s.AuthMiddleware(mux))
}

func (s *GatewayServer) RegisterRoutes(mux *http.ServeMux) {
	// PTY WebSocket for full interactive terminal remote control
	mux.HandleFunc("/ws/pty", pty.HandleTerminalWs)

	// Status & Discovery
	mux.HandleFunc("/api/v1/status", s.handleStatus)
	mux.HandleFunc("/api/v1/harnesses", s.handleHarnesses)
	mux.HandleFunc("/api/v1/harnesses/toggle", s.handleToggleHarness)

	// Autonomous Pipeline Execution
	mux.HandleFunc("/api/v1/pipeline/run", s.handlePipelineRun)

	// Harness Execution & Monitoring
	mux.HandleFunc("/api/v1/harnesses/exec", s.handleHarnessExec)
	mux.HandleFunc("/api/v1/harnesses/stop", s.handleHarnessStop)
	mux.HandleFunc("/api/v1/harnesses/task", s.handleHarnessTask)

	// Framework State & CAS Engine
	mux.HandleFunc("/api/v1/state", s.handleState)
	mux.HandleFunc("/api/v1/state/override", s.handleOverride)
	mux.HandleFunc("/api/v1/task/command", s.handleTaskCommand)

	// 3-Way File System & Storage Sync
	mux.HandleFunc("/api/v1/fs/list", s.handleFsList)
	mux.HandleFunc("/api/v1/fs/read", s.handleFsRead)
	mux.HandleFunc("/api/v1/fs/write", s.handleFsWrite)
	mux.HandleFunc("/api/v1/fs/delete", s.handleFsDelete)
	mux.HandleFunc("/api/v1/sync/manifest", s.handleSyncManifest)
	mux.HandleFunc("/api/v1/sync/policy", s.handleSyncPolicy)
}

func (s *GatewayServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"system":         "READ Framework Gateway Daemon",
		"version":        "0.1.0",
		"workspace_root": s.config.WorkspaceRoot,
		"harness_count":  len(s.registry.Harnesses),
		"storage_policy": s.syncEngine.DefaultPolicy,
		"status":         "online",
	})
}

func (s *GatewayServer) handleHarnesses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.registry)
}

func (s *GatewayServer) handleToggleHarness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	for i := range s.registry.Harnesses {
		if s.registry.Harnesses[i].ID == req.ID {
			s.registry.Harnesses[i].Enabled = req.Enabled
			found = true
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": found,
		"id":      req.ID,
		"enabled": req.Enabled,
	})
}

func (s *GatewayServer) handleHarnessExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req harness.HarnessRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Workspace == "" {
		req.Workspace = s.config.WorkspaceRoot
	} else {
		validWs, err := security.ResolveWorkspacePath(s.config.WorkspaceRoot, req.Workspace)
		if err != nil {
			http.Error(w, "Forbidden: workspace escapes allowed boundary", http.StatusForbidden)
			return
		}
		req.Workspace = validWs
	}

	task, err := s.runner.StartTask(req, s.registry)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *GatewayServer) handleHarnessStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := s.runner.StopTask(req.TaskID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"task_id": req.TaskID,
		"error":   func() string { if err != nil { return err.Error() }; return "" }(),
	})
}

func (s *GatewayServer) handleHarnessTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")
	task, found := s.runner.GetTask(taskID)
	if !found {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (s *GatewayServer) handleState(w http.ResponseWriter, r *http.Request) {
	statePath := filepath.Join(s.config.WorkspaceRoot, ".read", "STATE.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		legacyPath := filepath.Join(s.config.WorkspaceRoot, ".method", "STATE.json")
		if _, err2 := os.Stat(legacyPath); err2 == nil {
			statePath = legacyPath
		}
	}
	data, err := os.ReadFile(statePath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "uninitialized"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *GatewayServer) handleOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Rationale string `json:"rationale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Rationale == "" {
		http.Error(w, "rationale required", http.StatusBadRequest)
		return
	}

	cmd := exec.Command(s.config.ReadEngineBin, "override", "remote_operator", req.Rationale)
	cmd.Dir = s.config.WorkspaceRoot
	security.SetProcessGroup(cmd)

	output, err := cmd.CombinedOutput()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": err == nil,
		"output":  security.SanitizeText(string(output)),
	})
}

func (s *GatewayServer) handleFsList(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	items, err := s.fsManager.ListDirectory(relPath)
	if err != nil {
		if errors.Is(err, security.ErrPathForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (s *GatewayServer) handleFsRead(w http.ResponseWriter, r *http.Request) {
	relPath := r.URL.Query().Get("path")
	data, err := s.fsManager.ReadFile(relPath)
	if err != nil {
		if errors.Is(err, security.ErrPathForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

func (s *GatewayServer) handleFsWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relPath := r.URL.Query().Get("path")
	policy := r.URL.Query().Get("policy")
	if relPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.fsManager.WriteFile(relPath, data, policy); err != nil {
		if errors.Is(err, security.ErrPathForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    relPath,
		"bytes":   len(data),
	})
}

func (s *GatewayServer) handleFsDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	relPath := r.URL.Query().Get("path")
	if relPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	if err := s.fsManager.DeleteFile(relPath); err != nil {
		if errors.Is(err, security.ErrPathForbidden) {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"path":    relPath,
	})
}

func (s *GatewayServer) handleSyncManifest(w http.ResponseWriter, r *http.Request) {
	manifest, err := sync.ScanProjectManifest(s.config.WorkspaceRoot, s.syncEngine.DefaultPolicy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(manifest)
}

func (s *GatewayServer) handleSyncPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Policy string `json:"policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.syncEngine.DefaultPolicy = sync.StoragePolicy(req.Policy)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"policy":  s.syncEngine.DefaultPolicy,
	})
}

func (s *GatewayServer) handleTaskCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Invoke native read-engine executable
	cmdArgs := append([]string{req.Command}, req.Args...)
	cmd := exec.Command(s.config.ReadEngineBin, cmdArgs...)
	cmd.Dir = s.config.WorkspaceRoot
	security.SetProcessGroup(cmd)

	output, err := cmd.CombinedOutput()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   err == nil,
		"exit_code": exitCode,
		"output":    security.SanitizeText(string(output)),
	})
}

func (s *GatewayServer) handlePipelineRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Prompt         string `json:"prompt"`
		Harness        string `json:"harness"`
		AutoApprove    bool   `json:"auto_approve"`
		TimeoutSeconds int    `json:"timeout_seconds"`
		Timeout        int    `json:"timeout"`
		Workspace      string `json:"workspace"`
		TestCommand    string `json:"test_command"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Prompt) == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}

	timeoutSec := req.TimeoutSeconds
	if timeoutSec <= 0 && req.Timeout > 0 {
		timeoutSec = req.Timeout
	}

	ws := s.config.WorkspaceRoot
	if req.Workspace != "" {
		validWs, err := security.ResolveWorkspacePath(s.config.WorkspaceRoot, req.Workspace)
		if err != nil {
			http.Error(w, "Forbidden: workspace escapes allowed boundary", http.StatusForbidden)
			return
		}
		ws = validWs
	}

	var logBuf bytes.Buffer
	cfg := orchestrator.PipelineConfig{
		Workspace:           ws,
		Prompt:              req.Prompt,
		HarnessID:           req.Harness,
		AutoApprove:         req.AutoApprove,
		TimeoutSeconds:      timeoutSec,
		EngineBin:           s.config.ReadEngineBin,
		TestCommand:         req.TestCommand,
		CustomHarnessConfig: s.config.CustomHarness,
		Stdout:              &logBuf,
		Stderr:              &logBuf,
	}

	orch, err := orchestrator.NewOrchestrator(cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := orch.Run(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if err != nil && result == nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(result)
}
