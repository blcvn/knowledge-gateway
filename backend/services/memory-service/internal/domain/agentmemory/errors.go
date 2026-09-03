package agentmemory

import "errors"

var (
    ErrMemoryNotFound  = errors.New("memory not found")
    ErrSlotReadOnly    = errors.New("memory slot is read-only")
    ErrSlotSizeExceeded = errors.New("content exceeds slot size limit")
    ErrInvalidType     = errors.New("invalid memory type")
)
