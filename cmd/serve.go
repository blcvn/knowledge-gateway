package cmd

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"kg-service/internal/bootstrap"
	"kg-service/internal/config"
)

var serveCmd = &Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start kg-service with the configured HTTP, Postgres, Redis, and bootstrap settings.",
	run: func(args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		app, err := bootstrap.New(cfg)
		if err != nil {
			return err
		}

		server := &http.Server{
			Addr:              cfg.HTTP.Address(),
			Handler:           app.Handler(),
			ReadHeaderTimeout: 5 * time.Second,
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go func() {
			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("shutdown server: %v", err)
			}
		}()

		log.Printf("kg-service listening on %s", cfg.HTTP.Address())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	RootCmd.run = serveCmd.run
}
