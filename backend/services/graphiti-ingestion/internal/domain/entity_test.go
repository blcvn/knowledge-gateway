package domain

import (
	"testing"
	"time"
)

func TestSagaStateTransition(t *testing.T) {
	state := NewSagaState("ep-123", "grp-456")
	
	if state.Status != SagaStatusRunning {
		t.Errorf("Expected initial status to be Running, got %s", state.Status)
	}

	err := state.Transition(StepExtractEntities, SagaStatusRunning, "")
	if err != nil {
		t.Errorf("Expected successful transition, got %v", err)
	}

	err = state.Transition(StepResolveEntities, SagaStatusCompleted, "")
	if err != nil {
		t.Errorf("Expected successful transition, got %v", err)
	}
	
	if state.Status != SagaStatusCompleted {
		t.Errorf("Expected status to be Completed, got %s", state.Status)
	}
}

func TestEpisodeCreation(t *testing.T) {
	ep, err := NewEpisode("Test", "grp-1", "Body content", SourceText, time.Now())
	if err != nil {
		t.Errorf("Failed to create episode: %v", err)
	}

	if ep.ContentHash == "" {
		t.Errorf("Expected ContentHash to be set")
	}
}
