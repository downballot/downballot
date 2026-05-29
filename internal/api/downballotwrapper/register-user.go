package downballotwrapper

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/emicklei/go-restful/v3"
	"github.com/threatmate/restfulwrapper"
)

// init registers the custom user-related tags for `restfulwrapper`.
func init() {
	const localMapAuthentication = "downballot.authentication"

	restfulwrapper.Register("downballot.currentUser", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		requireAuthentication := false
		switch field.Type.String() {
		case "downballotwrapper.User":
			requireAuthentication = true
		case "*downballotwrapper.User":
		default:
			return nil, fmt.Errorf("bad type for field %s", field.Name)
		}

		if info.LocalMap[localMapAuthentication] == "" {
			info.Do = append(info.Do, doRequireAuthentication(requireAuthentication))
			info.LocalMap[localMapAuthentication] = "true"
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()

			user, err := getUserFromRequest(req)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Could not get current user: %v", err))
				switch v.Interface().(type) {
				case User:
					return restfulwrapper.NewAPIResponseError(http.StatusForbidden, "Forbidden")
				case *User:
					v.Set(reflect.ValueOf((*User)(nil)))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			} else {
				switch v.Interface().(type) {
				case User:
					v.Set(reflect.ValueOf(*user))
				case *User:
					v.Set(reflect.ValueOf(user))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			}
			return nil
		}, nil
	})
}
