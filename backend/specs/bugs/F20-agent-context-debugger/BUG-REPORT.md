# Bug Report — F20: Agent Context Debugger

> Feature: Trace creation, trace retrieval, list traces
> Luồng: `apps/memory → gateway/console.go (DebuggerHandler) → obs-service`

---

## BUG-F20-001: Auth Middleware Không Apply (BUG-F14-001 Cross-reference)

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/console.go:352-374`

---

## BUG-F20-002: Debugger Forward Tới `obs-service` — Cần Verify Implementation

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/console.go:352-374`

**Mô tả:**  
CreateTrace, GetTrace, ListTraces tất cả forward tới `obs-service`. Cần verify `obs-service` implement context debugger protocol:
- Simulate context assembly (context window building)
- Store và retrieve traces
- Show which memories were selected và why

**Impact:**  
- Nếu `obs-service` không implement debugger protocol, trace endpoints sẽ fail hoặc trả về wrong data.

---

## BUG-F20-003: Không Có Trace Schema Definition

**Severity:** MEDIUM  
**File:** `gateway/domain/`

**Mô tả:**  
Không có domain types cho `DebugTrace`, `ContextAssemblyResult` trong `gateway/domain/`. Debugger handler chỉ forward raw bytes — không có input validation hay response marshaling.

**Impact:**  
- API không có well-defined request/response schema → client integration khó.
