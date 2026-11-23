package sync

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"unicode/utf8"
)

// StoragePolicy defines where files live in the READ ecosystem.
type StoragePolicy string

const (
	PolicyPCOnly    StoragePolicy = "pc_only"
	PolicyPhoneOnly StoragePolicy = "phone_only"
	PolicyBothSync  StoragePolicy = "both_sync"
)

// NodeRole designates the architectural role of a device.
type NodeRole string

const (
	RolePC    NodeRole = "pc"
	RolePhone NodeRole = "phone"
	RoleVPS   NodeRole = "vps"
)

// FileManifest records synchronized file state across nodes.
type FileManifest struct {
	RelativePath string        `json:"relative_path"`
	Policy       StoragePolicy `json:"policy"`
	SHA256       string        `json:"sha256"`
	SizeBytes    int64         `json:"size_bytes"`
	ModifiedUnix int64         `json:"modified_unix"`
	Mode         uint32        `json:"mode,omitempty"`
	IsSymlink    bool          `json:"is_symlink,omitempty"`
	LinkTarget   string        `json:"link_target,omitempty"`
	IsBinary     bool          `json:"is_binary,omitempty"`
}

// MerkleNode represents a deterministic hierarchical directory hash vector.
type MerkleNode struct {
	Path       string        `json:"path"`
	Name       string        `json:"name"`
	IsDir      bool          `json:"is_dir"`
	Hash       string        `json:"hash"`
	SizeBytes  int64         `json:"size_bytes"`
	Mode       uint32        `json:"mode"`
	IsSymlink  bool          `json:"is_symlink,omitempty"`
	LinkTarget string        `json:"link_target,omitempty"`
	IsBinary   bool          `json:"is_binary,omitempty"`
	Policy     StoragePolicy `json:"policy,omitempty"`
	Children   []*MerkleNode `json:"children,omitempty"`
}

// SyncActionType specifies the required synchronization operation.
type SyncActionType string

const (
	ActionNone     SyncActionType = "none"
	ActionUpload   SyncActionType = "upload"
	ActionDownload SyncActionType = "download"
	ActionDelete   SyncActionType = "delete"
	ActionConflict SyncActionType = "conflict_resolved"
	ActionSkip     SyncActionType = "skip_policy"
)

// SyncDeltaItem describes an action for a single file during delta sync.
type SyncDeltaItem struct {
	RelativePath string         `json:"relative_path"`
	Action       SyncActionType `json:"action"`
	Reason       string         `json:"reason"`
	SourceHash   string         `json:"source_hash,omitempty"`
	TargetHash   string         `json:"target_hash,omitempty"`
}

// DeltaPlan encapsulates the computed synchronization plan between two nodes.
type DeltaPlan struct {
	LocalRole  NodeRole        `json:"local_role"`
	RemoteRole NodeRole        `json:"remote_role"`
	Items      []SyncDeltaItem `json:"items"`
}

// SyncEngine coordinates file policies and bi-directional manifests
type SyncEngine struct {
	WorkspaceRoot string
	DefaultPolicy StoragePolicy
	FilePolicies  map[string]StoragePolicy
	mutex         sync.RWMutex
}

func NewSyncEngine(root string, defaultPolicy StoragePolicy) *SyncEngine {
	if defaultPolicy == "" {
		defaultPolicy = PolicyBothSync
	}
	return &SyncEngine{
		WorkspaceRoot: root,
		DefaultPolicy: defaultPolicy,
		FilePolicies:  make(map[string]StoragePolicy),
	}
}

func (e *SyncEngine) SetPolicy(relPath string, policy StoragePolicy) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	clean := filepath.Clean(relPath)
	e.FilePolicies[clean] = policy
}

func (e *SyncEngine) GetPolicy(relPath string) StoragePolicy {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	clean := filepath.Clean(relPath)
	if p, ok := e.FilePolicies[clean]; ok {
		return p
	}
	return e.DefaultPolicy
}

// ShouldFileExistOnNode determines if a file with the given policy belongs on node role.
func ShouldFileExistOnNode(policy StoragePolicy, role NodeRole) bool {
	switch policy {
	case PolicyBothSync:
		return true
	case PolicyPCOnly:
		return role == RolePC || role == RoleVPS
	case PolicyPhoneOnly:
		return role == RolePhone
	default:
		return true
	}
}

// ComputeFileHash calculates SHA-256 for a local file in streaming chunks.
func ComputeFileHash(path string) (string, int64, error) {
	hash, size, _, err := InspectFile(path)
	return hash, size, err
}

