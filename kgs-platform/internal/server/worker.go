package server

import (
	"context"
	"strings"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/outbox"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/overlay"

	"github.com/go-co-op/gocron/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// WorkerServer implements transport.Server for background workers in Kratos
type WorkerServer struct {
	scheduler       *biz.RuleRunner
	events          *biz.EventRunner
	policySync      *biz.PolicySyncRunner
	overlayListener *overlay.SessionCloseListener
	outboxWorker    *outbox.OutboxWorker
	reconcileJob    *outbox.ReconcileJob
	reconcileSched  gocron.Scheduler
	reconcileCron   string
	outboxCancel    context.CancelFunc
	log             *log.Helper
}

func NewWorkerServer(
	cfg *conf.Data,
	scheduler *biz.RuleRunner,
	events *biz.EventRunner,
	policySync *biz.PolicySyncRunner,
	overlayListener *overlay.SessionCloseListener,
	outboxWorker *outbox.OutboxWorker,
	reconcileJob *outbox.ReconcileJob,
	logger log.Logger,
) *WorkerServer {
	reconcileCron := "0 3 * * *"
	if cfg != nil && cfg.GetReconcile() != nil {
		if cron := strings.TrimSpace(cfg.GetReconcile().GetCron()); cron != "" {
			reconcileCron = cron
		}
	}
	return &WorkerServer{
		scheduler:       scheduler,
		events:          events,
		policySync:      policySync,
		overlayListener: overlayListener,
		outboxWorker:    outboxWorker,
		reconcileJob:    reconcileJob,
		reconcileCron:   reconcileCron,
		log:             log.NewHelper(logger),
	}
}

// Start runs the background worker
func (s *WorkerServer) Start(ctx context.Context) error {
	s.log.Info("[WorkerServer] starting...")
	if err := s.scheduler.Start(ctx); err != nil {
		return err
	}
	if err := s.events.Start(ctx); err != nil {
		return err
	}
	if err := s.policySync.Start(ctx); err != nil {
		return err
	}
	if s.overlayListener != nil {
		if err := s.overlayListener.Start(ctx); err != nil {
			return err
		}
	}
	if s.outboxWorker != nil {
		workerCtx, cancel := context.WithCancel(ctx)
		s.outboxCancel = cancel
		go s.outboxWorker.Run(workerCtx)
	}
	if s.reconcileJob != nil {
		scheduler, err := gocron.NewScheduler()
		if err != nil {
			return err
		}
		if _, err := scheduler.NewJob(
			gocron.CronJob(s.reconcileCron, false),
			gocron.NewTask(func() {
				if err := s.reconcileJob.Run(context.Background()); err != nil {
					s.log.Warnf("reconcile job failed: %v", err)
				}
			}),
		); err != nil {
			return err
		}
		s.reconcileSched = scheduler
		scheduler.Start()
	}
	return nil
}

// Stop shuts down the background worker
func (s *WorkerServer) Stop(ctx context.Context) error {
	s.log.Info("[WorkerServer] stopping...")
	if err := s.policySync.Stop(ctx); err != nil {
		s.log.Errorf("failed to stop policy sync runner: %v", err)
	}
	if s.outboxCancel != nil {
		s.outboxCancel()
	}
	if s.reconcileSched != nil {
		if err := s.reconcileSched.Shutdown(); err != nil {
			s.log.Errorf("failed to shutdown reconcile scheduler: %v", err)
		}
	}
	if s.overlayListener != nil {
		if err := s.overlayListener.Stop(ctx); err != nil {
			s.log.Errorf("failed to stop overlay listener: %v", err)
		}
	}
	if err := s.events.Stop(ctx); err != nil {
		s.log.Errorf("failed to stop event runner: %v", err)
	}
	if err := s.scheduler.Stop(ctx); err != nil {
		s.log.Errorf("failed to stop rule scheduler: %v", err)
	}
	return nil
}

// Ensure WorkerServer implements transport.Server
var _ transport.Server = (*WorkerServer)(nil)
