package db

import (
	"fmt"
	"os"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func resolveDBPath() string {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return "ghostwriter.db"
	}
	return dbPath
}

func initDBAtPath(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}
	return finishInitDB(db)
}

func finishInitDB(db *gorm.DB) (*gorm.DB, error) {
	if err := migrateAll(db); err != nil {
		return nil, fmt.Errorf("auto-migration failed: %w", err)
	}
	return db, nil
}

func InitDB() (*gorm.DB, error) {
	return initDBAtPath(resolveDBPath())
}

func migrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.CreatorProfile{},
		&models.Prompt{},
		&models.Script{},
		&models.PersonaProfile{},
		&models.ScriptFeedback{},
	)
}
