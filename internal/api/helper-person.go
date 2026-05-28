package api

import (
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

func buildPersonQuery(ctx context.Context, db *gorm.DB, organizationID uint64, groupHierarchies [][]*schema.Group, filterString *string, fieldDefinitionByNameMap map[string]*schema.PersonFieldDefinition) (*gorm.DB, error) {
	query := db.Session(&gorm.Session{}).
		Model(&schema.Person{}).
		Where("organization_id IN (SELECT id FROM organization WHERE id = ?)", organizationID)

	fieldJoinMap := map[string]string{} // This maps a field name to the the kind of join to use.
	var f2 func(clause filter.Clause) error
	f2 = func(clause filter.Clause) error {
		slog.DebugContext(ctx, fmt.Sprintf("f2: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("f2: condition: %+v", typedClause))

			personFieldDefinition := fieldDefinitionByNameMap[typedClause.Name]
			if personFieldDefinition == nil {
				return fmt.Errorf("unknown field: %s", typedClause.Name)
			}

			fieldJoinType := "INNER JOIN"
			if fieldJoinMap[typedClause.Name] != "" {
				fieldJoinType = fieldJoinMap[typedClause.Name]
			} else {
				fieldJoinMap[typedClause.Name] = fieldJoinType
			}

			switch typedClause.Operation {
			case filter.OperationEquals:
				// Inner join required.
			case filter.OperationNotEquals:
				fieldJoinType = "LEFT OUTER JOIN"
			case filter.OperationIs:
				if typedClause.Value == "null" {
					fieldJoinType = "LEFT OUTER JOIN"
				} else {
					return fmt.Errorf("invalid value for is operation: %s", typedClause.Value)
				}
			case filter.OperationIsNot:
				if typedClause.Value == "null" {
					// Inner join is fine.
				} else {
					return fmt.Errorf("invalid value for is not operation: %s", typedClause.Value)
				}
			case filter.OperationGreaterThan:
				// Inner join is fine.
			case filter.OperationGreaterThanOrEqual:
				// Inner join is fine.
			case filter.OperationLessThan:
				// Inner join is fine.
			case filter.OperationLessThanOrEqual:
				// Inner join is fine.
			case filter.OperationWildcard:
				// Inner join is fine.
			default:
				return fmt.Errorf("unknown operation: %s", typedClause.Operation)
			}

			if fieldJoinType != "INNER JOIN" {
				fieldJoinMap[typedClause.Name] = fieldJoinType
			}
		case *filter.ClauseGroup:
			slog.DebugContext(ctx, fmt.Sprintf("f2: group: %+v", typedClause))

			for _, groupClause := range typedClause.Clauses {
				switch typedClause.Operation {
				case filter.ClauseGroupOperationAnd:
					err := f2(groupClause)
					if err != nil {
						return err
					}
				case filter.ClauseGroupOperationOr:
					err := f2(groupClause)
					if err != nil {
						return err
					}
				}
			}
		default:
			return fmt.Errorf("unknown clause type: %T", typedClause)
		}
		return nil
	}

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

				fieldJoinType := fieldJoinMap[typedClause.Name]
				if fieldJoinType == "" {
					slog.WarnContext(ctx, fmt.Sprintf("no field join type for field: %s", typedClause.Name))
					fieldJoinType = "LEFT OUTER JOIN"
				}

				query = query.Joins(fieldJoinType+" person_field AS "+fieldTableName+" ON person.id = "+fieldTableName+".person_id AND "+fieldTableName+".person_field_definition_id = ?", personFieldDefinition.ID)
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
					err := f(groupClause, newQuery)
					if err != nil {
						return err
					}
					groupQuery.Where(newQuery)
				case filter.ClauseGroupOperationOr:
					newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
					err := f(groupClause, newQuery)
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
		for _, groupHierarchy := range groupHierarchies {
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

			err = f2(groupClause)
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

	return query, nil
}

func filterPersons(ctx context.Context, db *gorm.DB, userID uint64, organizationID uint64, groupID *uint64, filterString *string, returnFields *[]string, limit int) ([]*downballotapi.Person, error) {
	groupHierarchies, err := getGroupHierarchiesForUser(db, userID, organizationID)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(groupHierarchies)))

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
		groupHierarchies = condenseHierarchies(groupHierarchies)
		slog.InfoContext(ctx, fmt.Sprintf("Consensed hierarchies: (%d)", len(groupHierarchies)))
	} else {
		var groupHierarchy []*schema.Group
		for _, hierarchy := range groupHierarchies {
			if len(hierarchy) > 0 && hierarchy[len(hierarchy)-1].ID == *groupID {
				groupHierarchy = hierarchy
				break
			}
		}

		if len(groupHierarchy) == 0 {
			return nil, fmt.Errorf("could not find hierarchy for group.id=%d", *groupID)
		}

		groupHierarchies = [][]*schema.Group{groupHierarchy}
		slog.InfoContext(ctx, fmt.Sprintf("Group-limited hierarchies: (%d)", len(groupHierarchies)))
	}

	query, err := buildPersonQuery(ctx, db, organizationID, groupHierarchies, filterString, fieldDefinitionByNameMap)
	if err != nil {
		return nil, err
	}

	var persons []*schema.Person
	err = query.
		Distinct().
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

func filterPersonsCount(ctx context.Context, db *gorm.DB, userID uint64, organizationID uint64, groupIDs []uint64, filterString *string) (map[uint64]int64, error) {
	groupHierarchies, err := getGroupHierarchiesForUser(db, userID, organizationID)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(groupHierarchies)))

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

	groupIDToCountMap := map[uint64]int64{}
	for _, groupID := range groupIDs {
		groupIDToCountMap[groupID] = 0
	}

	slog.InfoContext(ctx, fmt.Sprintf("Group IDs: (%d)", len(groupIDs)))
	for _, hierarchy := range groupHierarchies {
		if len(hierarchy) == 0 {
			continue
		}
		if !slices.Contains(groupIDs, hierarchy[len(hierarchy)-1].ID) {
			continue
		}
		groupID := hierarchy[len(hierarchy)-1].ID
		groupHierarchy := hierarchy

		groupHierarchies = [][]*schema.Group{groupHierarchy}
		slog.InfoContext(ctx, fmt.Sprintf("Group-limited hierarchies: (%d)", len(groupHierarchies)))

		query, err := buildPersonQuery(ctx, db, organizationID, groupHierarchies, filterString, fieldDefinitionByNameMap)
		if err != nil {
			return nil, err
		}

		var count int64
		err = query.
			Distinct().
			Count(&count).
			Error
		if err != nil {
			return nil, err
		}

		groupIDToCountMap[groupID] = count
	}

	return groupIDToCountMap, nil
}
