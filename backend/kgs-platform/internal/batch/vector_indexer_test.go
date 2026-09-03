package batch

import "testing"

func TestToQdrantPointID_IsUUIDAndDeterministic(t *testing.T) {
	a := toQdrantPointID("app-1", "tenant-1", "USER_STORY", "doc-1_US-01")
	b := toQdrantPointID("app-1", "tenant-1", "USER_STORY", "doc-1_US-01")
	c := toQdrantPointID("app-1", "tenant-1", "USER_STORY", "doc-1_US-02")

	if a == "" {
		t.Fatalf("point id must not be empty")
	}
	if len(a) != 36 {
		t.Fatalf("point id must be UUID format, got=%q", a)
	}
	if a != b {
		t.Fatalf("point id must be deterministic, got a=%q b=%q", a, b)
	}
	if a == c {
		t.Fatalf("different entity ids must produce different point ids, got=%q", a)
	}
}
