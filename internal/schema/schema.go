package schema

type Organization struct {
	ID   uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Name string `gorm:"column:name;size:256"`
}

func (Organization) TableName() string {
	return "organization"
}

type User struct {
	ID       uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Username string `gorm:"column:username;size:256;unique"`
	Name     string `gorm:"column:name;size:256"`
}

func (User) TableName() string {
	return "user"
}

type UserOrganizationMap struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	UserID         uint64        `gorm:"column:user_id;not null"`
	User           *User         `json:"-" gorm:"constraint:fk_user_organization_map_user,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:user_id;references:id"`
	OrganizationID uint64        `gorm:"column:organization_id;not null"`
	Organization   *Organization `json:"-" gorm:"constraint:fk_user_organization_map_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
}

func (UserOrganizationMap) TableName() string {
	return "user_organization_map"
}

type Person struct {
	ID             uint64        `gorm:"column:id;primaryKey;not null;autoIncrement"`
	OrganizationID uint64        `gorm:"column:organization_id;not null"`
	Organization   *Organization `json:"-" gorm:"constraint:fk_user_organization_map_organization,OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:organization_id;references:id"`
	VoterID        string        `gorm:"column:voter_id;not null;size:256"`
}

func (Person) TableName() string {
	return "person"
}
