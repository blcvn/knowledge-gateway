package server

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

// WorkerServer implements transport.Server for background workers in Kratos
type WorkerServer struct {
	scheduler  *biz.RuleRunner
	events     *biz.EventRunner
	policySync *biz.PolicySyncRunner
	log        *log.Helper
}

func NewWorkerServer(scheduler *biz.RuleRunner, events *biz.EventRunner, policySync *biz.PolicySyncRunner, logger log.Logger) *WorkerServer {
	return &WorkerServer{
		scheduler:  scheduler,
		events:     events,
		policySync: policySync,
		log:        log.NewHelper(logger),
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
	return nil
}

// Stop shuts down the background worker
func (s *WorkerServer) Stop(ctx context.Context) error {
	s.log.Info("[WorkerServer] stopping...")
	if err := s.policySync.Stop(ctx); err != nil {
		s.log.Errorf("failed to stop policy sync runner: %v", err)
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
