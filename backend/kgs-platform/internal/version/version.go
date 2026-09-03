package version

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VersionManager interface {
	CreateDelta(ctx context.Context, namespace string, changes ChangeSet) (string, error)
	GetVersion(ctx context.Context, namespace, versionID string) (*GraphVersion, error)
	ListVersions(ctx context.Context, namespace string) ([]GraphVersion, error)
	DiffVersions(ctx context.Context, namespace, fromVersionID, toVersionID string) (*DiffResult, error)
	Rollback(ctx context.Context, namespace, targetVersionID, reason string) (string, error)
}

type Manager struct {
	db  *gorm.DB
	log *log.Helper
}

func NewManager(db *gorm.DB, logger log.Logger) *Manager {
	return &Manager{
		db:  db,
		log: log.NewHelper(logger),
	}
}

func (m *Manager) CreateDelta(ctx context.Context, namespace string, changes ChangeSet) (string, error) {
	parentID, err := m.latestVersionID(ctx, namespace)
	if err != nil {
		return "", err
	}
	versionID := uuid.NewString()
	record := GraphVersion{
		ID:               versionID,
		Namespace:        namespace,
		ParentID:         parentID,
		CommitMessage:    changes.CommitMessage,
		EntitiesAdded:    changes.EntitiesAdded,
		EntitiesModified: changes.EntitiesModified,
		EntitiesDeleted:  changes.EntitiesDeleted,
		EdgesAdded:       changes.EdgesAdded,
		EdgesModified:    changes.EdgesModified,
		EdgesDeleted:     changes.EdgesDeleted,
		CreatedAt:        time.Now().UTC(),
	}
	if err := m.db.WithContext(ctx).Create(&record).Error; err != nil {
		return "", err
	}
	return versionID, nil
}

func (m *Manager) GetVersion(ctx context.Context, namespace, versionID string) (*GraphVersion, error) {
	var out GraphVersion
	err := m.db.WithContext(ctx).
		Where("namespace = ? AND id = ?", namespace, versionID).
		Take(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (m *Manager) ListVersions(ctx context.Context, namespace string) ([]GraphVersion, error) {
	var out []GraphVersion
	if err := m.db.WithContext(ctx).
		Where("namespace = ?", namespace).
		Order("created_at DESC").
		Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (m *Manager) DiffVersions(ctx context.Context, namespace, fromVersionID, toVersionID string) (*DiffResult, error) {
	if fromVersionID == "" || toVersionID == "" {
		return nil, fmt.Errorf("from_version_id and to_version_id are required")
	}
	if fromVersionID == toVersionID {
		return &DiffResult{
			FromVersionID: fromVersionID,
			ToVersionID:   toVersionID,
		}, nil
	}
	chain, err := m.walkBackTo(ctx, namespace, fromVersionID, toVersionID)
	if err != nil {
		return nil, err
	}
	diff := aggregateDiff(fromVersionID, toVersionID, chain)
	return &diff, nil
}

func (m *Manager) Rollback(ctx context.Context, namespace, targetVersionID, reason string) (string, error) {
	target, err := m.GetVersion(ctx, namespace, targetVersionID)
	if err != nil {
		return "", err
	}
	commitMessage := "rollback to " + target.ID
	if reason != "" {
		commitMessage = commitMessage + ": " + reason
	}
	versionID, err := m.CreateDelta(ctx, namespace, ChangeSet{
		CommitMessage: commitMessage,
	})
	if err != nil {
		return "", err
	}
	if err := m.db.WithContext(ctx).
		Model(&GraphVersion{}).
		Where("id = ? AND namespace = ?", versionID, namespace).
		Update("rollback_from_id", target.ID).Error; err != nil {
		return "", err
	}
	return versionID, nil
}

func (m *Manager) latestVersionID(ctx context.Context, namespace string) (string, error) {
	var latest GraphVersion
	err := m.db.WithContext(ctx).
		Where("namespace = ?", namespace).
		Order("created_at DESC").
		Take(&latest).Error
	if err == nil {
		return latest.ID, nil
	}
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return "", err
}

func (m *Manager) walkBackTo(ctx context.Context, namespace, fromVersionID, toVersionID string) ([]GraphVersion, error) {
	currentID := toVersionID
	out := make([]GraphVersion, 0)
	guard := 0
	for currentID != "" {
		if guard > 10000 {
			return nil, fmt.Errorf("version chain too deep")
		}
		guard++
		item, err := m.GetVersion(ctx, namespace, currentID)
		if err != nil {
			return nil, err
		}
		if item.ID == fromVersionID {
			return out, nil
		}
		out = append(out, *item)
		currentID = item.ParentID
	}
	return nil, fmt.Errorf("version %s is not ancestor of %s", fromVersionID, toVersionID)
}

func aggregateDiff(fromVersionID, toVersionID string, chain []GraphVersion) DiffResult {
	out := DiffResult{
		FromVersionID: fromVersionID,
		ToVersionID:   toVersionID,
	}
	for _, version := range chain {
		out.EntitiesAdded += version.EntitiesAdded
		out.EntitiesModified += version.EntitiesModified
		out.EntitiesDeleted += version.EntitiesDeleted
		out.EdgesAdded += version.EdgesAdded
		out.EdgesModified += version.EdgesModified
		out.EdgesDeleted += version.EdgesDeleted
	}
	return out
}

var _ VersionManager = (*Manager)(nil)
