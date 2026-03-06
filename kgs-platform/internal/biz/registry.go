package biz

import (
	"time"

	"gorm.io/gorm"
)

// App represents a client application registered in the KGS platform.
type App struct {
	AppID       string `gorm:"primaryKey;type:varchar(50)"`
	AppName     string `gorm:"type:varchar(200);not null"`
	Description string `gorm:"type:text"`
	Owner       string `gorm:"type:varchar(100);not null"`
	Status      string `gorm:"type:varchar(20);default:'ACTIVE'"` // ACTIVE, INACTIVE, SUSPENDED
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`

	APIKeys []APIKey `gorm:"foreignKey:AppID"`
	Quotas  []Quota  `gorm:"foreignKey:AppID"`
}

// APIKey represents an authentication key for an App.
type APIKey struct {
	KeyHash   string `gorm:"primaryKey;type:varchar(80)"` // SHA-256 hash of the key
	AppID     string `gorm:"type:varchar(50);not null;index"`
	KeyPrefix string `gorm:"type:varchar(10);not null"` // First few chars for identification
	Name      string `gorm:"type:varchar(100)"`
	Scopes    string `gorm:"type:varchar(500)"` // Comma-separated scopes (e.g., "read,write")
	IsRevoked bool   `gorm:"default:false"`
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Quota defines rate limits and resource limits for an App.
type Quota struct {
	ID        uint   `gorm:"primaryKey"`
	AppID     string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"`
	QuotaType string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"` // e.g., "requests_per_minute", "max_nodes"
	Limit     int64  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AuditLog tracks administrative actions.
type AuditLog struct {
	ID        uint      `gorm:"primaryKey"`
	AppID     string    `gorm:"type:varchar(50);index"`
	Action    string    `gorm:"type:varchar(100);not null"`
	Actor     string    `gorm:"type:varchar(100);not null"`
	Details   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"index"`
}
