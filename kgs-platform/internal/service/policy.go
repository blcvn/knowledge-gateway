package service

import (
	"context"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/accesscontrol/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
)

type PolicyService struct {
	pb.UnimplementedAccessControlServer
	uc *biz.PolicyUsecase
}

func NewPolicyService(uc *biz.PolicyUsecase) *PolicyService {
	return &PolicyService{uc: uc}
}

func (s *PolicyService) CreatePolicy(ctx context.Context, req *pb.CreatePolicyRequest) (*pb.PolicyReply, error) {
	appID := "demo-app"

	policy := &biz.Policy{
		AppID:       appID,
		Name:        req.Name,
		Description: req.Description,
		RegoContent: req.RegoContent,
	}

	created, err := s.uc.CreatePolicy(ctx, policy)
	if err != nil {
		return nil, err
	}

	return &pb.PolicyReply{
		Id:          int64(created.ID),
		Name:        created.Name,
		Description: created.Description,
		RegoContent: created.RegoContent,
		IsActive:    created.IsActive,
	}, nil
}

func (s *PolicyService) ListPolicies(ctx context.Context, req *pb.ListPoliciesRequest) (*pb.ListPoliciesReply, error) {
	appID := "demo-app"
	policies, err := s.uc.ListPolicies(ctx, appID)
	if err != nil {
		return nil, err
	}

	var reply pb.ListPoliciesReply
	for _, p := range policies {
		reply.Policies = append(reply.Policies, &pb.PolicyReply{
			Id:          int64(p.ID),
			Name:        p.Name,
			Description: p.Description,
			RegoContent: p.RegoContent,
			IsActive:    p.IsActive,
		})
	}
	return &reply, nil
}

func (s *PolicyService) GetPolicy(ctx context.Context, req *pb.GetPolicyRequest) (*pb.PolicyReply, error) {
	policy, err := s.uc.GetPolicy(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}

	return &pb.PolicyReply{
		Id:          int64(policy.ID),
		Name:        policy.Name,
		Description: policy.Description,
		RegoContent: policy.RegoContent,
		IsActive:    policy.IsActive,
	}, nil
}
