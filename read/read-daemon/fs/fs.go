package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"read-daemon/security"
	"read-daemon/sync"
)

// ErrForbidden indicates a path traversal or security boundary violation.
var ErrForbidden = security.ErrPathForbidden

type FileItem struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	SHA256  string `json:"sha256,omitempty"`
}

type FsManager struct {
	WorkspaceRoot string
	SyncEngine    *sync.SyncEngine
}

func NewFsManager(root string, syncEngine *sync.SyncEngine) *FsManager {
	return &FsManager{
		WorkspaceRoot: root,
		SyncEngine:    syncEngine,
	}
}

// sanitizePath ensures that requested paths never escape the WorkspaceRoot
func (m *FsManager) sanitizePath(relPath string) (string, error) {
	return security.ResolveWorkspacePath(m.WorkspaceRoot, relPath)
}

// ListDirectory lists contents of a relative directory
func (m *FsManager) ListDirectory(relPath string) ([]FileItem, error) {
	fullPath, err := m.sanitizePath(relPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	items := make([]FileItem, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		itemRelPath := filepath.Join(relPath, entry.Name())
		item := FileItem{
			Name:    entry.Name(),
			Path:    itemRelPath,
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		}

		// Security check: ensure item does not resolve outside workspace before computing hash
		entryFullPath := filepath.Join(fullPath, entry.Name())
		if _, err := security.ResolveWorkspacePath(m.WorkspaceRoot, itemRelPath); err == nil {
			if !entry.IsDir() && info.Size() < 1024*1024 {
				hash, err := hashFile(entryFullPath)
				if err == nil {
					item.SHA256 = hash
				}
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// ReadFile returns raw content of requested file
func (m *FsManager) ReadFile(relPath string) ([]byte, error) {
	fullPath, err := m.sanitizePath(relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

// WriteFile writes content and updates sync metadata
func (m *FsManager) WriteFile(relPath string, data []byte, policy string) error {
	fullPath, err := m.sanitizePath(relPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Atomic write
	tmpFile := fullPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, fullPath); err != nil {
		return err
	}

	// If policy specified, register with sync engine
	if policy != "" && m.SyncEngine != nil {
		m.SyncEngine.SetPolicy(relPath, sync.StoragePolicy(policy))
	}

	return nil
}

// DeleteFile deletes a file or directory
func (m *FsManager) DeleteFile(relPath string) error {
	fullPath, err := m.sanitizePath(relPath)
	if err != nil {
		return err
	}

	realRoot, err := filepath.EvalSymlinks(m.WorkspaceRoot)
	if err != nil {
		realRoot = filepath.Clean(m.WorkspaceRoot)
	}
	if fullPath == realRoot || fullPath == m.WorkspaceRoot {
		return fmt.Errorf("security violation: cannot delete workspace root")
	}

	return os.RemoveAll(fullPath)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
