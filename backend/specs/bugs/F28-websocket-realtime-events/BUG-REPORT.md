# Bug Report — F28: WebSocket Real-time Events

> Feature: Real-time event streaming cho Console UI
> Luồng: `apps/memory → gateway/ws.go (WSHandler) → NATS → clients`

---

## BUG-F28-001: WSHandler Không Subscribe NATS — Chỉ Là SSE Passthrough

**Severity:** CRITICAL  
**File:** `gateway/adapter/handler/ws.go`

**Mô tả:**  
Feature spec yêu cầu WSHandler subscribe NATS subjects (tenant-scoped) và forward events tới WebSocket clients. Tuy nhiên implementation hiện tại:
1. Không có WebSocket upgrade — dùng SSE (Server-Sent Events) thay thế.
2. **Không có NATS subscription** — không subscribe bất kỳ subject nào.
3. `send` channel của connection chỉ được populated bởi `Broadcast()` method — nhưng không ai gọi `Broadcast()`.

```go
// ws.go: Không có NATS subscription code
// stream loop chỉ chờ conn.send channel
for {
    select {
    case msg, ok := <-conn.send:  // Channel này luôn empty!
```

**Impact:**  
- Console UI connect WS → nhận "connected" event → sau đó không nhận thêm event nào.
- Real-time updates cho Dashboard, Memory Stored, Pipeline Complete đều không hoạt động.

---

## BUG-F28-002: WebSocket Upgrade Không Được Implement

**Severity:** HIGH  
**File:** `gateway/adapter/handler/ws.go:52-54`

**Mô tả:**  
Comment trong code xác nhận đây là SSE fallback, không phải WebSocket:
```go
// Note: This is a simplified SSE-based implementation using Server-Sent Events
// for environments where native WebSocket upgrade is not available.
// For production, replace with gorilla/websocket or nhooyr.io/websocket.
```

Feature doc nói "WebSocket" nhưng implementation dùng SSE với header `Content-Type: text/event-stream`.

**Impact:**  
- Clients expect WebSocket protocol, nhận SSE — có thể gây incompatibility với WebSocket client libraries.

---

## BUG-F28-003: Auth Middleware Không Apply → WSHandler Auth Manual Nhưng Dựa Vào AuthContext Nil

**Severity:** HIGH  
**File:** `gateway/adapter/handler/ws.go:57-74`

**Mô tả:**  
WSHandler check `middleware.AuthFromContext(r.Context())` để validate admin role. Vì Auth middleware không được apply trong chain (BUG-F14-001), `AuthFromContext` luôn trả về `nil, false` → WSHandler luôn trả về 401 Unauthenticated.

**Impact:**  
- Không ai có thể connect tới WebSocket endpoint.

---

## BUG-F28-004: `channels` Query Parameter Parsing Không Handle Quoted String

**Severity:** LOW  
**File:** `gateway/adapter/handler/ws.go:92-100`

**Mô tả:**  
```go
channelsParam := r.URL.Query().Get("channels")
if err := json.Unmarshal([]byte(channelsParam), &channels); err == nil {
```

Nếu `channels` được truyền như URL-encoded JSON (`?channels=["memory","pipeline"]`), có thể bị URL-decode sai. Cần URL-decode trước khi JSON parse.

---

## BUG-F28-005: `removeConnection` Gọi `close(conn.done)` Có Thể Panic Nếu Done Channel Đã Closed

**Severity:** MEDIUM  
**File:** `gateway/adapter/handler/ws.go:175-184`

**Mô tả:**  
```go
func (h *WSHandler) removeConnection(id string) {
    if conn, ok := h.connections[id]; ok {
        close(conn.done)  // Panic nếu đã close trước đó
```

Không có check nào đảm bảo `conn.done` chưa bị closed. Race condition có thể xảy ra nếu `removeConnection` được gọi nhiều lần.

**Impact:**  
- Panic → recovery middleware bắt được nhưng gây connection handling instability.

---

## BUG-F28-006: Tenant-Scoping Cho Events Không Implement

**Severity:** HIGH  
**File:** `gateway/adapter/handler/ws.go`

**Mô tả:**  
Feature spec yêu cầu "zero cross-tenant event leakage" — chỉ gửi events thuộc tenant của user. `Broadcast()` gửi tới tất cả connections không cần biết tenant:

```go
func (h *WSHandler) Broadcast(channel, event string, data any) {
    for _, conn := range h.connections {
        if conn.channels[channel] || len(conn.channels) == 0 {
            conn.send <- msg  // Không check tenantID!
```

**Impact:**  
- Cross-tenant event leakage có thể xảy ra.
- Tenant A nhận events của Tenant B nếu subscribe cùng channel name.
