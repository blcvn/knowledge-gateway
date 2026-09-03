package model

import "time"

// FileType determines if a node is a directory or a file.
type FileType string

const (
\tFileTypeFile      FileType = "FILE"
\tFileTypeDirectory FileType = "DIRECTORY"
\tFileTypeSymlink   FileType = "SYMLINK"
)

// FSNode represents a file or folder in the OpenViking virtual file system.
type FSNode struct {
\tID             string    `json:"id"`
\tTenantID       string    `json:"tenant_id"`     // Multi-tenant isolation
\tParentID       string    `json:"parent_id"`     // Nullable for root
\tName           string    `json:"name"`          // Base name of the file/folder
\tType           FileType  `json:"type"`          // FILE or DIRECTORY
\tSize           int64     `json:"size"`          // File size in bytes
\tMimeType       string    `json:"mime_type"`     // E.g., application/pdf
\tChecksumSHA256 string    `json:"checksum"`      // Integrity hash
\tVersion        int32     `json:"version"`       // Optimistic concurrency control
\tCreatedAt      time.Time `json:"created_at"`
\tUpdatedAt      time.Time `json:"updated_at"`
}

// FullPath represents the resolved path of a node. Normally constructed by recursively joining Parents.
func (n *FSNode) IsRoot() bool {
\treturn n.ParentID == "" || n.ParentID == "root"
}

// Rename updates the node name and increments the version for concurrency control.
func (n *FSNode) Rename(newName string) {
\tn.Name = newName
\tn.Version++
\tn.UpdatedAt = time.Now()
}
