# Core Golang Skills

## Concurrency & Synchronization
- Mastery of Goroutines, Channels, and the `select` statement.
- Deep understanding of the `sync` package (`Mutex`, `RWMutex`, `WaitGroup`, `Cond`, `Map`, `Pool`).
- Expertise in using `context` for cancellation, timeouts, and passing request-scoped values.
- Preventing and debugging goroutine leaks and race conditions (using `go run -race`).

## Memory Management & Go Runtime
- Understanding the Go Garbage Collector (GC) mechanics (Mark and Sweep, Pacing).
- Managing memory allocation, heap vs. stack allocation (escape analysis).
- Utilizing `sync.Pool` to reduce GC pressure for short-lived, frequently allocated objects.
- Optimizing data structures for memory alignment and CPU cache locality.

## Error Handling & Observability
- Idiomatic error handling: wrapping errors (`fmt.Errorf("... %w", err)`), custom error types, and `errors.Is`/`errors.As`.
- Implementing structured logging (e.g., `slog`, `zap`, `logrus`).
- Distributed tracing (OpenTelemetry, Jaeger) and metrics collection (Prometheus) to monitor system health.

## Testing & QA
- Writing comprehensive table-driven tests.
- Utilizing `go test -bench` for performance benchmarking and `go test -fuzz` for fuzz testing.
- Mocking and interface-based testing for high test coverage without brittle tests.
