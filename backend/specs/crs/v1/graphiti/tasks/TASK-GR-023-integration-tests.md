# TASK-GR-023 — End-to-End Integration Tests

| Field | Value |
|-------|-------|
| **Task ID** | TASK-GR-023 |
| **Wave** | 4 (Admin & Observability) |
| **Component** | `tests/integration/graphiti/` |
| **Status** | 🔲 Pending |
| **Solution Ref** | SOL-001 §8, SOL-004 §7 |
| **Priority** | High |
| **Depends On** | TASK-GR-022 |
| **Estimated** | 4h |

---

## Context

Viết E2E integration tests cho toàn bộ graphiti pipeline: ingest → resolve → search → admin. Tests chạy against real Neo4j + Redis (docker compose test stack).

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `tests/integration/graphiti/ingest_test.go` |
| CREATE | `tests/integration/graphiti/search_test.go` |
| CREATE | `tests/integration/graphiti/temporal_test.go` |
| CREATE | `tests/integration/graphiti/ontology_test.go` |
| CREATE | `tests/integration/graphiti/admin_test.go` |
| CREATE | `tests/integration/graphiti/helpers.go` |

---

## Implementation

### File 1: `tests/integration/graphiti/helpers.go`

```go
package graphiti_integration

import (
    "context"
    "fmt"
    "os"
    "testing"
    "time"

    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    searchpb    "github.com/vnp-memory/api/proto/graphiti/search/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/metadata"
)

const (
    defaultIngestionAddr = "localhost:9094"
    defaultSearchAddr    = "localhost:9098"
    testGroupID          = "integration-test-group"
)

type TestClients struct {
    Ingestion ingestionpb.IngestionServiceClient
    Search    searchpb.SearchServiceClient
}

func SetupClients(t *testing.T) *TestClients {
    t.Helper()
    ingestionAddr := os.Getenv("GRAPHITI_INGESTION_ADDR")
    if ingestionAddr == "" { ingestionAddr = defaultIngestionAddr }
    searchAddr := os.Getenv("GRAPHITI_SEARCH_ADDR")
    if searchAddr == "" { searchAddr = defaultSearchAddr }

    ingestionConn, err := grpc.Dial(ingestionAddr, grpc.WithInsecure())
    if err != nil { t.Fatalf("dial ingestion: %v", err) }
    searchConn, err := grpc.Dial(searchAddr, grpc.WithInsecure())
    if err != nil { t.Fatalf("dial search: %v", err) }

    t.Cleanup(func() { ingestionConn.Close(); searchConn.Close() })
    return &TestClients{
        Ingestion: ingestionpb.NewIngestionServiceClient(ingestionConn),
        Search:    searchpb.NewSearchServiceClient(searchConn),
    }
}

func TestCtx(groupID string) context.Context {
    ctx := context.Background()
    return metadata.AppendToOutgoingContext(ctx, "x-group-id", groupID)
}

func IngestEpisode(t *testing.T, client ingestionpb.IngestionServiceClient, body, source, groupID string) string {
    t.Helper()
    resp, err := client.IngestEpisode(TestCtx(groupID), &ingestionpb.IngestEpisodeRequest{
        Name:   "test-episode-" + fmt.Sprintf("%d", time.Now().UnixNano()),
        Body:   body,
        Source: source,
    })
    if err != nil { t.Fatalf("ingest episode: %v", err) }
    return resp.EpisodeUuid
}

// WaitForIndexing gives Neo4j time to update indices after ingestion
func WaitForIndexing() { time.Sleep(500 * time.Millisecond) }
```

### File 2: `tests/integration/graphiti/ingest_test.go`

