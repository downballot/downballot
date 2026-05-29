package schema

// Group is a group within an organziation.
//
// By default, there will be a group called "Root", but an organization could subdivide itself
// any way that it wants.
//
// A group has a particular filter that limits its voter access.
type Group struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64        `gorm:"column:organization_id;not null"`
	Organization   *Organization `gorm:"belongsTo;constraint:fk_group_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id" json:"-"`
	ParentID       *uint64       `gorm:"column:parent_id"`
	Parent         *Group        `gorm:"belongsTo;constraint:fk_group_parent,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:parent_id;references:id" json:"-"`
	Name           string        `gorm:"column:name;size:256;type:varchar(256) collate nocase"`
	Filter         string        `gorm:"column:filter;type:text collate nocase"`
}

func (Group) TableName() string {
	return "group"
}
