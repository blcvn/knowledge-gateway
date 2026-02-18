package usecases

import (
	"context"
	"fmt"
	"time"

	v32 "github.com/blcvn/backend/services/ba-agent-service/domain/v3.2"
	"github.com/google/uuid"
)

type ReviewUseCase struct {
	reviewRepo   v32.ReviewRepository
	docRepo      v32.DocumentRepository
	eventEmitter v32.EventEmitter
}

func NewReviewUseCase(
	reviewRepo v32.ReviewRepository,
	docRepo v32.DocumentRepository,
	eventEmitter v32.EventEmitter,
) *ReviewUseCase {
	return &ReviewUseCase{
		reviewRepo:   reviewRepo,
		docRepo:      docRepo,
		eventEmitter: eventEmitter,
	}
}

// SubmitReview interprets the review comment and determines action type
func (uc *ReviewUseCase) SubmitReview(ctx context.Context, documentID string, comment string) (*v32.Review, error) {
	// Get document to validate it exists and get Tier
	doc, err := uc.docRepo.Get(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Create Review
	review := &v32.Review{
		ID:         uuid.New().String(),
		DocumentID: documentID,
		Tier:       doc.Tier,
		Comment:    comment,
		ActionType: v32.ActionPending, // Analysis will be done asynchronously by ChatAgent
		CreatedAt:  time.Now(),
	}

	if err := uc.reviewRepo.Create(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	// Emit EventReviewSubmitted
	event := &v32.PlannerEvent{
		ID:         uuid.New().String(),
		Type:       v32.EventReviewSubmitted,
		DocumentID: documentID,
		Tier:       doc.Tier,
		ProjectID:  doc.ProjectID,
		Metadata: map[string]interface{}{
			"review_id": review.ID,
		},
		CreatedAt: time.Now(),
	}
	if err := uc.eventEmitter.Emit(event); err != nil {
		fmt.Printf("Warning: Failed to emit review submitted event: %v\n", err)
	}

	return review, nil
}

func (uc *ReviewUseCase) GetReviewsByDocument(ctx context.Context, documentID string) ([]*v32.Review, error) {
	return uc.reviewRepo.GetByDocumentID(ctx, documentID)
}
