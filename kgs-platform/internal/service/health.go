package service

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthProviderSet wires health service dependencies.
var HealthProviderSet = wire.NewSet(NewHealthService)

type HealthService struct {
	db      *gorm.DB
	redis   *redis.Client
	neo4j   neo4j.DriverWithContext
	qdrant  *data.QdrantClient
	nats    *data.NATSClient
	log     *log.Helper
	timeout time.Duration
}

func NewHealthService(db *gorm.DB, redisCli *redis.Client, neo4jDriver neo4j.DriverWithContext, qdrant *data.QdrantClient, nats *data.NATSClient, logger log.Logger) *HealthService {
	return &HealthService{
		db:      db,
		redis:   redisCli,
		neo4j:   neo4jDriver,
		qdrant:  qdrant,
		nats:    nats,
		log:     log.NewHelper(logger),
		timeout: 2 * time.Second,
	}
}

type healthResponse struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks,omitempty"`
	CheckedAt string            `json:"checked_at"`
}

func (s *HealthService) Liveness(w http.ResponseWriter, r *http.Request) {
	_ = r
	s.writeJSON(w, http.StatusOK, healthResponse{
		Status:    "ok",
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *HealthService) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.timeout)
	defer cancel()

	checks := map[string]string{
		"postgres": s.checkPostgres(ctx),
		"redis":    s.checkRedis(ctx),
		"neo4j":    s.checkNeo4j(ctx),
		"qdrant":   s.checkQdrant(ctx),
		"nats":     s.checkNATS(ctx),
	}

	statusCode := http.StatusOK
	status := "ready"
	for _, state := range checks {
		if state == "error" {
			statusCode = http.StatusServiceUnavailable
			status = "not_ready"
			break
		}
	}

	s.writeJSON(w, statusCode, healthResponse{
		Status:    status,
		Checks:    checks,
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *HealthService) checkPostgres(ctx context.Context) string {
	if s.db == nil {
		return "skip"
	}
	sqlDB, err := s.db.DB()
	if err != nil {
		return "error"
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func (s *HealthService) checkRedis(ctx context.Context) string {
	if s.redis == nil {
		return "skip"
	}
	if err := s.redis.Ping(ctx).Err(); err != nil {
		return "error"
	}
	return "ok"
}

func (s *HealthService) checkNeo4j(ctx context.Context) string {
	if s.neo4j == nil {
		return "skip"
	}
	session := s.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, "RETURN 1 AS ok", nil)
		if err != nil {
			return nil, err
		}
		if result.Next(ctx) {
			return 1, nil
		}
		return nil, result.Err()
	})
	if err != nil {
		return "error"
	}
	return "ok"
}

func (s *HealthService) checkQdrant(ctx context.Context) string {
	if s.qdrant == nil {
		return "skip"
	}
	if err := s.qdrant.Ping(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func (s *HealthService) checkNATS(ctx context.Context) string {
	if s.nats == nil {
		return "skip"
	}
	if err := s.nats.Ping(ctx); err != nil {
		return "error"
	}
	return "ok"
}

func (s *HealthService) writeJSON(w http.ResponseWriter, code int, payload healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.log.Warnf("failed to write health response: %v", err)
	}
}
