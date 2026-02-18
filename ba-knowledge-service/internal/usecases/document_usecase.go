package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/generators/full"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/generators/index"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/generators/outline"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/parsers/prd"
	persistencepb "github.com/blcvn/backend/services/proto/persistence"
	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	promptpb "github.com/blcvn/kratos-proto/go/prompt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// DocumentUseCase handles 3-tier document generation
type DocumentUseCase struct {
	persistenceClient persistencepb.PersistenceServiceClient
	aiProxyClient     aiproxy.AIProxyServiceClient
	promptClient      promptpb.PromptServiceClient
	redisClient       *redis.Client
	prdParser         *prd.Parser
}

// NewDocumentUseCase creates a new document usecase
func NewDocumentUseCase(
	persistenceClient persistencepb.PersistenceServiceClient,
	aiProxyClient aiproxy.AIProxyServiceClient,
	promptClient promptpb.PromptServiceClient,
	redisClient *redis.Client,
) *DocumentUseCase {
	return &DocumentUseCase{
		persistenceClient: persistenceClient,
		aiProxyClient:     aiProxyClient,
		promptClient:      promptClient,
		redisClient:       redisClient,
		prdParser:         prd.NewParser(aiProxyClient, promptClient),
	}
}

// CreatePRD creates a PRD document and triggers the workflow
func (u *DocumentUseCase) CreatePRD(ctx context.Context, projectID, moduleName string, content string) (string, error) {
	fmt.Printf("[DocumentUseCase.CreatePRD] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.CreatePRD] ProjectID: %s, ModuleName: %s, ContentLength: %d\n", projectID, moduleName, len(content))

	docID := uuid.New().String()

	// Create document via Persistence Service
	req := &persistencepb.CreateDocumentRequest{
		Id:         docID,
		Title:      fmt.Sprintf("PRD - %s", moduleName),
		Content:    content,
		Author:     "system",
		ProjectId:  projectID,
		ParentId:   "",
		RootId:     docID,
		ModuleName: moduleName,
		Tier:       "PRD",
		Status:     "approved",
		Version:    1,
	}

	resp, err := u.persistenceClient.CreateDocument(ctx, req)
	if err != nil {
		fmt.Printf("[DocumentUseCase.CreatePRD] ERROR: Failed to create document: %v\n", err)
		return "", fmt.Errorf("failed to create PRD document: %w", err)
	}

	if !resp.Success {
		return "", fmt.Errorf("failed to create PRD document: %s", resp.Error)
	}

	fmt.Printf("[DocumentUseCase.CreatePRD] PRD saved successfully with ID: %s\n", docID)

	// Emit event to Redis queue for Knowledge Worker to process
	event := map[string]interface{}{
		"id":          uuid.New().String(),
		"type":        "PRDUploaded",
		"document_id": docID,
		"project_id":  projectID,
		"tier":        "PRD",
		"created_at":  time.Now().Format(time.RFC3339),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("[DocumentUseCase.CreatePRD] Warning: Failed to marshal event: %v\n", err)
	} else {
		if err := u.redisClient.Publish(ctx, "document_events", eventJSON).Err(); err != nil {
			fmt.Printf("[DocumentUseCase.CreatePRD] Warning: Failed to publish event: %v\n", err)
		}
	}

	fmt.Printf("[DocumentUseCase.CreatePRD] ====== SUCCESS ======\n")
	return docID, nil
}

// GenerateIndex generates a URD Index document
func (u *DocumentUseCase) GenerateIndex(ctx context.Context, parentID string) (string, error) {
	fmt.Printf("[DocumentUseCase.GenerateIndex] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.GenerateIndex] ParentID (PRD): %s\n", parentID)

	// 1. Get PRD document from Persistence Service
	prdResp, err := u.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: parentID})
	if err != nil {
		return "", fmt.Errorf("failed to get PRD document: %w", err)
	}

	fmt.Printf("[DocumentUseCase.GenerateIndex] PRD found: %s\n", prdResp.Title)

	// 2. Parse PRD and build knowledge graph
	// TODO: Implement KG building logic
	// For now, we'll generate index directly from PRD content

	// 3. Generate index using AI Proxy
	generator := index.NewGenerator(u.aiProxyClient, u.promptClient)
	indexContent, err := generator.GenerateIndex(ctx, prdResp.Content, prdResp.Title)
	if err != nil {
		return "", fmt.Errorf("failed to generate index: %w", err)
	}

	// 4. Save index document
	indexID := uuid.New().String()
	createReq := &persistencepb.CreateDocumentRequest{
		Id:         indexID,
		Title:      fmt.Sprintf("URD Index - %s", prdResp.Title),
		Content:    indexContent,
		Author:     "system",
		ProjectId:  "", // TODO: Get from PRD
		ParentId:   parentID,
		RootId:     parentID,
		ModuleName: prdResp.Title,
		Tier:       "URD_INDEX",
		Status:     "reviewing",
		Version:    1,
	}

	createResp, err := u.persistenceClient.CreateDocument(ctx, createReq)
	if err != nil {
		return "", fmt.Errorf("failed to save index document: %w", err)
	}

	if !createResp.Success {
		return "", fmt.Errorf("failed to save index document: %s", createResp.Error)
	}

	fmt.Printf("[DocumentUseCase.GenerateIndex] ====== SUCCESS ======\n")
	return indexID, nil
}