```go
package graphiti_integration

import (
    "testing"
    "time"

    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestIngestEpisode_Text_BasicEntityEdgeExtraction(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-ingest-basic"
    ctx := TestCtx(groupID)

    resp, err := clients.Ingestion.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
        Name:   "org-chart",
        Body:   "Alice works as a Software Engineer at TechCorp. She reports to Bob who is the CTO.",
        Source: "text",
    })

    require.NoError(t, err)
    assert.NotEmpty(t, resp.EpisodeUuid)
    assert.Greater(t, resp.Stats.EntitiesExtracted, int32(0), "should extract some entities")
    assert.Greater(t, resp.Stats.EdgesExtracted, int32(0), "should extract some edges")

    t.Logf("Episode UUID: %s, Entities: %d, Edges: %d",
        resp.EpisodeUuid, resp.Stats.EntitiesExtracted, resp.Stats.EdgesExtracted)
}

func TestIngestEpisode_EntityResolution_SameEntityMerged(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-entity-dedup"
    ctx := TestCtx(groupID)

    // First mention: "Alice Johnson"
    resp1, err := clients.Ingestion.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
        Body:   "Alice Johnson joined the engineering team as a backend developer.",
        Source: "text",
    })
    require.NoError(t, err)

    WaitForIndexing()

    // Second mention: "Alice" (same person, different name form)
    resp2, err := clients.Ingestion.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
        Body:   "Alice is working on the new API gateway project.",
        Source: "text",
    })
    require.NoError(t, err)

    // Second episode should have fewer NEW entities (Alice merged, not new)
    assert.Less(t, resp2.Stats.EntitiesNew, resp2.Stats.EntitiesExtracted,
        "Alice should be merged with existing entity, not created as new")

    t.Logf("Ep1 new: %d, Ep2 new: %d (expected < extracted)", resp1.Stats.EntitiesNew, resp2.Stats.EntitiesNew)
}

func TestIngestEpisode_Triplet(t *testing.T) {
    clients := SetupClients(t)
    ctx := TestCtx(testGroupID + "-triplet")

    resp, err := clients.Ingestion.AddTriplet(ctx, &ingestionpb.AddTripletRequest{
        SourceEntity: "Alice",
        Relation:     "WORKS_AT",
        TargetEntity: "TechCorp",
        Fact:         "Alice works at TechCorp as an engineer",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, resp.EpisodeUuid)
}

func TestIngestEpisode_RemoveEpisode_CascadesMentions(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-remove"
    ctx := TestCtx(groupID)

    episodeUUID := IngestEpisode(t, clients.Ingestion, "Bob leads the Platform Team.", "text", groupID)
    WaitForIndexing()

    _, err := clients.Ingestion.RemoveEpisode(ctx, &ingestionpb.RemoveEpisodeRequest{EpisodeUuid: episodeUUID})
    require.NoError(t, err)

    // Episode should no longer appear in list
    listResp, err := clients.Ingestion.ListEpisodes(ctx, &ingestionpb.ListEpisodesRequest{LastN: 20})
    require.NoError(t, err)
    for _, ep := range listResp.Episodes {
        assert.NotEqual(t, episodeUUID, ep.Uuid, "removed episode should not appear in list")
    }
}
```

### File 3: `tests/integration/graphiti/temporal_test.go`

```go
package graphiti_integration

import (
    "testing"
    "time"

    ingestionpb "github.com/vnp-memory/api/proto/graphiti/ingestion/v1"
    searchpb    "github.com/vnp-memory/api/proto/graphiti/search/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestTemporalEdgeInvalidation verifies that CONTRADICTION resolution invalidates old edges
func TestTemporalEdgeInvalidation_ContradictingFacts(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-temporal"
    ctx := TestCtx(groupID)

    past := time.Now().AddDate(0, -1, 0).Format(time.RFC3339)

    // First fact: Alice works at OldCorp
    _, err := clients.Ingestion.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
        Body:          "Alice has been working at OldCorp as a developer.",
        Source:        "text",
        ReferenceTime: past,
    })
    require.NoError(t, err)
    WaitForIndexing()

    // Second fact: Alice now works at NewCorp (contradicts first)
    _, err = clients.Ingestion.IngestEpisode(ctx, &ingestionpb.IngestEpisodeRequest{
        Body:   "Alice recently joined NewCorp as a senior engineer, leaving OldCorp.",
        Source: "text",
    })
    require.NoError(t, err)
    WaitForIndexing()

    // Search for Alice's current employer should return NewCorp
    searchResp, err := clients.Search.Search(ctx, &searchpb.SearchRequest{
        Query:      "Alice employer workplace",
        GroupIds:   []string{groupID},
        NumResults: 5,
    })
    require.NoError(t, err)

    found := false
    for _, edge := range searchResp.Edges {
        if containsFact(edge, "NewCorp") { found = true; break }
    }
    assert.True(t, found, "search should return NewCorp fact as current employer")
    t.Logf("Search returned %d edges", len(searchResp.Edges))
}

func containsFact(edge any, keyword string) bool {
    // Type assertion to check fact field
    if m, ok := edge.(map[string]any); ok {
        if fact, ok := m["fact"].(string); ok {
            return len(fact) > 0 && len(keyword) > 0 &&
                (len(fact) >= len(keyword) && containsStr(fact, keyword))
        }
    }
    return false
}

func containsStr(s, sub string) bool {
    for i := 0; i <= len(s)-len(sub); i++ {
        if s[i:i+len(sub)] == sub { return true }
    }
    return false
}
```

### File 4: `tests/integration/graphiti/search_test.go`

