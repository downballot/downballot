package schema

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
