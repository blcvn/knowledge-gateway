package postgres

import (
	"strings"
	"testing"

	"kg-service/internal/write"
)

func TestChunkRelationshipRecordsRespectsPostgresBindLimit(t *testing.T) {
	rels := make([]write.RelationshipRecord, 5960)

	chunks := chunkRelationshipRecords(rels)

	if len(chunks) != 2 {
		t.Fatalf("chunk count = %d, want 2", len(chunks))
	}
	if got := len(chunks[0]); got != 5957 {
		t.Fatalf("first chunk size = %d, want 5957", got)
	}
	if got := len(chunks[1]); got != 3 {
		t.Fatalf("second chunk size = %d, want 3", got)
	}
}

func TestChunkOutboxEventsRespectsPostgresBindLimit(t *testing.T) {
	events := make([]write.OutboxEvent, 7283)

	chunks := chunkOutboxEvents(events)

	if len(chunks) != 2 {
		t.Fatalf("chunk count = %d, want 2", len(chunks))
	}
	if got := len(chunks[0]); got != 7281 {
		t.Fatalf("first chunk size = %d, want 7281", got)
	}
	if got := len(chunks[1]); got != 2 {
		t.Fatalf("second chunk size = %d, want 2", got)
	}
}

func TestBuildGraphVersionEntitiesInsertQueryUsesVersionID(t *testing.T) {
	query, args, err := buildGraphVersionEntitiesInsertQuery("version-123", []write.GraphVersionEntityRecord{
		{EntityKind: "node", EntityID: "entity-1", ChangeKind: "UPSERT"},
		{EntityKind: "relationship", EntityID: "entity-2", ChangeKind: "DELETE"},
	})
	if err != nil {
		t.Fatalf("buildGraphVersionEntitiesInsertQuery() error = %v", err)
	}
	if !strings.Contains(query, "$1") || !strings.Contains(query, "$5") {
		t.Fatalf("query placeholders = %q, want version/entity placeholders", query)
	}
	if len(args) != 8 {
		t.Fatalf("args len = %d, want 8", len(args))
	}
	for i := 0; i < len(args); i += 4 {
		if args[i] != "version-123" {
			t.Fatalf("args[%d] = %v, want version-123", i, args[i])
		}
	}
}
