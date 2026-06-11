package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/config"
	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/repository"
	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/server"
	"github.com/blcvn/backend/services/ai-kg-service/mcp-kg-service/internal/service"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	if err := db.AutoMigrate(&repository.RequirementNodeModel{}, &repository.DependencyEdgeModel{}, &repository.DocumentArtifactModel{}); err != nil {
		log.Fatalf("auto migrate mcp-kg-service tables: %v", err)
	}

	repo := repository.New(db)
	svc := service.New(repo)
	srv := server.New(cfg.Port, svc, cfg.ReadTimeout, cfg.WriteTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("mcp-kg-service listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen and serve: %v", err)
	}
}
