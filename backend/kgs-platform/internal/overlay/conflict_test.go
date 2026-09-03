package overlay

import "testing"

func TestResolveConflictsPolicies(t *testing.T) {
	conflicts := []Conflict{{Type: "BASE_DRIFT", Message: "drift"}}

	tests := []struct {
		name      string
		policy    string
		wantErr   bool
		wantCount int
	}{
		{name: "keep overlay", policy: PolicyKeepOverlay, wantCount: 1},
		{name: "keep base", policy: PolicyKeepBase, wantCount: 1},
		{name: "merge", policy: PolicyMerge, wantCount: 1},
		{name: "manual", policy: PolicyRequireManual, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveConflicts(tc.policy, conflicts)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantCount {
				t.Fatalf("unexpected resolved count: %d", got)
			}
		})
	}
}

func TestDetectConflicts(t *testing.T) {
	if got := DetectConflicts("v1", "v1"); len(got) != 0 {
		t.Fatalf("expected no conflicts when base unchanged")
	}
	if got := DetectConflicts("v1", "v2"); len(got) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(got))
	}
}
