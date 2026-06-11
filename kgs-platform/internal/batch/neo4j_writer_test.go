package batch

import (
	"strings"
	"testing"
)

func TestBuildBatchUpsertQueryUsesMerge(t *testing.T) {
	query := buildBatchUpsertQuery("UserStory")
	required := []string{
		"MERGE (n:Entity:UserStory",
		"ON CREATE SET n += e, n.created_at = datetime(), n._unique_key = e._unique_key",
		"ON MATCH SET n += e, n.updated_at = datetime(), n._unique_key = e._unique_key",
		"RETURN count(n) AS upserted",
	}
	for _, token := range required {
		if !strings.Contains(query, token) {
			t.Fatalf("query missing token %q\nquery:\n%s", token, query)
		}
	}
	if strings.Contains(query, "CREATE (n:") {
		t.Fatalf("query should not use CREATE for node upsert")
	}
}

func TestBuildEntityUniqueKey(t *testing.T) {
	got := buildEntityUniqueKey(" app ", " tenant ", " id ")
	if got != "app|tenant|id" {
		t.Fatalf("unexpected unique key: %q", got)
	}
}
