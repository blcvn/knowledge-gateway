package data

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

// JSONMap stores arbitrary key/value payloads in JSONB columns.
type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return []byte("null"), nil
	}
	b, err := json.Marshal(map[string]any(m))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *JSONMap) Scan(src any) error {
	if m == nil {
		return fmt.Errorf("JSONMap: scan on nil pointer")
	}
	if src == nil {
		*m = nil
		return nil
	}

	var raw []byte
	switch v := src.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("JSONMap: unsupported scan type %T", src)
	}
	if len(raw) == 0 {
		*m = JSONMap{}
		return nil
	}

	decoded := map[string]any{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	*m = JSONMap(decoded)
	return nil
}

// StringArr stores text[] payloads in Postgres while remaining DB-agnostic in tests.
type StringArr []string

func (a StringArr) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}

	parts := make([]string, 0, len(a))
	for _, s := range a {
		escaped := strings.ReplaceAll(s, `\\`, `\\\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\\"`)
		parts = append(parts, `"`+escaped+`"`)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func (a *StringArr) Scan(src any) error {
	if a == nil {
		return fmt.Errorf("StringArr: scan on nil pointer")
	}
	if src == nil {
		*a = nil
		return nil
	}

	var raw string
	switch v := src.(type) {
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("StringArr: unsupported scan type %T", src)
	}

	parsed, err := parsePGTextArray(raw)
	if err != nil {
		return err
	}
	*a = parsed
	return nil
}

func parsePGTextArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return []string{}, nil
	}
	if len(raw) < 2 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		return nil, fmt.Errorf("StringArr: invalid postgres text[] payload %q", raw)
	}

	inner := raw[1 : len(raw)-1]
	if inner == "" {
		return []string{}, nil
	}

	out := make([]string, 0, 4)
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(inner); i++ {
		ch := inner[i]
		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuotes = !inQuotes
			continue
		}
		if ch == ',' && !inQuotes {
			out = append(out, current.String())
			current.Reset()
			continue
		}
		current.WriteByte(ch)
	}

	if inQuotes || escaped {
		return nil, fmt.Errorf("StringArr: malformed postgres text[] payload %q", raw)
	}
	out = append(out, current.String())
	return out, nil
}

