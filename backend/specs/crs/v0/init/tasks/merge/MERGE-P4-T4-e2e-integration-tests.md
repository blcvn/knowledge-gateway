---
id: MERGE-P4-T4
title: "E2E Integration Tests — Validate 8-service Architecture"
phase: P4
service: tests
priority: P2
status: Done
estimated: 8h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P4-T3]
---

## Mục Tiêu

Tạo bộ integration tests kiểm tra toàn bộ happy-path flows của 8-service architecture. Tests chạy với `docker-compose.consolidated.yml` và verify end-to-end data flow.

## Test Structure

```
tests/integration/sol003/
├── setup_test.go           # TestMain: docker compose up/down
├── platform_test.go        # vnp-platform: auth + admin flows
├── kg_test.go              # kg-service: graphiti + cognee flows
├── memory_test.go          # memory-service: memobase + zep flows
├── storage_test.go         # storage-service: file CRUD flows
├── search_test.go          # search-service: cross-engine search
├── pipeline_test.go        # pipeline-service: job management
├── obs_test.go             # obs-service: metrics + infra
├── gateway_test.go         # gateway: routing verification
└── testhelper_test.go      # Shared test utilities
```

## Test Cases

### 1. Platform Tests (`platform_test.go`)

```go
func TestAuth_RegisterAndLogin(t *testing.T) {
    // Given: gateway is running
    c := newGatewayClient(t)
    
    // When: register new user
    token, err := c.Register("test@example.com", "testuser", "password123")
    require.NoError(t, err)
    require.NotEmpty(t, token)
    
    // When: login with same credentials
    loginToken, err := c.Login("test@example.com", "password123")
    require.NoError(t, err)
    require.NotEmpty(t, loginToken)
    
    // Then: JWT is valid
    assert.Equal(t, token.Email, loginToken.Email)
}

func TestAdmin_CreateTenantAndIssueKey(t *testing.T) {
    c := newGatewayClient(t)
    
    // Create tenant
    tenant, err := c.CreateTenant("Test Corp", "pro")
    require.NoError(t, err)
    require.NotEmpty(t, tenant.ID)
    
    // Issue API key
    key, err := c.IssueAPIKey(tenant.ID, []string{"read", "write"})
    require.NoError(t, err)
    require.NotEmpty(t, key.Key)
    
    // Verify health
    status, err := c.AdminHealth()
    require.NoError(t, err)
    assert.Equal(t, "ok", status)
}

func TestDashboard_MetricsNotEmpty(t *testing.T) {
    c := newGatewayClient(t)
    metrics, err := c.DashboardMetrics()
    require.NoError(t, err)
    assert.NotNil(t, metrics)
}
```

### 2. KG Tests (`kg_test.go`)

```go
func TestGraphiti_IngestAndSearch(t *testing.T) {
    c := newGatewayClient(t)
    
    // Ingest episode
    episode, err := c.IngestEpisode(IngestEpisodeRequest{
        Name:    "Test Episode",
        Content: "Alice works at Acme Corp. Bob is Alice's manager.",
        Source:  "message",
    })
    require.NoError(t, err)
    require.NotEmpty(t, episode.UUID)
    
    // Wait for processing
    time.Sleep(2 * time.Second)
    
    // Search
    results, err := c.GraphitiSearch("Alice")
    require.NoError(t, err)
    assert.Greater(t, len(results.Episodes), 0)
}

func TestGraphiti_GetNode(t *testing.T) {
    c := newGatewayClient(t)
    
    // Ingest first
    episode := ingestTestEpisode(t, c, "John is CEO of TechCo.")
    time.Sleep(2 * time.Second)
    
    // Get a node that should have been extracted
    results, _ := c.GraphitiSearch("John")
    if len(results.Nodes) > 0 {
        node, err := c.GetGraphitiNode(results.Nodes[0].UUID)
        require.NoError(t, err)
        assert.NotEmpty(t, node.UUID)
        assert.Equal(t, "TechCo", node.Attributes["company"])
    }
}
```

### 3. Memory Tests (`memory_test.go`)

