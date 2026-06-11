package sqltype

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

const Format = "2006-01-02 15:04:05"

type DateTime time.Time

var _ gorm.Valuer = (*DateTime)(nil)
var _ migrator.GormDataTypeInterface = (*DateTime)(nil)
var _ schema.GormDataTypeInterface = (*DateTime)(nil)
var _ sql.Scanner = (*DateTime)(nil)
var _ driver.Valuer = (*DateTime)(nil)

func (DateTime) GormDataType() string {
	return string(schema.Time)
}

func (DateTime) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql":
		return "DATETIME"
	case "sqlite":
		return "TEXT"
	}
	return ""
}

func (t DateTime) GormValue(ctx context.Context, db *gorm.DB) (expr clause.Expr) {
	switch db.Dialector.Name() {
	case "mysql":
		return clause.Expr{SQL: "?", Vars: []any{time.Time(t).UTC()}}
	case "sqlite":
		return clause.Expr{SQL: "?", Vars: []any{time.Time(t).UTC().Format(Format)}}
	default:
		return clause.Expr{SQL: "?", Vars: []any{time.Time(t).UTC()}}
	}
}

// Scan a value from the database.
func (t *DateTime) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		newValue, err := time.Parse(Format, string(v))
		if err != nil {
			return err
		}
		*t = DateTime(newValue)
	case string:
		newValue, err := time.Parse(Format, v)
		if err != nil {
			return err
		}
		*t = DateTime(newValue)
	case time.Time:
		*t = DateTime(v)
	default:
		return fmt.Errorf("invalid type: %T", value)
	}

	return nil
}

// Value returns a value suitable for insert.
func (t DateTime) Value() (driver.Value, error) {
	return time.Time(t).UTC().Format(Format), nil
}
