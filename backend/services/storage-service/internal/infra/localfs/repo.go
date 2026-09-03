// Package localfs implements the filesystem repository using the local OS filesystem.
//
// Each tenant's files are sandboxed under: {BaseDir}/{tenantID}/
// Path traversal attacks are prevented at the usecase layer AND here.
package localfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vnp-memory/services/storage-service/internal/domain/fs"
)

// Repo implements usecase/fs.Repository using the local filesystem.
type Repo struct {
	BaseDir string // root directory for all tenant sandboxes
}

// NewRepo creates a LocalFSRepo.
func NewRepo(baseDir string) *Repo {
	return &Repo{BaseDir: baseDir}
}

// tenantPath returns the full, validated path for a tenant+relative path.
// Returns error if path would escape the tenant sandbox.
func (r *Repo) tenantPath(tenantID, relPath string) (string, error) {
	// Resolve base
	base := filepath.Join(r.BaseDir, filepath.Clean(tenantID))
	full := filepath.Join(base, filepath.Clean(relPath))

	// Verify it's still under base (prevent ../../../etc/passwd)
	rel, err := filepath.Rel(base, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", relPath)
	}
	return full, nil
}

// Read reads the content of a file.
func (r *Repo) Read(_ context.Context, tenantID, path string) ([]byte, error) {
	full, err := r.tenantPath(tenantID, path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(full)
}

// Write writes content to a file, creating parent directories as needed.
func (r *Repo) Write(_ context.Context, tenantID, path string, content []byte) error {
	full, err := r.tenantPath(tenantID, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("create dirs: %w", err)
	}
	return os.WriteFile(full, content, 0o644)
}

// Delete removes a file.
func (r *Repo) Delete(_ context.Context, tenantID, path string) error {
	full, err := r.tenantPath(tenantID, path)
	if err != nil {
		return err
	}
	return os.Remove(full)
}

// Tree returns a directory tree up to the given depth.
func (r *Repo) Tree(_ context.Context, tenantID, path string, depth int) (*fs.Directory, error) {
	full, err := r.tenantPath(tenantID, path)
	if err != nil {
		return nil, err
	}
	dir := &fs.Directory{Path: path}
	children, err := buildTree(full, 0, depth)
	if err != nil {
		return nil, fmt.Errorf("build tree: %w", err)
	}
	dir.Children = children
	return dir, nil
}

func buildTree(dirPath string, currentDepth, maxDepth int) ([]fs.TreeNode, error) {
	if currentDepth >= maxDepth {
		return nil, nil
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var nodes []fs.TreeNode
	for _, e := range entries {
		node := fs.TreeNode{
			Name:  e.Name(),
			Path:  filepath.Join(dirPath, e.Name()),
			IsDir: e.IsDir(),
		}
		if !e.IsDir() {
			if info, err := e.Info(); err == nil {
				node.Size = info.Size()
			}
		} else {
			children, _ := buildTree(filepath.Join(dirPath, e.Name()), currentDepth+1, maxDepth)
			node.Children = children
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
