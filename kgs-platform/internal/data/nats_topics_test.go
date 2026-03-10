package data

import "testing"

func TestNATSTopics(t *testing.T) {
	ns := "graph/app-1/tenant-42"
	if got := TopicOverlayCommitted(ns); got != "overlay.committed.tenant-42" {
		t.Fatalf("unexpected committed topic: %s", got)
	}
	if got := TopicOverlayDiscarded(ns); got != "overlay.discarded.tenant-42" {
		t.Fatalf("unexpected discarded topic: %s", got)
	}
	if got := TopicSessionClose("session-1"); got != "session.close.session-1" {
		t.Fatalf("unexpected session close topic: %s", got)
	}
	if got := TopicBudgetStop("session-1"); got != "budget.stop.session-1" {
		t.Fatalf("unexpected budget stop topic: %s", got)
	}
	if TopicSessionClosePattern() != "session.close.*" {
		t.Fatalf("unexpected session pattern")
	}
	if TopicBudgetStopPattern() != "budget.stop.*" {
		t.Fatalf("unexpected budget pattern")
	}
}
