package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type JobModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey"`
	TenantID  string         `gorm:"type:varchar(64);index"`
	DatasetID uuid.UUID      `gorm:"type:uuid;index"`
	Status    string         `gorm:"type:varchar(32)"`
	Stage     string         `gorm:"type:varchar(64)"`
	Progress  float64        
	Error     string         `gorm:"type:text"`
	Config    string         `gorm:"type:jsonb"`
	Metrics   string         `gorm:"type:jsonb"`
	StartedAt time.Time      
	EndedAt   *time.Time     
}

func (JobModel) TableName() string {
	return "cognify_jobs"
}

type jobRepository struct {
	db *gorm.DB
}

// NewJobRepository returns a new instance of port.JobRepository backed by Postgres/GORM.
func NewJobRepository(db *gorm.DB) port.JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) Create(ctx context.Context, job *domain.CognifyJob) error {
	model, err := toJobModel(job)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *jobRepository) GetByID(ctx context.Context, tenantID string, id uuid.UUID) (*domain.CognifyJob, error) {
	var model JobModel
	if err := r.db.WithContext(ctx).Where("id = ? AND tenant_id = ?", id, tenantID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrJobNotFound
		}
		return nil, err
	}
	return toDomainJob(&model)
}

func (r *jobRepository) Update(ctx context.Context, job *domain.CognifyJob) error {
	model, err := toJobModel(job)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Save(model).Error
}

func (r *jobRepository) ListByDataset(ctx context.Context, tenantID string, datasetID uuid.UUID) ([]*domain.CognifyJob, error) {
	var models []JobModel
	if err := r.db.WithContext(ctx).Where("dataset_id = ? AND tenant_id = ?", datasetID, tenantID).Find(&models).Error; err != nil {
		return nil, err
	}
	
	jobs := make([]*domain.CognifyJob, len(models))
	for i, m := range models {
		job, err := toDomainJob(&m)
		if err != nil {
			return nil, err
		}
		jobs[i] = job
	}
	return jobs, nil
}

func toJobModel(job *domain.CognifyJob) (*JobModel, error) {
	configBytes, err := json.Marshal(job.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	
	metricsBytes, err := json.Marshal(job.Metrics)
	if err != nil {
		return nil, fmt.Errorf("marshal metrics: %w", err)
	}

	return &JobModel{
		ID:        job.ID,
		TenantID:  job.TenantID,
		DatasetID: job.DatasetID,
		Status:    string(job.Status),
		Stage:     string(job.Stage),
		Progress:  job.Progress,
		Error:     job.Error,
		Config:    string(configBytes),
		Metrics:   string(metricsBytes),
		StartedAt: func() time.Time { if job.StartedAt != nil { return *job.StartedAt }; return time.Time{} }(),
		EndedAt:   job.EndedAt,
	}, nil
}

func toDomainJob(model *JobModel) (*domain.CognifyJob, error) {
	var config domain.CognifyConfig
	if err := json.Unmarshal([]byte(model.Config), &config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	var metrics domain.PipelineMetrics
	if err := json.Unmarshal([]byte(model.Metrics), &metrics); err != nil {
		return nil, fmt.Errorf("unmarshal metrics: %w", err)
	}

	return &domain.CognifyJob{
		ID:        model.ID,
		TenantID:  model.TenantID,
		DatasetID: model.DatasetID,
		Status:    domain.JobStatus(model.Status),
		Stage:     domain.StageType(model.Stage),
		Progress:  model.Progress,
		Error:     model.Error,
		Config:    config,
		Metrics:   metrics,
		StartedAt: &model.StartedAt,
		EndedAt:   model.EndedAt,
	}, nil
}
