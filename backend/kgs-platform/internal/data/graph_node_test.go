package data

import (
	"strings"
	"testing"
)

func TestBuildCreateNodeQueryUsesMerge(t *testing.T) {
	query := buildCreateNodeQuery("USER_STORY")
	required := []string{
		"MERGE (n:Entity:USER_STORY",
		"ON CREATE SET n += $props, n.created_at = datetime(), n._unique_key = $unique_key",
		"ON MATCH SET n += $props, n.updated_at = datetime(), n._unique_key = $unique_key",
		"RETURN n",
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

func TestBuildNodeUniqueKey(t *testing.T) {
	got := buildNodeUniqueKey(" app ", " tenant ", " node ")
	if got != "app|tenant|node" {
		t.Fatalf("unexpected unique key: %q", got)
	}
}
