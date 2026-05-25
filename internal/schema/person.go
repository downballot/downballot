package schema

import (
	"github.com/downballot/downballot/internal/schema/sqltype"
)

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
	UserID                  uint64                 `gorm:"column:user_id;not null"`
	User                    *User                  `json:"-" gorm:"belongsTo;constraint:fk_user_audit_user,OnDelete:RESTRICT,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	Timestamp               sqltype.DateTime       `gorm:"column:timestamp;not null"`
	PersonFieldDefinitionID uint64                 `gorm:"column:person_field_definition_id;not null"`
	PersonFieldDefinition   *PersonFieldDefinition `json:"-" gorm:"belongsTo;constraint:fk_person_audit_person_field_definition,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_field_definition_id;references:id"`
	OldValue                *string                `gorm:"column:old_value;type:text"` // If this is nil, then the field was added.
	NewValue                *string                `gorm:"column:new_value;type:text"` // If this is nil, then the field was deleted.
}

func (PersonAudit) TableName() string {
	return "person_audit"
}
