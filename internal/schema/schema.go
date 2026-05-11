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

// Organization is an organization using this system.
//
// This will be a candidate campaign.
type Organization struct {
	ID   uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Name string `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
}

func (Organization) TableName() string {
	return "organization"
}

// Group is a group within an organziation.
//
// By default, there will be a group called "Root", but an organization could subdivide itself
// any way that it wants.
//
// A group has a particular filter that limits its voter access.
type Group struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64        `gorm:"column:organization_id;not null"`
	Organization   *Organization `json:"-" gorm:"belongsTo;constraint:fk_group_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	ParentID       *uint64       `gorm:"column:parent_id"`
	Parent         *Group        `json:"-" gorm:"belongsTo;constraint:fk_group_parent,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:parent_id;references:id"`
	Name           string        `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
	Filter         string        `gorm:"column:filter;type:text collate nocase"`
}

func (Group) TableName() string {
	return "group"
}

// User is a user of the system.
//
// A user will be a candidate, a campaign volunteer, a campaign staffer, etc.
//
// A user can be part of many groups.  By being a member of a group, a user is indirectly
// a member of that group's organization.
type User struct {
	ID       uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Username string `gorm:"column:username;size:256;unique;type:varchar(256) collate nocase"`
	Name     string `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
	// TODO: Password hash
}

func (User) TableName() string {
	return "user"
}

// UserGruopMap maps a user to a group.
type UserGroupMap struct {
	ID      uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID  uint64 `gorm:"column:user_id;not null;uniqueIndex:idx_unique_user_group,priority:1"`
	User    *User  `json:"-" gorm:"belongsTo;constraint:fk_user_group_map_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	GroupID uint64 `gorm:"column:group_id;not null;uniqueIndex:idx_unique_user_group,priority:2"`
	Group   *Group `json:"-" gorm:"belongsTo;constraint:fk_user_group_map_group,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:group_id;references:id"`
}

func (UserGroupMap) TableName() string {
	return "user_group_map"
}

// UserOrganizationMap maps a user to an organization.
type UserOrganizationMap struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID         uint64        `gorm:"column:user_id;not null;uniqueIndex:idx_unique_user_organization,priority:1"`
	User           *User         `json:"-" gorm:"belongsTo;constraint:fk_user_organization_map_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	OrganizationID uint64        `gorm:"column:organization_id;not null;uniqueIndex:idx_unique_user_organization,priority:2"`
	Organization   *Organization `json:"-" gorm:"belongsTo;constraint:fk_user_organization_map_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
}

func (UserOrganizationMap) TableName() string {
	return "user_organization_map"
}

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

// Person represents a person.
//
// The ideal person is a registered voter, with a voter ID, but whatevs.
type Person struct {
	ID             uint64            `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64            `gorm:"column:organization_id;not null;uniqueIndex:idx_unique_person,priority:1"`
	Organization   *Organization     `json:"-" gorm:"belongsTo;constraint:fk_person_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	VoterID        string            `gorm:"column:voter_id;not null;size:256;type:varchar(256) collate nocase;uniqueIndex:idx_unique_person,priority:2"`
	Fields         map[string]string `gorm:"-"` // TODO: Use an intermediate structure, not the schema structure.
}

func (Person) TableName() string {
	return "person"
}

// PersonField represents a (key,value) field pair for a person.
//
// This is how we'll represent dynamic data, such as the person's name, address, phone number, etc.
type PersonField struct {
	ID                      uint64                 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	PersonID                uint64                 `gorm:"column:person_id;not null;uniqueIndex:idx_unique_person_field,priority:1;index:idx_person_field,priority:1"`
	Person                  *Person                `json:"-" gorm:"belongsTo;constraint:fk_person_field_person,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_id;references:id"`
	PersonFieldDefinitionID uint64                 `gorm:"column:person_field_definition_id;not null;uniqueIndex:idx_unique_person_field,priority:2;index:idx_person_field,priority:2"`
	PersonFieldDefinition   *PersonFieldDefinition `json:"-" gorm:"belongsTo;constraint:fk_person_field_person_field_definition,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_field_definition_id;references:id"`
	Value                   string                 `gorm:"column:value;not null;size:256;type:varchar(256) collate nocase;index:idx_person_field,priority:3"`
}

func (PersonField) TableName() string {
	return "person_field"
}

// PersonAudit represents a change to a person.
type PersonAudit struct {
	ID                      uint64                 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	PersonID                uint64                 `gorm:"column:person_id;not null"`
	Person                  *Person                `json:"-" gorm:"belongsTo;constraint:fk_person_audit_person,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_id;references:id"`
	Timestamp               sqltype.DateTime       `gorm:"column:timestamp;not null"`
	PersonFieldDefinitionID uint64                 `gorm:"column:person_field_definition_id;not null"`
	PersonFieldDefinition   *PersonFieldDefinition `json:"-" gorm:"belongsTo;constraint:fk_person_audit_person_field_definition,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_field_definition_id;references:id"`
	OldValue                *string                `gorm:"column:old_value;type:text"` // If this is nil, then the field was added.
	NewValue                *string                `gorm:"column:new_value;type:text"` // If this is nil, then the field was deleted.
}

func (PersonAudit) TableName() string {
	return "person_audit"
}
