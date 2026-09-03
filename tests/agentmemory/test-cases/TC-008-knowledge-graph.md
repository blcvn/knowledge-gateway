# TC-008: Knowledge Graph — Test Cases

**Test Design tham chiếu:** [TD-008](../designs/TD-008-knowledge-graph.md)  
**Requirements tham chiếu:** [TR-008](../requirements/TR-008-knowledge-graph.md)  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-008-001: Graph node được tạo với đúng structure

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-001 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-008-GRF-001 |

**Dữ liệu đầu vào:**
- name: `"jose_library"`, type: `"library"`, observationIds: `["obs_001"]`

**Các bước thực hiện:**
1. Gọi hàm tạo graph node với dữ liệu trên
2. Đọc node từ KV tại scope `mem:graph:nodes`
3. Kiểm tra tất cả fields

**Kết quả mong đợi:**
- `node.id` — string unique
- `node.name = "jose_library"`
- `node.type = "library"`
- `node.observationIds = ["obs_001"]`
- `node.createdAt` — ISO timestamp
- `node.degree = 0` (ban đầu)

---

## TC-008-002: Node dedup — cùng `{type}|{name}` → cùng nodeId

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-008-GRF-003 |

**Điều kiện tiên quyết:** Name index `mem:graph:name-index` trống

**Các bước thực hiện:**
1. Extract entity type=`"library"`, name=`"jose"` từ obs_A → lưu `nodeId1`
2. Extract entity type=`"library"`, name=`"jose"` từ obs_B → lưu `nodeId2`
3. So sánh `nodeId1` và `nodeId2`

**Kết quả mong đợi:** `nodeId1 === nodeId2` (không có duplicate node)

---

## TC-008-003: Node name index — key format `{type}|{name}`

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-003 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- Node với type=`"library"`, name=`"jose"`

**Kết quả mong đợi:**
- KV `mem:graph:name-index` có key `"library|jose"` → giá trị = nodeId

---

## TC-008-004: Edge được tạo với đúng structure

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-004 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-008-GRF-002 |

**Dữ liệu đầu vào:**
- sourceId: `"node_A"`, targetId: `"node_B"`, type: `"uses"`, observationIds: `["obs_001"]`

**Kết quả mong đợi:**
- `edge.id` — string unique
- `edge.sourceId = "node_A"`, `edge.targetId = "node_B"`
- `edge.type = "uses"`
- `edge.weight` — float
- `edge.observationIds = ["obs_001"]`
- `edge.createdAt` — ISO timestamp

---

## TC-008-005: Edge dedup — cùng `{src}|{tgt}|{type}` → cùng edgeId

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-008-GRF-004 |

**Các bước thực hiện:**
1. Create edge: src=nodeA, tgt=nodeB, type=`"uses"` → `edgeId1`
2. Create edge: src=nodeA, tgt=nodeB, type=`"uses"` → `edgeId2`
3. So sánh `edgeId1` và `edgeId2`

**Kết quả mong đợi:** `edgeId1 === edgeId2`

---

## TC-008-006: 6 relationship types được chấp nhận

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-006 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-008-GRF-005 |

**Dữ liệu đầu vào (test từng type):** `uses`, `implements`, `extends`, `calls`, `imports`, `defines`

**Kết quả mong đợi:** Tất cả 6 types được accept và lưu đúng

---

## TC-008-007: searchByEntities trả về observations liên quan

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-008-GRF-008 |

**Setup:**
- Node `"jose"` liên kết với obsIds: `["obs_jose_1", "obs_jose_2"]`

**Các bước thực hiện:**
1. Setup node "jose" với observationIds
2. Gọi `searchByEntities(["jose"], 1, 5)`
3. Kiểm tra results

**Kết quả mong đợi:** Results chứa obs_jose_1 và/hoặc obs_jose_2

---

## TC-008-008: Node degree tăng khi edge được thêm

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-008-GRF-010 |

**Setup:** Node A với degree = 0

**Các bước thực hiện:**
1. Tạo edge A → B
2. Đọc degree của A

**Kết quả mong đợi:** `degree(A) = 1`

---

## TC-008-009: Graph snapshot được tạo tại key `"current"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-008-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-008-GRF-007 |

**Setup:** 20 nodes với degrees khác nhau

**Kết quả mong đợi:**
- Snapshot tồn tại tại KV scope `mem:graph:snapshot`, key `"current"`
- Snapshot chứa top nodes theo degree

---

## Tổng kết TC-008

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-008-001 | Node structure | 🔴 P0 | Unit |
| TC-008-002 | Node dedup | 🔴 P0 | Integration |
| TC-008-003 | Name index format | 🟠 P1 | Unit |
| TC-008-004 | Edge structure | 🔴 P0 | Unit |
| TC-008-005 | Edge dedup | 🔴 P0 | Integration |
| TC-008-006 | 6 relationship types | 🟠 P1 | Unit |
| TC-008-007 | searchByEntities | 🔴 P0 | Integration |
| TC-008-008 | Degree increment | 🟠 P1 | Integration |
| TC-008-009 | Snapshot at "current" | 🟠 P1 | Integration |
