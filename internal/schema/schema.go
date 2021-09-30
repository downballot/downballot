package schema

type Organization struct {
	ID   uint64 `gorm:"column:id;primaryKey;not null;autoIncrement"`
	Name string `gorm:"column:name;size:256"`
}