type KGEntity struct {
	EntityID       string    `gorm:"column:entity_id;type:text;primaryKey"`
	AppID          string    `gorm:"column:app_id;type:varchar(64);not null;index:idx_entity_tenant"`
	TenantID       string    `gorm:"column:tenant_id;type:varchar(64);not null;index:idx_entity_tenant"`
	EntityType     string    `gorm:"column:entity_type;type:varchar(128);not null;index:idx_entity_type"`
	Name           string    `gorm:"column:name;type:text;not null"`
	Properties     JSONMap   `gorm:"column:properties;type:jsonb"`
	Confidence     float64   `gorm:"column:confidence;type:float;default:1.0"`
	SourceFile     string    `gorm:"column:source_file;type:text"`
	ChunkID        string    `gorm:"column:chunk_id;type:varchar(128)"`
	SkillID        string    `gorm:"column:skill_id;type:varchar(128)"`
	VersionID      string    `gorm:"column:version_id;type:uuid"`
	ProvenanceType string    `gorm:"column:provenance_type;type:varchar(32)"`
	Domains        StringArr `gorm:"column:domains;type:text[]"`
	Aliases        StringArr `gorm:"column:aliases;type:text[]"`
	Version        int       `gorm:"column:version;type:int;default:1"`
	IsDeleted      bool      `gorm:"column:is_deleted;type:boolean;default:false"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KGEntity) TableName() string { return "kg_entities" }

type KGEdge struct {
	EdgeID       string    `gorm:"column:edge_id;type:text;primaryKey"`
	AppID        string    `gorm:"column:app_id;type:varchar(64);not null;index:idx_edge_tenant"`
	TenantID     string    `gorm:"column:tenant_id;type:varchar(64);not null;index:idx_edge_tenant"`
	FromEntityID string    `gorm:"column:from_entity_id;type:text;not null;index:idx_edge_from"`
	ToEntityID   string    `gorm:"column:to_entity_id;type:text;not null;index:idx_edge_to"`
	RelationType string    `gorm:"column:relation_type;type:varchar(128);not null;index:idx_edge_relation"`
	Properties   JSONMap   `gorm:"column:properties;type:jsonb"`
	Confidence   float64   `gorm:"column:confidence;type:float;default:1.0"`
	VersionID    string    `gorm:"column:version_id;type:uuid"`
	IsDeleted    bool      `gorm:"column:is_deleted;type:boolean;default:false"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (KGEdge) TableName() string { return "kg_edges" }

type KGSyncOutbox struct {
	ID        int64      `gorm:"column:id;primaryKey;autoIncrement"`
	Op        string     `gorm:"column:op;type:varchar(32);not null"`
	EntityID  *string    `gorm:"column:entity_id;type:text;index:idx_outbox_entity"`
	EdgeID    *string    `gorm:"column:edge_id;type:text"`
	TenantID  string     `gorm:"column:tenant_id;type:varchar(64);not null"`
	AppID     string     `gorm:"column:app_id;type:varchar(64);not null"`
	Payload   JSONMap    `gorm:"column:payload;type:jsonb;not null"`
	Status    string     `gorm:"column:status;type:varchar(16);default:'PENDING'"`
	Attempts  int        `gorm:"column:attempts;type:int;default:0"`
	LastError string     `gorm:"column:last_error;type:text"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	SyncedAt  *time.Time `gorm:"column:synced_at"`
}

func (KGSyncOutbox) TableName() string { return "kg_sync_outbox" }

const (
	OutboxOpUpsertEntity = "UPSERT_ENTITY"
	OutboxOpUpsertEdge   = "UPSERT_EDGE"
	OutboxOpDeleteEntity = "DELETE_ENTITY"
	OutboxOpDeleteEdge   = "DELETE_EDGE"

	OutboxStatusPending = "PENDING"
	OutboxStatusSyncing = "SYNCING"
	OutboxStatusDone    = "DONE"
	OutboxStatusFailed  = "FAILED"
)

// NormalizePropertiesForNeo4j converts JSON-like maps into Neo4j property-compatible values.
// Unsupported nested structures are serialized into compact JSON strings so writes do not fail.
func NormalizePropertiesForNeo4j(props JSONMap) (JSONMap, int) {
	if len(props) == 0 {
		return JSONMap{}, 0
	}
	out := make(JSONMap, len(props))
	changed := 0
	for key, value := range props {
		normalized, keep, valueChanged := normalizeNeo4jPropertyValue(value)
		if valueChanged {
			changed++
		}
		if !keep {
			continue
		}
		out[key] = normalized
	}
	return out, changed
}

func normalizeNeo4jPropertyValue(v any) (normalized any, keep bool, changed bool) {
	switch x := v.(type) {
	case nil:
		return nil, false, true
	case string, bool, int64:
		return x, true, false
	case int:
		return int64(x), true, true
	case int8:
		return int64(x), true, true
	case int16:
		return int64(x), true, true
	case int32:
		return int64(x), true, true
	case uint:
		if uint64(x) > math.MaxInt64 {
			return fmt.Sprintf("%d", x), true, true
		}
		return int64(x), true, true
	case uint8:
		return int64(x), true, true
	case uint16:
		return int64(x), true, true
	case uint32:
		return int64(x), true, true
	case uint64:
		if x > math.MaxInt64 {
			return fmt.Sprintf("%d", x), true, true
		}
		return int64(x), true, true
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return fmt.Sprintf("%v", x), true, true
		}
		return x, true, false
	case float32:
		f := float64(x)
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return fmt.Sprintf("%v", x), true, true
		}
		return f, true, true
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i, true, true
		}
		if f, err := x.Float64(); err == nil {
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return x.String(), true, true
			}
			return f, true, true
		}
		return x.String(), true, true
	case time.Time:
		return x.UTC().Format(time.RFC3339Nano), true, true
	case []string, []bool, []int64, []float64:
		return x, true, false
	case []int:
		out := make([]int64, 0, len(x))
		for _, n := range x {
			out = append(out, int64(n))
		}
		return out, true, true
	case []int32:
		out := make([]int64, 0, len(x))
		for _, n := range x {
			out = append(out, int64(n))
		}
		return out, true, true
	case []uint64:
		out := make([]int64, 0, len(x))
		for _, n := range x {
			if n > math.MaxInt64 {
				return toJSONString(x), true, true
			}
			out = append(out, int64(n))
		}
		return out, true, true
	case []float32:
		out := make([]float64, 0, len(x))
		for _, n := range x {
			f := float64(n)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return toJSONString(x), true, true
			}
			out = append(out, f)
		}
		return out, true, true
	case []byte:
		return string(x), true, true
	default:
		rv := reflect.ValueOf(v)
		if !rv.IsValid() {
			return nil, false, true
		}
		if rv.Kind() == reflect.Ptr {
			if rv.IsNil() {
				return nil, false, true
			}
			nv, keep, _ := normalizeNeo4jPropertyValue(rv.Elem().Interface())
			return nv, keep, true
		}
		switch rv.Kind() {
		case reflect.Map, reflect.Struct, reflect.Array, reflect.Slice:
			return toJSONString(v), true, true
		default:
			return fmt.Sprintf("%v", v), true, true
		}
	}
}

func toJSONString(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(raw)
}
