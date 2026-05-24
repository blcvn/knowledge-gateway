# Core Skills — Software Architecture & System Design

## Distributed Systems Architecture

### Service Boundary Design
- **Domain-Driven Design (DDD):** Identifying Bounded Contexts and Aggregates to define service boundaries that align with business domains rather than technical layers.
- **Single Responsibility Services:** Each service owns exactly one business capability — it is the sole writer of its data and the authority on its domain.
- **Inter-Service Communication Patterns:**
  - **Synchronous (REST/gRPC):** For request-response interactions requiring immediate results.
  - **Asynchronous (Kafka/RabbitMQ):** For event-driven workflows, fan-out notifications, and decoupling services.
  - **Choosing wisely:** Synchronous coupling is a hidden dependency. Prefer async for anything not on the critical user path.

### Pipeline Architecture (Core for this Platform)
- **Multi-Stage DAG Design:** Modeling the processing pipeline as a Directed Acyclic Graph where each stage has clear input/output contracts.
- **Stage Isolation:** Each stage can fail, retry, and recover independently without corrupting other stages.
- **Checkpointing:** Storing intermediate results at each stage boundary so the pipeline can resume from the last successful checkpoint, not from scratch.
- **Backpressure Management:** Preventing fast producers from overwhelming slow consumers using buffering, rate limiting, and flow control.

## Architecture Decision Records (ADR)

### When to Write an ADR
Write an ADR for any decision that:
- Commits the team to a technology or platform (e.g., "We use Neo4j for the Knowledge Graph")
- Establishes a pattern that will be replicated (e.g., "All services expose health checks at `/health`")
- Has significant trade-offs that future engineers need to understand
- Cannot be reversed cheaply once implemented

### ADR Template
```markdown
# ADR-XXXX: [Title]
- **Date:** YYYY-MM-DD
- **Status:** Proposed | Accepted | Deprecated | Superseded by ADR-XXXX
- **Deciders:** [Names/roles of decision makers]

## Context
[What problem or constraint is driving this decision?]

## Considered Options
1. [Option A]
2. [Option B]
3. [Option C — status quo]

## Decision
[What was chosen and why — be explicit about the primary reasons]

## Consequences
### Positive
- [Benefit 1]
### Negative / Trade-offs
- [Risk or cost 1]
### Risks
- [What could go wrong and how to mitigate]
```

## Scalability & Operational Design
- **Stateless Services:** Services store no session state locally; all state is externalized (Redis, DB). This enables horizontal scaling.
- **Health Check Design:** Every service exposes `/health` (liveness) and `/ready` (readiness) endpoints for orchestrators (Kubernetes).
- **Circuit Breaker Integration:** All calls to external services (LLM APIs, Neo4j, downstream services) are wrapped in circuit breakers to prevent cascading failures.
- **Graceful Degradation:** When a non-critical dependency fails (e.g., traceability validator), the system continues functioning in a degraded mode rather than failing entirely.
