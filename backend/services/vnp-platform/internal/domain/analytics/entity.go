// Package analytics defines domain entities for usage tracking.
//
// Absorbed from: sm-analytics
package analytics

import (
	"time"

	"github.com/google/uuid"
)

// UsageRecord tracks API usage per tenant.
type UsageRecord struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Engine    string    `json:"engine"`
	Endpoint  string    `json:"endpoint"`
	Tokens    int64     `json:"tokens"`     // LLM tokens consumed
	Requests  int64     `json:"requests"`   // API call count
	Period    string    `json:"period"`     // daily, monthly
	Date      time.Time `json:"date"`
}
