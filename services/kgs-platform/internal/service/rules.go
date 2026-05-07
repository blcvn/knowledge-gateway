package service

import (
	"context"

	pb "kgs-platform/api/rules/v1"
	"kgs-platform/internal/biz"
)

type RulesService struct {
	pb.UnimplementedRulesServer
	uc *biz.RulesUsecase
}

func NewRulesService(uc *biz.RulesUsecase) *RulesService {
	return &RulesService{uc: uc}
}

func (s *RulesService) CreateRule(ctx context.Context, req *pb.CreateRuleRequest) (*pb.RuleReply, error) {
	// appID extraction from context would happen here via Auth Middleware
	appID := "demo-app"

	rule := &biz.Rule{
		AppID:       appID,
		Name:        req.Name,
		Description: req.Description,
		TriggerType: req.TriggerType,
		Cron:        req.Cron,
		CypherQuery: req.CypherQuery,
		Action:      req.Action,
	}

	created, err := s.uc.CreateRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	return &pb.RuleReply{
		Id:          int64(created.ID),
		Name:        created.Name,
		Description: created.Description,
		TriggerType: created.TriggerType,
		Cron:        created.Cron,
		CypherQuery: created.CypherQuery,
		Action:      created.Action,
		IsActive:    created.IsActive,
	}, nil
}

func (s *RulesService) ListRules(ctx context.Context, req *pb.ListRulesRequest) (*pb.ListRulesReply, error) {
	appID := "demo-app"
	rules, err := s.uc.ListRules(ctx, appID)
	if err != nil {
		return nil, err
	}

	var reply pb.ListRulesReply
	for _, r := range rules {
		reply.Rules = append(reply.Rules, &pb.RuleReply{
			Id:          int64(r.ID),
			Name:        r.Name,
			Description: r.Description,
			TriggerType: r.TriggerType,
			Cron:        r.Cron,
			CypherQuery: r.CypherQuery,
			Action:      r.Action,
			IsActive:    r.IsActive,
		})
	}
	return &reply, nil
}

func (s *RulesService) GetRule(ctx context.Context, req *pb.GetRuleRequest) (*pb.RuleReply, error) {
	rule, err := s.uc.GetRule(ctx, uint(req.Id))
	if err != nil {
		return nil, err
	}

	return &pb.RuleReply{
		Id:          int64(rule.ID),
		Name:        rule.Name,
		Description: rule.Description,
		TriggerType: rule.TriggerType,
		Cron:        rule.Cron,
		CypherQuery: rule.CypherQuery,
		Action:      rule.Action,
		IsActive:    rule.IsActive,
	}, nil
}
