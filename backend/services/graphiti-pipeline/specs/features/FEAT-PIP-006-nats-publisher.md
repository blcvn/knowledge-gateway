---
id: FEAT-PIP-006
title: NATS Event Publisher Adapter
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P1
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement NATS JetStream publisher adapter cho graphiti-pipeline. Publish domain events sau khi saga pipeline hoàn thành: `graphiti.episode.ingested`, `graphiti.entity.resolved`, `graphiti.community.rebuilt`.

## Scope

### In Scope
- `internal/adapter/event/nats_publisher.go` — implements EventPublisher port
- NATS JetStream stream configuration (`graphiti` stream)
- Message serialization (Protobuf or JSON)
- Retry logic for transient NATS failures
- Message deduplication via NATS dedup window

### Out of Scope
- NATS subscriber (graphiti-search handles cache invalidation)
- Stream creation (infra/config responsibility)

## Thiết Kế Kỹ Thuật

### Events Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `graphiti.episode.ingested` | `{episode_id, group_id, nodes_count, edges_count, timestamp}` | Saga completed |
| `graphiti.entity.resolved` | `{entity_id, group_id, merged_from[], timestamp}` | Entity merge during resolution |
| `graphiti.community.rebuilt` | `{community_id, group_id, member_count, timestamp}` | Community update completed |

### Implementation

```go
type NATSPublisher struct {
    js   nats.JetStreamContext
    opts []nats.PubOpt
}

func (p *NATSPublisher) PublishEpisodeIngested(ctx context.Context, event domain.EpisodeIngestedEvent) error {
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    _, err = p.js.Publish("graphiti.episode.ingested", data, p.opts...)
    return err
}
```

## Acceptance Criteria

- [ ] AC-1: Given saga completion, When PublishEpisodeIngested called, Then message appears on `graphiti.episode.ingested` subject
- [ ] AC-2: Given NATS temporarily unavailable, When publish fails, Then retry 3x with exponential backoff
- [ ] AC-3: Given duplicate message (same episode_id), Then NATS dedup window prevents duplicate delivery
- [ ] AC-4: All events include `group_id` for downstream tenant-scoped processing

## Test Requirements
- **Unit tests**: Publisher with mock JetStream context
- **Integration tests**: NATS testcontainer publish/subscribe roundtrip
- **Minimum coverage**: 80%
