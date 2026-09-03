# TR-007: Memory Management Test Requirements

**Module:** Long-term Memory (remember.ts, evict.ts, auto-forget.ts)  
**Nguồn:** SRS §3.4 (FR-MEM-001..005), Architecture §6, TDD §4.1-4.3, URD §3.4  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho long-term memory management: 6 memory types, versioning qua Jaccard similarity, strength & decay, TTL-based expiry, memory relations.

**Files:** `src/functions/remember.ts`, `src/functions/evict.ts`, `src/functions/auto-forget.ts`

---

## TR-007-MEM-001 — 6 memory types được hỗ trợ
🔴 P0 | `[UNIT]` | **FR-MEM-001**

**Given:** `mem::remember` được gọi với mỗi type  
**When:** Memory được tạo  
**Then:** Memory được lưu với đúng type:

| Type | Example |
|---|---|
| `pattern` | "Use async/await instead of .then() chaining" |
| `preference` | "Prefer functional components" |
| `architecture` | "Auth uses jose middleware" |
| `bug` | "N+1 query in UserRepository.findAll()" |
| `workflow` | "Deploy: build → test → stage → prod" |
| `fact` | "Project uses Tailwind v4, Vitest" |

**Traceability:** FR-MEM-001, SRS §3.4

---

## TR-007-MEM-002 — Memory structure đầy đủ
🔴 P0 | `[UNIT]`

**Given:** `mem::remember` được gọi với params đầy đủ  
**When:** Memory được tạo trong KV  
**Then:** Memory object có đầy đủ fields:
```typescript
{
  id: string,              // "mem_<nanoid>"
  createdAt: string,       // ISO
  updatedAt: string,
  type: MemoryType,
  title: string,
  content: string,
  concepts: string[],
  files: string[],
  sessionIds: string[],
  strength: number,        // default 0.7
  version: number,         // starts at 1
  isLatest: true,
  parentId?: string,
  supersedes: string[],    // empty array
  forgetAfter?: string
}
```

**Traceability:** SRS §6.1, TDD §4.1

---

## TR-007-MEM-003 — Jaccard similarity > 0.7 → supersede
🔴 P0 | `[UNIT]` | **FR-MEM-002**

**Given:**
- Memory M1: type="architecture", concepts=["jose", "authentication", "middleware", "JWT"]

**When:** `mem::remember` với concepts=["jose", "authentication", "JWT", "Edge", "middleware"] (Jaccard=4/5=0.8 > 0.7)  
**Then:**
- M2 được tạo với `supersedes=[M1.id]`, `version=2`, `parentId=M1.id`, `isLatest=true`
- M1 được update: `isLatest=false`
- M1 vẫn còn trong KV (không bị xóa, chỉ marked)

**Traceability:** FR-MEM-002, UR-017, TDD §4.1

---

## TR-007-MEM-004 — Jaccard similarity ≤ 0.7 → new memory (no supersede)
🔴 P0 | `[UNIT]` | **FR-MEM-002**

**Given:**
- Memory M1: concepts=["jose", "authentication"]

**When:** `mem::remember` với concepts=["postgres", "database", "connection-pool"] (Jaccard=0/5=0 < 0.7)  
**Then:**
- M2 được tạo với `version=1`, `supersedes=[]`, `isLatest=true`
- M1 KHÔNG bị thay đổi (`isLatest` vẫn true)

**Traceability:** FR-MEM-002, TDD §4.1

---

## TR-007-MEM-005 — Jaccard similarity calculation
🔴 P0 | `[UNIT]` | **FR-MEM-002**

**Given:** concepts_A = ["jose", "auth", "jwt"], concepts_B = ["jose", "auth", "middleware"]  
**When:** `jaccardSimilarity(A, B)` được tính  
**Then:**
- intersection = {jose, auth} = 2
- union = {jose, auth, jwt, middleware} = 4
- jaccard = 2/4 = 0.5 (< 0.7, không supersede)

**Test Cases:**
| A | B | Expected Jaccard |
|---|---|---|
| ["a","b","c"] | ["a","b","c"] | 1.0 |
| ["a","b"] | ["c","d"] | 0.0 |
| ["a","b","c"] | ["a","b"] | 0.67 |
| [] | [] | 0.0 |

**Traceability:** TDD §4.1 `jaccardSimilarity()`

---

## TR-007-MEM-006 — Jaccard case-insensitive
🟠 P1 | `[UNIT]`

**Given:** concepts_A = ["Auth", "JWT"], concepts_B = ["auth", "jwt"]  
**When:** Jaccard similarity được tính  
**Then:** Jaccard = 1.0 (case-insensitive comparison)

**Traceability:** TDD §4.1

---

## TR-007-MEM-007 — Supersede cascade: multiple old memories
🟠 P1 | `[UNIT]` | **FR-MEM-002**

