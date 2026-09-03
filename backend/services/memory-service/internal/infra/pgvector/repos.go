// Package pgvector implements persistence for memory-service.
//
// Covers: BlobRepository, ProfileRepository, EventRepository,
//         SMMemoryRepository, SMDocumentRepository
// (MERGE-P2-T3)
package pgvector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	pgv "github.com/pgvector/pgvector-go"

	"vnp-memory/services/memory-service/internal/domain/memobase"
	"vnp-memory/services/memory-service/internal/domain/sm"
)

// ── BlobRepository ─────────────────────────────────────────────────────────

// BlobRepo implements port.BlobRepository.
type BlobRepo struct{ pool *pgxpool.Pool }

// NewBlobRepo creates a BlobRepo.
func NewBlobRepo(pool *pgxpool.Pool) *BlobRepo { return &BlobRepo{pool: pool} }

func (r *BlobRepo) Create(ctx context.Context, b *memobase.Blob) error {
	var emb *pgv.Vector
	if len(b.Embedding) > 0 {
		v := pgv.NewVector(b.Embedding)
		emb = &v
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_blobs (id, user_id, tenant_id, type, content, metadata, embedding, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		b.ID, b.UserID, b.TenantID, b.Type, b.Content, pgxJSONB(b.Metadata), emb, b.CreatedAt,
	)
	return err
}

func (r *BlobRepo) List(ctx context.Context, userID, tenantID string, limit int) ([]*memobase.Blob, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, type, content, created_at
		 FROM memory_blobs WHERE user_id=$1 AND tenant_id=$2
		 ORDER BY created_at DESC LIMIT $3`, userID, tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blobs []*memobase.Blob
	for rows.Next() {
		b := &memobase.Blob{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.TenantID, &b.Type, &b.Content, &b.CreatedAt); err != nil {
			return nil, err
		}
		blobs = append(blobs, b)
	}
	return blobs, nil
}

func (r *BlobRepo) GetBufferSize(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM memory_blobs WHERE user_id=$1 AND created_at > NOW() - INTERVAL '1 hour'`,
		userID,
	).Scan(&count)
	return count, err
}

func (r *BlobRepo) SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*memobase.Blob, error) {
	v := pgv.NewVector(embedding)
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, tenant_id, type, content, created_at
		 FROM memory_blobs WHERE tenant_id=$1 AND embedding IS NOT NULL
		 ORDER BY embedding <=> $2 LIMIT $3`,
		tenantID, v, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var blobs []*memobase.Blob
	for rows.Next() {
		b := &memobase.Blob{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.TenantID, &b.Type, &b.Content, &b.CreatedAt); err != nil {
			return nil, err
		}
		blobs = append(blobs, b)
	}
	return blobs, nil
}

// ── ProfileRepository ──────────────────────────────────────────────────────

// ProfileRepo implements port.ProfileRepository.
type ProfileRepo struct{ pool *pgxpool.Pool }

// NewProfileRepo creates a ProfileRepo.
func NewProfileRepo(pool *pgxpool.Pool) *ProfileRepo { return &ProfileRepo{pool: pool} }

func (r *ProfileRepo) Upsert(ctx context.Context, userID, tenantID string, p *memobase.Profile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_profiles (user_id, tenant_id, key, value, category, score, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 ON CONFLICT (user_id, tenant_id, key) DO UPDATE SET value=$4, category=$5, score=$6, updated_at=$7`,
		userID, tenantID, p.Key, p.Value, p.Category, p.Score, time.Now(),
	)
	return err
}

func (r *ProfileRepo) GetByUser(ctx context.Context, userID, tenantID string) ([]*memobase.Profile, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT key, value, category, score, updated_at
		 FROM memory_profiles WHERE user_id=$1 AND tenant_id=$2
		 ORDER BY score DESC`, userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []*memobase.Profile
	for rows.Next() {
		p := &memobase.Profile{}
		if err := rows.Scan(&p.Key, &p.Value, &p.Category, &p.Score, &p.UpdatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// ── EventRepository (memobase) ─────────────────────────────────────────────

// EventRepo implements port.EventRepository.
type EventRepo struct{ pool *pgxpool.Pool }

// NewEventRepo creates an EventRepo.
func NewEventRepo(pool *pgxpool.Pool) *EventRepo { return &EventRepo{pool: pool} }

func (r *EventRepo) Create(ctx context.Context, evt *memobase.Event) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO memory_blobs (id, user_id, tenant_id, type, content, created_at)
		 VALUES ($1,$2,'','event',$3,$4)`,
		evt.ID, evt.UserID, evt.Content, evt.CreatedAt,
	)
	return err
}

