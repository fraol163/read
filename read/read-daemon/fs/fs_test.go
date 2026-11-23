package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFsManagerTraversalRejection(t *testing.T) {
	wsDir, err := os.MkdirTemp("", "fs_test_ws_*")
	if err != nil {
		t.Fatalf("failed to create temp workspace: %v", err)
	}
	defer os.RemoveAll(wsDir)

	mgr := NewFsManager(wsDir, nil)

	// Create valid file
	if err := mgr.WriteFile("hello.txt", []byte("world"), ""); err != nil {
		t.Fatalf("failed to write valid file: %v", err)
	}

	// Create symlink to /etc inside workspace
	symlinkEtc := filepath.Join(wsDir, "symlink_etc")
	_ = os.Symlink("/etc", symlinkEtc)

	maliciousPaths := []string{
		"../etc/passwd",
		"../../../../etc/shadow",
		"%2e%2e%2fetc/passwd",
		"%2e%2e/%2e%2e/etc/passwd",
		"%252e%252e%252fetc/passwd",
		"symlink_etc/passwd",
		"symlink_etc/hosts",
		"symlink_etc",
		"valid.txt\x00extra",
		"/etc/passwd",
	}

	// 1. ReadFile must reject all malicious paths with ErrForbidden
	for _, p := range maliciousPaths {
		_, err := mgr.ReadFile(p)
		if err == nil {
			t.Fatalf("ReadFile(%q) succeeded, expected ErrForbidden", p)
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("ReadFile(%q) returned %v, expected ErrForbidden", p, err)
		}
	}

	// 2. WriteFile must reject all malicious paths with ErrForbidden
	for _, p := range maliciousPaths {
		err := mgr.WriteFile(p, []byte("attack"), "")
		if err == nil {
			t.Fatalf("WriteFile(%q) succeeded, expected ErrForbidden", p)
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("WriteFile(%q) returned %v, expected ErrForbidden", p, err)
		}
	}

	// 3. ListDirectory must reject malicious paths
	for _, p := range []string{"../", "../../", "%2e%2e%2f", "symlink_etc"} {
		_, err := mgr.ListDirectory(p)
		if err == nil {
			t.Fatalf("ListDirectory(%q) succeeded, expected ErrForbidden", p)
		}
		if !errors.Is(err, ErrForbidden) {
			t.Fatalf("ListDirectory(%q) returned %v, expected ErrForbidden", p, err)
		}
	}

	// 4. DeleteFile must reject malicious paths and workspace root
	for _, p := range append(maliciousPaths, "", ".") {
		err := mgr.DeleteFile(p)
		if err == nil {
			t.Fatalf("DeleteFile(%q) succeeded, expected error", p)
		}
	}

	// 5. Valid operations should succeed
	data, err := mgr.ReadFile("hello.txt")
	if err != nil {
		t.Fatalf("ReadFile valid failed: %v", err)
	}
	if string(data) != "world" {
		t.Fatalf("ReadFile content mismatch: got %q, want 'world'", string(data))
	}

	items, err := mgr.ListDirectory("")
	if err != nil {
		t.Fatalf("ListDirectory root failed: %v", err)
	}
	found := false
	for _, item := range items {
		if item.Name == "hello.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected hello.txt in ListDirectory output")
	}

	if err := mgr.DeleteFile("hello.txt"); err != nil {
		t.Fatalf("DeleteFile valid failed: %v", err)
	}
}
