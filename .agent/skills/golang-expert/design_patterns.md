# Golang Design Patterns for Stability & Performance

## Concurrency Patterns
1. **Worker Pool:** Limiting the number of active goroutines to prevent resource exhaustion (OOM) when processing a massive number of tasks.
2. **Fan-Out / Fan-In:** Distributing work across multiple goroutines (Fan-Out) and multiplexing their results into a single channel (Fan-In).
3. **Pipeline Pattern:** Processing streams of data in stages, where each stage is a group of goroutines connected by channels.
4. **Context Cancellation:** Propagating cancellation signals across multiple goroutines to gracefully shut down operations and free resources.

## Stability & Resilience Patterns
1. **Circuit Breaker:** Preventing cascading failures by wrapping remote service calls. If a service is failing, the circuit "trips" and returns an immediate error, giving the failing service time to recover.
2. **Retry with Exponential Backoff and Jitter:** Handling transient network or service errors gracefully without overwhelming the downstream service.
3. **Bulkhead Pattern:** Isolating system components (e.g., using separate goroutine pools for different endpoints) so a failure in one does not bring down the entire system.
4. **Rate Limiting (Token Bucket / Leaky Bucket):** Throttling incoming requests to protect the system from sudden traffic spikes or DDoS attacks.

## Performance & Optimization Patterns
1. **Functional Options Pattern:** Providing clean and extensible APIs for configuring complex objects without breaking backwards compatibility.
2. **Object Pooling (`sync.Pool`):** Reusing objects to reduce heap allocations and minimize garbage collection overhead.
3. **Batching:** Grouping multiple small operations (e.g., database inserts, external API calls) into a single larger operation to reduce network and I/O overhead.
4. **Flyweight:** Minimizing memory usage by sharing as much data as possible with similar objects.
