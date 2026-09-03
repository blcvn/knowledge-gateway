package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/domain/repository"
	"openviking.com/ov-resource/internal/usecase/dto"
	"openviking.com/ov-resource/internal/usecase/port"
)

type watchUseCase struct {
	watchRepo      repository.WatchRepository
	ingestUseCase  port.IngestUseCase
	maxTasks       int
	defaultPollMs  int64
}

func NewWatchUseCase(
	watchRepo repository.WatchRepository,
	ingestUseCase port.IngestUseCase,
	maxTasks int,
	defaultPollMs int64,
) *watchUseCase {
	return &watchUseCase{
		watchRepo:     watchRepo,
		ingestUseCase: ingestUseCase,
		maxTasks:      maxTasks,
		defaultPollMs: defaultPollMs,
	}
}

func (u *watchUseCase) Execute(ctx context.Context, req dto.WatchRequest) (<-chan model.WatchEvent, error) {
	pollInterval := req.PollIntervalMs
	if pollInterval <= 0 {
		pollInterval = u.defaultPollMs
	}

	task := &model.WatchTask{
		ID:             uuid.New().String(),
		AccountID:      req.AccountID,
		SourcePath:     req.SourcePath,
		TargetPath:     req.TargetPath,
		Patterns:       req.Patterns,
		PollIntervalMs: pollInterval,
		Status:         model.WatchStatusActive,
	}
	
	err := u.watchRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	ch := make(chan model.WatchEvent, 10)
	
	go func() {
		ch <- model.WatchEvent{
			Type:      model.EventTypeCreated,
			Path:      req.SourcePath,
			Timestamp: time.Now(),
		}
		close(ch)
	}()

	return ch, nil
}

func (u *watchUseCase) StartManager(ctx context.Context) error {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				u.pollDirectories(ctx)
			}
		}
	}()
	return nil
}

func (u *watchUseCase) pollDirectories(ctx context.Context) {
	tasks, err := u.watchRepo.GetActiveTasks(ctx)
	if err != nil {
		return
	}

	count := 0
	for _, task := range tasks {
		if count >= u.maxTasks {
			break
		}
		_ = u.watchRepo.UpdateLastPoll(ctx, task.ID)
		count++
	}
}
