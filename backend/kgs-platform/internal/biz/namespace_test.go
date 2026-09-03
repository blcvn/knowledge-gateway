package biz

import "testing"

func TestComputeNamespace(t *testing.T) {
	tests := []struct {
		name     string
		appID    string
		tenantID string
		want     string
	}{
		{
			name:     "normal",
			appID:    "app-1",
			tenantID: "tenant-1",
			want:     "graph/app-1/tenant-1",
		},
		{
			name:  "default tenant",
			appID: "app-1",
			want:  "graph/app-1/default",
		},
		{
			name:     "trim spaces",
			appID:    " app-1 ",
			tenantID: " tenant-1 ",
			want:     "graph/app-1/tenant-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeNamespace(tc.appID, tc.tenantID)
			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}

