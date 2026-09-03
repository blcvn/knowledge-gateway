package batch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdlog "log"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PGWriter struct {
	db *gorm.DB
}

func NewPGWriter(db *gorm.DB) *PGWriter {
	return &PGWriter{db: db}
}

func (w *PGWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
	if w == nil || w.db == nil || len(entities) == 0 {
		return 0, nil
	}
	started := time.Now()
	stdlog.Printf("[KGS][PGWriter] BulkCreate start app_id=%s tenant_id=%s entities=%d", appID, tenantID, len(entities))

	created := 0
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, in := range entities {
			kgEntity := toKGEntity(in, appID, tenantID)
			op, err := data.UpsertEntityTx(tx, kgEntity)
			if err != nil {
				if errors.Is(err, data.ErrAlreadyExists) || errors.Is(err, data.ErrVersionConflict) || errors.Is(err, data.ErrNameConflict) {
					stdlog.Printf("[KGS][PGWriter] BulkCreate entity conflict skip app_id=%s tenant_id=%s entity_id=%s label=%s err=%v",
						appID, tenantID, kgEntity.EntityID, kgEntity.EntityType, err)
					continue
				}
				stdlog.Printf("[KGS][PGWriter] BulkCreate upsert entity failed app_id=%s tenant_id=%s entity_id=%s label=%s err=%v",
					appID, tenantID, kgEntity.EntityID, kgEntity.EntityType, err)
				return err
			}
			if op == data.UpsertOpCreated {
				created++
			}
			stdlog.Printf("[KGS][PGWriter] BulkCreate upsert entity done app_id=%s tenant_id=%s entity_id=%s label=%s op=%v",
				appID, tenantID, kgEntity.EntityID, kgEntity.EntityType, op)

			rec, err := toOutboxRecord(data.OutboxOpUpsertEntity, kgEntity)
			if err != nil {
				stdlog.Printf("[KGS][PGWriter] BulkCreate build outbox failed app_id=%s tenant_id=%s entity_id=%s err=%v",
					appID, tenantID, kgEntity.EntityID, err)
				return err
			}
			if err := data.EnqueueOutboxTx(tx, rec); err != nil {
				stdlog.Printf("[KGS][PGWriter] BulkCreate enqueue outbox failed app_id=%s tenant_id=%s entity_id=%s op=%s err=%v",
					appID, tenantID, kgEntity.EntityID, rec.Op, err)
				return err
			}
			stdlog.Printf("[KGS][PGWriter] BulkCreate enqueue outbox done app_id=%s tenant_id=%s entity_id=%s op=%s",
				appID, tenantID, kgEntity.EntityID, rec.Op)
		}
		return nil
	})
	if err != nil {
		stdlog.Printf("[KGS][PGWriter] BulkCreate failed app_id=%s tenant_id=%s entities=%d created=%d err=%v duration=%s",
			appID, tenantID, len(entities), created, err, time.Since(started))
		return 0, err
	}
	stdlog.Printf("[KGS][PGWriter] BulkCreate done app_id=%s tenant_id=%s entities=%d created=%d duration=%s",
		appID, tenantID, len(entities), created, time.Since(started))
	return created, nil
}

func toKGEntity(e Entity, appID, tenantID string) data.KGEntity {
	props := cloneProperties(e.Properties)
	entityID := toString(props["id"])
	if strings.TrimSpace(entityID) == "" {
		entityID = uuid.NewString()
		props["id"] = entityID
	}

	name := toString(props["name"])
	if strings.TrimSpace(name) == "" {
		name = entityID
	}

	return data.KGEntity{
		EntityID:       entityID,
		AppID:          appID,
		TenantID:       tenantID,
		EntityType:     e.Label,
		Name:           name,
		Properties:     data.JSONMap(props),
		Confidence:     toFloat(props["confidence"], 1.0),
		SourceFile:     firstNonEmpty(toString(props["source_file"]), toString(props["sourceFile"])),
		ChunkID:        firstNonEmpty(toString(props["chunk_id"]), toString(props["chunkId"])),
		SkillID:        firstNonEmpty(toString(props["skill_id"]), toString(props["skillId"])),
		VersionID:      firstNonEmpty(toString(props["version_id"]), toString(props["versionId"])),
		ProvenanceType: firstNonEmpty(toString(props["provenance_type"]), toString(props["provenanceType"])),
		Domains:        toStringArr(props["domains"]),
		Aliases:        toStringArr(props["aliases"]),
		Version:        toInt(props["version"]),
	}
}

func toOutboxRecord(op string, entity data.KGEntity) (data.KGSyncOutbox, error) {
	payload, err := structToJSONMap(entity)
	if err != nil {
		return data.KGSyncOutbox{}, err
	}
	entityID := entity.EntityID
	return data.KGSyncOutbox{
		Op:       op,
		EntityID: &entityID,
		TenantID: entity.TenantID,
		AppID:    entity.AppID,
		Payload:  payload,
		Status:   data.OutboxStatusPending,
	}, nil
}

func structToJSONMap(v any) (data.JSONMap, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return data.JSONMap(out), nil
}

func cloneProperties(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func toFloat(v any, def float64) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	default:
		return def
	}
}

func toInt(v any) int {
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

func toStringArr(v any) data.StringArr {
	switch x := v.(type) {
	case []string:
		return data.StringArr(x)
	case data.StringArr:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s := toString(item)
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return data.StringArr(out)
	default:
		return data.StringArr{}
	}
}
