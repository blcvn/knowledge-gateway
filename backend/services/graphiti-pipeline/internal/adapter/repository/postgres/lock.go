package postgres

import (
	"context"
	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type GroupLock struct {}

func NewGroupLock() port.GroupLock {
	return &GroupLock{}
}

func (l *GroupLock) Acquire(ctx context.Context, groupID ingestion.GroupID) (func(), error) {
	return func() {}, nil
}
