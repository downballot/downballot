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

	config := &gorm.Config{}

	switch driverName {
	case "sqlite3":
		db, err = gorm.Open(sqlite.Open(connectionString), config)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid database driver: %s", driverName)
	}

	db = db.Debug()

	return db, nil
}
