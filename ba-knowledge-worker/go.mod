module github.com/blcvn/backend/services/ba-knowledge-worker

go 1.24.0

require (
	github.com/blcvn/backend/services/pkg v0.0.0
	github.com/hibiken/asynq v0.24.1
)

replace github.com/blcvn/backend/services/pkg => ../pkg
