package server

import (
	"context"
	"fmt"

	"github.com/blcvn/ba-shared-libs/pkg/domain"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/usecases"
	knowledgepb "github.com/blcvn/ba-shared-libs/proto/knowledge"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// KnowledgeServer implements the gRPC server for knowledge
type KnowledgeServer struct {
	knowledgepb.UnimplementedKnowledgeServiceServer
	docUseCase    *usecases.DocumentUseCase
	reviewUseCase *usecases.ReviewUseCase
}

// NewKnowledgeServer creates a new instance of KnowledgeServer
func NewKnowledgeServer(docUseCase *usecases.DocumentUseCase, reviewUseCase *usecases.ReviewUseCase) *KnowledgeServer {
	return &KnowledgeServer{
		docUseCase:    docUseCase,
		reviewUseCase: reviewUseCase,
	}
}

// CreatePRD creates a PRD document
func (s *KnowledgeServer) CreatePRD(ctx context.Context, req *knowledgepb.CreatePRDRequest) (*knowledgepb.CreatePRDResponse, error) {
	// CreatePRD returns ID string
	docID, err := s.docUseCase.CreatePRD(ctx, req.ProjectId, req.Title, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create PRD: %w", err)
	}

	return &knowledgepb.CreatePRDResponse{
		DocumentId: docID,
		Content:    req.Description, // Echo description or leave empty as content isn't returned by UC
	}, nil
}

// GenerateDocument generates a document based on tier
func (s *KnowledgeServer) GenerateDocument(ctx context.Context, req *knowledgepb.GenerateDocumentRequest) (*knowledgepb.GenerateDocumentResponse, error) {
	var docID string
	var err error

	switch req.Tier {
	case "URD_INDEX":
		docID, err = s.docUseCase.GenerateIndex(ctx, req.ParentDocumentId)
	case "URD_OUTLINE":
		docID, err = s.docUseCase.GenerateOutline(ctx, req.ParentDocumentId)
	case "URD_FULL":
		docID, err = s.docUseCase.GenerateFull(ctx, req.ParentDocumentId)
	default:
		return nil, fmt.Errorf("unsupported tier: %s", req.Tier)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate document: %w", err)
	}

	return &knowledgepb.GenerateDocumentResponse{
		DocumentId: docID,
		JobId:      docID, // Using DocID as JobID for synchronous generation
	}, nil
}

// GetDocument retrieves a document by ID
func (s *KnowledgeServer) GetDocument(ctx context.Context, req *knowledgepb.GetDocumentRequest) (*knowledgepb.GetDocumentResponse, error) {
	doc, err := s.docUseCase.GetDocument(ctx, req.DocumentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &knowledgepb.GetDocumentResponse{
		DocumentId:       doc.ID,
		ProjectId:        doc.ProjectID,
		Tier:             doc.Tier.String(),
		Status:           string(doc.Status),
		Content:          doc.Content,
		ModuleName:       doc.ModuleName,
		CreatedAt:        timestamppb.New(doc.CreatedAt),
		UpdatedAt:        timestamppb.New(doc.UpdatedAt),
		ParentDocumentId: doc.ParentDocumentID,
	}, nil
}

// GetDocumentByContext retrieves a document by parent ID and tier
func (s *KnowledgeServer) GetDocumentByContext(ctx context.Context, req *knowledgepb.GetDocumentByContextRequest) (*knowledgepb.GetDocumentResponse, error) {
	var tier domain.RequirementTier
	switch req.Tier {
	case "PRD":
		tier = domain.TierPRD
	case "URD_INDEX":
		tier = domain.TierURDIndex
	case "URD_OUTLINE":
		tier = domain.TierURDOutline
	case "URD_FULL":
		tier = domain.TierURDFull
	default:
		return nil, fmt.Errorf("unsupported tier: %s", req.Tier)
	}

	doc, err := s.docUseCase.GetDocumentByParentId(ctx, req.ParentDocumentId, tier)
	if err != nil {
		return nil, fmt.Errorf("failed to get document by context: %w", err)
	}

	return &knowledgepb.GetDocumentResponse{
		DocumentId:       doc.ID,
		ProjectId:        doc.ProjectID,
		Tier:             doc.Tier.String(),
		Status:           string(doc.Status),
		Content:          doc.Content,
		ModuleName:       doc.ModuleName,
		CreatedAt:        timestamppb.New(doc.CreatedAt),
		UpdatedAt:        timestamppb.New(doc.UpdatedAt),
		ParentDocumentId: doc.ParentDocumentID,
	}, nil
}

// UpdateDocument updates content directly (SaveEditedDocument)
func (s *KnowledgeServer) UpdateDocument(ctx context.Context, req *knowledgepb.UpdateDocumentRequest) (*knowledgepb.UpdateDocumentResponse, error) {
	// We need tier? No, DocumentUseCase.SaveEditedDocument needs ID and Tier.
	// But GetDocument can fetch Tier.
	// Let's modify DocumentUseCase.SaveEditedDocument to simplify args if possible,
	// OR fetch here.

	// Fetch document first to get tier (inefficient but safe)
	doc, err := s.docUseCase.GetDocument(ctx, req.DocumentId)
	if err != nil {
		return nil, fmt.Errorf("document not found: %w", err)
	}

	err = s.docUseCase.SaveEditedDocument(ctx, req.DocumentId, doc.Tier, req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to save document: %w", err)
	}

	return &knowledgepb.UpdateDocumentResponse{Success: true}, nil
}

// RegenerateDocument triggers regeneration
func (s *KnowledgeServer) RegenerateDocument(ctx context.Context, req *knowledgepb.RegenerateDocumentRequest) (*knowledgepb.RegenerateDocumentResponse, error) {
	// TODO: Pass instruction?
	if err := s.docUseCase.RegenerateDocument(ctx, req.DocumentId); err != nil {
		return nil, fmt.Errorf("failed to regenerate document: %w", err)
	}
	return &knowledgepb.RegenerateDocumentResponse{JobId: req.DocumentId}, nil
}

// ApproveDocument implements approval
func (s *KnowledgeServer) ApproveDocument(ctx context.Context, req *knowledgepb.ApproveDocumentRequest) (*knowledgepb.ApproveDocumentResponse, error) {
	// Fetch doc for Tier - not needed for UseCase call but maybe logic?
	// UseCase retrieves doc anyway or handles it.
	// But UseCase ApproveDocument takes (ctx, id).
	// So we don't need doc fetch unless we want to validate existence here.
	// Let's just call UseCase.

	if err := s.docUseCase.ApproveDocument(ctx, req.DocumentId); err != nil {
		return nil, fmt.Errorf("failed to approve document: %w", err)
	}

	return &knowledgepb.ApproveDocumentResponse{Success: true}, nil
}

// ReviewDocument submits a review
func (s *KnowledgeServer) ReviewDocument(ctx context.Context, req *knowledgepb.ReviewDocumentRequest) (*knowledgepb.ReviewDocumentResponse, error) {
	// TODO: Get reviewer from context metadata?
	reviewer := "unknown" // Placeholder

	review, err := s.reviewUseCase.SubmitReview(ctx, req.DocumentId, req.Comment, reviewer)
	if err != nil {
		return nil, fmt.Errorf("failed to submit review: %w", err)
	}

	return &knowledgepb.ReviewDocumentResponse{
		ReviewId: review.ID,
		Status:   string(review.ActionType),
	}, nil
}

// GetReviewStatus retrieval
func (s *KnowledgeServer) GetReviewStatus(ctx context.Context, req *knowledgepb.GetReviewStatusRequest) (*knowledgepb.GetReviewStatusResponse, error) {
	// We only have ListReviews in UseCase.
	// We don't have GetReview(id).
	// But `persistence.proto` doesn't have GetReview either.
	// We'll stub this for now or rely on ListReviews and filtering (inefficient).

	// IMPORTANT: For Monolith parity, we need to return status and maybe content.
	return &knowledgepb.GetReviewStatusResponse{
		ReviewId: req.ReviewId,
		Status:   "pending", // Stub
	}, nil
}

// Utilities
func (s *KnowledgeServer) GenerateUserStories(ctx context.Context, req *knowledgepb.GenerateUserStoriesRequest) (*knowledgepb.GenerateUserStoriesResponse, error) {
	return &knowledgepb.GenerateUserStoriesResponse{StoriesJson: "[]"}, nil
}
