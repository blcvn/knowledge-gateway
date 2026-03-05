package batch

import (
	"context"
	"errors"
	"testing"
)

type fakeWriter struct {
	createFn func(ctx context.Context, appID, tenantID string, entities []Entity) (int, error)
}

func (w *fakeWriter) BulkCreate(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
	return w.createFn(ctx, appID, tenantID, entities)
}

func TestUsecaseExecute(t *testing.T) {
	tests := []struct {
		name    string
		req     BatchUpsertRequest
		wantErr bool
		assert  func(t *testing.T, result *BatchUpsertResult)
	}{
		{
			name:    "empty batch",
			req:     BatchUpsertRequest{AppID: "app", TenantID: "default", Entities: []Entity{}},
			wantErr: true,
		},
		{
			name: "max batch",
			req: func() BatchUpsertRequest {
				entities := make([]Entity, 1000)
				for i := range entities {
					entities[i] = Entity{Label: "User", Properties: map[string]any{"n": i}}
				}
				return BatchUpsertRequest{AppID: "app", TenantID: "default", Entities: entities}
			}(),
			assert: func(t *testing.T, result *BatchUpsertResult) {
				if result.Created != 1000 || result.Skipped != 0 {
					t.Fatalf("unexpected result: %#v", result)
				}
			},
		},
		{
			name: "duplicate detection",
			req: BatchUpsertRequest{
				AppID:    "app",
				TenantID: "default",
				Entities: []Entity{
					{Label: "User", Properties: map[string]any{"name": "alice"}},
					{Label: "User", Properties: map[string]any{"name": "alice"}},
				},
			},
			assert: func(t *testing.T, result *BatchUpsertResult) {
				if result.Created != 1 || result.Skipped != 1 {
					t.Fatalf("unexpected result: %#v", result)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewUsecase(&fakeWriter{
				createFn: func(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
					return len(entities), nil
				},
			}, NewExactDeduper())

			result, err := uc.Execute(context.Background(), tc.req)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.assert != nil {
				tc.assert(t, result)
			}
		})
	}
}

func TestUsecaseExecuteWriterError(t *testing.T) {
	uc := NewUsecase(&fakeWriter{
		createFn: func(ctx context.Context, appID, tenantID string, entities []Entity) (int, error) {
			return 0, errors.New("write failed")
		},
	}, NewExactDeduper())

	_, err := uc.Execute(context.Background(), BatchUpsertRequest{
		AppID:    "app",
		TenantID: "default",
		Entities: []Entity{{Label: "User", Properties: map[string]any{"name": "alice"}}},
	})
	if err == nil {
		t.Fatalf("expected writer error")
	}
}
