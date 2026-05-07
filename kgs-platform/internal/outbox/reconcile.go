package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"
)

type ReconcileJob struct {
	db     *gorm.DB
	neo4j  neo4j.DriverWithContext
	qdrant *data.QdrantClient
	log    *log.Helper
}

func NewReconcileJob(db *gorm.DB, driver neo4j.DriverWithContext, qdrant *data.QdrantClient, logger log.Logger) *ReconcileJob {
	return &ReconcileJob{
		db:     db,
		neo4j:  driver,
		qdrant: qdrant,
		log:    log.NewHelper(logger),
	}
}

func (r *ReconcileJob) Run(ctx context.Context) error {
	started := time.Now()
	if r == nil || r.db == nil {
		return nil
	}
	r.log.Infof("[KGS][ReconcileJob] start")
	entities, err := r.listPostgresEntities(ctx)
	if err != nil {
		r.log.Errorf("[KGS][ReconcileJob] list postgres entities failed err=%v", err)
		return err
	}

	entityByID := make(map[string]data.KGEntity, len(entities))
	pgVersions := make(map[string]int, len(entities))
	for _, entity := range entities {
		entityByID[entity.EntityID] = entity
		pgVersions[entity.EntityID] = entity.Version
	}

	staleSet := map[string]struct{}{}
	if r.neo4j != nil {
		neo4jVersions, err := r.listNeo4jEntityVersions(ctx)
		if err != nil {
			r.log.Errorf("[KGS][ReconcileJob] list neo4j versions failed err=%v", err)
			return err
		}
		for _, id := range diffStaleEntities(pgVersions, neo4jVersions) {
			staleSet[id] = struct{}{}
		}
	}

	if r.qdrant != nil {
		missingIDs, err := r.listMissingQdrantEntityIDs(ctx, entities)
		if err != nil {
			r.log.Errorf("[KGS][ReconcileJob] list missing qdrant ids failed err=%v", err)
			return err
		}
		for id := range missingIDs {
			staleSet[id] = struct{}{}
		}
	}

	observability.SetReconcileStale(len(staleSet))
	r.log.Infof("[KGS][ReconcileJob] stale entities=%d scanned_entities=%d", len(staleSet), len(entities))
	if len(staleSet) == 0 {
		r.log.Infof("[KGS][ReconcileJob] done no stale entities duration=%s", time.Since(started))
		return nil
	}

	staleIDs := make([]string, 0, len(staleSet))
	for entityID := range staleSet {
		staleIDs = append(staleIDs, entityID)
	}
	sort.Strings(staleIDs)

	for _, entityID := range staleIDs {
		entity, ok := entityByID[entityID]
		if !ok {
			continue
		}
		payload, err := entityToPayload(&entity)
		if err != nil {
			continue
		}
		id := entity.EntityID
		rec := data.KGSyncOutbox{
			Op:       data.OutboxOpUpsertEntity,
			EntityID: &id,
			TenantID: entity.TenantID,
			AppID:    entity.AppID,
			Payload:  payload,
			Status:   data.OutboxStatusPending,
		}
		if err := r.db.WithContext(ctx).Create(&rec).Error; err != nil {
			r.log.Warnf("reconcile enqueue failed for entity %s: %v", entity.EntityID, err)
			continue
		}
		r.log.Infof("[KGS][ReconcileJob] enqueued outbox for stale entity app_id=%s tenant_id=%s entity_id=%s op=%s",
			entity.AppID, entity.TenantID, entity.EntityID, rec.Op)
	}
	r.log.Infof("[KGS][ReconcileJob] done stale_entities=%d duration=%s", len(staleIDs), time.Since(started))
	return nil
}

func (r *ReconcileJob) listPostgresEntities(ctx context.Context) ([]data.KGEntity, error) {
	rows := make([]data.KGEntity, 0)
	if err := r.db.WithContext(ctx).Model(&data.KGEntity{}).
		Where("is_deleted = FALSE").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ReconcileJob) listNeo4jEntityVersions(ctx context.Context) (map[string]int, error) {
	session := r.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result := make(map[string]int)
	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (e:Entity)
			WHERE e.isDeleted IS NULL OR e.isDeleted = false
			RETURN e.id AS entity_id, coalesce(e.version, 0) AS version
		`, nil)
		if err != nil {
			return nil, err
		}
		for res.Next(ctx) {
			row := res.Record().AsMap()
			entityID := fmt.Sprint(row["entity_id"])
			if entityID == "" || entityID == "<nil>" {
				continue
			}
			result[entityID] = toIntAny(row["version"])
		}
		return nil, res.Err()
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func diffStaleEntities(pgVersions, neo4jVersions map[string]int) []string {
	stale := make([]string, 0)
	for entityID, pgVersion := range pgVersions {
		if neoVersion, ok := neo4jVersions[entityID]; !ok || neoVersion < pgVersion {
			stale = append(stale, entityID)
		}
	}
	return stale
}

func (r *ReconcileJob) listMissingQdrantEntityIDs(ctx context.Context, entities []data.KGEntity) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	if r.qdrant == nil || len(entities) == 0 {
		return out, nil
	}

	type group struct {
		pointToEntity map[string]string
		points        []string
	}
	byCollection := map[string]*group{}
	for _, entity := range entities {
		collection := qdrantCollection(entity.AppID)
		pointID := qdrantPointID(entity.AppID, entity.TenantID, entity.EntityID)

		item := byCollection[collection]
		if item == nil {
			item = &group{
				pointToEntity: map[string]string{},
				points:        make([]string, 0),
			}
			byCollection[collection] = item
		}
		item.pointToEntity[pointID] = entity.EntityID
		item.points = append(item.points, pointID)
	}

	const chunkSize = 256
	for collection, item := range byCollection {
		for _, chunk := range chunkStrings(item.points, chunkSize) {
			found, err := r.qdrant.ExistingPointIDs(ctx, collection, chunk)
			if err != nil {
				return nil, err
			}
			for _, pointID := range chunk {
				if _, ok := found[pointID]; ok {
					continue
				}
				entityID := item.pointToEntity[pointID]
				if entityID != "" {
					out[entityID] = struct{}{}
				}
			}
		}
	}
	return out, nil
}

func chunkStrings(items []string, chunkSize int) [][]string {
	if len(items) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = len(items)
	}
	chunks := make([][]string, 0, (len(items)+chunkSize-1)/chunkSize)
	for start := 0; start < len(items); start += chunkSize {
		end := start + chunkSize
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[start:end])
	}
	return chunks
}

func entityToPayload(entity *data.KGEntity) (data.JSONMap, error) {
	raw, err := json.Marshal(entity)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return data.JSONMap(out), nil
}

func toIntAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}