// GenerateOutline generates a URD Outline document
func (u *DocumentUseCase) GenerateOutline(ctx context.Context, parentID string) (string, error) {
	fmt.Printf("[DocumentUseCase.GenerateOutline] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.GenerateOutline] ParentID (Index): %s\n", parentID)

	// 1. Get parent (Index) document
	indexResp, err := u.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: parentID})
	if err != nil {
		return "", fmt.Errorf("failed to get index document: %w", err)
	}

	// 2. Generate outline using AI Proxy
	generator := outline.NewGenerator(u.aiProxyClient, u.promptClient)
	outlineContent, err := generator.GenerateOutline(ctx, indexResp.Content, indexResp.Title)
	if err != nil {
		return "", fmt.Errorf("failed to generate outline: %w", err)
	}

	// 3. Save outline document
	outlineID := uuid.New().String()
	createReq := &persistencepb.CreateDocumentRequest{
		Id:         outlineID,
		Title:      fmt.Sprintf("URD Outline - %s", indexResp.Title),
		Content:    outlineContent,
		Author:     "system",
		ProjectId:  "", // TODO: Get from Index
		ParentId:   parentID,
		RootId:     "", // TODO: Track root document
		ModuleName: indexResp.Title,
		Tier:       "URD_OUTLINE",
		Status:     "draft",
		Version:    1,
	}

	createResp, err := u.persistenceClient.CreateDocument(ctx, createReq)
	if err != nil {
		return "", fmt.Errorf("failed to save outline document: %w", err)
	}

	if !createResp.Success {
		return "", fmt.Errorf("failed to save outline document: %s", createResp.Error)
	}

	fmt.Printf("[DocumentUseCase.GenerateOutline] ====== SUCCESS ======\n")
	return outlineID, nil
}

// GenerateFull generates a Full URD document
func (u *DocumentUseCase) GenerateFull(ctx context.Context, parentID string) (string, error) {
	fmt.Printf("[DocumentUseCase.GenerateFull] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.GenerateFull] ParentID (Outline): %s\n", parentID)

	// 1. Get parent (Outline) document
	outlineResp, err := u.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: parentID})
	if err != nil {
		return "", fmt.Errorf("failed to get outline document: %w", err)
	}

	// 2. Generate full URD using AI Proxy
	generator := full.NewGenerator(u.aiProxyClient, u.promptClient)
	fullContent, err := generator.GenerateFull(ctx, outlineResp.Content, outlineResp.Title)
	if err != nil {
		return "", fmt.Errorf("failed to generate full URD: %w", err)
	}

	// 3. Save full URD document
	fullID := uuid.New().String()
	createReq := &persistencepb.CreateDocumentRequest{
		Id:         fullID,
		Title:      fmt.Sprintf("URD Full - %s", outlineResp.Title),
		Content:    fullContent,
		Author:     "system",
		ProjectId:  "", // TODO: Get from Outline
		ParentId:   parentID,
		RootId:     "", // TODO: Track root document
		ModuleName: outlineResp.Title,
		Tier:       "URD_FULL",
		Status:     "draft",
		Version:    1,
	}

	createResp, err := u.persistenceClient.CreateDocument(ctx, createReq)
	if err != nil {
		return "", fmt.Errorf("failed to save full URD document: %w", err)
	}

	if !createResp.Success {
		return "", fmt.Errorf("failed to save full URD document: %s", createResp.Error)
	}

	fmt.Printf("[DocumentUseCase.GenerateFull] ====== SUCCESS ======\n")
	return fullID, nil
}

// RegenerateDocument regenerates a document based on a review
func (u *DocumentUseCase) RegenerateDocument(ctx context.Context, reviewID string) error {
	fmt.Printf("[DocumentUseCase.RegenerateDocument] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.RegenerateDocument] ReviewID: %s\n", reviewID)

	// TODO: Implement review-based regeneration
	// This requires:
	// 1. Get review from Persistence Service
	// 2. Get original document
	// 3. Use ChatAgent to process comments and update document
	// 4. Save updated document version

	fmt.Printf("[DocumentUseCase.RegenerateDocument] TODO: Implement review-based regeneration\n")
	return fmt.Errorf("not implemented yet")
}

// ApproveDocument approves a document and triggers next steps
func (u *DocumentUseCase) ApproveDocument(ctx context.Context, id string) error {
	fmt.Printf("[DocumentUseCase.ApproveDocument] ====== START ======\n")
	fmt.Printf("[DocumentUseCase.ApproveDocument] DocumentID: %s\n", id)

	// Create approval record via Persistence Service
	approvalReq := &persistencepb.CreateApprovalRequest{
		DocumentId: id,
		Approver:   "system", // TODO: Get from context
		Status:     "approved",
	}

	approvalResp, err := u.persistenceClient.CreateApproval(ctx, approvalReq)
	if err != nil {
		return fmt.Errorf("failed to create approval: %w", err)
	}

	if !approvalResp.Success {
		return fmt.Errorf("failed to create approval")
	}

	// Emit approval event
	event := map[string]interface{}{
		"id":          uuid.New().String(),
		"type":        "DocumentApproved",
		"document_id": id,
		"created_at":  time.Now().Format(time.RFC3339),
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("[DocumentUseCase.ApproveDocument] Warning: Failed to marshal event: %v\n", err)
	} else {
		if err := u.redisClient.Publish(ctx, "document_events", eventJSON).Err(); err != nil {
			fmt.Printf("[DocumentUseCase.ApproveDocument] Warning: Failed to publish event: %v\n", err)
		}
	}

	fmt.Printf("[DocumentUseCase.ApproveDocument] ====== SUCCESS ======\n")
	return nil
}
