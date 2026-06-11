package downballotwrapper

import (
	"fmt"
	"reflect"

	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/restfulwrapper"
)

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
