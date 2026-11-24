package sync

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeFileHash_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyPath := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}

	hash, size, isBinary, err := InspectFile(emptyPath)
	if err != nil {
		t.Fatalf("unexpected error inspecting empty file: %v", err)
	}

	expectedHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if hash != expectedHash {
		t.Errorf("expected empty hash %s, got %s", expectedHash, hash)
	}
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}
	if isBinary {
		t.Errorf("empty file should not be flagged as binary")
	}
}

func TestScanProjectManifest_EdgeCases(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Text file with executable permissions
	execPath := filepath.Join(tmpDir, "script.sh")
	if err := os.WriteFile(execPath, []byte("#!/bin/sh\necho hello\n"), 0755); err != nil {
		t.Fatalf("failed to write script.sh: %v", err)
	}

	// 2. Binary file with null bytes
	binPath := filepath.Join(tmpDir, "data.bin")
	binData := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01, 0x01, 0x00}
	if err := os.WriteFile(binPath, binData, 0644); err != nil {
		t.Fatalf("failed to write data.bin: %v", err)
	}

	// 3. Symlink
	symPath := filepath.Join(tmpDir, "link_to_script")
	if err := os.Symlink("script.sh", symPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// 4. Empty file
	emptyPath := filepath.Join(tmpDir, "zero.dat")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to write zero.dat: %v", err)
	}

	// 5. Ignored dir
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("failed to create .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644); err != nil {
		t.Fatalf("failed to write git HEAD: %v", err)
	}

	eng := NewSyncEngine(tmpDir, PolicyBothSync)
	eng.SetPolicy("data.bin", PolicyPCOnly)

	manifests, err := eng.ScanManifest()
	if err != nil {
		t.Fatalf("unexpected scan error: %v", err)
	}

	mMap := make(map[string]FileManifest)
	for _, m := range manifests {
		mMap[m.RelativePath] = m
	}

	// Verify .git was ignored
	if _, ok := mMap[".git/HEAD"]; ok {
		t.Errorf(".git/HEAD should have been skipped")
	}

	// Verify script.sh
	script, ok := mMap["script.sh"]
	if !ok {
		t.Fatalf("script.sh missing in manifest")
	}
	if script.IsBinary {
		t.Errorf("script.sh should not be binary")
	}
	if script.Mode != 0755 {
		t.Errorf("expected mode 0755, got %o", script.Mode)
	}

	// Verify data.bin
	dataBin, ok := mMap["data.bin"]
	if !ok {
		t.Fatalf("data.bin missing in manifest")
	}
	if !dataBin.IsBinary {
		t.Errorf("data.bin should be detected as binary")
	}
	if dataBin.Policy != PolicyPCOnly {
		t.Errorf("expected policy pc_only for data.bin, got %s", dataBin.Policy)
	}

	// Verify symlink
	link, ok := mMap["link_to_script"]
	if !ok {
		t.Fatalf("link_to_script missing in manifest")
	}
	if !link.IsSymlink {
		t.Errorf("link_to_script should be detected as symlink")
	}
	if link.LinkTarget != "script.sh" {
		t.Errorf("expected link target 'script.sh', got '%s'", link.LinkTarget)
	}

	// Verify empty file
	zero, ok := mMap["zero.dat"]
	if !ok {
		t.Fatalf("zero.dat missing in manifest")
	}
	if zero.SizeBytes != 0 {
		t.Errorf("expected size 0, got %d", zero.SizeBytes)
	}
}

func TestMerkleTree_Determinism(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Readme\n"), 0644); err != nil {
		t.Fatalf("failed to write README.md: %v", err)
	}

	tree1, err := BuildMerkleTree(tmpDir, PolicyBothSync)
	if err != nil {
		t.Fatalf("failed to build tree1: %v", err)
	}

	tree2, err := BuildMerkleTree(tmpDir, PolicyBothSync)
	if err != nil {
		t.Fatalf("failed to build tree2: %v", err)
	}

	if tree1.Hash != tree2.Hash {
		t.Errorf("Merkle root hash should be deterministic: %s != %s", tree1.Hash, tree2.Hash)
	}

	// Modify one file and verify root hash changes
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte("package main\n// updated\n"), 0644); err != nil {
		t.Fatalf("failed to update main.go: %v", err)
	}

	tree3, err := BuildMerkleTree(tmpDir, PolicyBothSync)
	if err != nil {
		t.Fatalf("failed to build tree3: %v", err)
	}

	if tree1.Hash == tree3.Hash {
		t.Errorf("Merkle root hash should change when file content changes")
	}
}

func TestStoragePolicies_DeterministicBehavior(t *testing.T) {
	// 1. ShouldFileExistOnNode verification
	if !ShouldFileExistOnNode(PolicyBothSync, RolePC) || !ShouldFileExistOnNode(PolicyBothSync, RolePhone) {
		t.Errorf("both_sync must exist on pc and phone")
	}
	if !ShouldFileExistOnNode(PolicyPCOnly, RolePC) || !ShouldFileExistOnNode(PolicyPCOnly, RoleVPS) {
		t.Errorf("pc_only must exist on pc and vps")
	}
	if ShouldFileExistOnNode(PolicyPCOnly, RolePhone) {
		t.Errorf("pc_only must not exist on phone")
	}
	if !ShouldFileExistOnNode(PolicyPhoneOnly, RolePhone) {
		t.Errorf("phone_only must exist on phone")
	}
	if ShouldFileExistOnNode(PolicyPhoneOnly, RolePC) {
		t.Errorf("phone_only must not exist on pc")
	}

	// 2. ComputeDelta scenario
	now := time.Now().Unix()

	local := []FileManifest{
		{RelativePath: "sync.txt", Policy: PolicyBothSync, SHA256: "hash_old", ModifiedUnix: now - 100},
		{RelativePath: "pc_file.bin", Policy: PolicyPCOnly, SHA256: "hash_pc", ModifiedUnix: now},
		{RelativePath: "orphaned.txt", Policy: PolicyPhoneOnly, SHA256: "hash_orph", ModifiedUnix: now},
	}

	remote := []FileManifest{
		{RelativePath: "sync.txt", Policy: PolicyBothSync, SHA256: "hash_new", ModifiedUnix: now},
		{RelativePath: "phone_log.txt", Policy: PolicyPhoneOnly, SHA256: "hash_phone", ModifiedUnix: now},
	}

	// Local = PC, Remote = Phone
	plan := ComputeDelta(local, remote, RolePC, RolePhone)

	actionMap := make(map[string]SyncActionType)
	for _, item := range plan.Items {
		actionMap[item.RelativePath] = item.Action
	}

	// sync.txt: remote is newer -> download
	if actionMap["sync.txt"] != ActionDownload {
		t.Errorf("expected sync.txt to be download, got %s", actionMap["sync.txt"])
	}

	// pc_file.bin: pc_only, remote is phone -> skip
	if actionMap["pc_file.bin"] != ActionSkip {
		t.Errorf("expected pc_file.bin to be skip on phone remote, got %s", actionMap["pc_file.bin"])
	}

	// orphaned.txt: phone_only, local is PC -> delete
	if actionMap["orphaned.txt"] != ActionDelete {
		t.Errorf("expected orphaned.txt to be deleted on PC, got %s", actionMap["orphaned.txt"])
	}

	// phone_log.txt: phone_only, exists on remote, local is PC -> skip
	if actionMap["phone_log.txt"] != ActionSkip {
		t.Errorf("expected phone_log.txt to be skipped on PC, got %s", actionMap["phone_log.txt"])
	}
}
