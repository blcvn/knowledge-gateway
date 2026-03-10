package service

import (
	"context"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/registry/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
)

type RegistryService struct {
	pb.UnimplementedRegistryServer
	uc *biz.RegistryUsecase
}

func NewRegistryService(uc *biz.RegistryUsecase) *RegistryService {
	return &RegistryService{uc: uc}
}

func (s *RegistryService) CreateApp(ctx context.Context, req *pb.CreateAppRequest) (*pb.CreateAppReply, error) {
	app, err := s.uc.CreateApp(ctx, req.AppName, req.Description, req.Owner)
	if err != nil {
		return nil, err
	}
	return &pb.CreateAppReply{
		AppId:  app.AppID,
		Status: app.Status,
	}, nil
}
func (s *RegistryService) GetApp(ctx context.Context, req *pb.GetAppRequest) (*pb.GetAppReply, error) {
	app, err := s.uc.GetApp(ctx, req.AppId)
	if err != nil {
		return nil, err
	}
	return &pb.GetAppReply{
		AppId:       app.AppID,
		AppName:     app.AppName,
		Description: app.Description,
		Owner:       app.Owner,
		Status:      app.Status,
	}, nil
}
func (s *RegistryService) ListApps(ctx context.Context, req *pb.ListAppsRequest) (*pb.ListAppsReply, error) {
	apps, err := s.uc.ListApps(ctx)
	if err != nil {
		return nil, err
	}
	out := &pb.ListAppsReply{Apps: make([]*pb.GetAppReply, 0, len(apps))}
	for _, app := range apps {
		out.Apps = append(out.Apps, &pb.GetAppReply{
			AppId:       app.AppID,
			AppName:     app.AppName,
			Description: app.Description,
			Owner:       app.Owner,
			Status:      app.Status,
		})
	}
	return out, nil
}
func (s *RegistryService) IssueApiKey(ctx context.Context, req *pb.IssueApiKeyRequest) (*pb.IssueApiKeyReply, error) {
	rawKey, key, err := s.uc.IssueAPIKey(ctx, req.AppId, req.Name, req.Scopes, req.TtlSeconds)
	if err != nil {
		return nil, err
	}
	return &pb.IssueApiKeyReply{
		ApiKey:    rawKey,
		KeyHash:   key.KeyHash,
		KeyPrefix: key.KeyPrefix,
	}, nil
}
func (s *RegistryService) RevokeApiKey(ctx context.Context, req *pb.RevokeApiKeyRequest) (*pb.RevokeApiKeyReply, error) {
	if err := s.uc.RevokeAPIKey(ctx, req.KeyHash); err != nil {
		return nil, err
	}
	return &pb.RevokeApiKeyReply{Success: true}, nil
}
