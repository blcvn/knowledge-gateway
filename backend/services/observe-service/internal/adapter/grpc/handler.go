package grpc

import (
	"context"
	"encoding/json"

	observepb "github.com/vnp-memory/api/proto/observe/v1"
	"github.com/vnp-memory/services/observe-service/internal/observe"
	"github.com/vnp-memory/services/observe-service/internal/usecase"
	"github.com/vnp-memory/services/observe-service/internal/usecase/port"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ObserveHandler struct {
	observepb.UnimplementedObserveServiceServer
	observeUC       *usecase.ObserveUseCase
	createSessionUC *usecase.CreateSessionUseCase
	endSessionUC    *usecase.EndSessionUseCase
	sessionRepo     port.ISessionRepo
	stream          *observe.StreamBroker
}

func (h *ObserveHandler) Observe(ctx context.Context, req *observepb.ObserveRequest) (*observepb.ObserveResponse, error) {
	ucReq := observe.ObserveRequest{
		SessionID:         req.SessionId,
		HookType:          req.HookType,
		ToolName:          req.ToolName,
		ToolInput:         req.ToolInput,
		ToolOutput:        req.ToolOutput,
		UserPrompt:        req.UserPrompt,
		AssistantResponse: req.AssistantResponse,
		AgentID:           req.AgentId,
		TenantID:          req.TenantId,
	}
	if req.Timestamp != nil {
		ucReq.Timestamp = req.Timestamp.AsTime()
	}

	resp, err := h.observeUC.Execute(ctx, ucReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "observe: %v", err)
	}

	return mapObserveResponse(resp), nil
}

func (h *ObserveHandler) StartSession(ctx context.Context, req *observepb.StartSessionRequest) (*observepb.StartSessionResponse, error) {
	resp, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionRequest{
		TenantID: req.TenantId, Project: req.Project,
		CWD: req.Cwd, Model: req.Model, AgentID: req.AgentId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "start session: %v", err)
	}
	return &observepb.StartSessionResponse{SessionId: resp.SessionID, Status: resp.Status}, nil
}

func (h *ObserveHandler) EndSession(ctx context.Context, req *observepb.EndSessionRequest) (*observepb.EndSessionResponse, error) {
	resp, err := h.endSessionUC.Execute(ctx, usecase.EndSessionRequest{
		SessionID: req.SessionId, TenantID: req.TenantId,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "end session: %v", err)
	}
	return &observepb.EndSessionResponse{
		SessionId:        resp.SessionID,
		Status:           resp.Status,
		ObservationCount: int32(resp.ObservationCount),
	}, nil
}

func (h *ObserveHandler) ListSessions(ctx context.Context, req *observepb.ListSessionsRequest) (*observepb.ListSessionsResponse, error) {
	sessions, err := h.sessionRepo.List(ctx, req.TenantId, req.Status, req.Project, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
	}
	return mapListSessionsResponse(sessions), nil
}

func (h *ObserveHandler) GetSession(ctx context.Context, req *observepb.GetSessionRequest) (*observepb.GetSessionResponse, error) {
	sess, err := h.sessionRepo.GetByID(ctx, req.SessionId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "session not found")
	}
	return &observepb.GetSessionResponse{Session: mapSession(sess)}, nil
}

func (h *ObserveHandler) DeleteSession(ctx context.Context, req *observepb.DeleteSessionRequest) (*observepb.DeleteSessionResponse, error) {
	if err := h.sessionRepo.UpdateStatus(ctx, req.SessionId, "abandoned"); err != nil {
		return nil, status.Errorf(codes.Internal, "delete session: %v", err)
	}
	return &observepb.DeleteSessionResponse{Deleted: true}, nil
}

func (h *ObserveHandler) GetObservations(ctx context.Context, req *observepb.GetObservationsRequest) (*observepb.GetObservationsResponse, error) {
	obs, err := h.observeUC.GetObservations(ctx, req.SessionId, int(req.Limit), int(req.Offset))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get observations: %v", err)
	}
	return mapObservationsResponse(obs), nil
}

// StreamEvents — server-side streaming (SSE via gRPC)
func (h *ObserveHandler) StreamEvents(req *observepb.StreamEventsRequest, stream observepb.ObserveService_StreamEventsServer) error {
	ch, cancel := h.stream.Subscribe(req.SessionId)
	defer cancel()
	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			data, _ := json.Marshal(event.Data)
			stream.Send(&observepb.StreamEvent{
				EventType: event.Type,
				Data:      data,
			})
		case <-stream.Context().Done():
			return nil
		}
	}
}
