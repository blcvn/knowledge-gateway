# SOL-AM-007 — Solution: Governance, Audit & Diagnostics

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-007 |
| **CR** | CR-AM-007 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/vnp-admin` |

---

## 1. Giải pháp

AgentMemory-specific governance:
1. Hook event audit: who submitted what hook, when
2. Session diagnostics: anomalies, suspicious patterns
3. Memory inventory for compliance

```go
// services/vnp-admin/internal/usecase/diagnostics.go [NEW]
func (u *DiagnosticsUseCase) AnalyzeSession(ctx context.Context, sessionID string) (*DiagnosticReport, error) {
    obs, _ := u.obsRepo.GetAll(ctx, sessionID)
    return &DiagnosticReport{
        TotalHooks:      len(obs),
        HookBreakdown:   countByType(obs),
        Anomalies:       detectAnomalies(obs),  // e.g., too many errors
        PIIDetected:     scanForPII(obs),
        Duration:        calcDuration(obs),
        CompressionRatio: calcCompressionRatio(obs),
    }, nil
}
```

## 2. Acceptance Criteria

- [ ] GET /v1/admin/sessions/{id}/diagnostics returns report
- [ ] PII detection in hook payloads
- [ ] Anomaly alerts (e.g., error rate > 20%)

