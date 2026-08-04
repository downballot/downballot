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
		TableName               string // The name of the table.
		ColumnName              string // The name of the column.
		ColumnExpression        string // The expression for the column.
		PersonFieldDefinitionID uint64 // The ID of the person field definition.
	}

	fieldInfoMap := map[string]*FieldInfo{} // This maps a field name the table info for it.

	// registerFieldTableIfNecessary registers a field table if it is not already registered.
	var registerFieldTableIfNecessary func(fieldName string) error
	registerFieldTableIfNecessary = func(fieldName string) error {
		// Handle any special cases first.
		switch fieldName {
		case "voter_id":
			return nil
		}

		personFieldDefinition := fieldDefinitionByNameMap[fieldName]
		if personFieldDefinition == nil {
			return fmt.Errorf("unknown field: %s", fieldName)
		}

		fieldInfo := fieldInfoMap[fieldName]
		if fieldInfo != nil {
			return nil
		}

		if personFieldDefinition.ComputedExpression != "" {
			tokenList, err := filter.Tokenize(personFieldDefinition.ComputedExpression)
			if err != nil {
				return fmt.Errorf("could not tokenize computed expression: %w", err)
			}

			var parts []string
			for !tokenList.IsEmpty() {
				token, err := tokenList.Next()
				if err != nil {
					return fmt.Errorf("error reading token: %w", err)
				}

				slog.DebugContext(ctx, fmt.Sprintf("* %s (quoted: %t)", token.Value, token.Quote != ""))

				newPart := token.String()
				if !token.Quoted() {
					if fieldDefinitionByNameMap[token.Value] != nil {
						registerFieldTableIfNecessary(token.Value)
						if fieldInfoMap[token.Value] != nil {
							newPart = fieldInfoMap[token.Value].ColumnName
						}
					} else {
						switch token.Value {
						case "-", "+", "*", "/", "(", ")", "=", ">", "<", ">=", "<=", "!=", "~":
							// This is legit.
						default:
							_, err := strconv.ParseFloat(token.Value, 64)
							if err != nil {
								return fmt.Errorf("could not parse float: %w", err)
							}
						}
					}
				}

				parts = append(parts, newPart)
			}
			computedExpression := strings.Join(parts, " ")

			fieldInfo = &FieldInfo{
				FieldName:               fieldName,
				ColumnName:              "computed_" + fmt.Sprintf("%d", len(fieldInfoMap)+1),
				ColumnExpression:        computedExpression,
				PersonFieldDefinitionID: personFieldDefinition.ID,
			}
		} else {
			tableName := "person_field_join" + fmt.Sprintf("%d", len(fieldInfoMap)+1)
			fieldInfo = &FieldInfo{
				FieldName:               fieldName,
				TableName:               tableName,
				ColumnName:              tableName + ".value",
				PersonFieldDefinitionID: personFieldDefinition.ID,
			}
		}
		fieldInfoMap[fieldName] = fieldInfo

		return nil
	}

	var recursiveBuildInfo func(clause filter.Clause) error
	recursiveBuildInfo = func(clause filter.Clause) error {
		slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseCondition: %+v", typedClause))

			err := registerFieldTableIfNecessary(typedClause.Name)
			if err != nil {
				return err
			}
		case *filter.ClauseIsNull:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseIsNull: %+v", typedClause))

			err := registerFieldTableIfNecessary(typedClause.Name)
			if err != nil {
				return err
			}
		case *filter.ClauseIsNotNull:
			slog.DebugContext(ctx, fmt.Sprintf("recursiveBuildInfo: ClauseIsNotNull: %+v", typedClause))

			err := registerFieldTableIfNecessary(typedClause.Name)
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

	// getFieldColumn returns the column name for a field.
	getFieldColumn := func(fieldName string) (string, error) {
		// Handle any special cases first.
		switch fieldName {
		case "voter_id":
			return "person.voter_id", nil
		}

		fieldInfo := fieldInfoMap[fieldName]
		if fieldInfo == nil {
			return "", fmt.Errorf("unknown field: %s", fieldName)
		}
		return fieldInfo.ColumnName, nil
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

			fieldColumn, err := getFieldColumn(typedClause.Name)
			if err != nil {
				return fmt.Errorf("could not get field column (%T) %q: %w", typedClause, typedClause.Name, err)
			}

			// We need to create a parenthetical subquery and add everything to that.
			subquery := db.Session(&gorm.Session{NewDB: true, Initialized: true})
			for _, value := range typedClause.Values {
				switch typedClause.Operation {
				case filter.OperationEquals:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldColumn+" AS INTEGER) = ?", value)
					default:
						subquery = subquery.Or(fieldColumn+" = ?", value)
					}
				case filter.OperationNotEquals:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Where(fieldColumn+" IS NULL OR CAST("+fieldColumn+" AS INTEGER) != ?", value)
					default:
						subquery = subquery.Where(fieldColumn+" IS NULL OR "+fieldColumn+" != ?", value)
					}
				case filter.OperationGreaterThan:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldColumn+" AS INTEGER) > ?", value)
					default:
						subquery = subquery.Or(fieldColumn+" > ?", value)
					}
				case filter.OperationGreaterThanOrEqual:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldColumn+" AS INTEGER) >= ?", value)
					default:
						subquery = subquery.Or(fieldColumn+" >= ?", value)
					}
				case filter.OperationLessThan:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldColumn+" AS INTEGER) < ?", value)
					default:
						subquery = subquery.Or(fieldColumn+" < ?", value)
					}
				case filter.OperationLessThanOrEqual:
					switch personFieldDefinition.Type {
					case "integer":
						subquery = subquery.Or("CAST("+fieldColumn+" AS INTEGER) <= ?", value)
					default:
						subquery = subquery.Or(fieldColumn+" <= ?", value)
					}
				case filter.OperationSetHasOne:
					subquery = subquery.Or(fieldColumn+" LIKE ?", "%,"+value+",%")
				case filter.OperationSetHasAll:
					subquery = subquery.Where(fieldColumn+" LIKE ?", "%,"+value+",%")
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
								Where("CAST(SUBSTR("+fieldColumn+", 1, INSTR("+fieldColumn+", ',')) AS REAL) BETWEEN ? AND ?", latitude-100*oneMeter, latitude+100*oneMeter).
								Where("CAST(SUBSTR("+fieldColumn+", INSTR("+fieldColumn+", ',') + 1) AS REAL) BETWEEN ? AND ?", longitude-100*oneMeter, longitude+100*oneMeter),
						)
					default:
						subquery = subquery.Or(fieldColumn+" LIKE ?", strings.ReplaceAll(value, "*", "%"))
					}
				case filter.OperationNotWildcard:
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
						subquery = subquery.Where(
							db.Session(&gorm.Session{NewDB: true, Initialized: true}).
								Or("CAST(SUBSTR("+fieldColumn+", 1, INSTR("+fieldColumn+", ',')) AS REAL) NOT BETWEEN ? AND ?", latitude-100*oneMeter, latitude+100*oneMeter).
								Or("CAST(SUBSTR("+fieldColumn+", INSTR("+fieldColumn+", ',') + 1) AS REAL) NOT BETWEEN ? AND ?", longitude-100*oneMeter, longitude+100*oneMeter),
						)
					default:
						subquery = subquery.Where(fieldColumn+" NOT LIKE ?", strings.ReplaceAll(value, "*", "%"))
					}
				default:
					return fmt.Errorf("unknown operation: %s", typedClause.Operation)
				}
			}
			groupQuery = groupQuery.Where(subquery)
		case *filter.ClauseIsNull:
			fieldColumn, err := getFieldColumn(typedClause.Name)
			if err != nil {
				return fmt.Errorf("could not get field column (%T) %q: %w", typedClause, typedClause.Name, err)
			}

			groupQuery = groupQuery.Where(fieldColumn + " IS NULL")
		case *filter.ClauseIsNotNull:
			fieldColumn, err := getFieldColumn(typedClause.Name)
			if err != nil {
				return fmt.Errorf("could not get field column (%T) %q: %w", typedClause, typedClause.Name, err)
			}

			groupQuery = groupQuery.Where(fieldColumn + " IS NOT NULL")
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

	slog.DebugContext(ctx, "Group hierarchies.", "groupHierarchies", groupHierarchies)
	{
		var hierarchyStrings []string
		if len(groupHierarchies) == 0 {
			if filterString != nil && *filterString != "" {
				hierarchyStrings = append(hierarchyStrings, *filterString)
			}
		} else {
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
				if left.TableName == "" && right.TableName != "" {
					return 1
				}
				if left.TableName != "" && right.TableName == "" {
					return -1
				}
				if left.TableName != "" && right.TableName != "" {
					return cmp.Compare(left.TableName, right.TableName)
				}

				return cmp.Compare(left.ColumnName, right.ColumnName)
			})
			slog.DebugContext(ctx, fmt.Sprintf("Field info list: (%d)", len(fieldInfoList)))

			// Set up the joins.
			for _, fieldInfo := range fieldInfoList {
				if fieldInfo.TableName == "" {
					continue
				}

				joinType := "LEFT OUTER JOIN" // Note: there's no clear, easy way to force the use of an INNER JOIN, given the ability to have "OR" expressions everywhere.
				query = query.Joins("/* "+fieldInfo.FieldName+" */ "+joinType+" person_field AS "+fieldInfo.TableName+" ON person.id = "+fieldInfo.TableName+".person_id AND "+fieldInfo.TableName+".person_field_definition_id = ?", fieldInfo.PersonFieldDefinitionID)
			}

			// Tack on the WHERE clause based on the filter.
			newQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true})

			err = f(groupClause, newQuery)
			if err != nil {
				return nil, err
			}
			query = query.Where(newQuery)

			columns := []string{"person.*"}
			for _, fieldInfo := range fieldInfoList {
				if fieldInfo.TableName != "" {
					continue
				}

				columnExpression := fieldInfo.ColumnExpression + " AS " + fieldInfo.ColumnName
				columns = append(columns, columnExpression)
			}
			query = query.Select(strings.Join(columns, ", "))
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
		for name, value := range personFieldsMap[person.ID] {
			o.Fields[name] = value
		}

		// Compute the computed fields.
		for _, personFieldDefinition := range fieldDefinitionByNameMap {
			if personFieldDefinition.ComputedExpression == "" {
				continue
			}

			var computedExpression string
			var variables []any
			{
				tokenList, err := filter.Tokenize(personFieldDefinition.ComputedExpression)
				if err != nil {
					return nil, fmt.Errorf("could not tokenize computed expression: %w", err)
				}

				var parts []string
				for !tokenList.IsEmpty() {
					token, err := tokenList.Next()
					if err != nil {
						return nil, fmt.Errorf("error reading token: %w", err)
					}
					slog.DebugContext(ctx, fmt.Sprintf("* %s (quoted: %t)", token.Value, token.Quote != ""))

					if token.Quoted() {
						parts = append(parts, token.String())
						continue
					}

					if fieldDefinitionByNameMap[token.Value] != nil {
						parts = append(parts, "?")
						variables = append(variables, o.Fields[token.Value])
						continue
					}

					switch strings.ToLower(token.Value) {
					case "-", "+", "*", "/", "(", ")", ">", "<", ">=", "<=":
						// This is legit and works the normal SQL way.
						nextToken, err := tokenList.Peek()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						_ = nextToken
						parts = append(parts, token.String())
					case "=":
						// This is legit, but we have special syntax.
						nextToken, err := tokenList.Peek()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsOpeningParen(nextToken) {
							_, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							parts = append(parts, token.String())
						} else {
							nextToken, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							group, err := filter.ReadParentheticalGroup(tokenList)
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							for _, groupToken := range group {
								// TODO:
								_ = groupToken
							}
						}
					case "!=":
						// This is legit, but we have special syntax.
						nextToken, err := tokenList.Peek()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsOpeningParen(nextToken) {
							_, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							parts = append(parts, token.String())
						} else {
							nextToken, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							group, err := filter.ReadParentheticalGroup(tokenList)
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							for _, groupToken := range group {
								// TODO:
								_ = groupToken
							}
						}
					case "~", "!~":
						// This is legit, but we have special syntax.
						nextToken, err := tokenList.Peek()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsOpeningParen(nextToken) {
							_, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							parts = append(parts, "LIKE", "?")
							variables = append(variables, "%,"+nextToken.Value+",%")
						} else {
							nextToken, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							group, err := filter.ReadParentheticalGroup(tokenList)
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							for _, groupToken := range group {
								// TODO:
								_ = groupToken
							}
						}
					case "has_one", "has_all":
						// This is legit, but we have special syntax.
						nextToken, err := tokenList.Peek()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsOpeningParen(nextToken) {
							nextToken, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							parts = append(parts, "LIKE", "?")
							variables = append(variables, "%,"+nextToken.Value+",%")
						} else {
							nextToken, err = tokenList.Next()
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							group, err := filter.ReadParentheticalGroup(tokenList)
							if err != nil {
								return nil, fmt.Errorf("error reading token: %w", err)
							}
							for _, groupToken := range group {
								// TODO:
								_ = groupToken
							}
						}
					case "current_year":
						// This is legit, but we have special syntax.
						nextToken, err := tokenList.Next()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsOpeningParen(nextToken) {
							return nil, fmt.Errorf("expected opening paren")
						}
						nextToken, err = tokenList.Next()
						if err != nil {
							return nil, fmt.Errorf("error reading token: %w", err)
						}
						if !filter.TokenIsClosingParen(nextToken) {
							return nil, fmt.Errorf("expected closing paren")
						}
						parts = append(parts, "CAST(STRFTIME('%Y', 'NOW') AS INTEGER)")
					default:
						_, err := strconv.ParseFloat(token.Value, 64)
						if err != nil {
							return nil, fmt.Errorf("could not parse float: %w", err)
						}
						parts = append(parts, token.String())
					}
				}
				computedExpression = strings.Join(parts, " ")
			}

			var expressionResult string
			err = db.Session(&gorm.Session{}).
				Raw(`SELECT `+computedExpression+` AS expression_result`, variables...).
				Scan(&expressionResult).
				Error
			if err != nil {
				return nil, fmt.Errorf("could not execute computed expression: %w", err)
			}
			o.Fields[personFieldDefinition.Name] = expressionResult
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
