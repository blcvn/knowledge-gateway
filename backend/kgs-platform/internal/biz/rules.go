package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Rule represents a business rule that runs either on a schedule or triggered by events.
type Rule struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;index:idx_app_tenant_rule"`
	TenantID    string         `gorm:"type:varchar(50);not null;default:'default';index:idx_app_tenant_rule"`
	Name        string         `gorm:"type:varchar(100);not null"`
	Description string         `gorm:"type:text"`
	TriggerType string         `gorm:"type:varchar(20);not null"` // e.g., "SCHEDULED", "ON_WRITE"
	Cron        string         `gorm:"type:varchar(50)"`          // e.g., "0 0 * * *"
	CypherQuery string         `gorm:"type:text;not null"`        // Cypher query to execute
	Action      string         `gorm:"type:varchar(50)"`          // webhook, push_notification
	Payload     datatypes.JSON `gorm:"type:jsonb"`                // Action payload
	IsActive    bool           `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Rule) TableName() string {
	return "kgs_rules"
}

// RuleExecution tracks the history of rule executions.
type RuleExecution struct {
	ID        uint      `gorm:"primaryKey"`
	AppID     string    `gorm:"type:varchar(50);not null;index:idx_app_tenant_ex"`
	TenantID  string    `gorm:"type:varchar(50);not null;default:'default';index:idx_app_tenant_ex"`
	RuleID    uint      `gorm:"not null"`
	Status    string    `gorm:"type:varchar(20);not null"` // SUCCESS, FAILED
	Message   string    `gorm:"type:text"`
	StartedAt time.Time `gorm:"index"`
	EndedAt   time.Time
}

func (RuleExecution) TableName() string {
	return "kgs_rule_executions"
}

// Policy defines OPA Rego policies managed via the database.
type Policy struct {
	ID          uint   `gorm:"primaryKey"`
	AppID       string `gorm:"type:varchar(50);not null;index:idx_app_tenant_policy"`
	TenantID    string `gorm:"type:varchar(50);not null;default:'default';index:idx_app_tenant_policy"`
	Name        string `gorm:"type:varchar(100);not null"`
	Description string `gorm:"type:text"`
	RegoContent string `gorm:"type:text;not null"`
	IsActive    bool   `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Policy) TableName() string {
	return "kgs_policies"
}

// RulesRepo defines the persistence interface for Rules
type RulesRepo interface {
	CreateRule(ctx context.Context, rule *Rule) (*Rule, error)
	GetRule(ctx context.Context, id uint) (*Rule, error)
	ListRules(ctx context.Context, appID string) ([]*Rule, error)
}

type RulesUsecase struct {
	repo RulesRepo
	log  *log.Helper
}

func NewRulesUsecase(repo RulesRepo, logger log.Logger) *RulesUsecase {
	return &RulesUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *RulesUsecase) CreateRule(ctx context.Context, rule *Rule) (*Rule, error) {
	// TODO: any validation specific to rules goes here
	return uc.repo.CreateRule(ctx, rule)
}

func (uc *RulesUsecase) GetRule(ctx context.Context, id uint) (*Rule, error) {
	return uc.repo.GetRule(ctx, id)
}

func (uc *RulesUsecase) ListRules(ctx context.Context, appID string) ([]*Rule, error) {
	return uc.repo.ListRules(ctx, appID)
}
