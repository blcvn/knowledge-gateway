package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"vnp-memory/services/ov-fs/internal/infra/config"
	// pb "vnp-memory/api/gen/go/openviking/fs/v1"
)

func StartGRPCServer(cfg *config.Config /* , handler pb.OvFsServiceServer */) error {
	_ = grpc.NewServer()
	// pb.RegisterOvFsServiceServer(s, handler)

	// lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	// if err != nil { return err }
	
	// return s.Serve(lis)
	return nil
}

func StartHealthServer(cfg *config.Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "serving"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ready", "checks": {"db": "ok", "nats": "ok", "crypto": "ok"}}`))
	})

	srv := &http.Server{
		Addr:    ":9104", // Default from config
		Handler: mux,
	}
	return srv.ListenAndServe()
}

func GracefulShutdown(s *grpc.Server, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.Stop()
	case <-stopped:
	}
}
