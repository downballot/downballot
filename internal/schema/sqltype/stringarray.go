package sqltype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringArray is a custom type for JSON arrays of strings
type StringArray []string

// Value implements driver.Valuer: converts Go slice to JSON for the DB
func (a StringArray) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner: converts JSON from the DB to Go slice
func (a *StringArray) Scan(src any) error {
	if src == nil {
		return nil
	}
	var content []byte
	switch v := src.(type) {
	case []byte:
		content = v
	case string:
		content = []byte(v)
	default:
		return fmt.Errorf("invalid underlying type: %T", src)
	}
	return json.Unmarshal(content, a)
}
