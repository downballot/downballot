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
	Name     string `gorm:"column:name;size:256"`
	Username string `gorm:"column:name;size:256"`
}

func (User) TableName() string {
	return "user"
}
