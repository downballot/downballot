package api

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/downballot/downballot/internal/schema"
	"gorm.io/gorm"
)

// getGroupsForUser returns the list of groups that the user can see.
func getGroupsForUser(db *gorm.DB, userID any, organizationID any) ([]*schema.Group, error) {
	var mappedGroupIDs []uint64
	err := db.Session(&gorm.Session{}).
		Model(&schema.Group{}).
		Where("organization_id = ?", organizationID).
		Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", userID).
		Pluck("id", &mappedGroupIDs).
		Error
	if err != nil {
		return nil, err
	}

	/*
		slog.Info("mappedGroupIDs", "mappedGroupIDs", len(mappedGroupIDs))
		for _, mappedGroupID := range mappedGroupIDs {
			slog.Info("mappedGroupID", "mappedGroupID", mappedGroupID)
		}
			//*/

	var groups []*schema.Group
	err = db.Session(&gorm.Session{}).
		Where("organization_id = ?", organizationID).
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}

	groupIDToParentIDMap := map[uint64]uint64{}
	for _, group := range groups {
		if group.ParentID != nil {
			groupIDToParentIDMap[group.ID] = *group.ParentID
		}
	}

	userGroupIDMap := map[uint64]bool{}
	for _, groupID := range mappedGroupIDs {
		userGroupIDMap[groupID] = true
	}

	var userGroups []*schema.Group
	for _, group := range groups {
		groupIsUserGroup := false
		currentGroupID := group.ID
		for currentGroupID != 0 {
			if userGroupIDMap[currentGroupID] {
				groupIsUserGroup = true
				break
			}

			currentGroupID = groupIDToParentIDMap[currentGroupID]
		}

		if !groupIsUserGroup {
			continue
		}

		userGroups = append(userGroups, group)
	}

	slices.SortFunc(userGroups, func(a, b *schema.Group) int {
		diff := cmp.Compare(a.Name, b.Name)
		if diff != 0 {
			return diff
		}
		return cmp.Compare(a.ID, b.ID)
	})

	return userGroups, nil
}

// getGroupHierarchiesForUser returns the hierarchies of groups for a user.
//
// One hierarchy will be returned for each bottom-level group that the can see.
//
// Each hierarchy goes from the organization's root group to the bottom-level group, and thus
// includes all of the information needed to properly build the filter for the group.
func getGroupHierarchiesForUser(db *gorm.DB, userID any, organizationID any) ([][]*schema.Group, error) {
	groupChildrenMap := map[uint64][]*schema.Group{}
	groupsByID := map[uint64]*schema.Group{}
	{
		var groups []*schema.Group
		err := db.Session(&gorm.Session{}).
			Where("organization_id = ?", organizationID).
			Order("id").
			Find(&groups).
			Error
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			groupsByID[group.ID] = group

			if group.ParentID != nil {
				groupChildrenMap[*group.ParentID] = append(groupChildrenMap[*group.ParentID], group)
			}
		}
	}

	userGroups, err := getGroupsForUser(db, userID, organizationID)
	if err != nil {
		return nil, err
	}

	//*
	for _, userGroup := range userGroups {
		slog.Info("userGroup", "userGroup", userGroup.Name, "id", userGroup.ID)
	}
	//*/

	hierarchies := [][]*schema.Group{}
	for _, bottomLevelGroup := range userGroups {
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

// condenseHierarchies prunes any hierarchies that are covered by ones higher up the chain.
//
// Since these hierarchies are used for permissions, we need to keep the most encompassing ones,
// not the derivative ones.
//
// For example, if the hierarchies include the root group, then all of other ones will be pruned.
// Similar, if a parent and child are in the list, then the child will be pruned.
func condenseHierarchies(hierarchies [][]*schema.Group) [][]*schema.Group {
	var newHierarchies [][]*schema.Group
	{
		pathToIndexMap := map[string]int{}
		indexToKeepMap := map[int]bool{}
		indexToPathMap := map[int]string{}
		for i, hierarchy := range hierarchies {
			indexToKeepMap[i] = true

			var pathParts []string
			for _, group := range hierarchy {
				pathParts = append(pathParts, fmt.Sprintf("%d", group.ID))
			}
			path := "/" + strings.Join(pathParts, "/") + "/"
			pathToIndexMap[path] = i
			indexToPathMap[i] = path
		}
		for i := range hierarchies {
			if !indexToKeepMap[i] {
				continue
			}

			iPath := indexToPathMap[i]
			for j := range hierarchies {
				if i == j {
					continue
				}
				jPath := indexToPathMap[j]
				if strings.HasPrefix(jPath, iPath) {
					indexToKeepMap[j] = false
				}
			}
		}
		for i, keep := range indexToKeepMap {
			if keep {
				newHierarchies = append(newHierarchies, hierarchies[i])
			}
		}
	}
	return newHierarchies
}
