package middleware

import (
	"context"
	"testing"
	"time"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
)

type fakeQuotaProvider struct {
	limit int64
}

func (f *fakeQuotaProvider) GetQuotaLimit(ctx context.Context, appID string, fallback int64) (int64, error) {
	if f.limit <= 0 {
		return fallback, nil
	}
	return f.limit, nil
}

type fakeRateLimitStore struct {
	countByKey map[string]int64
}

func (s *fakeRateLimitStore) Increment(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	if s.countByKey == nil {
		s.countByKey = map[string]int64{}
	}
	s.countByKey[key]++
	return s.countByKey[key], nil
}

func TestRateLimiter(t *testing.T) {
	tr := &testTransport{
		operation: "/api.graph.v1.Graph/GetContext",
		request:   testHeader{},
		reply:     testHeader{},
	}
	baseCtx := transport.NewServerContext(context.Background(), tr)
	baseCtx = context.WithValue(baseCtx, AppContextKey, AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	store := &fakeRateLimitStore{}
	mw := RateLimiter(&fakeQuotaProvider{limit: 1}, store)
	handler := mw(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	if _, err := handler(baseCtx, nil); err != nil {
		t.Fatalf("unexpected first call error: %v", err)
	}
	if _, err := handler(baseCtx, nil); err == nil {
		t.Fatalf("expected rate limit error on second call")
	} else {
		if kerrors.Code(err) != 429 {
			t.Fatalf("expected 429, got %d", kerrors.Code(err))
		}
		if tr.reply.Get("Retry-After") != "60" {
			t.Fatalf("expected Retry-After header to be set")
		}
	}
}
