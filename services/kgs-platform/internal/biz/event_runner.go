package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
)

// EventRunner listens for real-time events on Redis Streams
type EventRunner struct {
	rulesRepo RulesRepo
	graphRepo GraphRepo
	redisCli  *redis.Client
	log       *log.Helper
	stream    string
	group     string
}

func NewEventRunner(rulesRepo RulesRepo, graphRepo GraphRepo, redisCli *redis.Client, logger log.Logger) *EventRunner {
	return &EventRunner{
		rulesRepo: rulesRepo,
		graphRepo: graphRepo,
		redisCli:  redisCli,
		log:       log.NewHelper(logger),
		stream:    "kgs:events:nodes",
		group:     "kgs-worker-group",
	}
}

func (r *EventRunner) Start(ctx context.Context) error {
	r.log.Info("Starting Event Runner Component...")

	err := r.redisCli.XGroupCreateMkStream(ctx, r.stream, r.group, "$").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		r.log.Errorf("Failed to create consumer group: %v", err)
		return err
	}

	go r.consumeEvents()
	return nil
}

func (r *EventRunner) Stop(ctx context.Context) error {
	r.log.Info("Stopping Event Runner Component...")
	return nil
}

func (r *EventRunner) consumeEvents() {
	ctx := context.Background()

	for {
		res, err := r.redisCli.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    r.group,
			Consumer: "worker-1",
			Streams:  []string{r.stream, ">"},
			Count:    10,
			Block:    0,
			NoAck:    false,
		}).Result()

		if err != nil {
			r.log.Errorf("Error reading stream: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		for _, stream := range res {
			for _, message := range stream.Messages {
				r.processMessage(ctx, message)
				r.redisCli.XAck(ctx, r.stream, r.group, message.ID)
			}
		}
	}
}

func (r *EventRunner) processMessage(ctx context.Context, message redis.XMessage) {
	r.log.Infof("Processing event: %v", message.Values)

	appID, _ := message.Values["app_id"].(string)
	if appID == "" {
		appID = "demo-app" // fallback
	}

	rules, err := r.rulesRepo.ListRules(ctx, appID)
	if err != nil {
		r.log.Errorf("Failed to list rules for event runner: %v", err)
		return
	}

	for _, rule := range rules {
		if rule.IsActive && rule.TriggerType == "ON_WRITE" {
			r.log.Infof("Executing ON_WRITE rule: %s...", rule.Name)
			// Pass the event payload directly into the Cypher script parameters
			params := map[string]any{
				"app_id": appID,
				"event":  message.Values,
			}
			result, err := r.graphRepo.ExecuteQuery(ctx, rule.CypherQuery, params)
			if err != nil {
				r.log.Errorf("ON_WRITE rule %s execution failed: %v", rule.Name, err)
				continue
			}

			// Action Execution (e.g. webhook) based on rule.Action
			resultJSON, _ := json.Marshal(result)
			r.log.Infof("ON_WRITE rule %s execution succeeded. Action payload triggering... Result: %s", rule.Name, resultJSON)
			fmt.Printf("Trigging action [%s] for rule %s.\n", rule.Action, rule.Name)
		}
	}
}
