package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/filter"
	"github.com/downballot/downballot/internal/schema"
	"gorm.io/gorm"
)

func getGroupsForUser(db *gorm.DB, userID any, organizationID any, filters map[string]any) ([]*schema.Group, error) {
	var groups []*schema.Group
	query := db.Session(&gorm.Session{}).
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
		Order("id").
		Find(&groups).
		Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

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

	userGroups := []*schema.Group{}
	{
		var groups []*schema.Group
		err := db.Session(&gorm.Session{}).
			Where("organization_id = ?", organizationID).
			Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", userID).
			Order("id").
			Find(&groups).
			Error
		if err != nil {
			return nil, err
		}
		userGroups = append(userGroups, groups...)
	}

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

func filterPersons(ctx context.Context, db *gorm.DB, userID uint64, organizationID uint64, groupID *uint64, filterString *string, returnFields []string, limit int) ([]*downballotapi.Person, error) {
	hierarchies, err := getGroupHierarchiesForUser(db, userID, organizationID)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(hierarchies)))

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

	var joinCount uint
	var f func(clause filter.Clause, groupQuery *gorm.DB)
	f = func(clause filter.Clause, groupQuery *gorm.DB) {
		slog.DebugContext(ctx, fmt.Sprintf("f: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("f: condition: %+v", typedClause))

			joinCount++
			query = query.Joins("LEFT OUTER JOIN person_field AS person_field_join" + fmt.Sprintf("%d", joinCount) + " ON person.id = person_field_join" + fmt.Sprintf("%d", joinCount) + ".person_id")
			groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".name = ?", typedClause.Name)
			switch typedClause.Operation {
			case filter.OperationEquals:
				groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".value = ?", typedClause.Value)
			case filter.OperationWildcard:
				groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".value LIKE ?", strings.ReplaceAll(typedClause.Value, "*", "%"))
			default:
				slog.WarnContext(ctx, fmt.Sprintf("Unknown operation: %s", typedClause.Operation))
			}
		case *filter.ClauseGroup:
			slog.DebugContext(ctx, fmt.Sprintf("f: group: %+v", typedClause))

			for _, groupClause := range typedClause.Clauses {
				switch typedClause.Operation {
				case filter.ClauseGroupOperationAnd:
					newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
					f(groupClause, newQuery)
					groupQuery.Where(newQuery)
				case filter.ClauseGroupOperationOr:
					newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
					f(groupClause, newQuery)
					groupQuery.Or(newQuery)
				}
			}
		default:
			slog.WarnContext(ctx, fmt.Sprintf("Unknown clause type: %T", typedClause))
		}
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
			f(groupClause, newQuery)
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

	personFieldsMap := map[uint64][]*schema.PersonField{}
	{
		var fields []*schema.PersonField
		query := db.Session(&gorm.Session{}).
			Where("person_id IN (?)", personIDs)
		if len(returnFields) > 0 {
			query = query.Where("name IN (?)", returnFields)
		}
		err := query.
			Find(&fields).
			Error
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if personFieldsMap[field.PersonID] == nil {
				personFieldsMap[field.PersonID] = []*schema.PersonField{}
			}
			personFieldsMap[field.PersonID] = append(personFieldsMap[field.PersonID], field)
		}
	}

	for _, person := range persons {
		o := &downballotapi.Person{
			ID:      fmt.Sprintf("%d", person.ID),
			VoterID: person.VoterID,
			Fields:  map[string]string{},
		}

		fields := personFieldsMap[person.ID]
		for _, field := range fields {
			o.Fields[field.Name] = field.Value
		}

		output = append(output, o)
	}

	return output, nil
}
