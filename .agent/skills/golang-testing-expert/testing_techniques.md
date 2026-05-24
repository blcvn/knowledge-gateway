# Golang Testing Techniques & Automation

## Testing Strategy by Layer

### Unit Tests
- **Table-Driven Tests:** Using Go's idiomatic table-driven test pattern to cover multiple input/output scenarios concisely in a single test function.
```go
func TestParseAmount(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int64
        wantErr bool
    }{
        {"valid integer", "100", 100, false},
        {"negative value", "-50", -50, false},
        {"empty string", "", 0, true},
        {"overflow", "99999999999999999999", 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseAmount(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseAmount() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("ParseAmount() = %v, want %v", got, tt.want)
            }
        })
    }
}
```
- **Interface Mocking:** Using interfaces to inject mock dependencies (database, external services) for isolated unit testing without real I/O.

### Integration Tests
- **Test Containers (testcontainers-go):** Spinning up real Docker containers (PostgreSQL, Redis, Kafka) for integration tests to eliminate mocking inconsistencies.
- **In-Memory Databases (sqlitemock / pgmem):** For faster test cycles where full containerization is unnecessary.
- **HTTP Handler Testing:** Using Go's `net/http/httptest` package to test HTTP handlers without starting a real server.

### API / E2E Tests
- **`httptest.NewServer`:** Creating a real HTTP server in-process for black-box API testing.
- **Contract Testing:** Verifying that the API responses match the published OpenAPI/Swagger contract to catch regressions in the API surface.

## Key Testing Tools
| Tool | Purpose |
|---|---|
| `go test -race ./...` | Detect data races at runtime |
| `go test -cover ./...` | Measure code coverage |
| `go test -fuzz ./...` | Fuzz testing to discover edge cases |
| `go test -bench ./...` | Benchmark-driven performance regression detection |
| `testify/assert` | Cleaner assertion library |
| `testify/mock` | Interface mock generation |
| `testcontainers-go` | Real infrastructure for integration tests |
| `govulncheck` | Scan for known CVEs in dependencies |
| `gosec` | Static analysis for security issues |

## Fuzz Testing
Fuzz testing automatically generates random inputs to find edge cases that structured tests miss:
```go
func FuzzParseInput(f *testing.F) {
    // Seed corpus
    f.Add("valid-input-1")
    f.Add("")
    f.Fuzz(func(t *testing.T, s string) {
        // Must not panic on any input
        result, _ := ParseInput(s)
        _ = result
    })
}
```
