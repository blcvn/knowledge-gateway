# TASK-PLAT-019 — WebSocket Event Buffer (Redis Ring)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-019 |
| **Wave** | 4 (Events) |
| **Solution** | [SOL-PLAT-006](../solutions/SOL-PLAT-006-WebSocket-Realtime-Events.md) §2.3 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-PLAT-017 |
| **Estimated** | 2h |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** Event buffer in ws.go: basic in-memory buffer; no durable queue / missed-event replay
---

## Mục tiêu

Implement `WSEventBuffer` sử dụng Redis Sorted Set để buffer last 100 events per tenant (24h TTL). Support replay từ `last_event_id`.

---

## Công việc cụ thể

### 1. Tạo `gateway/adapter/handler/ws_buffer.go` [NEW]

```go
package handler

import (
    "context"
    "fmt"
    "github.com/redis/go-redis/v9"
)

const (
    wsEventBufferKeyFmt = "ws:events:%s"   // ws:events:{tenant_id}
    wsMaxBufferSize     = 100
    wsBufferTTL         = 24 * time.Hour
)

// WSEventBuffer stores recent events per tenant in Redis sorted set
// Score = UnixNano timestamp (enables chronological ordering + range queries)
type WSEventBuffer struct {
    redis redis.UniversalClient
}

func NewWSEventBuffer(redisClient redis.UniversalClient) *WSEventBuffer {
    return &WSEventBuffer{redis: redisClient}
}

// Store adds an event to the tenant's buffer ring (keeps last 100)
func (b *WSEventBuffer) Store(ctx context.Context, tenantID string, payload []byte) error {
    key := fmt.Sprintf(wsEventBufferKeyFmt, tenantID)
    score := float64(time.Now().UnixNano())

    // Add event: score = timestamp, member = timestamp:payload
    member := fmt.Sprintf("%d:%s", time.Now().UnixNano(), string(payload))

    pipe := b.redis.Pipeline()
    pipe.ZAdd(ctx, key, redis.Z{Score: score, Member: member})
    // Trim to last 100 events
    pipe.ZRemRangeByRank(ctx, key, 0, -(wsMaxBufferSize + 1))
    pipe.Expire(ctx, key, wsBufferTTL)
    _, err := pipe.Exec(ctx)
    return err
}

// Replay sends all events newer than lastEventID to the client
// lastEventID is the UnixNano timestamp (as string) of the last received event
func (b *WSEventBuffer) Replay(ctx context.Context, client *WSClient, tenantID, lastEventID string) {
    key := fmt.Sprintf(wsEventBufferKeyFmt, tenantID)

    // Parse last event ID as score
    lastScore, err := strconv.ParseFloat(lastEventID, 64)
    if err != nil {
        return
    }

    // Get events with score > lastScore (events after last received)
    members, err := b.redis.ZRangeByScore(ctx, key, &redis.ZRangeBy{
        Min: fmt.Sprintf("(%g", lastScore), // exclusive: strictly after lastEventID
        Max: "+inf",
    }).Result()
    if err != nil {
        return
    }

    count := 0
    for _, member := range members {
        // Strip "timestamp:" prefix to get raw payload
        idx := strings.Index(member, ":")
        if idx < 0 {
            continue
        }
        payload := member[idx+1:]
        if !client.Send([]byte(payload)) {
            break // client buffer full, stop replay
        }
        count++
    }

    if count > 0 {
        // Send replay complete notification
        client.SendJSON(map[string]interface{}{
            "event": "replay_complete",
            "data":  map[string]int{"replayed_count": count},
        })
    }
}

// GetLastEventID returns the score (UnixNano) of the most recent event for tenant
func (b *WSEventBuffer) GetLastEventID(ctx context.Context, tenantID string) (string, error) {
    key := fmt.Sprintf(wsEventBufferKeyFmt, tenantID)
    members, err := b.redis.ZRevRangeWithScores(ctx, key, 0, 0).Result()
    if err != nil || len(members) == 0 {
        return "", err
    }
    return fmt.Sprintf("%g", members[0].Score), nil
}
```

### 2. Unit tests `gateway/adapter/handler/ws_buffer_test.go` [NEW]

```go
package handler_test

func TestWSEventBuffer_StoreAndReplay(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    buf := NewWSEventBuffer(rdb)

    // Store 3 events
    buf.Store(ctx, "tenant-1", []byte(`{"event":"memory.stored","data":{"id":"1"}}`))
    time.Sleep(time.Millisecond)
    t1 := strconv.FormatInt(time.Now().UnixNano(), 10)
    time.Sleep(time.Millisecond)
    buf.Store(ctx, "tenant-1", []byte(`{"event":"memory.stored","data":{"id":"2"}}`))
    buf.Store(ctx, "tenant-1", []byte(`{"event":"memory.stored","data":{"id":"3"}}`))

    // Create test client
    client := &WSClient{tenantID: "tenant-1", send: make(chan []byte, 10)}

    // Replay from t1 — should receive events 2 and 3
    buf.Replay(ctx, client, "tenant-1", t1)

    var received [][]byte
    for len(client.send) > 0 {
        received = append(received, <-client.send)
    }
    // Expect 2 events + replay_complete notification
    assert.GreaterOrEqual(t, len(received), 2)
}

func TestWSEventBuffer_MaxBufferSize(t *testing.T) {
    mr := miniredis.RunT(t)
    rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    buf := NewWSEventBuffer(rdb)

    // Store 110 events (10 over limit)
    for i := 0; i < 110; i++ {
        buf.Store(ctx, "tenant-1", []byte(fmt.Sprintf(`{"i":%d}`, i)))
    }

    key := fmt.Sprintf(wsEventBufferKeyFmt, "tenant-1")
    count, _ := rdb.ZCard(ctx, key).Result()
    assert.Equal(t, int64(wsMaxBufferSize), count, "should keep only last 100")
}
```

---

## Acceptance Criteria

- [ ] `Store()` adds event with UnixNano score to Redis sorted set
- [ ] Buffer trimmed to last 100 events after each store
- [ ] Redis key has 24h TTL (auto-expires)
- [ ] `Replay()` returns only events AFTER `lastEventID` (exclusive)
- [ ] `Replay()` stops gracefully if client send buffer is full
- [ ] Multiple tenants isolated: separate Redis keys (`ws:events:{tenant_id}`)
- [ ] `go test ./gateway/adapter/handler/...` passes

## Files

```
gateway/adapter/handler/ws_buffer.go       [NEW]
gateway/adapter/handler/ws_buffer_test.go  [NEW]
```
