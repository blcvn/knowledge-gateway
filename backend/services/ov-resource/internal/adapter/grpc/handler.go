package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"openviking.com/ov-resource/internal/domain"
	"openviking.com/ov-resource/internal/usecase/dto"
	"openviking.com/ov-resource/internal/usecase/port"
)

// Define dummy PB interfaces locally since we don't have generated code
type IngestRequest struct { Content []byte; Filename string; Path string; AccountId string; ForceParser string }
type IngestResponse struct { ChunksCount int32; TotalTokens int32; Path string; ParseDurationMs int32 }
type ParseRequest struct { Content []byte; Filename string; ChunkSize int32; ChunkOverlap int32 }
type ParseResponse struct { Chunks []*Chunk }
type Chunk struct { Id string; Content string; StartLine int32; EndLine int32; TotalTokens int32 }
type WatchRequest struct { SourcePath string; TargetPath string; PollIntervalMs int64; Patterns []string; AccountId string }
type WatchEvent struct { Type string; Path string; Timestamp int64 }
type RefreshRequest struct { Paths []string; Force bool; AccountId string }
type RefreshResponse struct { Refreshed int32; Failed int32 }

type WatchStream interface {
	Send(*WatchEvent) error
}

type OvResourceServiceServer interface {
	Ingest(context.Context, *IngestRequest) (*IngestResponse, error)
	Parse(context.Context, *ParseRequest) (*ParseResponse, error)
	Watch(*WatchRequest, WatchStream) error
	Refresh(context.Context, *RefreshRequest) (*RefreshResponse, error)
}

type Handler struct {
	ingestUc  port.IngestUseCase
	parseUc   port.ParseUseCase
	watchUc   port.WatchUseCase
	refreshUc port.RefreshUseCase
}

func NewHandler(ingestUc port.IngestUseCase, parseUc port.ParseUseCase, watchUc port.WatchUseCase, refreshUc port.RefreshUseCase) *Handler {
	return &Handler{
		ingestUc:  ingestUc,
		parseUc:   parseUc,
		watchUc:   watchUc,
		refreshUc: refreshUc,
	}
}

func (h *Handler) Ingest(ctx context.Context, req *IngestRequest) (*IngestResponse, error) {
	dtoReq := dto.IngestRequest{
		Content:     req.Content,
		Filename:    req.Filename,
		Path:        req.Path,
		AccountID:   req.AccountId,
		ForceParser: req.ForceParser,
	}
	
	resp, err := h.ingestUc.Execute(ctx, dtoReq)
	if err != nil {
		if err == domain.ErrResourceExhausted {
			return nil, status.Errorf(codes.ResourceExhausted, err.Error())
		}
		if err == domain.ErrParseFailed {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	
	return &IngestResponse{
		ChunksCount:     int32(resp.ChunksCount),
		TotalTokens:     int32(resp.TotalTokens),
		Path:            resp.Path,
		ParseDurationMs: int32(resp.ParseDurationMs),
	}, nil
}

func (h *Handler) Parse(ctx context.Context, req *ParseRequest) (*ParseResponse, error) {
	dtoReq := dto.ParseRequest{
		Content:      req.Content,
		Filename:     req.Filename,
		ChunkSize:    req.ChunkSize,
		ChunkOverlap: req.ChunkOverlap,
	}
	
	resp, err := h.parseUc.Execute(ctx, dtoReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	
	var chunks []*Chunk
	for _, c := range resp.Chunks {
		chunks = append(chunks, &Chunk{
			Id:          c.ID,
			Content:     c.Content,
			StartLine:   int32(c.Metadata.StartLine),
			EndLine:     int32(c.Metadata.EndLine),
			TotalTokens: int32(c.Metadata.TotalTokens),
		})
	}
	return &ParseResponse{Chunks: chunks}, nil
}

func (h *Handler) Watch(req *WatchRequest, stream WatchStream) error {
	dtoReq := dto.WatchRequest{
		AccountID:      req.AccountId,
		SourcePath:     req.SourcePath,
		TargetPath:     req.TargetPath,
		PollIntervalMs: req.PollIntervalMs,
		Patterns:       req.Patterns,
	}
	
	ch, err := h.watchUc.Execute(context.Background(), dtoReq)
	if err != nil {
		return status.Errorf(codes.NotFound, err.Error())
	}
	
	for ev := range ch {
		if err := stream.Send(&WatchEvent{
			Type:      string(ev.Type),
			Path:      ev.Path,
			Timestamp: ev.Timestamp.Unix(),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Refresh(ctx context.Context, req *RefreshRequest) (*RefreshResponse, error) {
	dtoReq := dto.RefreshRequest{
		AccountID: req.AccountId,
		Paths:     req.Paths,
		Force:     req.Force,
	}
	
	resp, err := h.refreshUc.Execute(ctx, dtoReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	
	return &RefreshResponse{
		Refreshed: int32(resp.Refreshed),
		Failed:    int32(resp.Failed),
	}, nil
}
