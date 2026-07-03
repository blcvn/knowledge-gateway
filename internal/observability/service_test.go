package observability

import (
	"testing"
	"time"

	"kg-service/internal/telemetry"
	"kg-service/internal/write"
)

type observabilityStoreStub struct {
	outbox  []write.OutboxEvent
	records []write.ProjectionVersionRecord
}

type observabilityTelemetryStub struct {
	snapshot telemetry.Snapshot
}

func (s observabilityStoreStub) ListOutboxEvents() []write.OutboxEvent {
	return append([]write.OutboxEvent(nil), s.outbox...)
}

func (s observabilityStoreStub) ListProjectionVersions() []write.ProjectionVersionRecord {
	return append([]write.ProjectionVersionRecord(nil), s.records...)
}

func (s observabilityTelemetryStub) Snapshot() telemetry.Snapshot {
	return s.snapshot
}

func TestSnapshotIncludesRealtimeFallbackAndGraphScopeConflictCounters(t *testing.T) {
	before := telemetry.Default().Snapshot()
	telemetry.RecordRealtimeReadFallback("domain-a", "app-a")
	telemetry.RecordGraphScopeConflict("project:a", "project:b")

	now := time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC)
	svc := NewServiceWithTelemetry(
		observabilityStoreStub{
			outbox: []write.OutboxEvent{
				{Status: "PENDING"},
				{Status: "FAILED"},
				{Status: "DONE"},
			},
			records: []write.ProjectionVersionRecord{
				{LastGraphSyncedAt: now.Add(-10 * time.Second), LastVectorSyncedAt: now.Add(-20 * time.Second)},
			},
		},
		nil,
		telemetry.Default(),
	)
	svc.now = func() time.Time { return now }

	snap := svc.Snapshot()
	if snap.OutboxBacklog != 2 {
		t.Fatalf("OutboxBacklog = %d, want 2", snap.OutboxBacklog)
	}
	if snap.RealtimeReadFallbackCount < before.RealtimeReadFallbackCount+1 {
		t.Fatalf("RealtimeReadFallbackCount = %d, want at least %d", snap.RealtimeReadFallbackCount, before.RealtimeReadFallbackCount+1)
	}
	if snap.GraphScopeConflictCount < before.GraphScopeConflictCount+1 {
		t.Fatalf("GraphScopeConflictCount = %d, want at least %d", snap.GraphScopeConflictCount, before.GraphScopeConflictCount+1)
	}
	if snap.GraphLagSeconds.Count != 1 || snap.VectorLagSeconds.Count != 1 {
		t.Fatalf("lag stats = %+v / %+v, want single samples", snap.GraphLagSeconds, snap.VectorLagSeconds)
	}
}

func TestSnapshotUsesInjectedTelemetrySnapshotter(t *testing.T) {
	now := time.Date(2026, 6, 23, 9, 0, 0, 0, time.UTC)
	svc := NewServiceWithTelemetry(
		observabilityStoreStub{},
		nil,
		observabilityTelemetryStub{
			snapshot: telemetry.Snapshot{
				RealtimeReadFallbackCount: 7,
				GraphScopeConflictCount:   3,
				LastUpdatedAt:             now,
			},
		},
	)

	svc.now = func() time.Time { return now }

	snap := svc.Snapshot()
	if snap.RealtimeReadFallbackCount != 7 || snap.GraphScopeConflictCount != 3 {
		t.Fatalf("snapshot counters = %+v", snap)
	}
	if !snap.LastUpdatedAt.Equal(now) {
		t.Fatalf("LastUpdatedAt = %s, want %s", snap.LastUpdatedAt, now)
	}
}
