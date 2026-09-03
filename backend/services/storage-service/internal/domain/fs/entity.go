// Package fs defines domain entities for the filesystem sub-domain.
//
// Part of storage-service (MERGE-P1-T4: absorbs ov-fs, ov-crypto, ov-resource, ov-session)
package fs

import "time"

// File represents a file in the sandboxed tenant filesystem.
type File struct {
	Path      string    `json:"path"`
	Content   []byte    `json:"content,omitempty"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	Encrypted bool      `json:"encrypted"`
	ModTime   time.Time `json:"mod_time"`
}

// Directory represents a directory node.
type Directory struct {
	Path     string     `json:"path"`
	Children []TreeNode `json:"children"`
}

// TreeNode is a single node in the directory tree.
type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Size     int64      `json:"size,omitempty"`
	Children []TreeNode `json:"children,omitempty"`
}

// GrepResult is a single line match from a grep operation.
type GrepResult struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Content string `json:"content"`
	Match   string `json:"match"`
}
