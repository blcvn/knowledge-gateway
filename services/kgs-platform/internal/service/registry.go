package service

import (
	"context"

	pb "kgs-platform/api/registry/v1"
)

type RegistryService struct {
	pb.UnimplementedRegistryServer
}

func NewRegistryService() *RegistryService {
	return &RegistryService{}
}

func (s *RegistryService) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppReply, error) {
    return &pb.CreateAppReply{}, nil
}
func (s *RegistryService) GetApp(ctx context.Context, req *pb.GetAppRequest) (*pb.GetAppReply, error) {
    return &pb.GetAppReply{}, nil
}
func (s *RegistryService) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsReply, error) {
    return &pb.ListAppsReply{}, nil
}
func (s *RegistryService) IssueApiKey(ctx context.Context, req *pb.IssueApiKeyRequest) (*pb.IssueApiKeyReply, error) {
    return &pb.IssueApiKeyReply{}, nil
}
func (s *RegistryService) RevokeApiKey(ctx context.Context, req *pb.RevokeApiKeyRequest) (*pb.RevokeApiKeyReply, error) {
    return &pb.RevokeApiKeyReply{}, nil
}
