package middleware

import (
	"context"
	"testing"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/redis/go-redis/v9"
)

func TestPhase4E2EXOrgIDRoundTripThroughAuthMiddleware(t *testing.T) {
	mr := miniredis.RunT(t)
	rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })

	validateCalls := 0
	handler := Auth(&fakeRegistryValidator{
		validateFn: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
			validateCalls++
			return &biz.APIKey{AppID: "app-phase4", Scopes: "read,write"}, nil
		},
	}, rc)(func(ctx context.Context, req any) (any, error) {
		appCtx, _ := AppContextFromContext(ctx)
		return appCtx, nil
	})

	call := func(orgID string) (AppContext, error) {
		tr := &testTransport{
			operation: "/api.graph.v1.Graph/CreateNode",
			request:   testHeader{},
			reply:     testHeader{},
		}
		tr.request.Set("Authorization", "Bearer phase4-api-key")
		if orgID != "" {
			tr.request.Set("X-Org-ID", orgID)
		}
		ctx := transport.NewServerContext(context.Background(), tr)
		resp, err := handler(ctx, nil)
		if err != nil {
			return AppContext{}, err
		}
		out, _ := resp.(AppContext)
		return out, nil
	}

	first, err := call("org-123")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if first.OrgID != "org-123" {
		t.Fatalf("first call org mismatch: got=%q want=%q", first.OrgID, "org-123")
	}

	second, err := call("org-456")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if second.OrgID != "org-456" {
		t.Fatalf("second call org mismatch: got=%q want=%q", second.OrgID, "org-456")
	}

	third, err := call("")
	if err != nil {
		t.Fatalf("third call failed: %v", err)
	}
	if third.OrgID != "" {
		t.Fatalf("third call org should be empty, got=%q", third.OrgID)
	}

	if validateCalls != 1 {
		t.Fatalf("expected ValidateAPIKey called once (cache hit afterwards), got=%d", validateCalls)
	}
}
