package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/port"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// In a real scenario, this would implement the compiled proto interface.
// type memobaseIngestionServer struct { pb.UnimplementedMemobaseIngestionServiceServer }

type IngestionHandler struct {
	ingestUsecase port.IngestionUseCase
}

func NewIngestionHandler(ingestUsecase port.IngestionUseCase) *IngestionHandler {
	return &IngestionHandler{ingestUsecase: ingestUsecase}
}

// Simulated RPC Endpoint
type IngestRequest struct {
	Content  string
	Type     string
	Tokens   int
}

type IngestResponse struct {
	BlobID string
}

func (h *IngestionHandler) Ingest(ctx context.Context, req *IngestRequest) (*IngestResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing metadata")
	}

	tenantStrs := md.Get("x-tenant-id")
	if len(tenantStrs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing x-tenant-id")
	}

	tenantID, err := uuid.Parse(tenantStrs[0])
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid x-tenant-id")
	}

	// For simulation, using a dummy user ID or extracting from token
	userID := uuid.New()

	blob, err := h.ingestUsecase.IngestBlob(ctx, tenantID, userID, req.Content, req.Type, req.Tokens)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &IngestResponse{BlobID: blob.ID.String()}, nil
}

type Server struct {
	grpcServer *grpc.Server
	healthSrv  *health.Server
	port       int
	healthPort int
}

func NewServer(port, healthPort int, handler *IngestionHandler) *Server {
	s := grpc.NewServer()
	
	// pb.RegisterMemobaseIngestionServiceServer(s, handler)

	healthSrv := health.NewServer()
	
	return &Server{
		grpcServer: s,
		healthSrv:  healthSrv,
		port:       port,
		healthPort: healthPort,
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}

	go func() {
		hLis, _ := net.Listen("tcp", fmt.Sprintf(":%d", s.healthPort))
		hSrv := grpc.NewServer()
		grpc_health_v1.RegisterHealthServer(hSrv, s.healthSrv)
		s.healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		_ = hSrv.Serve(hLis)
	}()

	fmt.Printf("Starting gRPC server on port %d\n", s.port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
