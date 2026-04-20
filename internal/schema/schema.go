package schema

// Organization is an organization using this system.
//
// This will be a candidate campaign.
type Organization struct {
	ID   uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Name string `gorm:"column:name;size:256"`
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
	Name           string        `gorm:"column:name;size:256"`
	Filter         string        `gorm:"column:filter"`
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
	Username string `gorm:"column:username;size:256;unique"`
	Name     string `gorm:"column:name;size:256"`
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

// Person represents a person.
//
// The ideal person is a registered voter, with a voter ID, but whatevs.
type Person struct {
	ID             uint64            `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64            `gorm:"column:organization_id;not null;uniqueIndex:idx_unique_person,priority:1"`
	Organization   *Organization     `json:"-" gorm:"belongsTo;constraint:fk_person_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	VoterID        string            `gorm:"column:voter_id;not null;size:256;uniqueIndex:idx_unique_person,priority:2"`
	Fields         map[string]string `gorm:"-"`
}

func (Person) TableName() string {
	return "person"
}

// PersonField represents a (key,value) field pair for a person.
//
// This is how we'll represent dynamic data, such as the person's name, address, phone number, etc.
type PersonField struct {
	ID       uint64  `gorm:"column:id;primaryKey;not null;autoIncrement"`
	PersonID uint64  `gorm:"column:person_id;not null;uniqueIndex:idx_unique_person_field,priority:1"`
	Person   *Person `json:"-" gorm:"belongsTo;constraint:fk_person_field_person,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:person_id;references:id"`
	Name     string  `gorm:"column:name;not null;size:256;uniqueIndex:idx_unique_person_field,priority:2;index:idx_person_field_name_value,priority:1"`
	Value    string  `gorm:"column:value;not null;size:256;index:idx_person_field_name_value,priority:2"`
}

func (PersonField) TableName() string {
	return "person_field"
}
