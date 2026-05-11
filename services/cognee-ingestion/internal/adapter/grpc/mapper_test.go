package grpc

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vnp-community/vnp-memory/services/cognee-ingestion/internal/domain"
)

func TestDatasetToProto(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.Nil(t, DatasetToProto(nil))
	})

	t.Run("Success", func(t *testing.T) {
		id := uuid.New()
		now := time.Now()
		ds := &domain.Dataset{
			ID:             id,
			TenantID:       "tenant-1",
			Name:           "ds1",
			Description:    "desc1",
			Status:         domain.DatasetReady,
			FileCount:      10,
			TotalSizeBytes: 1024,
			Metadata:       map[string]string{"key": "value"},
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		proto := DatasetToProto(ds)
		assert.NotNil(t, proto)
		assert.Equal(t, id.String(), proto.Id)
		assert.Equal(t, "tenant-1", proto.TenantId)
		assert.Equal(t, "ds1", proto.Name)
		assert.Equal(t, "desc1", proto.Description)
		assert.Equal(t, string(domain.DatasetReady), proto.Status)
		assert.Equal(t, int32(10), proto.FileCount)
		assert.Equal(t, int64(1024), proto.TotalSizeBytes)
		assert.Equal(t, "value", proto.Metadata["key"])
		assert.Equal(t, now.Unix(), proto.CreatedAt.Seconds)
	})
}
