# Bug Report — F12: Memory Consolidation Pipeline

> Feature: 4-tier consolidation (compress → summarize → procedural → insights)
> Luồng: NATS `consolidation.trigger` → memory-service ConsolidationPipeline → LLM

---

## BUG-F12-001: ConsolidationHandler Không Được Đăng Ký Trong gRPC Server

**Severity:** CRITICAL  
**File:** `services/memory-service/cmd/server/main.go`

**Mô tả:**  
`ConsolidationHandler` đã được implement (`adapter/grpc/consolidation_handler.go`) nhưng không được register vào gRPC server trong `main.go`. Server chỉ register `forward.RegisterForwardService` và `grpc_health_v1`.

```go
// main.go - THIẾU:
// memorypb.RegisterConsolidationServiceServer(grpcServer, consolidationHandler)
```

**Impact:**  
- `ConsolidationService` RPCs (SummarizeSession, RunPipeline) không accessible.
- Consolidation pipeline không thể được triggered qua gRPC.

---

## BUG-F12-002: NATS Subscription Cho `consolidation.trigger` Không Được Setup

**Severity:** CRITICAL  
**File:** `services/memory-service/cmd/server/main.go`

**Mô tả:**  
Feature spec yêu cầu consolidation pipeline được trigger khi session kết thúc qua NATS event `consolidation.trigger`. `memory-service/main.go` không setup bất kỳ NATS subscription nào.

**Impact:**  
- Session end (từ `observe-service`) không trigger consolidation.
- Tier 2, 3, 4 không bao giờ chạy tự động.

---

## BUG-F12-003: Consolidation Pipeline Thiếu LLM Client

**Severity:** HIGH  
**File:** `services/memory-service/internal/consolidation/`

**Mô tả:**  
Consolidation pipeline (Tier 1 LLM compression, Tier 2 session summarization) yêu cầu LLM calls. Không có LLM client nào được inject hoặc configured trong `ConsolidationPipeline`.

```go
// consolidation_handler.go
type ConsolidationHandler struct {
    pipeline    *consolidation.ConsolidationPipeline
    // Không có LLM client!
}
```

**Impact:**  
- LLM compression và summarization không hoạt động.
- Không có circuit breaker fallback được implement.

---

## BUG-F12-004: Consolidation Pipeline Chưa Connect Tới Background Job Scheduler

**Severity:** HIGH  
**File:** `services/pipeline-service/`

**Mô tả:**  
`pipeline-service` directory tồn tại nhưng không có implementation. Tier 3 (daily) và Tier 4 (weekly) background jobs không có scheduler.

**Impact:**  
- Procedural memory extraction (Tier 3) và Insights (Tier 4) không bao giờ chạy.
