package schema

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/downballot/downballot/internal/schema/sqltype"
)

// PersonFieldDefinition represents a (key,value) field pair definition for a person.
//
// This is organization-specific.
type PersonFieldDefinition struct {
	ID             uint64                    `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64                    `gorm:"column:organization_id;not null;uniqueIndex:idx_unique_person_field_definition,priority:1"`
	Organization   *Organization             `json:"-" gorm:"belongsTo;constraint:fk_person_field_definition_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	Name           string                    `gorm:"column:name;not null;size:256;type:varchar(256) collate nocase;uniqueIndex:idx_unique_person_field_definition,priority:2"`
	Type           PersonFieldDefinitionType `gorm:"column:type;not null;size:256;type:varchar(256) collate nocase"`
	AllowEmpty     bool                      `gorm:"column:allow_empty;not null;default:0"`
	AllowedValues  sqltype.StringArray       `gorm:"column:allowed_values;type:text"`
	AllowedRegex   string                    `gorm:"column:allowed_regex;type:text"`
}

func (PersonFieldDefinition) TableName() string {
	return "person_field_definition"
}

type PersonFieldDefinitionType string

const (
	PersonFieldDefinitionTypeBoolean     PersonFieldDefinitionType = "boolean"
	PersonFieldDefinitionTypeCoordinates PersonFieldDefinitionType = "coordinates"
	PersonFieldDefinitionTypeDate        PersonFieldDefinitionType = "date"
	PersonFieldDefinitionTypeEnum        PersonFieldDefinitionType = "enum"
	PersonFieldDefinitionTypeInteger     PersonFieldDefinitionType = "integer"
	PersonFieldDefinitionTypeSet         PersonFieldDefinitionType = "set"
	PersonFieldDefinitionTypeString      PersonFieldDefinitionType = "string"
)

func (t PersonFieldDefinition) Validate(input string) error {
	if t.AllowEmpty && input == "" {
		return nil
	}

	switch t.Type {
	case PersonFieldDefinitionTypeBoolean:
		if input != "true" && input != "false" {
			return fmt.Errorf("invalid boolean value: %s", input)
		}
	case PersonFieldDefinitionTypeCoordinates:
		parts := strings.SplitN(input, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid coordinates value: %s", input)
		}
		latitude, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return fmt.Errorf("invalid latitude: %w", err)
		}
		longitude, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return fmt.Errorf("invalid longitude: %w", err)
		}
		if latitude < -90 || latitude > 90 {
			return fmt.Errorf("invalid latitude: %s", input)
		}
		if longitude < -180 || longitude > 180 {
			return fmt.Errorf("invalid longitude: %s", input)
		}
	case PersonFieldDefinitionTypeDate:
		_, err := time.Parse("2006-01-02", input)
		if err != nil {
			return fmt.Errorf("invalid date value: %s", input)
		}
	case PersonFieldDefinitionTypeEnum:
		if !slices.Contains(t.AllowedValues, input) {
			return fmt.Errorf("invalid enum value: %s", input)
		}
	case PersonFieldDefinitionTypeInteger:
		_, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value: %s", input)
		}
	case PersonFieldDefinitionTypeSet:
		if !strings.HasPrefix(input, ",") {
			return fmt.Errorf("set is missing leading comma: %s", input)
		}
		if !strings.HasSuffix(input, ",") {
			return fmt.Errorf("set is missing trailing comma: %s", input)
		}
		values := strings.Split(input[1:len(input)-1], ",")
		allValues := map[string]int{}
		for _, value := range values {
			allValues[value]++
			if len(t.AllowedValues) > 0 && !slices.Contains(t.AllowedValues, value) {
				return fmt.Errorf("invalid set value: %s", value)
			}
		}
		for value, count := range allValues {
			if count > 1 {
				return fmt.Errorf("duplicate set value: %s", value)
			}
		}
	case PersonFieldDefinitionTypeString:
		if t.AllowedRegex != "" {
			r, err := regexp.Compile(t.AllowedRegex)
			if err != nil {
				return fmt.Errorf("invalid regex: %w", err)
			}
			if !r.MatchString(input) {
				return fmt.Errorf("invalid string value: %s", input)
			}
		}
	default:
		return fmt.Errorf("unknown type: %s", t.Type)
	}
	return nil
}
