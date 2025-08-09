package db

import (
	"fmt"
	"os"
	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "ghostwriter.db"
	}
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	if err := db.AutoMigrate(&models.CreatorProfile{}); err != nil {
		return nil, fmt.Errorf("auto-migration failed: %w", err)
	}
	if err := db.AutoMigrate(&models.Prompt{}); err != nil {
		return nil, fmt.Errorf("auto-migration failed: %w", err)
	}
	if err := db.AutoMigrate(&models.Script{}); err != nil {
		return nil, fmt.Errorf("auto-migration failed: %w", err)
	}
	return db, nil
}
