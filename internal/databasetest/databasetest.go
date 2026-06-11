package databasetest

import (
	"context"
	"fmt"
	"strings"

	"github.com/downballot/downballot/internal/database"
	"github.com/downballot/downballot/internal/migrator"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"gorm.io/gorm"
)

func New(ctx context.Context) (*gorm.DB, error) {
	db, err := database.New(ctx, "sqlite3", "file::memory:?cache=shared&parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	// Set up an encryption key for the database.
	if sqltype.GetEncryptionKey() == "" {
		encryptionKey := strings.Repeat("00", 32)
		sqltype.SetEncryptionKey(encryptionKey)
	}

	err = migrator.Migrate(db)
	if err != nil {
		return nil, fmt.Errorf("could not migrate database: %w", err)
	}

	return db, nil
}