// InspectFile inspects a file or symlink, determining hash, size, and whether binary.
func InspectFile(path string) (string, int64, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, false, err
	}

	// Symlink handling: do not follow target into outside workspace or loops
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", 0, false, err
		}
		h := sha256.Sum256([]byte(target))
		return hex.EncodeToString(h[:]), int64(len(target)), false, nil
	}

	// Empty file handling
	if info.Size() == 0 {
		emptyHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
		return emptyHash, 0, false, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return "", 0, false, err
	}
	defer file.Close()

	hasher := sha256.New()
	buf := make([]byte, 32*1024)
	isBinary := false
	checkedBinary := false
	var totalSize int64

	for {
		n, err := file.Read(buf)
		if n > 0 {
			totalSize += int64(n)
			hasher.Write(buf[:n])

			if !checkedBinary {
				probe := buf[:min(n, 8192)]
				if bytes.IndexByte(probe, 0) != -1 || !utf8.Valid(probe) {
					isBinary = true
				}
				checkedBinary = true
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, false, err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), totalSize, isBinary, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isIgnoredDir checks if directory name should be excluded from sync.
func isIgnoredDir(name string) bool {
	return name == ".git" || name == "node_modules" || name == "__pycache__" ||
		name == "target" || name == ".method"
}

// ScanProjectManifest builds a manifest of the project adhering to storage policies.
func ScanProjectManifest(root string, defaultPolicy StoragePolicy) ([]FileManifest, error) {
	eng := NewSyncEngine(root, defaultPolicy)
	return eng.ScanManifest()
}

// ScanManifest scans the workspace root using policies configured on SyncEngine.
func (e *SyncEngine) ScanManifest() ([]FileManifest, error) {
	var list []FileManifest

	err := filepath.Walk(e.WorkspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if path != e.WorkspaceRoot && isIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(e.WorkspaceRoot, path)
		if err != nil {
			return err
		}

		// Lstat to handle symlinks correctly
		linfo, err := os.Lstat(path)
		if err != nil {
			return nil
		}

		isSymlink := linfo.Mode()&os.ModeSymlink != 0
		var linkTarget string
		if isSymlink {
			target, err := os.Readlink(path)
			if err == nil {
				linkTarget = target
			}
		}

		hash, size, isBinary, err := InspectFile(path)
		if err != nil {
			return nil
		}

		policy := e.GetPolicy(rel)

		list = append(list, FileManifest{
			RelativePath: rel,
			Policy:       policy,
			SHA256:       hash,
			SizeBytes:    size,
			ModifiedUnix: info.ModTime().Unix(),
			Mode:         uint32(linfo.Mode().Perm()),
			IsSymlink:    isSymlink,
			LinkTarget:   linkTarget,
			IsBinary:     isBinary,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].RelativePath < list[j].RelativePath
	})

	return list, nil
}

// BuildMerkleTree generates a deterministic Merkle tree from the workspace root.
func BuildMerkleTree(root string, defaultPolicy StoragePolicy) (*MerkleNode, error) {
	eng := NewSyncEngine(root, defaultPolicy)
	return eng.BuildMerkleTree()
}

// BuildMerkleTree computes the hierarchical directory Merkle tree.
func (e *SyncEngine) BuildMerkleTree() (*MerkleNode, error) {
	return e.buildMerkleDir(e.WorkspaceRoot, "")
}

func (e *SyncEngine) buildMerkleDir(fullPath, relPath string) (*MerkleNode, error) {
	info, err := os.Lstat(fullPath)
	if err != nil {
		return nil, err
	}

	node := &MerkleNode{
		Path:      relPath,
		Name:      filepath.Base(fullPath),
		IsDir:     true,
		Mode:      uint32(info.Mode().Perm()),
		Policy:    e.GetPolicy(relPath),
		SizeBytes: 0,
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var childNodes []*MerkleNode
	hasher := sha256.New()

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() && isIgnoredDir(name) {
			continue
		}

		childFull := filepath.Join(fullPath, name)
		var childRel string
		if relPath == "" {
			childRel = name
		} else {
			childRel = filepath.Join(relPath, name)
		}

		if entry.IsDir() {
			subTree, err := e.buildMerkleDir(childFull, childRel)
			if err != nil {
				continue
			}
			childNodes = append(childNodes, subTree)
			node.SizeBytes += subTree.SizeBytes
		} else {
			linfo, err := entry.Info()
			if err != nil {
				continue
			}

			isSymlink := linfo.Mode()&os.ModeSymlink != 0
			var linkTarget string
			if isSymlink {
				target, err := os.Readlink(childFull)
				if err == nil {
					linkTarget = target
				}
			}

			hash, size, isBinary, err := InspectFile(childFull)
			if err != nil {
				continue
			}

			childNode := &MerkleNode{
				Path:       childRel,
				Name:       name,
				IsDir:      false,
				Hash:       hash,
				SizeBytes:  size,
				Mode:       uint32(linfo.Mode().Perm()),
				IsSymlink:  isSymlink,
				LinkTarget: linkTarget,
				IsBinary:   isBinary,
				Policy:     e.GetPolicy(childRel),
			}
			childNodes = append(childNodes, childNode)
			node.SizeBytes += size
		}
	}

	// Sort child nodes deterministically by Name
	sort.Slice(childNodes, func(i, j int) bool {
		return childNodes[i].Name < childNodes[j].Name
	})

	for _, c := range childNodes {
		fmt.Fprintf(hasher, "%s:%s:%d:%d\n", c.Name, c.Hash, c.SizeBytes, c.Mode)
	}

	node.Hash = hex.EncodeToString(hasher.Sum(nil))
	node.Children = childNodes

	return node, nil
}

// ComputeDelta calculates deterministic delta synchronization actions between two nodes.
func ComputeDelta(
	localManifest, remoteManifest []FileManifest,
	localRole, remoteRole NodeRole,
) DeltaPlan {
	localMap := make(map[string]FileManifest, len(localManifest))
	for _, m := range localManifest {
		localMap[m.RelativePath] = m
	}

	remoteMap := make(map[string]FileManifest, len(remoteManifest))
	for _, m := range remoteManifest {
		remoteMap[m.RelativePath] = m
	}

	allPathsMap := make(map[string]struct{})
	for p := range localMap {
		allPathsMap[p] = struct{}{}
	}
	for p := range remoteMap {
		allPathsMap[p] = struct{}{}
	}

	allPaths := make([]string, 0, len(allPathsMap))
	for p := range allPathsMap {
		allPaths = append(allPaths, p)
	}
	sort.Strings(allPaths)

	plan := DeltaPlan{
		LocalRole:  localRole,
		RemoteRole: remoteRole,
		Items:      make([]SyncDeltaItem, 0, len(allPaths)),
	}

	for _, p := range allPaths {
		loc, hasLocal := localMap[p]
		rem, hasRemote := remoteMap[p]

		var policy StoragePolicy
		if hasLocal {
			policy = loc.Policy
		} else {
			policy = rem.Policy
		}

		localAllowed := ShouldFileExistOnNode(policy, localRole)
		remoteAllowed := ShouldFileExistOnNode(policy, remoteRole)

		if hasLocal && !localAllowed {
			plan.Items = append(plan.Items, SyncDeltaItem{
				RelativePath: p,
				Action:       ActionDelete,
				Reason:       fmt.Sprintf("policy '%s' forbids file on local role '%s'", policy, localRole),
				SourceHash:   loc.SHA256,
			})
			continue
		}

		if hasLocal && hasRemote {
			if loc.SHA256 == rem.SHA256 {
				plan.Items = append(plan.Items, SyncDeltaItem{
					RelativePath: p,
					Action:       ActionNone,
					Reason:       "content identical across nodes",
					SourceHash:   loc.SHA256,
					TargetHash:   rem.SHA256,
				})
			} else {
				// Last-Write-Wins (LWW) conflict resolution with deterministic hash tie-break
				if loc.ModifiedUnix > rem.ModifiedUnix {
					plan.Items = append(plan.Items, SyncDeltaItem{
						RelativePath: p,
						Action:       ActionUpload,
						Reason:       "conflict: local modified time is newer (LWW)",
						SourceHash:   loc.SHA256,
						TargetHash:   rem.SHA256,
					})
				} else if rem.ModifiedUnix > loc.ModifiedUnix {
					plan.Items = append(plan.Items, SyncDeltaItem{
						RelativePath: p,
						Action:       ActionDownload,
						Reason:       "conflict: remote modified time is newer (LWW)",
						SourceHash:   loc.SHA256,
						TargetHash:   rem.SHA256,
					})
				} else {
					// Timestamps equal: deterministic tie-break using lexicographical hash ordering
					if loc.SHA256 > rem.SHA256 {
						plan.Items = append(plan.Items, SyncDeltaItem{
							RelativePath: p,
							Action:       ActionUpload,
							Reason:       "conflict: tie-break local hash precedence",
							SourceHash:   loc.SHA256,
							TargetHash:   rem.SHA256,
						})
					} else {
						plan.Items = append(plan.Items, SyncDeltaItem{
							RelativePath: p,
							Action:       ActionDownload,
							Reason:       "conflict: tie-break remote hash precedence",
							SourceHash:   loc.SHA256,
							TargetHash:   rem.SHA256,
						})
					}
				}
			}
			continue
		}

		if hasLocal && !hasRemote {
			if !remoteAllowed {
				plan.Items = append(plan.Items, SyncDeltaItem{
					RelativePath: p,
					Action:       ActionSkip,
					Reason:       fmt.Sprintf("policy '%s' excludes remote role '%s'", policy, remoteRole),
					SourceHash:   loc.SHA256,
				})
			} else {
				plan.Items = append(plan.Items, SyncDeltaItem{
					RelativePath: p,
					Action:       ActionUpload,
					Reason:       "file exists locally and remote node requires it",
					SourceHash:   loc.SHA256,
				})
			}
			continue
		}

		if !hasLocal && hasRemote {
			if !localAllowed {
				plan.Items = append(plan.Items, SyncDeltaItem{
					RelativePath: p,
					Action:       ActionSkip,
					Reason:       fmt.Sprintf("policy '%s' excludes local role '%s'", policy, localRole),
					TargetHash:   rem.SHA256,
				})
			} else {
				plan.Items = append(plan.Items, SyncDeltaItem{
					RelativePath: p,
					Action:       ActionDownload,
					Reason:       "file exists remotely and local node requires it",
					TargetHash:   rem.SHA256,
				})
			}
			continue
		}
	}

	return plan
}
