package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	contextv1 "github.com/vnp-community/vnp-memory/services/memobase-context/api/proto/memobase/context/v1"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase/port"
)

type Handler struct {
	contextv1.UnimplementedMemobaseContextServiceServer
	assembler port.ContextAssembler
	fetcher   port.ProfileFetcher
	searcher  port.ProfileSearcher
}

func NewHandler(assembler port.ContextAssembler, fetcher port.ProfileFetcher, searcher port.ProfileSearcher) *Handler {
	return &Handler{
		assembler: assembler,
		fetcher:   fetcher,
		searcher:  searcher,
	}
}

func extractTenantInfo(ctx context.Context, reqProjectID string) string {
	if reqProjectID != "" {
		return reqProjectID
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if vals := md.Get("x-tenant-id"); len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}

func (h *Handler) GetContext(ctx context.Context, req *contextv1.GetContextRequest) (*contextv1.ContextResponse, error) {
	projectID := extractTenantInfo(ctx, req.ProjectId)
	if projectID == "" {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	usecaseReq := &dto.GetContextRequest{
		UserID:            req.UserId,
		ProjectID:         projectID,
		MaxTokenSize:      req.MaxTokenSize,
		PreferTopics:      req.PreferTopics,
		OnlyTopics:        req.OnlyTopics,
		ProfileEventRatio: req.ProfileEventRatio,
		ChatsContext:      req.ChatsContext,
	}

	resp, err := h.assembler.GetContext(ctx, usecaseReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get context: %v", err)
	}

	return &contextv1.ContextResponse{
		Context:      resp.Context,
		ProfileCount: resp.ProfileCount,
		EventCount:   resp.EventCount,
		TotalTokens:  resp.TotalTokens,
	}, nil
}

func (h *Handler) GetProfiles(ctx context.Context, req *contextv1.GetProfilesRequest) (*contextv1.ProfilesResponse, error) {
	projectID := extractTenantInfo(ctx, req.ProjectId)
	if projectID == "" {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	usecaseReq := &dto.GetProfilesRequest{
		UserID:          req.UserId,
		ProjectID:       projectID,
		Topics:          req.Topics,
		MaxSubtopicSize: req.MaxSubtopicSize,
		MaxTokenSize:    req.MaxTokenSize,
	}

	resp, err := h.fetcher.GetProfiles(ctx, usecaseReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get profiles: %v", err)
	}

	var grpcProfiles []*contextv1.UserProfile
	for _, p := range resp.Profiles {
		grpcProfiles = append(grpcProfiles, &contextv1.UserProfile{
			Id:        p.ID,
			Topic:     p.Topic,
			SubTopic:  p.SubTopic,
			Content:   p.Content,
			UpdatedAt: timestamppb.New(p.UpdatedAt),
		})
	}

	return &contextv1.ProfilesResponse{Profiles: grpcProfiles}, nil
}

func (h *Handler) SearchProfiles(ctx context.Context, req *contextv1.SearchProfilesRequest) (*contextv1.SearchProfilesResponse, error) {
	projectID := extractTenantInfo(ctx, req.ProjectId)
	if projectID == "" {
		return nil, status.Error(codes.InvalidArgument, "project_id is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	usecaseReq := &dto.SearchProfilesRequest{
		UserID:    req.UserId,
		ProjectID: projectID,
		Query:     req.Query,
		Limit:     req.Limit,
	}

	resp, err := h.searcher.SearchProfiles(ctx, usecaseReq)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search profiles: %v", err)
	}

	var grpcProfiles []*contextv1.UserProfile
	for _, p := range resp.Profiles {
		grpcProfiles = append(grpcProfiles, &contextv1.UserProfile{
			Id:        p.ID,
			Topic:     p.Topic,
			SubTopic:  p.SubTopic,
			Content:   p.Content,
			UpdatedAt: timestamppb.New(p.UpdatedAt),
		})
	}

	return &contextv1.SearchProfilesResponse{Profiles: grpcProfiles}, nil
}
