# TASK-007: Fix `HealthStatus` Enum Casing in `vnp-platform`

**Solution**: [SOL-004](../solutions/SOL-004-response-schema-contracts.md)  
**CR**: CR-004  
**Priority**: 🟠 Medium  
**Estimate**: 30 minutes  
**Status**: ✅ Implemented

---

## Context

The frontend expects `HealthStatus` values to be `"Healthy"`, `"Warning"`, `"Critical"` (capital first letter). The `vnp-platform` admin entity currently defines `HealthStatus` constants — the exact values must be verified and corrected if they use lowercase.

---

## Exact Task

### Step 1: Read the current `HealthStatus` definition

File: `services/vnp-platform/internal/domain/admin/entity.go`

Find the `HealthStatus` type and its constants.

### Step 2: Verify and fix the constants

```go
// MUST be (capital first letter):
const (
    StatusHealthy  HealthStatus = "Healthy"
    StatusWarning  HealthStatus = "Warning"
    StatusCritical HealthStatus = "Critical"
)

// If currently:
const (
    StatusHealthy  HealthStatus = "healthy"   // ← WRONG
    StatusWarning  HealthStatus = "warning"   // ← WRONG
    StatusCritical HealthStatus = "critical"  // ← WRONG
)
// Then change to the correct values above.
```

### Step 3: Grep for usages

Search for all places that use `StatusHealthy`, `StatusWarning`, `StatusCritical` or hardcoded `"healthy"`, `"warning"`, `"critical"` strings in the vnp-platform and vnp-dashboard services:

```bash
grep -rn '"healthy"\|"warning"\|"critical"\|StatusHealthy\|StatusWarning\|StatusCritical' \
  services/vnp-platform/ services/vnp-dashboard/ 2>/dev/null
```

Update any hardcoded lowercase strings to use the constant or the correctly-cased string.

### Step 4: Also check `HealthStatus` in `architecture.md`

Architecture doc shows: `HealthStatus — Service, Status(SERVING|NOT_SERVING|UNKNOWN)` (gRPC values). The HTTP-facing status returned to the frontend must be `"Healthy"`, `"Warning"`, `"Critical"` (not the gRPC values). Ensure the gRPC→HTTP mapping uses the correct casing.

---

## Files to Modify

| File | Change |
|------|--------|
| `services/vnp-platform/internal/domain/admin/entity.go` | Fix `HealthStatus` constant values |
| Any service that uses these constants | Update usages if changing string values |

---

## Acceptance Criteria

- [ ] `HealthStatus` constants are `"Healthy"`, `"Warning"`, `"Critical"` (capital first letter)
- [ ] No hardcoded `"healthy"`, `"warning"`, `"critical"` strings in HTTP response paths
- [ ] `go build ./services/vnp-platform/...` passes

---

**Audit Note:** HTTPHealthStatus type + StatusHealthy/Warning/Critical/Unknown constants added; GRPCToHTTPHealth mapper added
