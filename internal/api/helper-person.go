package api

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
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

	type FieldInfo struct {
		FieldName               string // The name of the field.
		InnerJoin               bool   // True if the join is an inner join, false if it is a left outer join.
		TableName               string // The name of the table.
		PersonFieldDefinitionID uint64 // The ID of the person field definition.
	}

	fieldInfoMap := map[string]*FieldInfo{} // This maps a field name the table info for it.

	// registerFieldTableIfNecessary registers a field table if it is not already registered.
	//
	// The field is assumed to be required, but if any request says that it's *not* required, then
	// the InnerJoin field will be set to false.
	registerFieldTableIfNecessary := func(fieldName string, required bool) error {
		personFieldDefinition := fieldDefinitionByNameMap[fieldName]
		if personFieldDefinition == nil {
			return fmt.Errorf("unknown field: %s", fieldName)
		}

		var fieldInfo *FieldInfo
		if fieldInfoMap[fieldName] != nil {
			fieldInfo = fieldInfoMap[fieldName]
		} else {
			fieldInfo = &FieldInfo{
				FieldName:               fieldName,
				InnerJoin:               true,
				TableName:               "person_field_join" + fmt.Sprintf("%d", len(fieldInfoMap)+1),
				PersonFieldDefinitionID: personFieldDefinition.ID,
			}
			fieldInfoMap[fieldName] = fieldInfo
		}

		if !required {
			fieldInfo.InnerJoin = false
		}
		return nil
	}

	var recursiveBuildInfo func(clause filter.Clause) error
	recursiveBuildInfo = func(clause filter.Clause) error {
		slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseCondition: %+v", typedClause))

			innerJoin := true
			switch typedClause.Operation {
			case filter.OperationEquals:
				// Inner join is fine.
			case filter.OperationNotEquals:
				innerJoin = false
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

			err := registerFieldTableIfNecessary(typedClause.Name, innerJoin)
			if err != nil {
				return err
			}
		case *filter.ClauseIsNull:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseIsNull: %+v", typedClause))

			err := registerFieldTableIfNecessary(typedClause.Name, false)
			if err != nil {
				return err
			}
		case *filter.ClauseIsNotNull:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseIsNotNull: %+v", typedClause))

			err := registerFieldTableIfNecessary(typedClause.Name, true)
			if err != nil {
				return err
			}
		case *filter.ClauseGroup:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: group: %+v", typedClause))

			for _, groupClause := range typedClause.Clauses {
				switch typedClause.Operation {
				case filter.ClauseGroupOperationAnd:
					err := recursiveBuildInfo(groupClause)
					if err != nil {
						return err
					}
				case filter.ClauseGroupOperationOr:
					err := recursiveBuildInfo(groupClause)
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

			fieldInfo := fieldInfoMap[typedClause.Name]
			if fieldInfo == nil {
				return fmt.Errorf("unknown field: %s", typedClause.Name)
			}

			// We need to create a parenthetical subquery and add everything to that.
			subquery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
			for _, value := range typedClause.Values {
				switch typedClause.Operation {
				case filter.OperationEquals:
					subquery = subquery.Or(fieldInfo.TableName+".value = ?", value)
				case filter.OperationNotEquals:
					subquery = subquery.Or(fieldInfo.TableName+".value IS NULL OR "+fieldInfo.TableName+".value != ?", value)
				case filter.OperationGreaterThan:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldInfo.TableName+".value AS INTEGER) > ?", value)
					default:
						subquery = subquery.Or(fieldInfo.TableName+".value > ?", value)
					}
				case filter.OperationGreaterThanOrEqual:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldInfo.TableName+".value AS INTEGER) >= ?", value)
					default:
						subquery = subquery.Or(fieldInfo.TableName+".value >= ?", value)
					}
				case filter.OperationLessThan:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldInfo.TableName+".value AS INTEGER) < ?", value)
					default:
						subquery = subquery.Or(fieldInfo.TableName+".value < ?", value)
					}
				case filter.OperationLessThanOrEqual:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldInfo.TableName+".value AS INTEGER) <= ?", value)
					default:
						subquery = subquery.Or(fieldInfo.TableName+".value <= ?", value)
					}
				case filter.OperationWildcard:
					switch personFieldDefinition.Type {
					case schema.PersonFieldDefinitionTypeCoordinates:
						parts := strings.SplitN(value, ",", 2)
						if len(parts) != 2 {
							return fmt.Errorf("invalid coordinates value: %s", value)
						}
						latitude, err := strconv.ParseFloat(parts[0], 64)
						if err != nil {
							return fmt.Errorf("invalid latitude: %w", err)
						}
						longitude, err := strconv.ParseFloat(parts[1], 64)
						if err != nil {
							return fmt.Errorf("invalid longitude: %w", err)
						}
						oneMeter := 0.000009
						subquery = subquery.Or(
							db.Session(&gorm.Session{NewDB: true, Initialized: true}).
								Where("CAST(SUBSTR("+fieldInfo.TableName+".value, 1, INSTR("+fieldInfo.TableName+".value, ',')) AS REAL) BETWEEN ? AND ?", latitude-100*oneMeter, latitude+100*oneMeter).
								Where("CAST(SUBSTR("+fieldInfo.TableName+".value, INSTR("+fieldInfo.TableName+".value, ',') + 1) AS REAL) BETWEEN ? AND ?", longitude-100*oneMeter, longitude+100*oneMeter),
						)
					default:
						subquery = subquery.Or(fieldInfo.TableName+".value LIKE ?", strings.ReplaceAll(value, "*", "%"))
					}
				default:
					return fmt.Errorf("unknown operation: %s", typedClause.Operation)
				}
			}
			groupQuery = groupQuery.Where(subquery)
		case *filter.ClauseIsNull:
			fieldInfo := fieldInfoMap[typedClause.Name]
			if fieldInfo == nil {
				return fmt.Errorf("unknown field: %s", typedClause.Name)
			}

			groupQuery = groupQuery.Where(fieldInfo.TableName + ".value IS NULL")
		case *filter.ClauseIsNotNull:
			fieldInfo := fieldInfoMap[typedClause.Name]
			if fieldInfo == nil {
				return fmt.Errorf("unknown field: %s", typedClause.Name)
			}

			groupQuery = groupQuery.Where(fieldInfo.TableName + ".value IS NOT NULL")
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

			// Build the field info map; this will populate `fieldInfoMap`.
			err = recursiveBuildInfo(groupClause)
			if err != nil {
				return nil, err
			}

			// Sort the fields so that all of the inner joins are first, followed by all of the left outer joins.
			var fieldInfoList []*FieldInfo
			for _, fieldInfo := range fieldInfoMap {
				fieldInfoList = append(fieldInfoList, fieldInfo)
			}
			slices.SortFunc(fieldInfoList, func(left, right *FieldInfo) int {
				leftInnerJoin := 1
				if !left.InnerJoin {
					leftInnerJoin = 0
				}
				rightInnerJoin := 1
				if !right.InnerJoin {
					rightInnerJoin = 0
				}
				diff := -cmp.Compare(leftInnerJoin, rightInnerJoin)
				if diff != 0 {
					return diff
				}
				return cmp.Compare(left.TableName, right.TableName)
			})
			slog.DebugContext(ctx, fmt.Sprintf("Field info list: (%d)", len(fieldInfoList)))

			// Set up the joins.
			for _, fieldInfo := range fieldInfoList {
				joinType := "INNER JOIN"
				if !fieldInfo.InnerJoin {
					joinType = "LEFT OUTER JOIN"
				}
				query = query.Joins("/* "+fieldInfo.FieldName+" */ "+joinType+" person_field AS "+fieldInfo.TableName+" ON person.id = "+fieldInfo.TableName+".person_id AND "+fieldInfo.TableName+".person_field_definition_id = ?", fieldInfo.PersonFieldDefinitionID)
			}

			// Tack on the WHERE clause based on the filter.
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
