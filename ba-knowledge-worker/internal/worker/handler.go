package worker

import (
	"context"
	"encoding/json"
	"log"

	"github.com/blcvn/ba-shared-libs/pkg/queue"
	"github.com/hibiken/asynq"
)

// HandleIndexPRD processes indexing tasks
func HandleIndexPRD(ctx context.Context, t *asynq.Task) error {
	var p queue.IndexPRDPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}
	log.Printf("Indexing PRD: ID=%s URL=%s", p.DocumentID, p.SourceURL)
	// Implement indexing logic here
	return nil
}

// HandleGenOutline processes outline generation tasks
func HandleGenOutline(ctx context.Context, t *asynq.Task) error {
	// Assuming simple payload for now
	log.Printf("Generating Outline for task: %s", t.Type())
	return nil
}
