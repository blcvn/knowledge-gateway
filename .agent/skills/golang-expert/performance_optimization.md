# Performance Optimization & Profiling

## Profiling Tools (`pprof`)
- **CPU Profiling:** Identifying functions that consume the most CPU time.
- **Memory Profiling (Heap):** Finding memory leaks, identifying excessive allocations, and understanding object lifecycles.
- **Goroutine Profiling:** Debugging blocked goroutines, deadlocks, and goroutine leaks.
- **Block & Mutex Profiling:** Identifying contention issues where goroutines are waiting on synchronization primitives.
- Utilizing `go tool pprof` and `go tool trace` to visualize execution timelines and identify latency bottlenecks.

## Code-Level Optimizations
- **Escape Analysis (`go build -gcflags="-m"`):** Understanding why variables escape to the heap and rewriting code to keep them on the stack.
- **Pre-allocating Slices and Maps:** Always using `make([]T, 0, capacity)` or `make(map[K]V, capacity)` when the final size is known to avoid costly reallocations during growth.
- **Avoiding Reflection:** Minimizing the use of the `reflect` package in hot paths, as it is significantly slower than direct type assertions or interface implementations.
- **String Manipulation:** Using `strings.Builder` or `bytes.Buffer` for efficient string concatenation instead of the `+` operator.
- **Pointer Semantics vs. Value Semantics:** Understanding when to pass by value (to avoid heap allocations) and when to pass by pointer (to avoid copying large structs).

## Database & I/O Optimization
- **Connection Pooling:** Configuring `database/sql` properly (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`).
- **Prepared Statements:** Using prepared statements for repeated queries to reduce parsing overhead and prevent SQL injection.
- **Caching:** Implementing multi-level caching strategies (in-memory caching with TTL, distributed caching with Redis) to reduce load on the primary data store.
