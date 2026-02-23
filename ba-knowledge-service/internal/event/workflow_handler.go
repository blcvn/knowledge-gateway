package event

import (
	"context"
	"log"
	"time"

	domain "github.com/blcvn/backend/services/pkg/domain"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/usecases"
)

// WorkflowEventHandler handles workflow progression events
type WorkflowEventHandler struct {
	docUseCase *usecases.DocumentUseCase
}

func NewWorkflowEventHandler(docUseCase *usecases.DocumentUseCase) *WorkflowEventHandler {
	return &WorkflowEventHandler{
		docUseCase: docUseCase,
	}
}

func (h *WorkflowEventHandler) Handle(event *domain.PlannerEvent) error {
	// context.Background() for now as event handling is async/background
	// TODO: context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	log.Printf("[WORKFLOW] Processing event: %s for document %s", event.Type, event.DocumentID)

	switch event.Type {
	case domain.EventPRDUploaded:
		// Trigger Index generation from PRD
		log.Printf("[WORKFLOW] PRD uploaded, triggering Index generation")
		if _, err := h.docUseCase.GenerateIndex(ctx, event.DocumentID); err != nil {
			log.Printf("[WORKFLOW] Failed to trigger Index generation: %v", err)
			return err
		}
		return nil

	case domain.EventIndexApproved:
		// Trigger Outline generation
		log.Printf("[WORKFLOW] Index approved, triggering Outline generation")
		// Event struct has ProjectID.

		if _, err := h.docUseCase.GenerateOutline(ctx, event.DocumentID); err != nil {
			log.Printf("[WORKFLOW] Failed to trigger Outline generation: %v", err)
			return err
		}
		return nil

	case domain.EventOutlineApproved:
		// Trigger Full URD generation
		log.Printf("[WORKFLOW] Outline approved, triggering Full URD generation")

		if _, err := h.docUseCase.GenerateFull(ctx, event.DocumentID); err != nil {
			log.Printf("[WORKFLOW] Failed to trigger Full generation: %v", err)
			return err
		}
		return nil

	case domain.EventFullApproved:
		// Mark as ready for publication
		log.Printf("[WORKFLOW] Full URD approved, ready for publication")
		return nil

	case domain.EventReviewSubmitted:
		// Trigger Document Regeneration based on Review
		log.Printf("[WORKFLOW] Review submitted for document %s", event.DocumentID)
		reviewID, ok := event.Metadata["review_id"].(string)
		if !ok {
			log.Printf("[WORKFLOW] Error: review_id missing in event metadata")
			return nil
		}
		if err := h.docUseCase.RegenerateDocument(ctx, reviewID); err != nil {
			log.Printf("[WORKFLOW] Failed to regenerate document: %v", err)
			return err
		}
		return nil

	default:
		// Ignore unknown events or log
		return nil
	}
}
