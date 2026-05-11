package fs

import (
\t"context"
\t"errors"
\t"fmt"
\t"time"

\t"vnp-memory/services/ov-fs/internal/domain/model"
\t"vnp-memory/services/ov-fs/internal/domain/model/lock"
)

// FileRepository defines the data access methods for FS nodes.
type FileRepository interface {
\tSave(ctx context.Context, node *model.FSNode) error
\tFindByID(ctx context.Context, id, tenantID string) (*model.FSNode, error)
\tListByParent(ctx context.Context, parentID, tenantID string) ([]*model.FSNode, error)
}

// FSUsecase orchestrates file system operations.
type FSUsecase struct {
\trepo        FileRepository
\tlockManager lock.PathLockManager
}

// NewFSUsecase creates a new FSUsecase instance.
func NewFSUsecase(repo FileRepository, lockManager lock.PathLockManager) *FSUsecase {
\treturn &FSUsecase{
\trepo:        repo,
\tlockManager: lockManager,
\t}
}

// CreateDirectory creates a new folder in the virtual file system.
func (uc *FSUsecase) CreateDirectory(ctx context.Context, tenantID, parentID, name, ownerID string) (*model.FSNode, error) {
\tif name == "" {
\t\treturn nil, errors.New("directory name cannot be empty")
\t}
\tif tenantID == "" {
\t\treturn nil, errors.New("tenant ID is required")
\t}

\t// 1. Acquire Distributed Lock for the parent path to prevent duplicate folder creation
\t// Ideally the path string would be the full resolved path. Here we use parentID + "/" + name as the key.
\tlockPath := fmt.Sprintf("%s/%s", parentID, name)
\tpLock, err := uc.lockManager.Acquire(ctx, tenantID, lockPath, ownerID, 5*time.Second)
\tif err != nil {
\t\treturn nil, fmt.Errorf("failed to acquire lock for directory creation: %w", err)
\t}
\tdefer func() {
\t\t_ = uc.lockManager.Release(context.Background(), pLock)
\t}()

\t// 2. Build the domain model
\tnode := &model.FSNode{
\t\tID:        generateID("dir_"),
\t\tTenantID:  tenantID,
\t\tParentID:  parentID,
\t\tName:      name,
\t\tType:      model.FileTypeDirectory,
\t\tSize:      0,
\t\tVersion:   1,
\t\tCreatedAt: time.Now(),
\t\tUpdatedAt: time.Now(),
\t}

\t// 3. Save via Repository
\tif err := uc.repo.Save(ctx, node); err != nil {
\t\treturn nil, fmt.Errorf("failed to save directory metadata: %w", err)
\t}

\treturn node, nil
}

// RenameNode securely renames a file or directory.
func (uc *FSUsecase) RenameNode(ctx context.Context, tenantID, nodeID, newName, ownerID string) (*model.FSNode, error) {
\tif newName == "" {
\t\treturn nil, errors.New("new name cannot be empty")
\t}

\t// 1. Acquire lock on the node itself
\tpLock, err := uc.lockManager.Acquire(ctx, tenantID, nodeID, ownerID, 5*time.Second)
\tif err != nil {
\t\treturn nil, fmt.Errorf("node is currently locked: %w", err)
\t}
\tdefer func() {
\t\t_ = uc.lockManager.Release(context.Background(), pLock)
\t}()

\t// 2. Retrieve existing node
\tnode, err := uc.repo.FindByID(ctx, nodeID, tenantID)
\tif err != nil {
\t\treturn nil, fmt.Errorf("failed to retrieve node: %w", err)
\t}
\tif node == nil {
\t\treturn nil, errors.New("node not found")
\t}

\t// 3. Apply Domain Logic (Rename increments version)
\tnode.Rename(newName)

\t// 4. Save via Repository (Enforces OCC)
\tif err := uc.repo.Save(ctx, node); err != nil {
\t\treturn nil, fmt.Errorf("failed to save renamed node (possible concurrent modification): %w", err)
\t}

\treturn node, nil
}

// generateID is a mock generator. 
// In production, use ULID or UUIDv7 for sortable primary keys.
func generateID(prefix string) string {
\treturn prefix + fmt.Sprintf("%d", time.Now().UnixNano())
}
