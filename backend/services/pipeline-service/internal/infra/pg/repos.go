// Package pg implements PostgreSQL repositories for pipeline-service.
//
// Covers: JobRepository, WorkerRegistry, PRDRepository, OutlineRepository
// (MERGE-P3-T1)
package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domk "vnp-memory/services/pipeline-service/internal/domain/knowledge"
	domp "vnp-memory/services/pipeline-service/internal/domain/pipeline"
)

// ── JobRepository ──────────────────────────────────────────────────────────

// JobRepo implements port.JobRepository.
type JobRepo struct{ pool *pgxpool.Pool }

// NewJobRepo creates a JobRepo.
func NewJobRepo(pool *pgxpool.Pool) *JobRepo { return &JobRepo{pool: pool} }

func (r *JobRepo) Create(ctx context.Context, job *domp.Job) error {
	payloadJSON, _ := json.Marshal(job.Payload)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO pipeline_jobs (id, engine, type, status, payload, priority, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		job.ID, job.Engine, job.Type, job.Status, payloadJSON, job.Priority, job.CreatedAt,
	)
	return err
}

func (r *JobRepo) GetByID(ctx context.Context, id string) (*domp.Job, error) {
	job := &domp.Job{}
	var payloadJSON, resultJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, engine, type, status, payload, result, error, priority, created_at, started_at, completed_at
		 FROM pipeline_jobs WHERE id=$1`, id,
	).Scan(&job.ID, &job.Engine, &job.Type, &job.Status, &payloadJSON, &resultJSON,
		&job.Error, &job.Priority, &job.CreatedAt, &job.StartedAt, &job.CompletedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(payloadJSON, &job.Payload)
	_ = json.Unmarshal(resultJSON, &job.Result)
	return job, nil
}

func (r *JobRepo) GetStats(ctx context.Context, engine string) (domp.PipelineJobCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM pipeline_jobs WHERE engine=$1 GROUP BY status`, engine,
	)
	if err != nil {
		return domp.PipelineJobCount{}, err
	}
	defer rows.Close()
	stats := domp.PipelineJobCount{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		switch status {
		case "pending":
			stats.Pending += count
		case "running":
			stats.Running += count
		case "completed":
			stats.Completed += count
		case "failed":
			stats.Failed += count
		}
	}
	return stats, nil
}

func (r *JobRepo) ListByEngine(ctx context.Context, engine string, filter domp.JobFilter) ([]*domp.Job, int, error) {
	query := `SELECT id, engine, type, status, payload, error, priority, created_at FROM pipeline_jobs WHERE engine=$1`
	args := []any{engine}
	argN := 2
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status=$%d", argN)
		args = append(args, filter.Status)
		argN++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type=$%d", argN)
		args = append(args, filter.Type)
		argN++
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var jobs []*domp.Job
	for rows.Next() {
		j := &domp.Job{}
		var payloadJSON []byte
		if err := rows.Scan(&j.ID, &j.Engine, &j.Type, &j.Status, &payloadJSON, &j.Error, &j.Priority, &j.CreatedAt); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal(payloadJSON, &j.Payload)
		jobs = append(jobs, j)
	}
	return jobs, len(jobs), nil
}

func (r *JobRepo) UpdateStatus(ctx context.Context, id, status, errMsg string) error {
	now := time.Now()
	if status == "running" {
		_, err := r.pool.Exec(ctx,
			`UPDATE pipeline_jobs SET status=$1, started_at=$2 WHERE id=$3`,
			status, now, id,
		)
		return err
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE pipeline_jobs SET status=$1, error=$2, completed_at=$3 WHERE id=$4`,
		status, errMsg, now, id,
	)
	return err
}

// ── WorkerRegistry ─────────────────────────────────────────────────────────

// WorkerRepo implements port.WorkerRegistry using PostgreSQL.
type WorkerRepo struct{ pool *pgxpool.Pool }

// NewWorkerRepo creates a WorkerRepo.
func NewWorkerRepo(pool *pgxpool.Pool) *WorkerRepo { return &WorkerRepo{pool: pool} }

func (r *WorkerRepo) Register(ctx context.Context, w *domp.Worker) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO pipeline_workers (id, engine, status, last_seen)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (id) DO UPDATE SET status=$3, last_seen=$4`,
		w.ID, w.Engine, w.Status, time.Now(),
	)
	return err
}

