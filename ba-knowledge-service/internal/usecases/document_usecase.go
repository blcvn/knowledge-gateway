package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/blcvn/backend/services/pkg/domain"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor"
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
	eventEmitter      domain.EventEmitter
	validatorAgent    *editor.ValidatorAgent
}

// NewDocumentUseCase creates a new document usecase
func NewDocumentUseCase(
	persistenceClient persistencepb.PersistenceServiceClient,
	aiProxyClient aiproxy.AIProxyServiceClient,
	promptClient promptpb.PromptServiceClient,
	redisClient *redis.Client,
	eventEmitter domain.EventEmitter,
	validatorAgent *editor.ValidatorAgent,
) *DocumentUseCase {
	return &DocumentUseCase{
		persistenceClient: persistenceClient,
		aiProxyClient:     aiProxyClient,
		promptClient:      promptClient,
		redisClient:       redisClient,
		prdParser:         prd.NewParser(aiProxyClient, promptClient),
		eventEmitter:      eventEmitter,
		validatorAgent:    validatorAgent,
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

	// Emit event
	event := &domain.PlannerEvent{
		Type:       domain.EventPRDUploaded,
		DocumentID: docID,
		ProjectID:  projectID,
		Tier:       domain.TierPRD,
		CreatedAt:  time.Now(),
	}

	if err := u.eventEmitter.Emit(event); err != nil {
		fmt.Printf("[DocumentUseCase.CreatePRD] Warning: Failed to emit event: %v\n", err)
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
	event := &domain.PlannerEvent{
		Type:       domain.EventFullApproved, // TODO: Determine event type based on input doc type!
		DocumentID: id,
		CreatedAt:  time.Now(),
	}
	// Identify correct event type
	// Ideally we fetch the document first to know its tier.
	doc, _ := u.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: id})
	if doc != nil {
		switch doc.Tier {
		case "URD_INDEX":
			event.Type = domain.EventIndexApproved
			event.Tier = domain.TierURDIndex
			event.ProjectID = doc.ProjectId
		case "URD_OUTLINE":
			event.Type = domain.EventOutlineApproved
			event.Tier = domain.TierURDOutline
			event.ProjectID = doc.ProjectId
		case "URD_FULL":
			event.Type = domain.EventFullApproved
			event.Tier = domain.TierURDFull
			event.ProjectID = doc.ProjectId
		}
	}

	if err := u.eventEmitter.Emit(event); err != nil {
		fmt.Printf("[DocumentUseCase.ApproveDocument] Warning: Failed to emit event: %v\n", err)
	}

	fmt.Printf("[DocumentUseCase.ApproveDocument] ====== SUCCESS ======\n")
	return nil
}

// GetDocument retrieves a document by ID
func (u *DocumentUseCase) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	resp, err := u.persistenceClient.GetDocument(ctx, &persistencepb.GetDocumentRequest{Id: id})
	if err != nil {
		return nil, err
	}

	// Map persistence response to domain document
	tier := domain.RequirementTier(0)
	switch resp.Tier {
	case "PRD":
		tier = domain.TierPRD
	case "URD_INDEX":
		tier = domain.TierURDIndex
	case "URD_OUTLINE":
		tier = domain.TierURDOutline
	case "URD_FULL":
		tier = domain.TierURDFull
	}

	return &domain.Document{
		ID:               resp.Id,
		ProjectID:        resp.ProjectId,
		Content:          resp.Content,
		Tier:             tier,
		Status:           domain.DocumentStatus(resp.Status),
		ModuleName:       resp.ModuleName,
		CreatedAt:        resp.CreatedAt.AsTime(),
		UpdatedAt:        resp.UpdatedAt.AsTime(),
		ParentDocumentID: resp.ParentId,
	}, nil
}

// GetDocumentByParentId retrieves a document by parent ID and tier
func (u *DocumentUseCase) GetDocumentByParentId(ctx context.Context, parentID string, tier domain.RequirementTier) (*domain.Document, error) {
	tierStr := ""
	switch tier {
	case domain.TierPRD:
		tierStr = "PRD"
	case domain.TierURDIndex:
		tierStr = "URD_INDEX"
	case domain.TierURDOutline:
		tierStr = "URD_OUTLINE"
	case domain.TierURDFull:
		tierStr = "URD_FULL"
	}

	req := &persistencepb.ListDocumentsRequest{
		ParentId: parentID,
		Tier:     tierStr,
	}

	resp, err := u.persistenceClient.ListDocuments(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	if len(resp.Documents) == 0 {
		return nil, fmt.Errorf("document not found for parent %s and tier %s", parentID, tierStr)
	}

	// Return first match
	doc := resp.Documents[0]

	// Map persistence response to domain document (Duplicated mapping logic, should extract helper)
	// For now, inline
	return &domain.Document{
		ID:               doc.Id,
		ProjectID:        doc.ProjectId,
		Content:          doc.Content,
		Tier:             tier,
		Status:           domain.DocumentStatus(doc.Status),
		ModuleName:       doc.ModuleName,
		CreatedAt:        doc.CreatedAt.AsTime(),
		UpdatedAt:        doc.UpdatedAt.AsTime(),
		ParentDocumentID: doc.ParentId,
	}, nil
}

// SaveEditedDocument validates and saves edited content
func (u *DocumentUseCase) SaveEditedDocument(ctx context.Context, id string, tier domain.RequirementTier, content string) error {
	fmt.Printf("[DocumentUseCase.SaveEditedDocument] Validating content for ID: %s\n", id)

	// Validate
	if u.validatorAgent != nil {
		doc := &domain.Document{ID: id, Tier: tier}
		result, err := u.validatorAgent.Execute(ctx, doc, content)
		if err != nil {
			return fmt.Errorf("validator execution failed: %w", err)
		}
		if !result.Success {
			// Convert validation errors to string for now
			errMsgs := ""
			for _, e := range result.Errors {
				errMsgs += fmt.Sprintf("[%s] %s\n", e.Type, e.Message)
			}
			return fmt.Errorf("validation failed:\n%s", errMsgs)
		}
	}

	// Update via Persistence
	req := &persistencepb.UpdateDocumentRequest{
		DocumentId: id,
		Content:    content,
	}

	resp, err := u.persistenceClient.UpdateDocument(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("failed to update document: %s", resp.Error)
	}

	return nil
}
