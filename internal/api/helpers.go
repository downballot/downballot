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

func getGroupsForUser(db *gorm.DB, userID interface{}, organizationID interface{}, filters map[string]interface{}) ([]*schema.Group, error) {
	var groups []*schema.Group
	query := db.Session(&gorm.Session{NewDB: true}).
		Where("organization_id = ?", organizationID).
		Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", userID)
	for key, value := range filters {
		if value == nil {
			query = query.Where(key + " IS NULL")
		} else {
			query = query.Where(key+" = ?", value)
		}
	}
	err := query.
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func getGroupHierarchiesForUser(db *gorm.DB, userID interface{}, organizationID interface{}) ([][]*schema.Group, error) {
	// TODO: Get the list of all groups in the organization.  This will be the master lookup list.
	// TODO: Get the list of groups for this particular user.  Do the logic based on them.

	var groups []*schema.Group
	err := db.Session(&gorm.Session{NewDB: true}).
		Where("organization_id = ?", organizationID).
		Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", userID).
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}

	groupChildrenMap := map[uint64][]*schema.Group{}
	groupsByID := map[uint64]*schema.Group{}
	for _, group := range groups {
		groupsByID[group.ID] = group

		if group.ParentID != nil {
			groupChildrenMap[*group.ParentID] = append(groupChildrenMap[*group.ParentID], group)
		}
	}
	bottomLevelGroups := []*schema.Group{}
	for _, group := range groups {
		if len(groupChildrenMap[group.ID]) == 0 {
			bottomLevelGroups = append(bottomLevelGroups, group)
		}
	}

	hierarchies := [][]*schema.Group{}
	for _, bottomLevelGroup := range bottomLevelGroups {
		hierarchy := []*schema.Group{}

		group := bottomLevelGroup
		for group != nil {
			hierarchy = append([]*schema.Group{group}, hierarchy...)

			if group.ParentID == nil {
				group = nil
			} else {
				group = groupsByID[*group.ParentID]
			}
		}
		hierarchies = append(hierarchies, hierarchy)
	}
	return hierarchies, nil
}

func getUsersForOrganization(db *gorm.DB, organizationID interface{}, filters map[string]interface{}) ([]*schema.User, error) {
	var users []*schema.User
	query := db.Session(&gorm.Session{NewDB: true}).
		Where("id IN (SELECT DISTINCT user_id FROM user_organization_map WHERE organization_id = ?)", organizationID)
	for key, value := range filters {
		if value == nil {
			query = query.Where(key + " IS NULL")
		} else {
			query = query.Where(key+" = ?", value)
		}
	}
	err := query.
		Find(&users).
		Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
