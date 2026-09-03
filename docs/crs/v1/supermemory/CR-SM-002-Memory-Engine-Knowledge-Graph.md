# Change Request: CR-SM-002 — Memory Engine & Knowledge Graph

**CR ID:** CR-SM-002  
**Component:** `services/memory-service` [NEW SERVICE]  
**Priority:** Critical  
**Status:** In Progress
**Reference:** Supermemory PRD §3.1, SRS §2.3, specs/services/03-memory-service.md  
**Benchmark target:** #1 LongMemEval (81.6%), #1 LoCoMo, #1 ConvoMem

---

## 1. Mô tả

Xây dựng **Memory Service** — lõi trí nhớ AI của VNP Memory:

1. **Fact Extraction**: Dùng LLM tự động trích xuất facts có cấu trúc (Facts, Preferences, Episodes) từ nội dung.
2. **Knowledge Graph**: Xây dựng đồ thị kiến thức sống với 3 loại quan hệ: `updates`, `extends`, `derives`.
3. **Version Chain**: Theo dõi lịch sử cập nhật bộ nhớ (`version`, `parentMemoryId`, `rootMemoryId`).
4. **Automatic Forgetting**: Hỗ trợ `forgetAfter` (time-based), semantic forget (tìm theo nghĩa để xóa), và forget theo ID.
5. **Memory Classification**: Static (bền vững), Dynamic (tạm thời), Inferred (AI suy luận).

---

## 2. Vấn đề hiện tại

- VNP Memory hiện tại lưu trữ nội dung thô chứ không trích xuất facts có cấu trúc.
- Chưa có cơ chế giải quyết mâu thuẫn: khi user cập nhật thông tin (ví dụ: thay đổi địa chỉ), bộ nhớ cũ vẫn tồn tại song song.
- Chưa hỗ trợ "automatic forgetting" — thông tin cũ không bao giờ tự xóa.
- Thiếu Knowledge Graph: mọi memories độc lập, không có quan hệ.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/memory-service/` (Port gRPC: 9002)

### 3.2. Domain Model

```go
type MemoryEntry struct {
    ID             string
    Memory         string                      // Fact text
    SpaceID        string                      // Scope vào project/space
    OrgID          string

    // Version chain
    Version        int                         // Tăng dần
    IsLatest       bool                        // false nếu bị cập nhật
    ParentMemoryID *string                     // Memory trực tiếp trước
    RootMemoryID   *string                     // Memory gốc của chain

    // Relations
    MemoryRelations map[string]RelationType    // {id: updates|extends|derives}

    // Classification
    IsStatic       bool                        // Long-term facts
    IsInference    bool                        // AI-derived
    IsForgotten    bool                        // Soft-deleted
    ForgetAfter    *time.Time                  // Auto-forget ngày
    ForgetReason   *string

    MemoryEmbedding []float32
    SourceCount    int
    Metadata       map[string]any
}

type RelationType string  // "updates" | "extends" | "derives"
```

### 3.3. Fact Extraction Pipeline

Khi nhận event `document.processed` từ Document Service:

```
1. Fetch document content + danh sách memories hiện có trong space
2. LLM Prompt: "Extract facts. Classify: fact|preference|episode. Is static? Set forgetAfter if temporal."
3. Với mỗi fact trích xuất:
   a. Generate embedding
   b. Cosine similarity search vs existing memories (threshold 0.8)
   c. Nếu match: LLM classify relation (updates/extends/derives)
   d. Tạo MemoryEntry với relations đúng
   e. Nếu "updates": mark memory cũ → isLatest=false
4. Publish event "memory.created" + "memory.relation.created"
```

### 3.4. Forget Algorithm (3 Phase)

```go
// Phase 1: Exact content match
// Phase 2: Semantic search (threshold 0.85) nếu không tìm được exact
// Phase 3: Forget by ID nếu được cung cấp
```

### 3.5. Auto-Forget Cron Job

Chạy định kỳ (mỗi 1 giờ) để kiểm tra `forgetAfter < now()` và đánh dấu `isForgotten=true`.

### 3.6. Memory Graph Visualization Data

Cung cấp endpoint `GetMemoryGraph` trả về nodes (MemoryEntry, Document) + edges (Relations) để frontend render force-directed graph.

---

## 4. Acceptance Criteria

- [ ] Gửi text "Tôi thích Python và ghét Java" → hệ thống tạo 2 facts riêng biệt (`IsStatic=false`).
- [ ] Gửi tiếp "Bây giờ tôi thích Go hơn Python" → fact Python cũ được đánh dấu `isLatest=false`, tạo fact mới với relation `updates`.
- [ ] Gửi memory với `forgetAfter = giờ hiện tại + 1 giây` → sau 60 giây cron chạy, memory bị đánh dấu `isForgotten=true`.
- [ ] Tool `forget` với nội dung gần đúng (nhưng không match 100%) → semantic search tìm được và xóa đúng memory.
- [ ] `GetMemoryGraph` trả về dữ liệu đủ để render graph với nodes và edges.
