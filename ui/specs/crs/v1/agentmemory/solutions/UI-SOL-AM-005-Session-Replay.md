# UI Solution: UI-SOL-AM-005 — Session Replay Viewer

**Solution ID:** UI-SOL-AM-005  
**CR References:** [CR-AM-005](../../../../docs/crs/v1/agentmemory/CR-AM-005-Session-Replay.md)  
**Backend Solution:** [SOL-005-Session-Replay.md](../../../../backend/specs/crs/v1/agentmemory/solutions/SOL-005-Session-Replay.md)  
**Feature:** Session Replay — Timeline Scrubbing, Real-time Viewer  
**Priority:** 🟡 High  
**Frontend Component:** `ui/src/pages/sessions/replay/`

---

## 1. Mục Đích

Xây dựng Session Replay Viewer với:
- Timeline scrubber để jump đến bất kỳ thời điểm nào trong session
- Real-time streaming replay qua SSE
- Hook detail panel với full payload
- JSONL import từ Claude Code sessions
- Session summary statistics

---

## 2. Backend API Alignment

### API Endpoints

| Method | Path | Mô tả |
|--------|------|--------|
| `GET` | `/v1/observe/sessions/{id}` | Session detail + all hooks |
| `GET` | `/v1/observe/sessions/{id}/replay` | SSE stream (sequential) |
| `POST` | `/v1/observe/sessions/import` | JSONL import (multipart) |

### TypeScript Types

```typescript
// ui/src/types/replay.ts

interface ReplayState {
  sessionId:      string;
  status:         'idle' | 'loading' | 'playing' | 'paused' | 'complete';
  currentIndex:   number;
  totalHooks:     number;
  playbackSpeed:  1 | 2 | 4;    // 1x, 2x, 4x speed
  visibleHooks:   HookEvent[];
}

interface ReplayControls {
  play:     () => void;
  pause:    () => void;
  seekTo:   (index: number) => void;
  setSpeed: (speed: 1 | 2 | 4) => void;
  reset:    () => void;
}
```

---

## 3. Components Architecture

```
SessionReplayPage
├── ReplayHeader
│   ├── SessionMeta         ← agent_id, started_at, duration, hook_count
│   └── SummaryStats        ← total_tokens, tool_calls count, llm_calls count
├── ReplayMainArea (split layout)
│   ├── TimelinePanel (left 40%)
│   │   ├── ScrubBar        ← progress bar with seek capability
│   │   ├── PlayControls    ← Play/Pause/Reset + Speed (1x/2x/4x)
│   │   └── HookList        ← scrollable list, current hook highlighted
│   │       └── HookItem    ← hook_type + timestamp + mini-preview
│   └── HookDetailPanel (right 60%)
│       ├── HookTypeHeader  ← large hook_type badge + latency
│       └── PayloadViewer   ← syntax-highlighted JSON with expand/collapse
├── ImportButton            ← "Import JSONL" in page header
└── JSONLImportModal        ← drag-drop area + upload progress
```

### Scrub Bar Design
```
──●──────────────────────────────  hook 5 of 47
00:00:05          [  ▶  ] [⏸]  [1x▾]         00:01:00
```

---

## 4. SSE Streaming Implementation

```typescript
// ui/src/api/hooks/useSessionReplayStream.ts

export function useSessionReplayStream(sessionId: string, speed: 1 | 2 | 4) {
  const [state, setState] = useState<ReplayState>({ status: 'idle', ... });
  
  const play = useCallback(() => {
    setState(s => ({ ...s, status: 'playing' }));
    const sse = new EventSource(`/v1/observe/sessions/${sessionId}/replay`);
    
    sse.onmessage = (e) => {
      const hook = JSON.parse(e.data) as HookEvent;
      setState(prev => ({
        ...prev,
        visibleHooks: [...prev.visibleHooks, hook],
        currentIndex: prev.currentIndex + 1,
      }));
    };
    
    sse.addEventListener('done', () => {
      setState(s => ({ ...s, status: 'complete' }));
      sse.close();
    });
    
    return () => sse.close();
  }, [sessionId]);
  
  return { state, play, pause, seekTo, reset };
}
```

---

## 5. JSONL Import

```typescript
// JSONL format (Claude Code):
// {"type":"session_start","session_id":"s_1","timestamp":"...","agent":"claude-code"}
// {"type":"llm_prompt","timestamp":"...","prompt":"...","tokens":2048}
// ...

// Upload handler
async function importJSONL(file: File): Promise<ImportResult> {
  const form = new FormData();
  form.append('file', file, file.name);
  return fetch('/v1/observe/sessions/import', {
    method: 'POST',
    body: form,
    headers: { 'Authorization': `Bearer ${getToken()}` },
  }).then(r => r.json());
}
```

---

## 6. Acceptance Criteria (Frontend)

- [ ] Replay starts playing hooks sequentially via SSE
- [ ] Scrub bar allows jumping to any hook (seek by index)
- [ ] Speed control: 1x/2x/4x playback
- [ ] Current hook highlighted in timeline list và detail panel updates
- [ ] JSONL drag-and-drop với file type validation (`.jsonl` only)
- [ ] Import progress: "Importing 47 of 52 hooks..."
- [ ] Hook detail panel renders JSON syntax-highlighted
- [ ] Session summary stats visible at all times (sticky header)
