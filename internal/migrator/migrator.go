package migrator

import (
	"fmt"

	"github.com/downballot/downballot/internal/schema"
	"gorm.io/gorm"
)

// Migrate the database schema.
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		schema.Organization{},
		schema.Group{},
		schema.User{},
		schema.UserGroupMap{},
		schema.UserOrganizationMap{},
		schema.Person{},
		schema.PersonField{},
		schema.PersonFieldDefinition{},
	)
	if err != nil {
		return fmt.Errorf("could not auto-migrate database: %w", err)
	}

	return nil
}
