package version

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type GC struct {
	manager *Manager
	log     *log.Helper
}

func NewGC(manager *Manager, logger log.Logger) *GC {
	return &GC{
		manager: manager,
		log:     log.NewHelper(logger),
	}
}

func (g *GC) Compact(ctx context.Context, namespace string, olderThan time.Duration, keepLatest int) (int, error) {
	if g == nil || g.manager == nil {
		return 0, nil
	}
	if keepLatest < 0 {
		keepLatest = 0
	}

	versions, err := g.manager.ListVersions(ctx, namespace)
	if err != nil {
		return 0, err
	}
	if len(versions) <= keepLatest {
		return 0, nil
	}

	protected := make(map[string]struct{}, keepLatest)
	for i := 0; i < keepLatest && i < len(versions); i++ {
		protected[versions[i].ID] = struct{}{}
	}
	cutoff := time.Now().UTC().Add(-olderThan)
	var deleteIDs []string
	for _, item := range versions {
		if _, ok := protected[item.ID]; ok {
			continue
		}
		if olderThan > 0 && item.CreatedAt.After(cutoff) {
			continue
		}
		deleteIDs = append(deleteIDs, item.ID)
	}
	if len(deleteIDs) == 0 {
		return 0, nil
	}
	if err := g.manager.db.WithContext(ctx).
		Where("namespace = ? AND id IN ?", namespace, deleteIDs).
		Delete(&GraphVersion{}).Error; err != nil {
		return 0, err
	}
	g.log.Infof("version GC compacted namespace=%s deleted=%d", namespace, len(deleteIDs))
	return len(deleteIDs), nil
}
