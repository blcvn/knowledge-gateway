package infrastructure

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/domain"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func ConnectPostgres(cfg Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	log.Println("Running AutoMigrate for Knowledge Service...")
	return db.AutoMigrate(
		&domain.Document{},
		&domain.DocumentLineage{},
		&domain.ExternalSource{},
	)
}
