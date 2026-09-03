# UI Solution: UI-SOL-PLAT-006 — WebSocket Realtime Events UI

**Solution ID:** UI-SOL-PLAT-006  
**CR References:** [CR-PLAT-006](../../../../docs/crs/v3/platform/CR-PLAT-006-WebSocket-Realtime-Events.md)  
**Feature:** WebSocket — Realtime Connection Status, Live Feed, Reconnect Logic  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/lib/websocket.ts`, `ui/src/components/layouts/RealtimeFeed.tsx`

---

## 1. Mục Đích

Implement WebSocket realtime event streaming:
- Global WS connection manager với auto-reconnect
- Connection status indicator (header bar)
- Live event feed (toast/notification tray)
- `queryClient.setQueryData` updates on incoming events
- Missed event replay on reconnect

---

## 2. Backend API Contract

```http
GET /v1/console/ws?token=<JWT>   → WebSocket upgrade
→ Server sends:
   {"event": "connected", "data": {"tenant_id": "..."}}
   {"event": "memory_stored", "data": {...}}
   {"event": "health_change", "data": {...}}
   {"event": "session_end", "data": {...}}
   {"event": "rate_limit_exceeded", "data": {...}}
   {"event": "observe_event", "data": {...}}
   {"event": "pipeline_complete", "data": {...}}

GET /v1/console/events → SSE fallback
```

---

## 3. WebSocket Manager

```typescript
// ui/src/lib/websocket.ts — singleton WS manager

class WebSocketManager {
  private ws: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;  // 1s → 2s → 4s → ... → 30s max
  private lastEventId: string | null = null;
  private listeners = new Map<string, Set<(data: unknown) => void>>();
  
  connect(token: string) {
    const url = `${WS_BASE_URL}/v1/console/ws?token=${encodeURIComponent(token)}`;
    this.ws = new WebSocket(url);
    
    this.ws.onopen = () => {
      this.reconnectDelay = 1000;   // reset backoff
      // Replay missed events if reconnecting
      if (this.lastEventId) {
        this.ws?.send(JSON.stringify({ last_event_id: this.lastEventId }));
      }
    };
    
    this.ws.onmessage = (e) => {
      const msg = JSON.parse(e.data);
      this.lastEventId = msg.id ?? this.lastEventId;
      this.emit(msg.event, msg.data);
    };
    
    this.ws.onclose = () => this.scheduleReconnect(token);
  }
  
  private scheduleReconnect(token: string) {
    this.reconnectTimer = setTimeout(() => {
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30_000);
      this.connect(token);
    }, this.reconnectDelay);
  }
  
  on(event: string, cb: (data: unknown) => void) {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event)!.add(cb);
    return () => this.listeners.get(event)?.delete(cb);
  }
  
  private emit(event: string, data: unknown) {
    this.listeners.get(event)?.forEach(cb => cb(data));
    this.listeners.get('*')?.forEach(cb => cb({ event, data }));
  }
}

export const wsManager = new WebSocketManager();
```

---

## 4. React Integration

```typescript
// ui/src/hooks/useRealtimeUpdates.ts
// Connects WS events to TanStack Query cache

export function useRealtimeUpdates() {
  const qc = useQueryClient();
  
  useEffect(() => {
    const handlers = [
      wsManager.on('health_change', (data: EngineHealth) => {
        qc.setQueryData(['dashboard', 'health'], (old: EngineHealth[]) =>
          old?.map(h => h.name === data.name ? { ...h, ...data } : h)
        );
      }),
      
      wsManager.on('memory_stored', (data) => {
        // Invalidate memory search results
        qc.invalidateQueries({ queryKey: ['memory', 'search'] });
      }),
      
      wsManager.on('session_end', (data: { session_id: string }) => {
        qc.invalidateQueries({ queryKey: ['sessions'] });
      }),
      
      wsManager.on('pipeline_complete', (data) => {
        qc.invalidateQueries({ queryKey: ['pipelines'] });
      }),
    ];
    
    return () => handlers.forEach(cleanup => cleanup());
  }, [qc]);
}
```

---

## 5. Connection Status Indicator (Header)

```typescript
// ui/src/components/layouts/Header/WSStatusBadge.tsx

type WSStatus = 'connected' | 'reconnecting' | 'disconnected';

const STATUS_STYLES: Record<WSStatus, string> = {
  connected:    'bg-green-500',
  reconnecting: 'bg-amber-500 animate-pulse',
  disconnected: 'bg-red-500',
};

const STATUS_LABELS: Record<WSStatus, string> = {
  connected:    'Live',
  reconnecting: 'Reconnecting...',
  disconnected: 'Offline',
};

// Shows in top navigation bar: ● Live  (green dot)
```

---

## 6. Live Event Feed (Notification Tray)

```
LiveEventsTray (right side panel, toggleable)
├── TrayHeader              ← "Live Events (23)" + Clear button
└── EventList
    └── EventEntry
        ├── EventIcon       ← per event type icon
        ├── EventType       ← "memory_stored" badge
        ├── EventData       ← brief summary (engine, user_id, etc.)
        └── Timestamp       ← "2s ago"
```

---

## 7. Acceptance Criteria (Frontend)

- [ ] WS connects with JWT in query param on app load
- [ ] Connection status indicator: green=connected, amber=reconnecting, red=offline
- [ ] `health_change` event → dashboard health grid updates without page refresh
- [ ] `memory_stored` event → invalidates memory search cache
- [ ] Exponential backoff: 1s → 2s → 4s → ... → 30s max
- [ ] `last_event_id` sent on reconnect to replay missed events
- [ ] SSE fallback: if WS unavailable, connect to `/v1/console/events`
- [ ] Live events tray: shows last 50 events, clearable
