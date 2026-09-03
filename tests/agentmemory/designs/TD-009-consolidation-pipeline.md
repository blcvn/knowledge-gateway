# TD-009: Consolidation Pipeline Test Design

**Liên kết Requirements:** [TR-009-consolidation-pipeline.md](../requirements/TR-009-consolidation-pipeline.md)  
**Source:** `references/agentmemory/src/functions/consolidation-pipeline.ts`  
**Test file:** `tests/agentmemory/specs/consolidation-pipeline.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Consolidation pipeline tổng hợp observations thành session summary và long-term memories, được trigger tự động khi đạt ngưỡng observation count.

**Các điểm kiểm thử:**
- Trigger điều kiện (observation count threshold)
- Session summary generation
- Long-term memory extraction
- Idempotency (trigger không xảy ra đồng thời 2 lần)
- Cleanup observations sau consolidation

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| Threshold | Boundary: threshold-1, threshold, threshold+1 |
| Idempotency | Concurrent trigger attempts |
| Summary content | Verify summary fields trong KV |
| Memory creation | Verify memories được tạo từ summary |

---

## 3. Test Cases

### Group A: Consolidation Trigger

#### TC-001 — Không trigger khi dưới ngưỡng
**Requirement:** TR-009-CON-001 | **Type:** integration | 🔴 P0

**Given:** Session có `CONSOLIDATION_THRESHOLD - 1` observations (ví dụ 9 nếu threshold=10)  
**When:** Hook mới được observe  
**Then:** Không có consolidation process nào bắt đầu

---

#### TC-002 — Trigger khi đạt đúng ngưỡng
**Requirement:** TR-009-CON-001 | **Type:** integration | 🔴 P0

**Given:** Session có đúng `CONSOLIDATION_THRESHOLD` observations  
**When:** Hook thứ `THRESHOLD` được observe  
**Then:** Consolidation được trigger (kiểm tra qua presence của summary trong KV)

---

#### TC-003 — Trigger mỗi N observations (không chỉ lần đầu)
**Type:** integration | 🟠 P1

**Given:** Threshold = 10, session đã được consolidate sau obs 10  
**When:** Thêm 10 observations nữa (obs 11-20)  
**Then:** Consolidation được trigger lại sau obs 20

---

### Group B: Session Summary

#### TC-004 — Session summary được tạo trong KV
**Requirement:** TR-009-CON-002 | **Type:** integration | 🔴 P0

**Given:** Consolidation được trigger  
**When:** Consolidation hoàn thành  
**Then:** KV `mem:summaries` có entry với key = sessionId

---

#### TC-005 — Summary có đúng structure
**Type:** integration | 🔴 P0

**Given:** Session với 10 observations consolidate  
**When:** Summary được đọc từ KV  
**Then:** Summary có:
- `sessionId`
- `observationCount` = số obs được consolidate
- `timeRangeStart`, `timeRangeEnd`
- `keyFacts[]` hoặc `narrative` (tổng hợp nội dung)
- `generatedAt`

---

### Group C: Memory Extraction

#### TC-006 — Memories được extract từ summary
**Requirement:** TR-009-CON-003 | **Type:** integration | 🟠 P1

**Given:** Observations chứa nội dung có thể trở thành memory (patterns, decisions)  
**When:** Consolidation hoàn thành  
**Then:** Ít nhất 1 memory mới được tạo trong `mem:memories` liên kết với session

---

#### TC-007 — Duplicate memories không được tạo (Jaccard check vẫn áp dụng)
**Type:** integration | 🟠 P1

**Given:** 2 observations về cùng topic tương tự (similarity > 0.7)  
**When:** Consolidation extract memories  
**Then:** Chỉ 1 memory cuối cùng có `isLatest = true`

---

### Group D: Idempotency và Concurrency

#### TC-008 — Concurrent consolidation attempts: chỉ 1 chạy
**Requirement:** TR-009-CON-006 | **Type:** integration | 🔴 P0

**Given:** 2 triggers xảy ra đồng thời cho cùng session  
**When:** Cả 2 được thực thi song song  
**Then:**
- Không có duplicate summaries
- Không có duplicate memories
- `consolidationCount = 1` (không phải 2)

---

#### TC-009 — Idempotent: trigger lại cùng batch → không tạo thêm
**Type:** integration | 🟡 P2

**Given:** Batch [obs1..obs10] đã được consolidate  
**When:** Trigger lại với cùng batch  
**Then:** Không có summary hoặc memory mới được tạo

---

### Group E: Observation Cleanup

#### TC-010 — Observations được đánh dấu "consolidated" sau khi xử lý
**Requirement:** TR-009-CON-004 | **Type:** integration | 🟠 P1

**Given:** 10 observations được consolidate  
**When:** Consolidation hoàn thành  
**Then:** Observations có flag `consolidated = true` trong KV (không xóa raw data)

---

### Group F: Configuration

#### TC-011 — `CONSOLIDATION_THRESHOLD` có thể cấu hình qua env var
**Type:** unit | 🟡 P2

**Given:** `CONSOLIDATION_THRESHOLD=5`  
**When:** `getConsolidationThreshold()` gọi  
**Then:** Trả về `5`

---

#### TC-012 — Default threshold là 10
**Type:** unit | 🟡 P2

**Given:** Không có env var  
**When:** `getConsolidationThreshold()` gọi  
**Then:** Trả về `10`

---

## 4. Coverage Notes

- Test cần mock LLM provider nếu consolidation dùng Claude để tóm tắt
- Trong `AGENTMEMORY_AUTO_COMPRESS=false` mode, test synthetic consolidation path
- Integration tests cần đủ observations để trigger threshold
