package downballotwrapper

import (
	"fmt"
	"reflect"

	"github.com/emicklei/go-restful/v3"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

// UseDatabase can be embedded in order to give the endpoint access to the database.
type UseDatabase struct {
	DB *gorm.DB `api:"database"`
}

// getDatabaseFromRequest returns the database from the request.
//
// If no database is found, then nil is returned.
func getDatabaseFromRequest(req *restful.Request) *gorm.DB {
	dbValue := req.Attribute(attributeDatabase)
	if dbValue == nil {
		return nil
	}
	db := dbValue.(*gorm.DB)
	return db.Session(&gorm.Session{}).WithContext(req.Request.Context())
}

func setDatabaseForRequest(req *restful.Request, db *gorm.DB) {
	req.SetAttribute(attributeDatabase, db)
}

// init registers the custom "database" tags for `restfulwrapper`.
func init() {
	// database can be used to get a handle to the database:
	// DB *gorm.DB `api:"database"`
	restfulwrapper.Register("database", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		switch field.Type.String() {
		case "*gorm.DB":
			// Good.
		default:
			return nil, fmt.Errorf("bad type for field %s", field.Name)
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()
			_ = ctx

			db := getDatabaseFromRequest(req)
			if db == nil {
				return fmt.Errorf("could not get database from request")
			}

			if v.CanSet() {
				v.Set(reflect.ValueOf(db))
			}
			return nil
		}, nil
	})
}
