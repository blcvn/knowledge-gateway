# TR-005: Vector Index Test Requirements

**Module:** VectorIndex  
**Nguồn:** SRS §3.5 (FR-SEARCH-001), Architecture §5.2, TDD §3.2  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho vector similarity search engine sử dụng cosine similarity với Float32Array embeddings (384 dimensions cho local model, configurable cho cloud providers).

**File:** `src/state/vector-index.ts`

---

## TR-005-VEC-001 — Cosine similarity: identical vectors = 1.0
🔴 P0 | `[UNIT]`

**Given:** 2 identical Float32Array[384]  
**When:** `cosineSimilarity(a, b)` được tính  
**Then:** Score = 1.0

**Traceability:** TDD §3.2, Architecture §5.2

---

## TR-005-VEC-002 — Cosine similarity: orthogonal vectors = 0.0
🟠 P1 | `[UNIT]`

**Given:** 2 perpendicular vectors (dot product = 0)  
**When:** `cosineSimilarity(a, b)` được tính  
**Then:** Score = 0.0 (hoặc xấp xỉ)

**Traceability:** TDD §3.2

---

## TR-005-VEC-003 — Top-K search: chính xác
🔴 P0 | `[UNIT]`

**Given:** VectorIndex với 100 documents, query vector gần giống doc_42  
**When:** `vector.search(queryVector, 5)`  
**Then:**
- doc_42 nằm trong top-5
- Results được sắp xếp DESC theo score
- Không có duplicates

**Traceability:** TDD §3.2, Architecture §5.2

---

## TR-005-VEC-004 — Online top-K: không sort toàn bộ
🟡 P2 | `[UNIT]`

**Given:** VectorIndex với 50K documents, limit=20  
**When:** `vector.search(query, 20)`  
**Then:** Chỉ maintain top-20 buffer, không sort toàn bộ 50K docs

**Traceability:** TDD §3.2, §12.1

---

## TR-005-VEC-005 — Dimension validation: reject mismatch
🔴 P0 | `[UNIT]` | **FR-OBS-003**

**Given:** VectorIndex với active embedding provider = 384-dim  
**When:** Embedding 768-dim được add  
**Then:**
- `validateDimensions(384)` trả về `{mismatches: [{obsId, actualDim: 768}]}`
- Mismatch observation được loại khỏi search

**Traceability:** TDD §14.2, SRS §4.6, Architecture §5.2

---

## TR-005-VEC-006 — Dimension validation on restore
🔴 P0 | `[UNIT]`

**Given:** `vector-index.json` được lưu với 768-dim embeddings  
**When:** VectorIndex load với provider mới (384-dim)  
**Then:**
- `validateDimensions(384)` fail
- IF `AGENTMEMORY_DROP_STALE_INDEX=true` → index cleared và rebuilt
- IF flag không set → startup error logged

**Traceability:** SRS §4.6, TDD §11.2, Architecture §5.2

---

## TR-005-VEC-007 — Serialization: Buffer pool slicing bug
🔴 P0 | `[UNIT]`

**Given:** Float32Array được tạo từ Buffer pool (slice)  
**When:** `serialize()` → `deserialize()` round-trip  
**Then:**
- Mỗi float value preserved với precision ≥ 5 decimal places
- Không có data corruption từ buffer slicing

**Test Method:**
```typescript
const original = new Float32Array(384).fill(Math.random());
const serialized = idx.serialize();
const restored = VectorIndex.deserialize(serialized);
for (let i = 0; i < 384; i++) {
  expect(restoredVec[i]).toBeCloseTo(original[i], 5);
}
```

**Traceability:** TDD §14.2, Architecture §5.2

---

## TR-005-VEC-008 — Serialization format
🟠 P1 | `[UNIT]`

**Given:** VectorIndex với documents  
**When:** `serialize()`  
**Then:** JSON format:
```json
[[obsId, {"embedding": "base64...", "sessionId": "sess_xxx"}], ...]
```

**Traceability:** TDD §3.2

---

## TR-005-VEC-009 — Add và remove
🔴 P0 | `[UNIT]`

**Given:** Document `obs1` trong index  
**When:** `vector.remove("obs1")`  
**Then:**
- `obs1` không xuất hiện trong search results
- `vectors.size` giảm 1

**Traceability:** TDD §3.2

---

## TR-005-VEC-010 — Empty index search
🟡 P2 | `[UNIT]`

**Given:** VectorIndex trống  
**When:** `vector.search(queryVec, 10)`  
**Then:** Trả về `[]`, không throw

**Traceability:** TDD §3.2

---

## TR-005-VEC-011 — Local embedding provider: all-MiniLM-L6-v2
🟠 P1 | `[INT]` | **FR-SEARCH-001**

**Given:** `EMBEDDING_PROVIDER=local`  
**When:** `embeddingProvider.embed("JWT authentication middleware")`  
**Then:**
- Trả về Float32Array với `length = 384`
- Vector normalized (L2 norm ≈ 1.0)
- Lazy load ONNX model (load lần đầu, cache sau)

**Traceability:** TDD §8.2, Architecture §5.2

---

## TR-005-VEC-012 — Semantic similarity: related concepts
🟠 P1 | `[INT]` | **FR-SEARCH-001**

**Given:** Documents:
- obs1 với text "N+1 database query optimization using eager loading"
- obs2 với text "React component lifecycle"

**When:** Query: "SQL performance improvement"  
**Then:** cosine_similarity(embed(query), embed(obs1)) > cosine_similarity(embed(query), embed(obs2))

**Traceability:** UR-012, UR-013, SRS §3.5

---

## TR-005-VEC-013 — Memory usage: Float32Array vs number[]
🟡 P2 | `[PERF]`

**Given:** 10K documents với 384-dim embeddings  
**When:** Tất cả được load vào VectorIndex  
**Then:** Heap usage ≤ 16MB (10K × 384 × 4 bytes = 15.36MB)

**Traceability:** TDD §12.2
