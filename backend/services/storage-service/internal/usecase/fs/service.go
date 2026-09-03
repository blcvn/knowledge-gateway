// Package fs implements the filesystem usecase for storage-service.
//
// Provides: ReadFile, WriteFile, DeleteFile, Tree, Grep
// Backend: local filesystem sandboxed per tenant (MERGE-P1-T4)
package fs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vnp-memory/services/storage-service/internal/domain/fs"
)

// Repository is the output port for filesystem operations.
type Repository interface {
	Read(ctx context.Context, tenantID, path string) ([]byte, error)
	Write(ctx context.Context, tenantID, path string, content []byte) error
	Delete(ctx context.Context, tenantID, path string) error
	Tree(ctx context.Context, tenantID, path string, depth int) (*fs.Directory, error)
}

// Service implements filesystem use cases.
type Service struct {
	repo    Repository
	baseDir string // root sandbox directory
}

// NewService creates a filesystem Service.
func NewService(repo Repository, baseDir string) *Service {
	return &Service{repo: repo, baseDir: baseDir}
}

// ReadFile reads a file for a tenant.
func (s *Service) ReadFile(ctx context.Context, tenantID, path string) (*fs.File, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	content, err := s.repo.Read(ctx, tenantID, path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	info, _ := os.Stat(filepath.Join(s.baseDir, tenantID, path))
	f := &fs.File{
		Path:    path,
		Content: content,
		Size:    int64(len(content)),
	}
	if info != nil {
		f.Size = info.Size()
		f.ModTime = info.ModTime()
	}
	return f, nil
}

// WriteFile writes content to a tenant file path.
func (s *Service) WriteFile(ctx context.Context, tenantID, path string, content []byte) error {
	if err := validatePath(path); err != nil {
		return err
	}
	return s.repo.Write(ctx, tenantID, path, content)
}

// DeleteFile removes a file from the tenant sandbox.
func (s *Service) DeleteFile(ctx context.Context, tenantID, path string) error {
	if err := validatePath(path); err != nil {
		return err
	}
	return s.repo.Delete(ctx, tenantID, path)
}

// Tree returns a directory tree for the given path.
func (s *Service) Tree(ctx context.Context, tenantID, path string, depth int) (*fs.Directory, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}
	if depth <= 0 {
		depth = 3
	}
	return s.repo.Tree(ctx, tenantID, path, depth)
}

// Grep searches for a pattern in files under path.
func (s *Service) Grep(ctx context.Context, tenantID, path, pattern string) ([]*fs.GrepResult, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	// Walk files and grep
	rootDir := filepath.Join(s.baseDir, tenantID, filepath.Clean(path))
	var results []*fs.GrepResult

	err := filepath.Walk(rootDir, func(fpath string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, fpath)

		f, err := os.Open(fpath)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				results = append(results, &fs.GrepResult{
					Path:    rel,
					Line:    lineNum,
					Content: line,
					Match:   pattern,
				})
				if len(results) >= 500 {
					return filepath.SkipAll
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("grep walk: %w", err)
	}
	return results, nil
}

// validatePath ensures no path traversal.
func validatePath(path string) error {
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}
	return nil
}
