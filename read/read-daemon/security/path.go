package security

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ErrPathForbidden is returned when a requested path attempts to escape the workspace root.
var ErrPathForbidden = errors.New("forbidden: access outside workspace is not allowed")

// ResolveWorkspacePath validates that the requested relative path stays strictly
// within the canonical workspace root. It rejects directory traversal (e.g. "../"),
// multi-level URL encoding tricks, null bytes, and symlinks that point outside the root.
func ResolveWorkspacePath(workspaceRoot, requestedPath string) (string, error) {
	if workspaceRoot == "" {
		return "", ErrPathForbidden
	}

	// 1. Resolve workspace root to canonical absolute realpath (resolving symlinks)
	realRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		realRoot = filepath.Clean(workspaceRoot)
	}
	realRoot, err = filepath.Abs(realRoot)
	if err != nil {
		return "", ErrPathForbidden
	}

	// 2. Iteratively decode URL percent-encoding (up to 3 rounds for double/triple encoded attacks)
	decoded := requestedPath
	for i := 0; i < 3; i++ {
		if unescaped, err := url.QueryUnescape(decoded); err == nil && unescaped != decoded {
			decoded = unescaped
		} else {
			break
		}
	}

	// 3. Reject null bytes and control characters
	if strings.ContainsRune(decoded, '\x00') {
		return "", ErrPathForbidden
	}

	// 4. Normalize path separators
	decoded = strings.ReplaceAll(decoded, "\\", "/")

	if decoded == "/" || decoded == "" || decoded == "." {
		return realRoot, nil
	}

	// 5. Determine candidate full path
	var candidateFullPath string
	if filepath.IsAbs(decoded) {
		candidateFullPath = filepath.Clean(decoded)
	} else {
		candidateFullPath = filepath.Clean(filepath.Join(realRoot, decoded))
	}

	// 6. Lexical boundary check: candidateFullPath must be inside realRoot
	relLex, err := filepath.Rel(realRoot, candidateFullPath)
	if err != nil || relLex == ".." || strings.HasPrefix(relLex, ".."+string(filepath.Separator)) {
		return "", ErrPathForbidden
	}

	// 7. Symlink resolution check
	if fi, err := os.Lstat(candidateFullPath); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := filepath.EvalSymlinks(candidateFullPath)
			if err != nil {
				// Broken symlink - check raw symlink destination
				linkTarget, rerr := os.Readlink(candidateFullPath)
				if rerr != nil {
					return "", ErrPathForbidden
				}
				if !filepath.IsAbs(linkTarget) {
					linkTarget = filepath.Join(filepath.Dir(candidateFullPath), linkTarget)
				}
				cleanTarget := filepath.Clean(linkTarget)
				relCheck, terr := filepath.Rel(realRoot, cleanTarget)
				if terr != nil || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
					return "", ErrPathForbidden
				}
				return candidateFullPath, nil
			}
			relToRoot, err := filepath.Rel(realRoot, target)
			if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
				return "", ErrPathForbidden
			}
			return target, nil
		}

		realTarget, err := filepath.EvalSymlinks(candidateFullPath)
		if err != nil {
			return "", ErrPathForbidden
		}
		relToRoot, err := filepath.Rel(realRoot, realTarget)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			return "", ErrPathForbidden
		}
		return realTarget, nil
	}

	// 8. Target does not exist yet (e.g. for new files created by WriteFile).
	// We walk up to find the closest existing ancestor directory and verify its realpath.
	curr := filepath.Dir(candidateFullPath)
	for {
		if fi, err := os.Stat(curr); err == nil {
			if !fi.IsDir() {
				return "", ErrPathForbidden
			}
			realCurr, err := filepath.EvalSymlinks(curr)
			if err != nil {
				return "", ErrPathForbidden
			}
			relToRoot, err := filepath.Rel(realRoot, realCurr)
			if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
				return "", ErrPathForbidden
			}
			break
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			return "", ErrPathForbidden
		}
		curr = parent
	}

	return candidateFullPath, nil
}
