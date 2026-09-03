# Bug Report — F10: Hybrid Search Engine

> Feature: Cross-engine recall (`POST /v1/memory/recall`), vnp-search-hub fan-out, rerank
> Luồng: `apps/memory → gateway → memory.Recall → ForwardToService("vnp-search-hub") → search engines`

---

## BUG-F10-001: `vnp-search-hub` Không Có Implementation

**Severity:** CRITICAL  
**File:** `services/vnp-search-hub/`

**Mô tả:**  
`vnp-search-hub` service được liệt kê trong `allServices` và `searchEngines` nhưng không có implementation code. `memory.Recall` forward mọi search requests tới service này.

```go
// handler.go:146
func (h *MemoryHandler) Recall(w http.ResponseWriter, r *http.Request) {
    ForwardToService(h.registry, "vnp-search-hub", h.logger)(w, r)
}
```

**Impact:**  
- `POST /v1/memory/recall` sẽ fail hoàn toàn — connection refused.
- Cross-engine hybrid search không hoạt động.

---

## BUG-F10-002: `SearchUseCase.FanOutSearch` Không Được Dùng Trong `Recall`

**Severity:** HIGH  
**File:**
- `gateway/usecase/console.go:239-290` (`SearchUseCase.FanOutSearch`)
- `gateway/adapter/handler/handler.go:145-147` (`Recall`)

**Mô tả:**  
`SearchUseCase` với `FanOutSearch()` đã được implement — fan-out tới 6 search engines song song với timeout. Tuy nhiên `MemoryHandler.Recall` không dùng `SearchUseCase` — nó chỉ forward tới `vnp-search-hub`. Nếu `vnp-search-hub` không tồn tại, không có fallback fan-out nào.

**Impact:**  
- Fan-out search capability đã được implement nhưng không được wire vào Recall handler.
- `SearchUseCase` được tạo trong `main.go` nhưng bị gán vào `_`:
  ```go
  _ = usecase.NewSearchUseCase(registry, logger)
  ```

---

## BUG-F10-003: `SearchUseCase` Sử Dụng `registry.Forward()` Deprecated Method

**Severity:** MEDIUM  
**File:** `gateway/usecase/console.go:260`

**Mô tả:**  
`FanOutSearch` dùng `registry.Forward()` (deprecated) thay vì `registry.ForwardWithContext()`. Comment trong code đánh dấu `Forward()` là deprecated.

```go
resp, err := uc.registry.Forward(ctx, target, query)  // Deprecated
```

**Impact:**  
- HTTP path/method không được propagate tới downstream services.
- Search engines không biết endpoint nào cần query.

---

## BUG-F10-004: Merge và Rerank Logic Không Tồn Tại

**Severity:** HIGH  
**File:** `gateway/usecase/console.go:271-289`

**Mô tả:**  
`FanOutSearch` collect results từ các engines nhưng không có merge hoặc rerank logic. `UnifiedSearchResult.Results` chỉ là list của raw bytes từ mỗi engine — không normalized, không ranked.

```go
// Chỉ collect raw bytes, không merge hay rerank
results = append(results, esr)
```

**Impact:**  
- Kết quả search sẽ là array của engine-specific raw JSON — không unified, không ranked.
- AI Agent nhận được unprocessed data thay vì top-K merged results.
