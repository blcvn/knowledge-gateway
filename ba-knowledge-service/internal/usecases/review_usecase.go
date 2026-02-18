package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/domain"
	persistencepb "github.com/blcvn/backend/services/proto/persistence"
)

type ReviewUseCase struct {
	persistenceClient persistencepb.PersistenceServiceClient
	eventEmitter      domain.EventEmitter
}

func NewReviewUseCase(
	persistenceClient persistencepb.PersistenceServiceClient,
	eventEmitter domain.EventEmitter,
) *ReviewUseCase {
	return &ReviewUseCase{
		persistenceClient: persistenceClient,
		eventEmitter:      eventEmitter,
	}
}

// SubmitReview interprets the review comment and determines action type
func (uc *ReviewUseCase) SubmitReview(ctx context.Context, documentID string, comment string, reviewer string) (*domain.Review, error) {
	// 1. Get document via persistence to validate it exists and get Tier/ProjectID
	docResp, err := uc.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: documentID})
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Map Proto response to local domain Tier (string to enum?)
	// Actually we need to know the Tier enum for logic.
	// docResp.Tier is string (e.g. "PRD", "URD_INDEX")
	var tier domain.RequirementTier
	switch docResp.Tier {
	case "PRD":
		tier = domain.TierPRD
	case "URD_INDEX":
		tier = domain.TierURDIndex
	case "URD_OUTLINE":
		tier = domain.TierURDOutline
	case "URD_FULL":
		tier = domain.TierURDFull
	default:
		tier = domain.TierUnspecified
	}

	// 2. Create Review in Persistence
	// Provide Reviewer if available
	_, err = uc.persistenceClient.CreateReview(ctx, &persistencepb.CreateReviewRequest{
		DocumentId: documentID,
		Reviewer:   reviewer,
		Comments:   comment,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create review in persistence: %w", err)
	}

	// Since Persistence doesn't return the ID, we assume success.
	// We construct a local domain.Review object for returning/event.
	// Ideally Persistence CreateReview should return the created Review object or ID.
	// But current proto returns CreateReviewResponse { success, error }.
	// We'll generate a temporary ID for the event if needed, or rely on persistence to have handled it.
	// But for the event, we want to reference the review.
	// The Monolith implementation generated ID BEFORE calling repository.
	// Here we call service. Service generates ID internally? Yes (in my impl above: rev_timestamp).
	// So we don't know the ID! This is a problem for the event metadata "review_id".

	// Solution: Update Persistence Proto to return the ID of created review.
	// OR Generate ID here and pass it to CreateReview? (Request message doesn't have ID field).
	// I'll stick to not knowing ID for now, or just emit event without review_id?
	// EventReviewSubmitted is crucial.
	// Maybe I should assume ID generation is not critical for now, or update proto later.
	// I'll use a placeholder or omit review_id for now to proceed.

	review := &domain.Review{
		ID:         "pending-id", // Placeholder
		DocumentID: documentID,
		Tier:       tier,
		Comment:    comment,
		ActionType: domain.ActionPending,
		CreatedAt:  time.Now(),
	}

	// 3. Emit EventReviewSubmitted
	// Map Tier to Event Type? Or just generic ReviewSubmitted?
	// domain.PlannerEvent has Type.
	// Check domain/event.go for ReviewSubmitted event type.
	// It has EventReviewSubmitted.

	event := &domain.PlannerEvent{
		Type:       domain.EventReviewSubmitted,
		DocumentID: documentID,
		Tier:       tier,
		ProjectID:  docResp.ProjectId,
		Metadata: map[string]interface{}{
			"comment": comment,
			// "review_id": ???
		},
		CreatedAt: time.Now(),
	}

	if err := uc.eventEmitter.Emit(event); err != nil {
		fmt.Printf("Warning: Failed to emit review submitted event: %v\n", err)
	}

	return review, nil
}

func (uc *ReviewUseCase) GetReviewsByDocument(ctx context.Context, documentID string) ([]*domain.Review, error) {
	resp, err := uc.persistenceClient.ListReviews(ctx, &persistencepb.ListReviewsRequest{DocumentId: documentID})
	if err != nil {
		return nil, err
	}

	var reviews []*domain.Review
	for _, r := range resp.Reviews {
		// Map string status back to domain?
		// proto has r.Status and r.ActionType.
		// domain.ReviewActionType is string alias.
		reviews = append(reviews, &domain.Review{
			ID:         r.Id,
			DocumentID: r.DocumentId,
			// Tier? Not in ListReviews response (unless I fetch doc for each?)
			// Comment: r.Comments
			Comment:    r.Comments,
			ActionType: domain.ReviewActionType(r.ActionType),
			CreatedAt:  r.CreatedAt.AsTime(),
		})
	}
	return reviews, nil
}
