package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor/validator"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/event"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/server"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/usecases"
	knowledgepb "github.com/blcvn/ba-shared-libs/proto/knowledge"
	persistencepb "github.com/blcvn/ba-shared-libs/proto/persistence"
	aiproxy "github.com/blcvn/kratos-proto/go/ai-proxy"
	promptpb "github.com/blcvn/kratos-proto/go/prompt"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("[KNOWLEDGE] Starting Knowledge Service...")

	// 1. Initialize Redis Client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test Redis connection
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Printf("[KNOWLEDGE] Warning: Failed to connect to Redis: %v", err)
	}

	// 2. Initialize gRPC Clients

	// Persistence Service
	persistenceAddr := os.Getenv("PERSISTENCE_ADDR")
	if persistenceAddr == "" {
		persistenceAddr = "localhost:50052"
	}
	persistenceConn, err := grpc.Dial(persistenceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[KNOWLEDGE] Failed to connect to Persistence Service: %v", err)
	}
	defer persistenceConn.Close()
	persistenceClient := persistencepb.NewPersistenceServiceClient(persistenceConn)

	// AI Proxy Service
	aiProxyAddr := os.Getenv("AI_PROXY_ADDR")
	if aiProxyAddr == "" {
		aiProxyAddr = "localhost:8087"
	}
	aiProxyConn, err := grpc.Dial(aiProxyAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[KNOWLEDGE] Failed to connect to AI Proxy Service: %v", err)
	}
	defer aiProxyConn.Close()
	aiProxyClient := aiproxy.NewAIProxyServiceClient(aiProxyConn)

	// Prompt Service
	promptAddr := os.Getenv("PROMPT_ADDR")
	if promptAddr == "" {
		promptAddr = "localhost:8086"
	}
	promptConn, err := grpc.Dial(promptAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("[KNOWLEDGE] Failed to connect to Prompt Service: %v", err)
	}
	defer promptConn.Close()
	promptClient := promptpb.NewPromptServiceClient(promptConn)

	// 3. Initialize Event System
	eventEmitter := event.NewSimpleEventEmitter()

	// 4. Initialize UseCases (partially, reviewUseCase first)
	reviewUseCase := usecases.NewReviewUseCase(
		persistenceClient,
		eventEmitter,
	)

	// Validator / Editor
	schemaPath := "templates/schema"
	templateLoader := validator.NewTemplateLoader(schemaPath)
	// Try to load templates, log warning if fails
	if err := templateLoader.LoadAllTemplates(); err != nil {
		log.Printf("Warning: Failed to load validation templates from %s: %v", schemaPath, err)
	}

	validatorConfig := &editor.AgentConfig{
		EnableKGUpdate:    false,
		EnableAutoFix:     false,
		ValidationTimeout: 30 * time.Second,
	}

	// Pass nil for LLM client as AutoFix is disabled
	validatorAgent := editor.NewValidatorAgent(templateLoader, nil, validatorConfig)

	// UseCases (DocumentUseCase now, with validatorAgent)
	docUseCase := usecases.NewDocumentUseCase(
		persistenceClient,
		aiProxyClient,
		promptClient,
		redisClient,
		eventEmitter,
		validatorAgent,
	)

	// 5. Initialize and Register Event Handlers
	workflowHandler := event.NewWorkflowEventHandler(docUseCase)
	eventEmitter.RegisterHandler(workflowHandler)

	// 6. Start Event Processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventEmitter.Start(ctx)

	// 7. Start gRPC Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "50053"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("[KNOWLEDGE] Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	knowledgeServer := server.NewKnowledgeServer(docUseCase, reviewUseCase)
	knowledgepb.RegisterKnowledgeServiceServer(s, knowledgeServer)

	// Enable reflection
	reflection.Register(s)

	log.Printf("[KNOWLEDGE] Knowledge Service listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("[KNOWLEDGE] Failed to serve: %v", err)
	}
}