func (r *EventRepo) GetByUser(ctx context.Context, userID string, limit int) ([]*memobase.Event, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, type, content, created_at
		 FROM memory_blobs WHERE user_id=$1 AND type='event'
		 ORDER BY created_at DESC LIMIT $2`, userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []*memobase.Event
	for rows.Next() {
		e := &memobase.Event{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.EventType, &e.Content, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// ── SMMemoryRepository ─────────────────────────────────────────────────────

// SMMemoryRepo implements port.SMMemoryRepository.
type SMMemoryRepo struct{ pool *pgxpool.Pool }

// NewSMMemoryRepo creates an SMMemoryRepo.
func NewSMMemoryRepo(pool *pgxpool.Pool) *SMMemoryRepo { return &SMMemoryRepo{pool: pool} }

func (r *SMMemoryRepo) Create(ctx context.Context, m *sm.SMMemory) error {
	var emb *pgv.Vector
	if len(m.Embedding) > 0 {
		v := pgv.NewVector(m.Embedding)
		emb = &v
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sm_memories (id, tenant_id, content, tags, metadata, embedding, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.TenantID, m.Content, m.Tags, pgxJSONB(m.Metadata), emb, m.CreatedAt,
	)
	return err
}

func (r *SMMemoryRepo) List(ctx context.Context, tenantID string, limit int) ([]*sm.SMMemory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, content, tags, created_at
		 FROM sm_memories WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2`,
		tenantID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSMMemories(rows)
}

func (r *SMMemoryRepo) SemanticSearch(ctx context.Context, tenantID string, embedding []float32, limit int) ([]*sm.SMMemory, error) {
	v := pgv.NewVector(embedding)
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, content, tags, created_at
		 FROM sm_memories WHERE tenant_id=$1 AND embedding IS NOT NULL
		 ORDER BY embedding <=> $2 LIMIT $3`,
		tenantID, v, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSMMemories(rows)
}

func scanSMMemories(rows interface{ Next() bool; Scan(...any) error }) ([]*sm.SMMemory, error) {
	var mems []*sm.SMMemory
	for rows.Next() {
		m := &sm.SMMemory{}
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Content, &m.Tags, &m.CreatedAt); err != nil {
			return nil, err
		}
		mems = append(mems, m)
	}
	return mems, nil
}

// ── SMDocumentRepository ───────────────────────────────────────────────────

// SMDocumentRepo implements port.SMDocumentRepository.
type SMDocumentRepo struct{ pool *pgxpool.Pool }

// NewSMDocumentRepo creates an SMDocumentRepo.
func NewSMDocumentRepo(pool *pgxpool.Pool) *SMDocumentRepo { return &SMDocumentRepo{pool: pool} }

func (r *SMDocumentRepo) Create(ctx context.Context, doc *sm.SMDocument) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sm_documents (id, tenant_id, title, content, type, url, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		doc.ID, doc.TenantID, doc.Title, doc.Content, doc.Type, doc.URL, doc.CreatedAt,
	)
	return err
}

func (r *SMDocumentRepo) FindByID(ctx context.Context, id string) (*sm.SMDocument, error) {
	doc := &sm.SMDocument{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, content, type, url, created_at
		 FROM sm_documents WHERE id=$1`, id,
	).Scan(&doc.ID, &doc.TenantID, &doc.Title, &doc.Content, &doc.Type, &doc.URL, &doc.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	return doc, nil
}

// pgxJSONB is a helper to convert map to JSON bytes for pgx.
func pgxJSONB(m map[string]any) []byte {
	if m == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}
