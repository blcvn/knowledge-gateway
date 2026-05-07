package biz

import (
	"context"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-kratos/kratos/v2/log"
)

type RuleRunner struct {
	rulesRepo RulesRepo
	graphRepo GraphRepo
	scheduler gocron.Scheduler
	log       *log.Helper
}

func NewRuleRunner(rulesRepo RulesRepo, graphRepo GraphRepo, logger log.Logger) *RuleRunner {
	s, err := gocron.NewScheduler()
	if err != nil {
		log.NewHelper(logger).Errorf("failed to create gocron scheduler: %v", err)
	}

	return &RuleRunner{
		rulesRepo: rulesRepo,
		graphRepo: graphRepo,
		scheduler: s,
		log:       log.NewHelper(logger),
	}
}

func (r *RuleRunner) Start(ctx context.Context) error {
	r.log.Info("Starting Rule Runner Component...")

	// Fetch all SCHEDULED rules from database
	// Mock: appID is ignored in a global runner, or we iterate through all apps
	rules, err := r.rulesRepo.ListRules(ctx, "demo-app")
	if err != nil {
		r.log.Errorf("Failed to list rules for runner: %v", err)
		return err
	}

	for _, rule := range rules {
		if rule.IsActive && rule.TriggerType == "SCHEDULED" && rule.Cron != "" {
			_, err := r.scheduler.NewJob(
				gocron.CronJob(rule.Cron, false),
				gocron.NewTask(r.executeRule, rule),
			)
			if err != nil {
				r.log.Errorf("Failed to schedule rule %s (ID: %d): %v", rule.Name, rule.ID, err)
			} else {
				r.log.Infof("Scheduled rule %s (ID: %d) with cron %s", rule.Name, rule.ID, rule.Cron)
			}
		}
	}

	r.scheduler.Start()
	return nil
}

func (r *RuleRunner) Stop(ctx context.Context) error {
	r.log.Info("Stopping Rule Runner Component...")
	if r.scheduler != nil {
		return r.scheduler.Shutdown()
	}
	return nil
}

func (r *RuleRunner) executeRule(rule *Rule) {
	r.log.Infof("Executing scheduled rule: %s...", rule.Name)

	ctx := context.Background()

	// Execute the embedded cypher query against the Graph
	result, err := r.graphRepo.ExecuteQuery(ctx, rule.CypherQuery, map[string]any{"app_id": rule.AppID})

	// In production, we should log to RuleExecution table
	if err != nil {
		r.log.Errorf("Rule %s execution failed: %v", rule.Name, err)
		return
	}

	// Based on rule.Action, trigger webhooks, emails, alerts, etc...
	r.log.Infof("Rule %s execution succeeded. Result mapping: %v", rule.Name, result)
}