**Given:** 3 memories M1, M2, M3 (tất cả `isLatest=true`, similar concepts)  
**When:** M4 được create với Jaccard > 0.7 với tất cả 3  
**Then:**
- M4.supersedes = [M1.id, M2.id, M3.id]
- M1, M2, M3: `isLatest=false`
- M4: `isLatest=true`, `version = max(M1.v, M2.v, M3.v) + 1`

**Traceability:** TDD §4.1

---

## TR-007-MEM-008 — Memory được index sau khi tạo
🔴 P0 | `[INT]` | **FR-MEM-001**

**Given:** Memory M mới được tạo  
**When:** `GET /smart-search?q=concepts_from_M`  
**Then:** M xuất hiện trong search results

**Traceability:** TDD §4.1 step 6

---

## TR-007-MEM-009 — Superseded memories bị remove khỏi index
🔴 P0 | `[INT]` | **FR-MEM-002**

**Given:** M1 tồn tại trong BM25 và Vector index  
**When:** M2 supersede M1  
**Then:**
- M1 được remove khỏi BM25 index
- M1 được remove khỏi Vector index
- Search chỉ trả về M2 (không còn M1)

**Traceability:** TDD §4.1 step 4

---

## TR-007-MEM-010 — Strength decay formula
🔴 P0 | `[UNIT]` | **FR-MEM-003**

**Given:** Memory với `strength = 1.0`, `decayDays = 30`  
**When:** `applyDecay()` chạy sau 30 ngày  
**Then:** `strength = 1.0 × 0.9^(30/30) = 0.9`

**Formula:** `strength_new = strength × 0.9^(daysSinceAccess / decayDays)`

**Traceability:** FR-MEM-003, TDD §6.1 Tier 4

---

## TR-007-MEM-011 — Strength minimum floor
🟠 P1 | `[UNIT]`

**Given:** Memory với strength gần 0  
**When:** Decay chạy nhiều lần  
**Then:** `strength` không giảm dưới `0.1` (minimum floor)

**Traceability:** TDD §6.1 Tier 4

---

## TR-007-MEM-012 — TTL-based expiry
🔴 P0 | `[INT]` | **FR-MEM-004**

**Given:** Memory với `forgetAfter = "2026-01-01T00:00:00Z"` (past timestamp)  
**When:** `mem::auto-forget` sweep chạy  
**Then:**
- Memory bị xóa khỏi KV
- Memory bị remove khỏi BM25 và Vector index
- Audit record được tạo cho deletion

**Traceability:** FR-MEM-004, UR-018

---

## TR-007-MEM-013 — Auto-forget interval default 60 phút
🟡 P2 | `[UNIT]` | **FR-MEM-004**

**Given:** Không có `AUTO_FORGET_INTERVAL_MS` env var  
**When:** Config được load  
**Then:** `autoForgetInterval = 3_600_000` (1 giờ tính bằng ms)

**Traceability:** SRS §9.3

---

## TR-007-MEM-014 — Eviction scoring formula
🟠 P1 | `[UNIT]` | **FR-MEM-004**

**Given:** Memory với:
- `strength = 0.8`
- Last accessed 10 days ago
- `sessionIds.length = 5`

**When:** `computeEvictionScore()` chạy  
**Then:**
```
recencyFactor = exp(-10 / halfLifeDays)
frequencyFactor = log(1 + 5) = log(6)
score = 0.8 × recencyFactor × frequencyFactor
```

**Traceability:** TDD §4.2, Architecture §6.2

---

## TR-007-MEM-015 — Memory relations: 5 relation types
🟠 P1 | `[UNIT]` | **FR-MEM-005**

**Given:** 2 memories M1 và M2  
**When:** Relation được tạo  
**Then:** Hỗ trợ đầy đủ 5 relation types:
- `supersedes` — M2 replaces M1
- `extends` — M2 adds to M1
- `derives` — M2 derived from M1
- `contradicts` — M2 conflicts with M1
- `related` — M2 is related to M1

**Traceability:** FR-MEM-005

---

## TR-007-MEM-016 — Memory list: chỉ trả về `isLatest=true`
🟠 P1 | `[INT]`

**Given:** 5 memories, 2 superseded (`isLatest=false`), 3 active (`isLatest=true`)  
**When:** `GET /memories`  
**Then:** Chỉ trả về 3 active memories (không trả về superseded)

**Traceability:** SRS §7.1

---

## TR-007-MEM-017 — Memory search bao gồm memories (không chỉ observations)
🔴 P0 | `[INT]`

**Given:** Memory M được tạo với keyword "jose"  
**When:** Smart search với query "jose middleware"  
**Then:** M xuất hiện trong search results (từ KV.memories fallback trong enrich)

**Traceability:** TDD §3.3 step 7 (Fallback: try KV.memories)

---

## TR-007-MEM-018 — Default strength = 0.7
🟡 P2 | `[UNIT]`

**Given:** `mem::remember` không chỉ định `strength`  
**When:** Memory được tạo  
**Then:** `memory.strength = 0.7`

**Traceability:** TDD §4.1
