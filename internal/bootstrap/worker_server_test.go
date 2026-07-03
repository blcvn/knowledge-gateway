package bootstrap

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/runtimeobs"
	"kg-service/internal/workers"
	"kg-service/internal/write"
)

type testOntologyResolver struct{}

func (testOntologyResolver) GetStatusFieldConfig(string) (*ontology.StatusFieldConfig, error) {
	return nil, nil
}

func TestWorkerServerStartAndStop(t *testing.T) {
	app := &App{
		config: config.Config{},
		logger: runtimeobs.NewLoggerWithWriter(config.Config{}, "bootstrap", &discardWriter{}),
	}

	runtime := workers.NewRuntime(write.NewMemoryStore(), testOntologyResolver{}, (*rediscache.Client)(nil))
	runtime.SetEmbeddingRouter(nil)
	runtime.SetGraphAdapter(nil)
	runtime.SetVectorAdapter(nil)
	runtime.SetFTSAdapter(nil)
	app.runtimeWorker = runtime

	server := newWorkerServer(app)
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start() did not return after cancellation")
	}

	if err := server.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) {
	return len(p), nil
}