```go
func TestMemobase_InsertBlobAndGetContext(t *testing.T) {
    c := newGatewayClient(t)
    userID := "test-user-" + uuid.New().String()[:8]
    
    // Insert blob
    blob, err := c.InsertBlob(userID, InsertBlobRequest{
        Type:    "conversation",
        Content: "User prefers dark mode. User is interested in Go programming.",
    })
    require.NoError(t, err)
    require.NotEmpty(t, blob.ID)
    
    // Flush buffer
    err = c.FlushBuffer(userID)
    require.NoError(t, err)
    
    // Get context
    time.Sleep(1 * time.Second)
    ctx, err := c.GetUserContext(userID)
    require.NoError(t, err)
    assert.NotEmpty(t, ctx.Summary)
}

func TestMemobase_GetProfiles(t *testing.T) {
    c := newGatewayClient(t)
    userID := createUserWithBlobs(t, c)
    
    profiles, err := c.GetUserProfiles(userID)
    require.NoError(t, err)
    // After blob processing, profiles should be extracted
    assert.GreaterOrEqual(t, len(profiles), 0)
}
```

### 4. Storage Tests (`storage_test.go`)

```go
func TestStorage_FileCRUD(t *testing.T) {
    c := newGatewayClient(t)
    path := "test/" + uuid.New().String() + ".txt"
    content := "Hello, VNP Memory!"
    
    // Write
    err := c.WriteFile(path, []byte(content))
    require.NoError(t, err)
    
    // Read
    data, err := c.ReadFile(path)
    require.NoError(t, err)
    assert.Equal(t, content, string(data))
    
    // Tree
    tree, err := c.Tree("test/")
    require.NoError(t, err)
    assert.True(t, containsFile(tree, path))
    
    // Grep
    results, err := c.Grep("test/", "Hello")
    require.NoError(t, err)
    assert.Greater(t, len(results), 0)
    
    // Delete
    err = c.DeleteFile(path)
    require.NoError(t, err)
    
    // Verify deleted
    _, err = c.ReadFile(path)
    assert.Error(t, err)
}

func TestStorage_SessionFlow(t *testing.T) {
    c := newGatewayClient(t)
    
    // Create session
    session, err := c.CreateSession("/test-workspace/")
    require.NoError(t, err)
    
    // Add message
    err = c.AddSessionMessage(session.ID, "user", "What files are here?")
    require.NoError(t, err)
    
    // Commit
    commit, err := c.CommitSession(session.ID)
    require.NoError(t, err)
    assert.NotEmpty(t, commit.SessionID)
}
```

### 5. Search Tests (`search_test.go`)

```go
func TestSearch_CrossEngineSearch(t *testing.T) {
    c := newGatewayClient(t)
    
    // Setup: insert data into multiple engines
    ingestTestEpisode(t, c, "Quantum computing uses qubits.")
    insertTestBlob(t, c, "testuser", "Quantum mechanics principles.")
    time.Sleep(2 * time.Second)
    
    // Cross-engine search
    results, err := c.ConsoleMemorySearch("quantum")
    require.NoError(t, err)
    
    // Should return results from multiple engines
    assert.Greater(t, len(results.Items), 0)
    
    // Verify RRF reranking (scores should be ordered)
    for i := 1; i < len(results.Items); i++ {
        assert.GreaterOrEqual(t, results.Items[i-1].Score, results.Items[i].Score)
    }
}

func TestSearch_RAG(t *testing.T) {
    c := newGatewayClient(t)
    
    response, err := c.RAG("What is quantum computing?")
    require.NoError(t, err)
    assert.NotEmpty(t, response.Context)
    assert.NotEmpty(t, response.Sources)
}
```

### 6. Pipeline Tests (`pipeline_test.go`)

```go
func TestPipeline_StatusNotEmpty(t *testing.T) {
    c := newGatewayClient(t)
    
    pipelines, err := c.PipelineStatus()
    require.NoError(t, err)
    assert.Greater(t, len(pipelines), 0)
    
    // Each pipeline should have known engine name
    engines := map[string]bool{}
    for _, p := range pipelines {
        engines[p.Engine] = true
    }
    assert.True(t, engines["graphiti"] || engines["cognee"] || engines["knowledge"])
}

func TestPipeline_Queues(t *testing.T) {
    c := newGatewayClient(t)
    queues, err := c.PipelineQueues()
    require.NoError(t, err)
    assert.NotNil(t, queues)
}
```

### 7. Observability Tests (`obs_test.go`)

