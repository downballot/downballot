package schema

// Filter is a filter on persons within an organization.
//
// You can think of a filter as a "saved search" that can be re-used.
//
// A filter can be public (everyone can see it) or private (only the user who created it can see it).
type Filter struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64        `gorm:"column:organization_id;not null"`
	Organization   *Organization `gorm:"belongsTo;constraint:fk_group_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id" json:"-"`
	UserID         *uint64       `gorm:"column:user_id"`
	User           *User         `gorm:"belongsTo;constraint:fk_filter_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id" json:"-"`
	Name           string        `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
	Description    string        `gorm:"column:description;type:text collate nocase"`
	Filter         string        `gorm:"column:filter;type:text collate nocase"`
}

func (Filter) TableName() string {
	return "filter"
}
