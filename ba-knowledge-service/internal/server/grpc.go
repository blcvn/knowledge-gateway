package server

import (
	"context"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/usecases"
	knowledgepb "github.com/blcvn/backend/services/proto/knowledge"
)

type KnowledgeServer struct {
	knowledgepb.UnimplementedKnowledgeServiceServer
	docUseCase *usecases.DocumentUseCase
}

func NewKnowledgeServer() *KnowledgeServer {
	return &KnowledgeServer{
		docUseCase: nil, // Should be injected in cmd/server/main.go
	}
}

func (s *KnowledgeServer) CreatePRD(ctx context.Context, req *knowledgepb.CreatePRDRequest) (*knowledgepb.CreatePRDResponse, error) {
	// Logic would invoke docUseCase.CreatePRD(...)
	return &knowledgepb.CreatePRDResponse{
		DocumentId: "stub-id",
		Content:    "Stub Content",
	}, nil
}

func (s *KnowledgeServer) GenerateUserStories(ctx context.Context, req *knowledgepb.GenerateUserStoriesRequest) (*knowledgepb.GenerateUserStoriesResponse, error) {
	return &knowledgepb.GenerateUserStoriesResponse{StoriesJson: "[]"}, nil
}