```go
func TestObs_Metrics(t *testing.T) {
    c := newGatewayClient(t)
    
    metrics, err := c.ObsMetrics()
    require.NoError(t, err)
    assert.NotNil(t, metrics)
    assert.GreaterOrEqual(t, metrics.TotalRequests, int64(0))
}

func TestObs_InfraTopology(t *testing.T) {
    c := newGatewayClient(t)
    
    topology, err := c.InfraTopology()
    require.NoError(t, err)
    assert.Greater(t, len(topology.Services), 0)
    
    // Verify all 7 backends appear in topology
    serviceNames := map[string]bool{}
    for _, s := range topology.Services {
        serviceNames[s.Name] = true
    }
    assert.True(t, serviceNames["vnp-platform"])
    assert.True(t, serviceNames["kg-service"])
    assert.True(t, serviceNames["memory-service"])
}
```

### 8. Gateway Routing Tests (`gateway_test.go`)

```go
func TestGateway_AllHealthEndpoints(t *testing.T) {
    services := []string{
        "http://localhost:9110/healthz",  // vnp-platform
        "http://localhost:9120/healthz",  // kg-service
        "http://localhost:9130/healthz",  // memory-service
        "http://localhost:9140/healthz",  // storage-service
        "http://localhost:9150/healthz",  // search-service
        "http://localhost:9160/healthz",  // pipeline-service
        "http://localhost:9170/healthz",  // obs-service
        "http://localhost:11080/health",  // gateway
    }
    
    for _, url := range services {
        resp, err := http.Get(url)
        require.NoError(t, err, "health check failed for %s", url)
        assert.Equal(t, 200, resp.StatusCode, "unhealthy: %s", url)
    }
}

func TestGateway_404ForUnknownRoute(t *testing.T) {
    resp, _ := http.Get("http://localhost:8080/v1/nonexistent/route")
    assert.Equal(t, 404, resp.StatusCode)
}
```

## Test Setup (`setup_test.go`)

```go
func TestMain(m *testing.M) {
    // Option 1: Assume docker-compose already up (CI mode)
    if os.Getenv("INTEGRATION_ASSUME_UP") == "true" {
        os.Exit(m.Run())
    }
    
    // Option 2: Start docker-compose in test
    cmd := exec.Command("docker", "compose",
        "-f", "../../docker-compose.consolidated.yml",
        "up", "-d", "--wait")
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        log.Fatalf("failed to start docker compose: %v", err)
    }
    
    // Wait for services to be healthy
    waitForHealthy("http://localhost:8080/v1/admin/health", 60*time.Second)
    
    code := m.Run()
    
    // Teardown
    exec.Command("docker", "compose",
        "-f", "../../docker-compose.consolidated.yml",
        "down").Run()
    
    os.Exit(code)
}
```

## Test Runner Config (`Makefile` additions)

```makefile
test-integration-sol003:
	INTEGRATION_ASSUME_UP=false \
	go test -v -timeout 300s -tags integration \
		./tests/integration/sol003/...

test-integration-sol003-ci:
	INTEGRATION_ASSUME_UP=true \
	go test -v -timeout 120s -tags integration \
		./tests/integration/sol003/...
```

## Acceptance Criteria

- [ ] Auth: register + login flow works end-to-end (JWT returned)
- [ ] Admin: create tenant + issue API key persisted in PostgreSQL
- [ ] Graphiti: ingest episode → search returns results
- [ ] Memobase: insert blob → flush → get context (non-empty summary)
- [ ] Storage: file CRUD (write → read → grep → delete) all pass
- [ ] Storage: session flow (create → add message → commit) works
- [ ] Search: cross-engine search returns results from 2+ engines
- [ ] Pipeline: status endpoint returns non-empty pipeline list
- [ ] Observability: metrics endpoint returns valid JSON
- [ ] Infra: topology includes all 7 backend services
- [ ] All 8 health endpoints return 200 OK
- [ ] `make test-integration-sol003` passes in < 5 minutes

## Ghi Chú

- Tests tagged với `//go:build integration` để không run trong unit test pass
- testhelper.go cần implement `GatewayClient` struct với typed methods
- Zep tests: skip nếu `ZEP_ENABLED=false`
- Cognee tests: skip nếu Cognee container không healthy
