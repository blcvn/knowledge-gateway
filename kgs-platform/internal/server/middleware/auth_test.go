package middleware

import (
	"context"
	"errors"
	"testing"

	registryv1 "kgs-platform/api/registry/v1"
	"kgs-platform/internal/biz"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport"
)

type fakeRegistryValidator struct {
	validateFn func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error)
}

func (f *fakeRegistryValidator) ValidateAPIKey(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
	return f.validateFn(ctx, rawAPIKey)
}

type testHeader map[string][]string

func (h testHeader) Get(key string) string {
	values := h[key]
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (h testHeader) Set(key string, value string) {
	h[key] = []string{value}
}

func (h testHeader) Add(key string, value string) {
	h[key] = append(h[key], value)
}

func (h testHeader) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

func (h testHeader) Values(key string) []string {
	return h[key]
}

type testTransport struct {
	operation string
	request   testHeader
	reply     testHeader
}

func (t *testTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (t *testTransport) Endpoint() string                { return "http://localhost:8000" }
func (t *testTransport) Operation() string               { return t.operation }
func (t *testTransport) RequestHeader() transport.Header { return t.request }
func (t *testTransport) ReplyHeader() transport.Header   { return t.reply }

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		auth      string
		validate  func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error)
		wantErr   bool
		wantAppID string
	}{
		{
			name:      "valid key",
			operation: "/api.graph.v1.Graph/CreateNode",
			auth:      "Bearer valid-key",
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				if rawAPIKey != "valid-key" {
					t.Fatalf("unexpected raw key: %s", rawAPIKey)
				}
				return &biz.APIKey{AppID: "app-1", Scopes: "read,write"}, nil
			},
			wantAppID: "app-1",
		},
		{
			name:      "invalid key",
			operation: "/api.graph.v1.Graph/CreateNode",
			auth:      "Bearer bad-key",
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				return nil, errors.New("invalid")
			},
			wantErr: true,
		},
		{
			name:      "expired key",
			operation: "/api.graph.v1.Graph/CreateNode",
			auth:      "Bearer expired-key",
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				return nil, errors.New("expired")
			},
			wantErr: true,
		},
		{
			name:      "revoked key",
			operation: "/api.graph.v1.Graph/CreateNode",
			auth:      "Bearer revoked-key",
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				return nil, errors.New("revoked")
			},
			wantErr: true,
		},
		{
			name:      "missing key",
			operation: "/api.graph.v1.Graph/CreateNode",
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				return nil, nil
			},
			wantErr: true,
		},
		{
			name:      "skip auth for registry create app",
			operation: registryv1.OperationRegistryCreateApp,
			validate: func(ctx context.Context, rawAPIKey string) (*biz.APIKey, error) {
				return nil, errors.New("should not be called")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := &testTransport{
				operation: tc.operation,
				request:   testHeader{},
				reply:     testHeader{},
			}
			if tc.auth != "" {
				tr.request.Set("Authorization", tc.auth)
			}

			ctx := transport.NewServerContext(context.Background(), tr)
			handler := Auth(&fakeRegistryValidator{validateFn: tc.validate}, nil)(func(ctx context.Context, req any) (any, error) {
				appCtx, _ := AppContextFromContext(ctx)
				return appCtx, nil
			})

			resp, err := handler(ctx, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				if !kerrors.IsUnauthorized(err) {
					t.Fatalf("expected unauthorized error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := resp.(AppContext)
			if tc.wantAppID != "" && got.AppID != tc.wantAppID {
				t.Fatalf("expected app id %s, got %s", tc.wantAppID, got.AppID)
			}
		})
	}
}
