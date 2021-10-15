package api

import (
	"github.com/downballot/downballot/internal/schema"
	"gorm.io/gorm"
)

func getOrganizationForUser(db *gorm.DB, userID interface{}, organizationID interface{}) (*schema.Organization, error) {
	var organizations []*schema.Organization
	err := db.Session(&gorm.Session{NewDB: true}).
		Where("id = ?", organizationID).
		Where("id IN (SELECT organization_id FROM user_organization_map WHERE user_id = ?)", userID).
		Find(&organizations).
		Error
	if err != nil {
		return nil, err
	}
	if len(organizations) == 0 {
		return nil, nil
	}
	return organizations[0], nil
}

func getGroupsForUser(db *gorm.DB, userID interface{}, organizationID interface{}) ([]*schema.Group, error) {
	var groups []*schema.Group
	err := db.Session(&gorm.Session{NewDB: true}).
		Where("organization_id = ?", organizationID).
		Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", userID).
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}
