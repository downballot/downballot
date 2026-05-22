package api

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/filter"
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

func filterPersons(ctx context.Context, db *gorm.DB, userID uint64, organizationID uint64, groupID *uint64, filterString *string, returnFields *[]string, limit int) ([]*downballotapi.Person, error) {
	hierarchies, err := getGroupHierarchiesForUser(db, userID, organizationID)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(hierarchies)))

	fieldDefinitionByIDMap := map[uint64]*schema.PersonFieldDefinition{}
	fieldDefinitionByNameMap := map[string]*schema.PersonFieldDefinition{}
	{
		var fieldDefinitions []*schema.PersonFieldDefinition
		err = db.Session(&gorm.Session{}).
			Where("organization_id = ?", organizationID).
			Find(&fieldDefinitions).
			Error
		if err != nil {
			return nil, fmt.Errorf("could not find field definitions: %w", err)
		}
		for _, fieldDefinition := range fieldDefinitions {
			fieldDefinitionByIDMap[fieldDefinition.ID] = fieldDefinition
			fieldDefinitionByNameMap[fieldDefinition.Name] = fieldDefinition
		}
	}

	if groupID == nil {
		hierarchies = condenseHierarchies(hierarchies)
		slog.InfoContext(ctx, fmt.Sprintf("Consensed hierarchies: (%d)", len(hierarchies)))
	} else {
		var groupHierarchy []*schema.Group
		for _, hierarchy := range hierarchies {
			if len(hierarchy) > 0 && hierarchy[len(hierarchy)-1].ID == *groupID {
				groupHierarchy = hierarchy
				break
			}
		}

		if len(groupHierarchy) == 0 {
			return nil, fmt.Errorf("could not find hierarchy for group.id=%d", *groupID)
		}

		hierarchies = [][]*schema.Group{groupHierarchy}
		slog.InfoContext(ctx, fmt.Sprintf("Group-limited hierarchies: (%d)", len(hierarchies)))
	}

	query := db.Session(&gorm.Session{})

	fieldTableMap := map[string]string{} // This maps a field name to the table that represents it.
	var f func(clause filter.Clause, groupQuery *gorm.DB) error
	f = func(clause filter.Clause, groupQuery *gorm.DB) error {
		slog.DebugContext(ctx, fmt.Sprintf("f: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("f: condition: %+v", typedClause))

			personFieldDefinition := fieldDefinitionByNameMap[typedClause.Name]
			if personFieldDefinition == nil {
				return fmt.Errorf("unknown field: %s", typedClause.Name)
			}

			var fieldTableName string
			if fieldTableMap[typedClause.Name] != "" {
				fieldTableName = fieldTableMap[typedClause.Name]
			} else {
				fieldTableName = "person_field_join" + fmt.Sprintf("%d", len(fieldTableMap)+1)
				fieldTableMap[typedClause.Name] = fieldTableName

				query = query.Joins("LEFT OUTER JOIN person_field AS "+fieldTableName+" ON person.id = "+fieldTableName+".person_id AND "+fieldTableName+".person_field_definition_id = ?", personFieldDefinition.ID)
			}
			switch typedClause.Operation {
			case filter.OperationEquals:
				groupQuery = groupQuery.Where(fieldTableName+".value = ?", typedClause.Value)
			case filter.OperationNotEquals:
				groupQuery = groupQuery.Where(fieldTableName+".value IS NULL OR "+fieldTableName+".value != ?", typedClause.Value)
			case filter.OperationIs:
				if typedClause.Value == "null" {
					groupQuery = groupQuery.Where(fieldTableName + ".value IS NULL")
				} else {
					return fmt.Errorf("invalid value for is operation: %s", typedClause.Value)
				}
			case filter.OperationIsNot:
				if typedClause.Value == "null" {
					groupQuery = groupQuery.Where(fieldTableName + ".value IS NOT NULL")
				} else {
					return fmt.Errorf("invalid value for is not operation: %s", typedClause.Value)
				}
			case filter.OperationGreaterThan:
				switch personFieldDefinition.Type {
				case "integer":
					groupQuery = groupQuery.Where("CAST("+fieldTableName+".value AS INTEGER) > ?", typedClause.Value)
				default:
					groupQuery = groupQuery.Where(fieldTableName+".value > ?", typedClause.Value)
				}
			case filter.OperationGreaterThanOrEqual:
				switch personFieldDefinition.Type {
				case "integer":
					groupQuery = groupQuery.Where("CAST("+fieldTableName+".value AS INTEGER) >= ?", typedClause.Value)
				default:
					groupQuery = groupQuery.Where(fieldTableName+".value >= ?", typedClause.Value)
				}
			case filter.OperationLessThan:
				switch personFieldDefinition.Type {
				case "integer":
					groupQuery = groupQuery.Where("CAST("+fieldTableName+".value AS INTEGER) < ?", typedClause.Value)
				default:
					groupQuery = groupQuery.Where(fieldTableName+".value < ?", typedClause.Value)
				}
			case filter.OperationLessThanOrEqual:
				switch personFieldDefinition.Type {
				case "integer":
					groupQuery = groupQuery.Where("CAST("+fieldTableName+".value AS INTEGER) <= ?", typedClause.Value)
				default:
					groupQuery = groupQuery.Where(fieldTableName+".value <= ?", typedClause.Value)
				}
			case filter.OperationWildcard:
				groupQuery = groupQuery.Where(fieldTableName+".value LIKE ?", strings.ReplaceAll(typedClause.Value, "*", "%"))
			default:
				return fmt.Errorf("unknown operation: %s", typedClause.Operation)
			}
		case *filter.ClauseGroup:
			slog.DebugContext(ctx, fmt.Sprintf("f: group: %+v", typedClause))

			for _, groupClause := range typedClause.Clauses {
				switch typedClause.Operation {
				case filter.ClauseGroupOperationAnd:
					newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
					err = f(groupClause, newQuery)
					if err != nil {
						return err
					}
					groupQuery.Where(newQuery)
				case filter.ClauseGroupOperationOr:
					newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
					err = f(groupClause, newQuery)
					if err != nil {
						return err
					}
					groupQuery.Or(newQuery)
				}
			}
		default:
			return fmt.Errorf("unknown clause type: %T", typedClause)
		}
		return nil
	}

	{
		var hierarchyStrings []string
		for _, groupHierarchy := range hierarchies {
			// This shouldn't be possible, but skip any broken hierarchies.
			if len(groupHierarchy) == 0 {
				continue
			}

			var groupStrings []string
			for _, group := range groupHierarchy {
				//slog.DebugContext(ctx, fmt.Sprintf("Group: id=%d, name=%s", group.ID, group.Name))
				if group.Filter != "" {
					groupStrings = append(groupStrings, group.Filter)
				}
			}
			if filterString != nil && *filterString != "" {
				groupStrings = append(groupStrings, *filterString)
			}

			if len(groupStrings) > 0 {
				hierarchyString := "((" + strings.Join(groupStrings, ") AND (") + "))"
				tailGroup := groupHierarchy[len(groupHierarchy)-1]
				slog.DebugContext(ctx, fmt.Sprintf("Group: id=%d, name=%s, hierarchyString: %s", tailGroup.ID, tailGroup.Name, hierarchyString))
				hierarchyStrings = append(hierarchyStrings, hierarchyString)
			}
		}

		if len(hierarchyStrings) > 0 {
			finalString := "((" + strings.Join(hierarchyStrings, ") OR (") + "))"
			slog.DebugContext(ctx, fmt.Sprintf("Final string: %s", finalString))

			groupClause, err := filter.Parse(ctx, finalString)
			if err != nil {
				return nil, err
			}
			newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
			err = f(groupClause, newQuery)
			if err != nil {
				return nil, err
			}
			query = query.Where(newQuery)
		}
	}

	var persons []*schema.Person
	err = query.
		Distinct().
		Where("organization_id IN (SELECT id FROM organization WHERE id = ?)", organizationID).
		Limit(limit).
		Find(&persons).
		Error
	if err != nil {
		return nil, err
	}

	var personIDs []uint64
	for _, person := range persons {
		personIDs = append(personIDs, person.ID)
	}

	output := make([]*downballotapi.Person, 0, len(persons))

	personFieldsMap := map[uint64]map[string]string{}
	{
		var fields []*schema.PersonField
		query := db.Session(&gorm.Session{}).
			Where("person_id IN (?)", personIDs)
		if returnFields != nil {
			fieldDefinitionIDs := []uint64{}
			for _, fieldName := range *returnFields {
				fieldDefinition := fieldDefinitionByNameMap[fieldName]
				if fieldDefinition == nil {
					return nil, fmt.Errorf("unknown field: %s", fieldName)
				}
				fieldDefinitionIDs = append(fieldDefinitionIDs, fieldDefinition.ID)
			}
			query = query.Where("person_field_definition_id IN (?)", fieldDefinitionIDs)
		}
		err := query.
			Find(&fields).
			Error
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if personFieldsMap[field.PersonID] == nil {
				personFieldsMap[field.PersonID] = map[string]string{}
			}
			personFieldDefinition := fieldDefinitionByIDMap[field.PersonFieldDefinitionID]
			if personFieldDefinition == nil {
				return nil, fmt.Errorf("unknown field definition: %d", field.PersonFieldDefinitionID)
			}
			personFieldsMap[field.PersonID][personFieldDefinition.Name] = field.Value
		}
	}

	for _, person := range persons {
		o := &downballotapi.Person{
			ID:      fmt.Sprintf("%d", person.ID),
			VoterID: person.VoterID,
			Fields:  map[string]string{},
		}

		fields := personFieldsMap[person.ID]
		for name, value := range fields {
			o.Fields[name] = value
		}

		output = append(output, o)
	}

	return output, nil
}
