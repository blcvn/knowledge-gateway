# llm-embedding-adapter

## Requirements

### Requirement: Pluggable EmbeddingProvider interface
The system SHALL define an `EmbeddingProvider` interface in `internal/platform/vector` with `Embed(ctx, text) ([]float64, error)`, `Dimensions() int`, and `ModelID() string` so that real LLM backends can be swapped without changing service code.

#### Scenario: Embed text via a configured LLM provider
- WHEN the system needs to embed a node's searchable text
- THEN it SHALL call the active `EmbeddingProvider.Embed(ctx, text)`
- AND SHALL propagate the returned error to the caller rather than silently falling back
- AND SHALL NOT call the deterministic fallback if a real provider is configured

#### Scenario: Use the deterministic provider as a test fallback
- WHEN no LLM embedding endpoint is configured
- THEN the system SHALL use `DeterministicProvider` wrapped to satisfy the new interface
- AND tests SHALL pass without any external network calls

### Requirement: HTTPEmbeddingProvider for direct LLM calls
The system SHALL provide an `HTTPEmbeddingProvider` that calls a configurable LLM HTTP endpoint (URL, model ID, API key, timeout).

#### Scenario: Successful LLM embed
- WHEN the LLM endpoint returns a valid embedding vector
- THEN `HTTPEmbeddingProvider.Embed` SHALL return that vector

#### Scenario: Transient LLM endpoint error
- WHEN the LLM endpoint returns 5xx or times out
- THEN `HTTPEmbeddingProvider.Embed` SHALL return an error
- AND the outbox worker SHALL retry the event according to its existing retry/dead-letter policy

### Requirement: EmbeddingRouter as the injection point for service code
The system SHALL define an `EmbeddingRouter` interface that wraps one or more `EmbeddingProvider` instances. `workers.Runtime` and `search.Service` SHALL accept an `EmbeddingRouter`, not a bare `EmbeddingProvider`.

```
EmbeddingRouter interface {
    EmbeddingProvider  // inherits Embed / Dimensions / ModelID
    RouteContext(tenantID, domainID string) EmbeddingProvider
}
```

`RouteContext` returns the appropriate provider for a given caller context. The default `DirectRouter` delegates all calls to a single provider. A `RoutingRouter` can select different models for different tenants or domains.

#### Scenario: Route different tenants to different LLM models
- WHEN a `RoutingRouter` is configured with per-tenant model assignments
- AND `workers.Runtime` processes a node belonging to tenant T
- THEN `EmbeddingRouter.RouteContext(T, domainID)` SHALL return the provider configured for T
- AND `workers.Runtime` SHALL call that provider's `Embed` without any tenant-aware logic in runtime code

### Requirement: Middleware chain for cross-cutting embedding concerns
The system SHALL allow composing `EmbeddingProvider` middleware layers (cache, retry, proxy) by wrapping providers. Each middleware implements `EmbeddingProvider` and delegates to the next layer.

```
EmbeddingRouter
  └─ CachingProvider(ttl=5m)
       └─ RetryProvider(maxAttempts=3, backoff=exponential)
            └─ ProxyHTTPProvider(proxyURL="https://proxy.internal/embed")
                 └─ HTTPEmbeddingProvider(url, model, apiKey)
```

#### Scenario: Call LLM through an HTTP proxy
- WHEN the deployment requires all outbound LLM calls to pass through an internal proxy
- THEN a `ProxyHTTPProvider` wrapping an `HTTPEmbeddingProvider` SHALL be registered at bootstrap
- AND `workers.Runtime` and `search.Service` SHALL be unchanged — they call `EmbeddingRouter.Embed` as always

#### Scenario: Cache embedding results for repeated text
- WHEN a `CachingProvider` is in the chain with a configured TTL
- AND `Embed` is called for a text that was embedded recently
- THEN the cached vector SHALL be returned without a downstream HTTP call

#### Scenario: Switch from direct call to proxy without code change
- WHEN the proxy layer is added or removed at bootstrap configuration
- THEN no changes SHALL be required in `workers.Runtime`, `search.Service`, or any service test

### Requirement: Embedding is computed in the outbox worker, not in the write transaction
- WHEN a `NODE_UPSERTED` or `NODE_DELETED` event is processed
- THEN `workers.Runtime` SHALL call `EmbeddingRouter.Embed` (via the resolved provider) after the transaction commits
- AND SHALL NOT call the embedding provider inside the Postgres write transaction
