package dto

import "vnp-memory/services/ov-fs/internal/domain/model"

type TieredAbstracts struct {
	L0 string
	L1 string
}

type ReadFileRequest struct {
	AccountID    string
	Path         string
	ContextLevel model.ContextLevel
}

type ReadFileResponse struct {
	Content      []byte
	Metadata     *model.FileMetadata
	ContextLevel model.ContextLevel
}

type WriteFileRequest struct {
	AccountID        string
	UserID           string
	Path             string
	Content          []byte
	CreateParents    bool
	ContextAbstracts *TieredAbstracts
}

type WriteFileResponse struct {
	Path      string
	SizeBytes int64
	Encrypted bool
}

type DeleteFileRequest struct {
	AccountID string
	Path      string
	Recursive bool
}

type MkDirRequest struct {
	AccountID     string
	Path          string
	CreateParents bool
}

type ListDirRequest struct {
	AccountID       string
	Path            string
	Recursive       bool
	IncludeMetadata bool
}

type ListDirResponse struct {
	Entries []*model.DirEntry
}

type TreeRequest struct {
	AccountID        string
	Root             string
	MaxDepth         int32
	IncludeAbstracts bool
}

type TreeResponse struct {
	Root *model.TreeNode
}

type GrepRequest struct {
	AccountID       string
	Pattern         string
	Path            string
	CaseInsensitive bool
	MaxResults      int32
}

type GrepMatch struct {
	Path       string
	LineNumber int
	Content    string
	Score      float64
}

type GrepResponse struct {
	Matches []*GrepMatch
}

type GlobRequest struct {
	AccountID string
	Pattern   string
	Root      string
}

type GlobResponse struct {
	Paths []string
}

type MoveRequest struct {
	AccountID   string
	Source      string
	Destination string
	Overwrite   bool
}

type GetRelationsRequest struct {
	AccountID    string
	Path         string
	RelationType *model.RelationType
}

type RelationsResponse struct {
	Relations []*model.FileRelation
}

type AddRelationRequest struct {
	AccountID    string
	SourcePath   string
	TargetPath   string
	RelationType model.RelationType
	Metadata     map[string]interface{}
}
