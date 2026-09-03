# TR-009: Memory Consolidation Pipeline Test Requirements

**Module:** Consolidation Pipeline (consolidation-pipeline.ts, summarize.ts)  
**Nguồn:** SRS §3.12 (FR-CONSOL-001..004), Architecture §4.3, TDD §6.1-6.2  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho 4-tier memory consolidation pipeline: Raw Observations → Compressed → Session Summaries → Long-term Memories, cùng với decay scoring và procedural memory extraction.

---

## TR-009-CON-001 — 4-tier pipeline structure
🔴 P0 | `[INT]` | **FR-CONSOL-001**

**Given:** Timer trigger (hoặc manual `mem::consolidate-pipeline`)  
**When:** Pipeline chạy  
**Then:** 4 tiers được xử lý theo thứ tự:
1. **Semantic tier**: SessionSummary[] → SemanticMemory[]
2. **Reflect tier**: Memory[] → Insight[]
3. **Procedural tier**: Pattern Memory[] → ProceduralMemory[]
4. **Decay tier**: Strength decay applied

**Traceability:** FR-CONSOL-001, TDD §6.1

---

## TR-009-CON-002 — Consolidation interval default 2 giờ
🟡 P2 | `[UNIT]`

**Given:** Không có `CONSOLIDATION_INTERVAL_MS`  
**When:** Config load  
**Then:** `consolidationInterval = 7_200_000` (2 giờ)

**Traceability:** SRS §9.3

---

## TR-009-CON-003 — Semantic tier: min 5 summaries required
🔴 P0 | `[INT]` | **FR-CONSOL-002**

**Given:** Chỉ có 3 SessionSummaries  
**When:** Semantic tier của consolidation chạy  
**Then:** LLM không được gọi, semantic extraction skip

**Traceability:** TDD §6.1, Architecture §4.3

---

## TR-009-CON-004 — Semantic tier: extract facts từ summaries
🟠 P1 | `[INT]` | **FR-CONSOL-002**

**Given:** 5+ SessionSummaries với content về JWT auth  
**When:** Semantic tier chạy với LLM provider  
**Then:**
- `SemanticMemory` được tạo: "Project uses jose for JWT authentication"
- `confidence` score được parse từ `<fact confidence="0.8">...</fact>` XML

**Traceability:** TDD §6.1, Architecture §4.3

---

## TR-009-CON-005 — Semantic tier: merge duplicate facts
🟠 P1 | `[UNIT]` | **FR-CONSOL-002**

**Given:** SemanticMemory đã tồn tại: "Project uses jose" (confidence=0.7)  
**When:** Consolidation extract cùng fact (case-insensitive match)  
**Then:**
- Không tạo duplicate, update existing
- `confidence` updated (tăng)
- `accessCount++`

**Traceability:** TDD §6.1

---

## TR-009-CON-006 — Procedural tier: chỉ extract recurring patterns (freq ≥ 2)
🔴 P0 | `[INT]` | **FR-CONSOL-003**

**Given:** Memory patterns với `sessionIds.length`:
- P1: 1 session (not recurring)
- P2: 3 sessions (recurring)
- P3: 2 sessions (recurring)

**When:** Procedural tier chạy  
**Then:**
- P2 và P3 được sent to LLM
- P1 bị skip
- `ProceduralMemory` được tạo chỉ cho P2 và P3

**Traceability:** FR-CONSOL-003, TDD §6.1

---

## TR-009-CON-007 — Procedural extraction: XML parsing
🟠 P1 | `[UNIT]`

**Given:** LLM output XML:
```xml
<procedure name="deploy-sequence" trigger="ready to deploy">
  <step>Run npm run build</step>
  <step>Execute tests</step>
  <step>Deploy to staging</step>
</procedure>
```
**When:** XML được parsed  
**Then:**
- `ProceduralMemory.name = "deploy-sequence"`
- `triggerCondition = "ready to deploy"`
- `steps = ["Run npm run build", "Execute tests", "Deploy to staging"]`

**Traceability:** TDD §6.1, Architecture §4.3

---

## TR-009-CON-008 — Procedural merge: tăng frequency
🟠 P1 | `[UNIT]` | **FR-CONSOL-003**

**Given:** ProceduralMemory "deploy-sequence" đã tồn tại  
**When:** Consolidation extract cùng procedure (exact name match)  
**Then:**
- `frequency++`
- `strength += 0.1` (capped at 1.0)
- Không tạo duplicate

**Traceability:** TDD §6.1

---

## TR-009-CON-009 — Decay tier: formula đúng
🔴 P0 | `[UNIT]` | **FR-CONSOL-004**

**Given:** SemanticMemory với `strength = 0.8`, `decayDays = 30`  
**When:** Decay chạy sau 15 ngày không access  
**Then:**
- `strength_new = 0.8 × 0.9^(15/30) = 0.8 × 0.9^0.5 ≈ 0.758`

**Traceability:** FR-CONSOL-004, TDD §6.1 Tier 4

---

## TR-009-CON-010 — Decay: minimum floor 0.1
🟠 P1 | `[UNIT]`

**Given:** Memory với strength = 0.05  
**When:** Decay chạy  
**Then:** strength không giảm xuống dưới 0.1

**Traceability:** TDD §6.1

---

## TR-009-CON-011 — Session summarization: LLM path
🔴 P0 | `[INT]` | **FR-COMPRESS-003**

**Given:** Session với 10 CompressedObservations, LLM provider available  
**When:** `mem::summarize({sessionId})` được gọi  
**Then:** SessionSummary được tạo với:
```typescript
{
  sessionId: string,
  title: string,           // 1 câu
  narrative: string,       // 2-3 đoạn
  keyDecisions: string[],
  filesModified: string[],
  concepts: string[],
  observationCount: number
}
```

**Traceability:** FR-COMPRESS-003, TDD §6.2

---

## TR-009-CON-012 — Session summarization: idempotent
🟠 P1 | `[INT]`

**Given:** SessionSummary đã tồn tại cho session `sess_abc`  
**When:** `mem::summarize({sessionId: "sess_abc"})` gọi lần 2  
**Then:**
- LLM KHÔNG được gọi lại
- Existing summary được trả về
- `force=true` parameter override this behavior

**Traceability:** TDD §6.2
