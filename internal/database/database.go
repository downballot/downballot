package database

import (
	"context"
	"fmt"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// New creates a new database connection.
func New(ctx context.Context, driverName string, connectionString string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	switch driverName {
	case "sqlite3":
		db, err = gorm.Open(sqlite.Open(connectionString), &gorm.Config{})
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid database driver: %s", driverName)
	}

	return db, nil
}
