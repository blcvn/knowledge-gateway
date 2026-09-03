# TC-009: Consolidation Pipeline — Test Cases

**Design ref:** [TD-009](../designs/TD-009-consolidation-pipeline.md) | **Ngày:** 2026-06-11

---

## TC-009-001: Không trigger consolidation dưới ngưỡng

| **ID** | TC-009-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** `CONSOLIDATION_THRESHOLD = 5`, session `sess_con` có 4 observations

**Các bước:**
1. Gửi hook thứ 4 (total = 4 obs)
2. Đọc KV `mem:summaries[sess_con]`

**Kết quả mong đợi:** Không có summary entry — consolidation KHÔNG trigger

---

## TC-009-002: Trigger khi đúng ngưỡng

| **ID** | TC-009-002 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Tiếp tục từ TC-009-001 (4 obs)

**Các bước:**
1. Gửi hook thứ 5 (total = 5 obs = threshold)
2. Đợi consolidation xử lý
3. Đọc KV `mem:summaries[sess_con]`

**Kết quả mong đợi:** `mem:summaries[sess_con]` tồn tại

---

## TC-009-003: Session summary có đúng structure

| **ID** | TC-009-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi — summary có fields:**
- `sessionId = "sess_con"`
- `observationCount >= 5`
- `timeRangeStart`, `timeRangeEnd` — ISO timestamps
- `generatedAt` — ISO timestamp

---

## TC-009-004: Concurrent triggers → chỉ 1 consolidation chạy

| **ID** | TC-009-004 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Các bước:**
1. Dispatch 2 consolidation triggers đồng thời cho cùng session
2. Chờ cả 2 hoàn thành
3. Đếm số summaries và memories được tạo

**Kết quả mong đợi:**
- Chỉ 1 summary (không duplicate)
- Số memories được tạo như consolidation 1 lần (không nhân đôi)

---

## TC-009-005: Trigger lặp lại sau mỗi N observations

| **ID** | TC-009-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Threshold = 5

**Các bước:**
1. Send 5 obs → verify consolidation 1 trigger (summary #1 được tạo)
2. Send thêm 5 obs → verify consolidation 2 trigger (summary được update)

**Kết quả mong đợi:** Consolidation trigger sau mỗi batch 5 observations

---

## TC-009-006: Observations được đánh dấu "consolidated"

| **ID** | TC-009-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:**
- Sau consolidation: observations trong batch có flag `consolidated = true`
- Raw data không bị xóa (chỉ flagged)

---

## Tổng kết TC-009

| ID | Priority | Loại |
|---|---|---|
| TC-009-001 | 🔴 P0 | Integration |
| TC-009-002 | 🔴 P0 | Integration |
| TC-009-003 | 🔴 P0 | Integration |
| TC-009-004 | 🔴 P0 | Integration |
| TC-009-005 | 🟠 P1 | Integration |
| TC-009-006 | 🟠 P1 | Integration |
