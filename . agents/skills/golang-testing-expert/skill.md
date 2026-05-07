---
skill_id: SKILL-012
version: 1.0.0
status: active
priority: existing
group: Quality Assurance
created_at: 2026-04-24
---

# SKILL-012 · Backend Testing — Golang Security & Quality

## Mô tả

Đọc code tìm lỗ hổng, viết table-driven tests, integration tests với testcontainers, fuzz testing, security vulnerability analysis.

## Agents sử dụng

- `qa-pipeline-agent`
- `traceability-validator-agent`

---

## Năng lực cốt lõi

### 1. Table-Driven Tests

```go
// Chuẩn Go: table-driven tests với subtests
func TestExtractEntities(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        want     []Entity
        wantErr  bool
    }{
        {
            name:  "extract single actor",
            input: "Người dùng đăng nhập vào hệ thống",
            want:  []Entity{{Name: "User", Type: "ACTOR"}},
        },
        {
            name:    "empty input returns error",
            input:   "",
            wantErr: true,
        },
        {
            name:  "Vietnamese and English mixed",
            input: "The Admin phê duyệt đơn hàng",
            want: []Entity{
                {Name: "Admin", Type: "ACTOR"},
                {Name: "Order", Type: "BUSINESS_OBJECT"},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExtractEntities(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.ElementsMatch(t, tt.want, got)
        })
    }
}
```

### 2. Integration Tests với Testcontainers

```go
// Integration test với real PostgreSQL
func TestDocumentRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()

    // Start PostgreSQL container
    container, err := testcontainers.GenericContainer(ctx,
        testcontainers.GenericContainerRequest{
            ContainerRequest: testcontainers.ContainerRequest{
                Image:        "postgres:16-alpine",
                ExposedPorts: []string{"5432/tcp"},
                Env: map[string]string{
                    "POSTGRES_PASSWORD": "test",
                    "POSTGRES_DB":       "testdb",
                },
                WaitingFor: wait.ForListeningPort("5432/tcp"),
            },
            Started: true,
        },
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)

    // Get connection string
    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "5432")
    dsn := fmt.Sprintf("postgres://postgres:test@%s:%s/testdb", host, port.Port())

    // Run migrations
    db, _ := pgxpool.New(ctx, dsn)
    runMigrations(db)

    // Test
    repo := NewDocumentRepository(db)
    doc, err := repo.Create(ctx, &Document{Title: "Test PRD", ProjectID: testProjectID})
    
    assert.NoError(t, err)
    assert.NotEmpty(t, doc.ID)
    
    fetched, err := repo.Get(ctx, doc.ID)
    assert.NoError(t, err)
    assert.Equal(t, "Test PRD", fetched.Title)
}
```

### 3. Race Detection

```bash
# Luôn chạy tests với -race flag trong CI
go test -race ./...

# Common race conditions to detect:
# - Concurrent map read/write (dùng sync.Map hoặc mutex)
# - Goroutine accessing shared state without sync
# - Channel send on closed channel
```

```go
// Fix: protect shared state với mutex
type SafeCache struct {
    mu    sync.RWMutex
    items map[string]string
}

func (c *SafeCache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    v, ok := c.items[key]
    return v, ok
}

func (c *SafeCache) Set(key, value string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.items[key] = value
}
```

### 4. Fuzz Testing

```go
// Fuzz test cho input parsing (Go 1.18+)
func FuzzExtractEntities(f *testing.F) {
    // Seed corpus
    f.Add("Người dùng đăng nhập")
    f.Add("Admin creates Order")
    f.Add("")
    f.Add(strings.Repeat("a", 10000))
    
    f.Fuzz(func(t *testing.T, input string) {
        // Must not panic, must not crash
        result, err := ExtractEntities(input)
        
        // Invariants that must always hold
        if err == nil {
            for _, entity := range result {
                assert.NotEmpty(t, entity.Name, "entity name must not be empty")
                assert.Contains(t, validEntityTypes, entity.Type)
            }
        }
    })
}

// Run: go test -fuzz=FuzzExtractEntities -fuzztime=60s
```

### 5. Security Code Audit Checklist

```
Code Review Security Checklist:

[ ] SQL Injection
    - All queries use parameterized statements ($1, $2...)
    - No string concatenation in SQL
    - Verify: grep -r "fmt.Sprintf.*SELECT\|INSERT\|UPDATE\|DELETE" .

[ ] Command Injection
    - No exec.Command with user input
    - Verify: grep -r "exec.Command\|os.exec" .

[ ] Path Traversal
    - File paths validated with filepath.Clean + base directory check
    - Verify: grep -r "os.Open\|os.ReadFile" . (manual review)

[ ] Integer Overflow
    - Explicit type conversion checked for overflow
    - Verify: grep -r "int(" . (manual review of conversions)

[ ] Goroutine Leak
    - All goroutines have exit conditions
    - Context cancellation propagated
    - Verify: goleak in tests

[ ] Sensitive Data in Logs
    - No passwords, tokens, PII in log statements
    - Verify: grep -r 'log.*password\|log.*token\|log.*secret' .
```

### 6. Security Scanning Tools

```yaml
# gosec — static security analysis
# .gosec.yml
rules:
  - G101  # Potential hardcoded credentials
  - G102  # Bind to all interfaces
  - G104  # Errors unhandled
  - G201  # SQL query construction using format string
  - G202  # SQL query construction using string concatenation
  - G301  # Poor file permissions used when creating a directory
  - G304  # File path provided as taint input
  - G401  # Use of weak cryptographic primitive
  - G501  # Import blocklist: crypto/md5

# govulncheck — known CVEs in dependencies
# govulncheck ./...

# goleak — goroutine leak detection in tests
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

### 7. Mock Generation

```go
// Using mockery or gomock
//go:generate mockery --name=DocumentRepository --outpkg=mocks
type DocumentRepository interface {
    Get(ctx context.Context, id string) (*Document, error)
    Create(ctx context.Context, doc *Document) (*Document, error)
}

// Test with mock
func TestDocumentService_Get(t *testing.T) {
    mockRepo := mocks.NewDocumentRepository(t)
    
    mockRepo.EXPECT().
        Get(mock.Anything, "doc-123").
        Return(&Document{ID: "doc-123", Title: "Test PRD"}, nil)
    
    svc := NewDocumentService(mockRepo)
    doc, err := svc.GetDocument(context.Background(), "doc-123")
    
    assert.NoError(t, err)
    assert.Equal(t, "Test PRD", doc.Title)
}
```

---

## Coverage Requirements

```
Minimum coverage by package:
├── internal/service/     → 85%+ (business logic)
├── internal/repository/  → 70%+ (integration tested)
├── internal/handler/     → 80%+ (happy + error paths)
└── pkg/                  → 90%+ (utility functions)

Run coverage:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Checklist

- [ ] Table-driven tests cho tất cả business logic functions
- [ ] Integration tests với testcontainers (không mock DB)
- [ ] `-race` flag trong CI test run
- [ ] Fuzz tests cho input parsing functions
- [ ] `govulncheck` pass (zero known vulnerabilities)
- [ ] `gosec` pass (hoặc documented exceptions)
- [ ] Test coverage ≥ 80% overall
- [ ] Mocks generated (không viết tay)
- [ ] `goleak.VerifyTestMain` để detect goroutine leaks
