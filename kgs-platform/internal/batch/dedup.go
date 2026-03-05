package batch

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"kgs-platform/internal/data"
)

const semanticThreshold = 0.95

type ExactDeduper struct{}

func NewExactDeduper() *ExactDeduper {
	return &ExactDeduper{}
}

func (d *ExactDeduper) Dedup(ctx context.Context, appID, tenantID string, entities []Entity) ([]Entity, int, error) {
	_ = ctx
	_ = appID
	_ = tenantID
	seen := make(map[string]struct{}, len(entities))
	unique := make([]Entity, 0, len(entities))
	skipped := 0

	for _, entity := range entities {
		keyBytes, err := json.Marshal(entity.Properties)
		if err != nil {
			return nil, 0, err
		}
		key := entity.Label + "|" + string(keyBytes)
		if _, ok := seen[key]; ok {
			skipped++
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, entity)
	}
	return unique, skipped, nil
}

type SemanticDeduper struct {
	exact  *ExactDeduper
	qdrant *data.QdrantClient
}

func NewSemanticDeduper(qdrant *data.QdrantClient) *SemanticDeduper {
	return &SemanticDeduper{
		exact:  NewExactDeduper(),
		qdrant: qdrant,
	}
}

func (d *SemanticDeduper) Dedup(ctx context.Context, appID, tenantID string, entities []Entity) ([]Entity, int, error) {
	unique, skipped, err := d.exact.Dedup(ctx, appID, tenantID, entities)
	if err != nil || d.qdrant == nil || appID == "" {
		return unique, skipped, err
	}

	collection := buildCollectionName(appID)
	final := make([]Entity, 0, len(unique))
	for _, entity := range unique {
		text := entityToText(entity)
		if text == "" {
			final = append(final, entity)
			continue
		}
		vector := embedDeterministic(text, 1536)
		if err := d.qdrant.EnsureCollection(ctx, collection, len(vector)); err != nil {
			return nil, 0, err
		}
		hits, err := d.qdrant.SearchVectors(ctx, collection, vector, 1, semanticThreshold)
		if err != nil {
			return nil, 0, err
		}
		if len(hits) > 0 {
			skipped++
			continue
		}
		final = append(final, entity)
	}
	return final, skipped, nil
}

func entityToText(entity Entity) string {
	parts := []string{entity.Label}
	if entity.Properties != nil {
		for _, key := range []string{"name", "title", "content", "description"} {
			if value, ok := entity.Properties[key].(string); ok && strings.TrimSpace(value) != "" {
				parts = append(parts, value)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func embedDeterministic(text string, size int) []float32 {
	if size <= 0 {
		size = 1536
	}
	out := make([]float32, size)
	for i := 0; i < size; i++ {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", text, i)))
		raw := binary.BigEndian.Uint32(hash[:4])
		out[i] = float32(raw%10000)/5000 - 1
	}
	normalize(out)
	return out
}

func normalize(vector []float32) {
	norm := 0.0
	for _, value := range vector {
		norm += float64(value * value)
	}
	if norm == 0 {
		return
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] = float32(float64(vector[i]) / norm)
	}
}

var sanitizePattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

func buildCollectionName(appID string) string {
	app := strings.ToLower(strings.TrimSpace(appID))
	if app == "" {
		app = "default"
	}
	app = sanitizePattern.ReplaceAllString(app, "_")
	app = strings.Trim(app, "_")
	if app == "" {
		app = "default"
	}
	return "kgs-vectors-" + app
}
