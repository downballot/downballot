package databasetest

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/internal/database"
	"github.com/downballot/downballot/internal/migrator"
	"gorm.io/gorm"
)

func New(ctx context.Context) (*gorm.DB, error) {
	db, err := database.New(ctx, "sqlite3", "file::memory:?cache=shared&parseTime=true")
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	err = migrator.Migrate(db)
	if err != nil {
		return nil, fmt.Errorf("could not migrate database: %w", err)
	}

	return db, nil
}
