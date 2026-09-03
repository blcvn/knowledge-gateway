# TC-010: Context Injection (Recall) — Test Cases

**Design ref:** [TD-010](../designs/TD-010-context-injection.md) | **Ngày:** 2026-06-11

---

## TC-010-001: Recall trả về kết quả khi có observations liên quan

| **ID** | TC-010-001 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Session `sess_recall` có 5 observations, trong đó 3 chứa từ "auth"

**Dữ liệu đầu vào:**
- `query = "auth"`, `sessionId = "sess_recall"`, `maxObs = 5`

**Kết quả mong đợi:**
- `results.length >= 1`
- Mỗi result có `observation` (object) và `combinedScore` (number > 0)
- Sorted by `combinedScore` descending (kết quả [0].combinedScore >= kết quả [1].combinedScore)

---

## TC-010-002: Query rỗng trả về recent observations

| **ID** | TC-010-002 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Dữ liệu đầu vào:** `query = ""`, `sessionId = "sess_recall"`, `maxObs = 5`

**Kết quả mong đợi:**
- 5 observations gần đây nhất (sorted by timestamp desc)
- Không có error

---

## TC-010-003: `maxObs` giới hạn số observations trong result

| **ID** | TC-010-003 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 20 observations liên quan query

**Dữ liệu đầu vào:** `maxObs = 5`

**Kết quả mong đợi:** `result.observations.length ≤ 5`

---

## TC-010-004: `maxMemories` giới hạn số memories trong result

| **ID** | TC-010-004 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** 10 memories liên quan

**Kết quả mong đợi:** `result.memories.length ≤ 3` (với maxMemories=3)

---

## TC-010-005: Recall bao gồm session summary nếu có

| **ID** | TC-010-005 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Session đã được consolidate, có summary trong `mem:summaries`

**Kết quả mong đợi:** `result.sessionSummary` không null, chứa nội dung từ KV

---

## TC-010-006: Recall chỉ trả về `isLatest = true` memories

| **ID** | TC-010-006 | **Priority** | 🟠 P1 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:**
- M1: `isLatest = false` (đã bị supersede bởi M2)
- M2: `isLatest = true`

**Kết quả mong đợi:**
- `result.memories` chứa M2
- `result.memories` KHÔNG chứa M1

---

## TC-010-007: Default chỉ recall trong cùng sessionId

| **ID** | TC-010-007 | **Priority** | 🔴 P0 | **Type** | Integration |
|---|---|---|---|---|---|

**Setup:** Session A và Session B, cả 2 có obs về "auth"

**Kết quả mong đợi:**
- Recall với `sessionId = "sess_A"` chỉ trả về obs từ sess_A
- Obs từ sess_B không xuất hiện

---

## TC-010-008: `includeOtherSessions=true` bao gồm obs từ sessions khác

| **ID** | TC-010-008 | **Priority** | 🟡 P2 | **Type** | Integration |
|---|---|---|---|---|---|

**Kết quả mong đợi:** Obs từ cả 2 sessions xuất hiện trong results

---

## Tổng kết TC-010

| ID | Priority | Loại |
|---|---|---|
| TC-010-001 | 🔴 P0 | Integration |
| TC-010-002 | 🟠 P1 | Integration |
| TC-010-003 | 🔴 P0 | Integration |
| TC-010-004 | 🟠 P1 | Integration |
| TC-010-005 | 🟠 P1 | Integration |
| TC-010-006 | 🟠 P1 | Integration |
| TC-010-007 | 🔴 P0 | Integration |
| TC-010-008 | 🟡 P2 | Integration |
