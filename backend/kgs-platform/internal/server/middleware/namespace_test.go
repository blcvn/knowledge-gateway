package middleware

import (
	"context"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
)

func TestNamespaceMiddleware(t *testing.T) {
	mw := Namespace()
	handler := mw(func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	tr := &testTransport{
		operation: "/api.graph.v1.Graph/GetNode",
		request:   testHeader{},
		reply:     testHeader{},
	}
	tr.request.Set("X-KG-Namespace", "graph/app-1/tenant-2")

	ctx := transport.NewServerContext(context.Background(), tr)
	ctx = context.WithValue(ctx, AppContextKey, AppContext{
		AppID:    "app-1",
		TenantID: "tenant-1",
		Scopes:   "read",
	})

	_, err := handler(ctx, nil)
	if err == nil {
		t.Fatalf("expected forbidden error")
	}
	if !kerrors.IsForbidden(err) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
}
