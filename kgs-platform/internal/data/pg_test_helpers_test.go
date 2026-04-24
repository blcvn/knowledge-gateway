package data

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newKGTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if err := db.AutoMigrate(&KGEntity{}, &KGEdge{}, &KGSyncOutbox{}); err != nil {
		t.Fatalf("auto migrate kg models: %v", err)
	}
	return db
}
