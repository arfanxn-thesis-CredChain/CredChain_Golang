package db

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"CredChain_Golang/infrastructure/database/gorm/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenInMemorySQLite creates a fresh in-memory SQLite DB with all GORM models
// auto-migrated. Each call gets a unique shared-cache name so tests are
// parallel-safe. The DB is closed via t.Cleanup.
func OpenInMemorySQLite(t *testing.T) *gorm.DB {
	t.Helper()

	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	dsn := fmt.Sprintf("file:memdb_%s?mode=memory&cache=shared", hex.EncodeToString(suffix))

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	if err := db.AutoMigrate(
		&model.UserUnit{},
		&model.CredentialType{},
		&model.CredentialIssuerOrganization{},
		&model.Competency{},
		&model.CompetencyCredential{},
		&model.User{},
		&model.UserToken{},
		&model.Credential{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err != nil {
			return
		}
		_ = sqlDB.Close()
	})

	return db
}
