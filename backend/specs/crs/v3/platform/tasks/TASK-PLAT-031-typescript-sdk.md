# TASK-PLAT-031 — TypeScript SDK Implementation

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-031 |
| **Wave** | 4 |
| **Solution** | [SOL-PLAT-010](../solutions/SOL-PLAT-010-SDK-Client-Libraries.md) §3 |
| **Component** | `sdk/typescript/` |
| **Priority** | 🟡 High |
| **Depends On** | — |
| **Estimated** | 4h |

---

## Mục tiêu

Tạo TypeScript SDK với full type safety, WebSocket EventsAPI, auto-retry.

---

## Công việc cụ thể

### 1. Tạo `sdk/typescript/package.json` [NEW]

```json
{
  "name": "@vnp/memory-sdk",
  "version": "0.1.0",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": { "build": "tsc", "test": "jest" },
  "dependencies": {},
  "devDependencies": { "typescript": "^5", "jest": "^29", "@types/jest": "^29" }
}
```

### 2. Tạo `sdk/typescript/src/transport.ts` [NEW]

```typescript
const RETRYABLE = new Set([429, 500, 502, 503, 504]);
const MAX_RETRIES = 3;

export class Transport {
  constructor(private config: { apiKey: string; baseUrl: string }) {}

  async post<T>(path: string, body: unknown): Promise<T> {
    let delay = 1000;
    for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
      const resp = await fetch(this.config.baseUrl + path, {
        method: 'POST',
        headers: { 'X-API-Key': this.config.apiKey, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!RETRYABLE.has(resp.status)) {
        if (!resp.ok) throw new VNPError(resp.status, await resp.text());
        return resp.json() as T;
      }
      if (attempt === MAX_RETRIES) throw new VNPError(resp.status, 'max retries exceeded');
      const retryAfter = resp.headers.get('Retry-After');
      delay = retryAfter ? parseInt(retryAfter) * 1000 : delay;
      await sleep(delay);
      delay = Math.min(delay * 2, 30000);
    }
    throw new VNPError(0, 'unreachable');
  }
}

const sleep = (ms: number) => new Promise(r => setTimeout(r, ms));
```

### 3. Tạo `sdk/typescript/src/events.ts` [NEW]

```typescript
import { EventEmitter } from 'events';

export class EventsAPI extends EventEmitter {
  private ws: WebSocket | null = null;
  private reconnectDelay = 1000;
  private shouldReconnect = true;

  constructor(private baseWsUrl: string) { super(); }

  async connect(token: string): Promise<void> {
    this.shouldReconnect = true;
    const url = `${this.baseWsUrl}/v1/console/ws?token=${encodeURIComponent(token)}`;
    this.ws = new WebSocket(url);

    this.ws.onopen = () => { this.reconnectDelay = 1000; this.emit('connected'); };
    this.ws.onmessage = (e: MessageEvent) => {
      const event = JSON.parse(e.data as string);
      this.emit(event.event, event.data, event.timestamp);
    };
    this.ws.onerror = (e) => this.emit('error', e);
    this.ws.onclose = () => {
      if (this.shouldReconnect) {
        setTimeout(() => this.connect(token), this.reconnectDelay);
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
      }
    };
  }

  disconnect(): void {
    this.shouldReconnect = false;
    this.ws?.close();
    this.ws = null;
  }
}
```

### 4. Tests `sdk/typescript/tests/memory.test.ts` [NEW]

```typescript
import { MemoryAPI } from '../src/memory';
import { Transport } from '../src/transport';

jest.mock('../src/transport');

test('store sends correct payload', async () => {
  const mockPost = jest.fn().mockResolvedValue({ id: 'mem-1', engine: 'graphiti' });
  const transport = { post: mockPost } as any;
  const api = new MemoryAPI(transport);
  const result = await api.store({ content: 'Hello', type: 'episodic', userId: 'u1' });
  expect(result.id).toBe('mem-1');
  expect(mockPost).toHaveBeenCalledWith('/v1/memory/store', expect.objectContaining({ content: 'Hello' }));
});
```

---

## Acceptance Criteria

- [ ] `VNPClient.memory.store()` → TypeScript strict mode compiles
- [ ] `EventsAPI.connect(token)` → WebSocket upgrade, events emitted
- [ ] `events.on('memory_stored', handler)` works
- [ ] Auto-reconnect with exponential backoff
- [ ] `npm test` passes

## Files

```
sdk/typescript/package.json
sdk/typescript/tsconfig.json
sdk/typescript/src/client.ts
sdk/typescript/src/transport.ts
sdk/typescript/src/memory.ts
sdk/typescript/src/observe.ts
sdk/typescript/src/events.ts
sdk/typescript/src/types.ts
sdk/typescript/tests/memory.test.ts
sdk/typescript/tests/events.test.ts
```
