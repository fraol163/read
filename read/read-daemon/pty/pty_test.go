package pty

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

func TestPtySessionManager(t *testing.T) {
	mgr := &SessionManager{sessions: make(map[string]*Session)}

	cmd := exec.Command("sleep", "10")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatalf("failed to start pty: %v", err)
	}

	pid := cmd.Process.Pid
	pgid, _ := syscall.Getpgid(pid)

	session := &Session{
		ID:        "test_s1",
		PID:       pid,
		PGID:      pgid,
		Command:   "sleep 10",
		Active:    true,
		CreatedAt: time.Now(),
		Cmd:       cmd,
		Ptmx:      ptmx,
	}

	mgr.AddSession(session)
	s, found := mgr.GetSession("test_s1")
	if !found || s.ID != "test_s1" {
		t.Fatalf("expected session test_s1 to be found")
	}

	if err := mgr.CloseSession("test_s1"); err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	_, found = mgr.GetSession("test_s1")
	if found {
		t.Fatalf("expected session to be removed from manager after close")
	}

	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("process %d is still alive after CloseSession", pid)
	}
}

func TestPtyWebSocketLifecycleAndCleanup(t *testing.T) {
	pidFile, err := os.CreateTemp("", "pty_child_pids_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	pidFilePath := pidFile.Name()
	pidFile.Close()
	defer os.Remove(pidFilePath)

	server := httptest.NewServer(http.HandlerFunc(HandleTerminalWs))
	defer server.Close()

	u, _ := url.Parse(server.URL)
	u.Scheme = "ws"
	u.Path = "/ws/pty"
	q := u.Query()
	q.Set("cmd", "/bin/sh")
	q.Set("session_id", "pty_test_session")
	u.RawQuery = q.Encode()

	wsConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}

	// Drain output in background so websocket buffer doesn't block
	go func() {
		for {
			_, _, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Send command to sh via websocket input message
	inputMsg := TerminalMessage{
		Type: "input",
		Data: fmt.Sprintf("sleep 100 & echo $! > %s\n", pidFilePath),
	}
	err = wsConn.WriteJSON(inputMsg)
	if err != nil {
		t.Fatalf("failed to send command to pty: %v", err)
	}

	// Wait for child PID to be written
	var childPid int
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		content, err := os.ReadFile(pidFilePath)
		if err == nil {
			trimmed := strings.TrimSpace(string(content))
			if p, err := strconv.Atoi(trimmed); err == nil && p > 0 {
				childPid = p
				break
			}
		}
	}

	if childPid <= 0 {
		wsConn.Close()
		t.Fatalf("failed to read child PID from pty script")
	}

	// Verify child process is running
	if err := syscall.Kill(childPid, 0); err != nil {
		t.Fatalf("child process %d is not running: %v", childPid, err)
	}

	// Send resize message to test resize handling
	resizeMsg := TerminalMessage{
		Type: "resize",
		Cols: 120,
		Rows: 40,
	}
	_ = wsConn.WriteJSON(resizeMsg)

	// Send ping message
	_ = wsConn.WriteJSON(TerminalMessage{Type: "ping"})

	// Send signal SIGTERM to verify signal forwarding
	signalMsg := TerminalMessage{
		Type: "signal",
		Data: "SIGTERM",
	}
	_ = wsConn.WriteJSON(signalMsg)

	// Close websocket (simulate client disconnect / detached session)
	_ = wsConn.Close()

	// Verify child process was cleaned up (no orphan!)
	dead := false
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := syscall.Kill(childPid, 0); err != nil {
			dead = true
			break
		}
	}
	if !dead {
		t.Fatalf("orphan child process %d is still alive after websocket disconnect", childPid)
	}

	// Verify session removed from DefaultManager
	cleanedUp := false
	for i := 0; i < 20; i++ {
		if _, found := DefaultManager.GetSession("pty_test_session"); !found {
			cleanedUp = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !cleanedUp {
		t.Fatalf("session pty_test_session was not cleaned up in DefaultManager")
	}
}

func BenchmarkPtyStreaming(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(HandleTerminalWs))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?cmd=/bin/cat&session_id=bench_session"
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatalf("failed to connect to pty ws: %v", err)
	}
	defer wsConn.Close()

	payload := []byte("benchmarking pty data streaming throughput\n")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := wsConn.WriteMessage(websocket.BinaryMessage, payload)
		if err != nil {
			b.Fatalf("failed to write message: %v", err)
		}
		_, _, err = wsConn.ReadMessage()
		if err != nil {
			b.Fatalf("failed to read message: %v", err)
		}
	}
}