```go
package graphiti_integration

import (
    "testing"

    searchpb "github.com/vnp-memory/api/proto/graphiti/search/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSearch_HybridSearch_ReturnsRelevantFacts(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-search"
    ctx := TestCtx(groupID)

    // Ingest test data
    episodes := []string{
        "Alice leads the Platform Team at Acme Corp. She reports to the CTO.",
        "Bob is a Senior Frontend Engineer at Acme Corp. He works on the customer dashboard.",
        "The Platform Team is responsible for infrastructure and API services.",
    }
    for _, ep := range episodes {
        IngestEpisode(t, clients.Ingestion, ep, "text", groupID)
    }
    WaitForIndexing()

    resp, err := clients.Search.Search(ctx, &searchpb.SearchRequest{
        Query:      "who leads the Platform Team",
        GroupIds:   []string{groupID},
        NumResults: 5,
    })
    require.NoError(t, err)
    assert.Greater(t, len(resp.Edges), 0, "should return results for known data")
    t.Logf("Search returned %d edges, %d nodes", len(resp.Edges), len(resp.Nodes))
}

func TestSearch_RecipeName_EdgeHybridMMR(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-search-mmr"
    ctx := TestCtx(groupID)

    IngestEpisode(t, clients.Ingestion, "Charlie is the VP of Engineering. He manages 3 teams.", "text", groupID)
    WaitForIndexing()

    resp, err := clients.Search.SearchAdvanced(ctx, &searchpb.SearchAdvancedRequest{
        Query:            "Charlie role responsibility",
        GroupIds:         []string{groupID},
        NumResults:       5,
        SearchConfigName: "edge_hybrid_mmr",
    })
    require.NoError(t, err)
    t.Logf("MMR search returned %d results", len(resp.Edges))
}
```

### File 5: `tests/integration/graphiti/admin_test.go`

```go
package graphiti_integration

import (
    "testing"
    "time"

    adminpb "github.com/vnp-memory/api/proto/graphiti/admin/v1"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"
)

func TestAdminBuildCommunities(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-admin-community"
    ctx := TestCtx(groupID)

    // Ingest enough data to form communities
    episodes := []string{
        "Alice, Bob, and Carol all work in the Engineering department.",
        "Dave and Eve work in the Sales department. Dave reports to Eve.",
        "Alice and Bob collaborate on the Platform project.",
    }
    for _, ep := range episodes {
        IngestEpisode(t, clients.Ingestion, ep, "text", groupID)
    }
    WaitForIndexing()

    // Build communities
    adminConn, err := grpc.Dial("localhost:9096", grpc.WithInsecure())
    require.NoError(t, err)
    defer adminConn.Close()

    adminClient := adminpb.NewAdminServiceClient(adminConn)
    resp, err := adminClient.BuildCommunities(ctx, &adminpb.BuildCommunitiesRequest{
        GroupId: groupID, DeleteExisting: true,
    })
    require.NoError(t, err)
    assert.GreaterOrEqual(t, int(resp.CommunitiesBuilt), 1, "should detect at least 1 community")
    t.Logf("Communities built: %d, Entities grouped: %d", resp.CommunitiesBuilt, resp.EntitiesGrouped)
}

func TestAdminGetGroupStats(t *testing.T) {
    clients := SetupClients(t)
    groupID := testGroupID + "-stats"
    ctx := TestCtx(groupID)

    IngestEpisode(t, clients.Ingestion, "Test entity for stats.", "text", groupID)
    WaitForIndexing()

    adminConn, err := grpc.Dial("localhost:9096", grpc.WithInsecure())
    require.NoError(t, err)
    defer adminConn.Close()

    adminClient := adminpb.NewAdminServiceClient(adminConn)
    resp, err := adminClient.GetGroupStats(ctx, &adminpb.GetGroupStatsRequest{GroupId: groupID})
    require.NoError(t, err)
    assert.Greater(t, resp.EpisodeCount, int64(0), "should have at least 1 episode")
    t.Logf("Stats: entities=%d, episodes=%d, edges=%d", resp.EntityCount, resp.EpisodeCount, resp.EdgeCount)
}
```

---

## Running Tests

```bash
# Start test stack
cd deploy/dev
docker compose -f docker-compose.server.yaml up -d neo4j graphiti-store graphiti-knowledge graphiti-ingestion graphiti-admin

# Wait for services to be ready
sleep 30

# Run integration tests
cd /path/to/vnp-memory
go test ./tests/integration/graphiti/... -v -timeout 5m \
    -run "TestIngest|TestSearch|TestTemporal|TestAdmin"
```

---

## Acceptance Criteria

| Test | Expected Outcome |
|------|-----------------|
| `TestIngestEpisode_Text_BasicEntityEdgeExtraction` | `entities_extracted > 0`, `edges_extracted > 0` |
| `TestIngestEpisode_EntityResolution_SameEntityMerged` | Ep2 `entities_new < entities_extracted` |
| `TestTemporalEdgeInvalidation_ContradictingFacts` | Search returns NewCorp (not OldCorp) as current employer |
| `TestSearch_HybridSearch_ReturnsRelevantFacts` | `len(edges) > 0` for known data |
| `TestAdminBuildCommunities` | `communities_built >= 1` |
| `TestAdminGetGroupStats` | `episode_count > 0` |
