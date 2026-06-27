package db

import (
	"strings"
	"testing"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitDB(t *testing.T) {
	t.Setenv("DB_PATH", ":memory:")
	conn, err := InitDB()
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	if conn == nil {
		t.Fatal("expected db connection")
	}
}

func TestResolveDBPathDefault(t *testing.T) {
	t.Setenv("DB_PATH", "")
	if resolveDBPath() != "ghostwriter.db" {
		t.Fatal("expected default db path")
	}
	t.Setenv("DB_PATH", "custom.db")
	if resolveDBPath() != "custom.db" {
		t.Fatal("expected custom db path")
	}
}

func TestInitDBDefaultPath(t *testing.T) {
	t.Setenv("DB_PATH", "")
	conn, err := InitDB()
	if err != nil {
		t.Fatalf("InitDB with default path failed: %v", err)
	}
	if conn == nil {
		t.Fatal("expected db connection")
	}
	t.Cleanup(func() {
		_ = conn.Exec("PRAGMA journal_mode=DELETE").Error
	})
}

func TestFinishInitDBMigrateError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
	if _, err := finishInitDB(db); err == nil || !strings.Contains(err.Error(), "auto-migration failed") {
		t.Fatalf("expected wrapped migrate error, got %v", err)
	}
}

func TestInitDBAtPathOpenFailure(t *testing.T) {
	if _, err := initDBAtPath("/root/forbidden/ghostwriter.db"); err == nil {
		t.Skip("open succeeded in this environment")
	} else if !strings.Contains(err.Error(), "failed to connect") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitDBAtPathMigrateFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
	if err := migrateAll(db); err == nil {
		t.Fatal("expected migrateAll error on closed db")
	}
	if _, err := initDBAtPath(":memory:"); err != nil {
		t.Fatalf("fresh memory init should succeed: %v", err)
	}
}

func TestMigrateAll(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateAll(db); err != nil {
		t.Fatalf("migrateAll failed: %v", err)
	}
}

func TestMigrateAllError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.Close()
	if err := migrateAll(db); err == nil {
		t.Fatal("expected migrateAll error on closed db")
	}
}

func TestInitDBMigrateAllModels(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	modelsToMigrate := []interface{}{
		&models.CreatorProfile{},
		&models.Prompt{},
		&models.Script{},
		&models.PersonaProfile{},
		&models.ScriptFeedback{},
	}
	if err := db.AutoMigrate(modelsToMigrate...); err != nil {
		t.Fatalf("migrate all models failed: %v", err)
	}
}
