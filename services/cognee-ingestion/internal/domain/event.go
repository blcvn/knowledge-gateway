package domain

import (
	"time"

	"github.com/google/uuid"
)

// DataIngestedEvent is a domain event raised when new data items are
// successfully ingested into a dataset and are ready for cognification.
type DataIngestedEvent struct {
	DatasetID uuid.UUID `json:"dataset_id"`
	TenantID  string    `json:"tenant_id"`
	ItemIDs   []string  `json:"item_ids"`
	Timestamp time.Time `json:"timestamp"`
}

// NewDataIngestedEvent constructs a DataIngestedEvent.
func NewDataIngestedEvent(datasetID uuid.UUID, tenantID string, itemIDs []string) DataIngestedEvent {
	return DataIngestedEvent{
		DatasetID: datasetID,
		TenantID:  tenantID,
		ItemIDs:   itemIDs,
		Timestamp: time.Now().UTC(),
	}
}
