package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

type PolicySyncRunner struct {
	repo   PolicyRepo
	opa    *OPAClient
	log    *log.Helper
	ticker *time.Ticker
	quit   chan struct{}
}

func NewPolicySyncRunner(repo PolicyRepo, opa *OPAClient, logger log.Logger) *PolicySyncRunner {
	return &PolicySyncRunner{
		repo: repo,
		opa:  opa,
		log:  log.NewHelper(logger),
		quit: make(chan struct{}),
	}
}

func (r *PolicySyncRunner) Start(ctx context.Context) error {
	r.log.Info("Starting Policy Sync Runner...")

	// Sync every 30 seconds
	r.ticker = time.NewTicker(30 * time.Second)

	go func() {
		// Run once immediately on start
		r.syncPolicies()

		for {
			select {
			case <-r.ticker.C:
				r.syncPolicies()
			case <-r.quit:
				r.ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (r *PolicySyncRunner) Stop(ctx context.Context) error {
	r.log.Info("Stopping Policy Sync Runner...")
	close(r.quit)
	return nil
}

func (r *PolicySyncRunner) syncPolicies() {
	r.log.Debug("Running policy sync job...")
	ctx := context.Background()

	// In a real system, you'd iterate over apps/namespaces. Mocking for demo-app.
	appID := "demo-app"
	policies, err := r.repo.ListPolicies(ctx, appID)
	if err != nil {
		r.log.Errorf("Failed to retrieve policies for sync: %v", err)
		return
	}

	for _, p := range policies {
		if !p.IsActive {
			continue // skip inactive policies
		}

		policyID := fmt.Sprintf("policy_%d", p.ID)
		err := r.opa.PutPolicy(ctx, policyID, p.RegoContent)
		if err != nil {
			r.log.Errorf("Failed to sync policy %s (ID %d) to OPA: %v", p.Name, p.ID, err)
		} else {
			r.log.Debugf("Successfully synced policy %s to OPA", p.Name)
		}
	}
}
