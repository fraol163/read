package pty

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"read-daemon/security"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow remote clients
	},
}

// TerminalMessage defines WebSocket control and data frames
type TerminalMessage struct {
	Type string `json:"type"` // "input", "resize", "signal", "ping"
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	PID       int       `json:"pid"`
	PGID      int       `json:"pgid"`
	Command   string    `json:"command"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	Cmd       *exec.Cmd `json:"-"`
	Ptmx      *os.File  `json:"-"`
	mutex     sync.Mutex
}

func (s *Session) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.Active {
		return nil
	}
	s.Active = false

	_ = security.KillProcessTree(s.PID, s.PGID)

	if s.Ptmx != nil {
		_ = s.Ptmx.Close()
	}

	if s.Cmd != nil {
		_ = s.Cmd.Wait()
	}

	return nil
}

type SessionManager struct {
	sessions map[string]*Session
	mutex    sync.RWMutex
}

var DefaultManager = &SessionManager{
	sessions: make(map[string]*Session),
}

func (m *SessionManager) AddSession(s *Session) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sessions[s.ID] = s
}

func (m *SessionManager) GetSession(id string) (*Session, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	s, found := m.sessions[id]
	return s, found
}

func (m *SessionManager) RemoveSession(id string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.sessions, id)
}

func (m *SessionManager) CloseSession(id string) error {
	m.mutex.RLock()
	s, found := m.sessions[id]
	m.mutex.RUnlock()

	if !found {
		return fmt.Errorf("session not found: %s", id)
	}

	err := s.Close()
	m.RemoveSession(id)
	return err
}

func (m *SessionManager) CloseAll() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for id, s := range m.sessions {
		_ = s.Close()
		delete(m.sessions, id)
	}
}

// HandleTerminalWs handles bidirectional terminal streaming over WebSocket
func HandleTerminalWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Parse query params: command and session id
	command := r.URL.Query().Get("cmd")
	if command == "" {
		command = os.Getenv("SHELL")
		if command == "" {
			command = "/bin/bash"
		}
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = fmt.Sprintf("pty_%d", time.Now().UnixNano())
	}

	cmd := exec.Command(command)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to spawn PTY: %v\r\n", err)))
		return
	}

	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil || pgid <= 1 {
		pgid = pid
	}

	session := &Session{
		ID:        sessionID,
		PID:       pid,
		PGID:      pgid,
		Command:   command,
		Active:    true,
		CreatedAt: time.Now(),
		Cmd:       cmd,
		Ptmx:      ptmx,
	}

	DefaultManager.AddSession(session)

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = session.Close()
			DefaultManager.RemoveSession(sessionID)
		})
	}
	defer cleanup()

	// Parse initial terminal dimensions
	cols := uint16(80)
	rows := uint16(24)
	if c, err := strconv.ParseUint(r.URL.Query().Get("cols"), 10, 16); err == nil && c > 0 {
		cols = uint16(c)
	}
	if rw, err := strconv.ParseUint(r.URL.Query().Get("rows"), 10, 16); err == nil && rw > 0 {
		rows = uint16(rw)
	}
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: rows, Cols: cols})

	var wsWriteMutex sync.Mutex
	safeWrite := func(msgType int, data []byte) error {
		wsWriteMutex.Lock()
		defer wsWriteMutex.Unlock()
		return conn.WriteMessage(msgType, data)
	}
	safeWriteJSON := func(v interface{}) error {
		wsWriteMutex.Lock()
		defer wsWriteMutex.Unlock()
		return conn.WriteJSON(v)
	}

	// Output routine: PTMX -> WebSocket
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				writeErr := safeWrite(websocket.BinaryMessage, buf[:n])
				if writeErr != nil {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		close(done)
	}()

	// Input routine: WebSocket -> PTMX & signal/resize controls
	for {
		msgType, msgBytes, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}

		if msgType == websocket.BinaryMessage {
			// Raw terminal bytes: escape sequences, arrow keys, Ctrl+C (\x03), Ctrl+D (\x04), etc.
			_, _ = ptmx.Write(msgBytes)
		} else if msgType == websocket.TextMessage {
			var termMsg TerminalMessage
			if err := json.Unmarshal(msgBytes, &termMsg); err == nil && termMsg.Type != "" {
				switch termMsg.Type {
				case "resize":
					if termMsg.Cols > 0 && termMsg.Rows > 0 {
						_ = pty.Setsize(ptmx, &pty.Winsize{
							Rows: termMsg.Rows,
							Cols: termMsg.Cols,
						})
					}
				case "signal":
					sig := security.ParseSignal(termMsg.Data)
					if sig != 0 && session.PGID > 1 {
						_ = security.SendSignalToProcessGroup(session.PGID, sig)
					}
				case "ping":
					_ = safeWriteJSON(map[string]string{"type": "pong"})
				case "input":
					_, _ = io.WriteString(ptmx, termMsg.Data)
				}
			} else {
				// Raw string text input
				_, _ = ptmx.Write(msgBytes)
			}
		}
	}

	cleanup()
	<-done
}