func (r *WorkerRepo) Heartbeat(ctx context.Context, workerID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pipeline_workers SET last_seen=$1 WHERE id=$2`,
		time.Now(), workerID,
	)
	return err
}

func (r *WorkerRepo) ListByEngine(ctx context.Context, engine string) ([]*domp.Worker, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, engine, status, job_id, last_seen
		 FROM pipeline_workers WHERE engine=$1 AND last_seen > NOW() - INTERVAL '5 minutes'`,
		engine,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var workers []*domp.Worker
	for rows.Next() {
		w := &domp.Worker{}
		var jobID *string
		if err := rows.Scan(&w.ID, &w.Engine, &w.Status, &jobID, &w.LastSeen); err != nil {
			continue
		}
		if jobID != nil {
			w.JobID = *jobID
		}
		workers = append(workers, w)
	}
	return workers, nil
}

func (r *WorkerRepo) GetQueues(ctx context.Context) ([]*domp.Queue, error) {
	// Aggregate queue depths from job table
	rows, err := r.pool.Query(ctx,
		`SELECT engine, COUNT(*) FROM pipeline_jobs WHERE status='pending' GROUP BY engine`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	queueMap := make(map[string]*domp.Queue)
	for rows.Next() {
		var engine string
		var count int
		if err := rows.Scan(&engine, &count); err != nil {
			continue
		}
		queueMap[engine] = &domp.Queue{
			Name:   engine + "-queue",
			Engine: engine,
			Size:   count,
		}
	}
	queues := make([]*domp.Queue, 0, len(queueMap))
	for _, q := range queueMap {
		queues = append(queues, q)
	}
	return queues, nil
}

// ── PRDRepository ──────────────────────────────────────────────────────────

// PRDRepo implements port.PRDRepository.
type PRDRepo struct{ pool *pgxpool.Pool }

// NewPRDRepo creates a PRDRepo.
func NewPRDRepo(pool *pgxpool.Pool) *PRDRepo { return &PRDRepo{pool: pool} }

func (r *PRDRepo) Create(ctx context.Context, prd *domk.PRD) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO knowledge_prds (id, title, content, tags, status, tenant_id, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		prd.ID, prd.Title, prd.Content, prd.Tags, prd.Status, prd.TenantID, prd.CreatedAt,
	)
	return err
}

func (r *PRDRepo) GetByID(ctx context.Context, id string) (*domk.PRD, error) {
	prd := &domk.PRD{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, title, content, tags, status, tenant_id, created_at, updated_at
		 FROM knowledge_prds WHERE id=$1`, id,
	).Scan(&prd.ID, &prd.Title, &prd.Content, &prd.Tags, &prd.Status, &prd.TenantID, &prd.CreatedAt, &prd.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("prd not found: %s", id)
	}
	return prd, err
}

func (r *PRDRepo) List(ctx context.Context, tenantID string, limit, offset int) ([]*domk.PRD, int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, title, tags, status, created_at FROM knowledge_prds
		 WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var prds []*domk.PRD
	for rows.Next() {
		p := &domk.PRD{TenantID: tenantID}
		if err := rows.Scan(&p.ID, &p.Title, &p.Tags, &p.Status, &p.CreatedAt); err != nil {
			continue
		}
		prds = append(prds, p)
	}
	return prds, len(prds), nil
}

func (r *PRDRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE knowledge_prds SET status=$1, updated_at=NOW() WHERE id=$2`,
		status, id,
	)
	return err
}

// ── OutlineRepository ──────────────────────────────────────────────────────

// OutlineRepo implements port.OutlineRepository.
type OutlineRepo struct{ pool *pgxpool.Pool }

// NewOutlineRepo creates an OutlineRepo.
func NewOutlineRepo(pool *pgxpool.Pool) *OutlineRepo { return &OutlineRepo{pool: pool} }

func (r *OutlineRepo) Create(ctx context.Context, o *domk.Outline) error {
	sectionsJSON, _ := json.Marshal(o.Sections)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO knowledge_outlines (id, prd_id, sections, status)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (prd_id) DO UPDATE SET sections=$3, status=$4`,
		o.ID, o.PRDID, sectionsJSON, o.Status,
	)
	return err
}

func (r *OutlineRepo) GetByPRD(ctx context.Context, prdID string) (*domk.Outline, error) {
	o := &domk.Outline{}
	var sectionsJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, prd_id, sections, status FROM knowledge_outlines WHERE prd_id=$1`, prdID,
	).Scan(&o.ID, &o.PRDID, &sectionsJSON, &o.Status)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("outline not found for prd: %s", prdID)
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(sectionsJSON, &o.Sections)
	return o, nil
}

func (r *OutlineRepo) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE knowledge_outlines SET status=$1 WHERE id=$2`,
		status, id,
	)
	return err
}
