package downballotwrapper

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/downballot/downballot/internal/permissionset"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/restfulwrapper"
)

func init() {
	restfulwrapper.Register("downballot.permission", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		requireAuthentication := false
		switch field.Type.String() {
		case "string":
			requireAuthentication = true
		case "*string":
		default:
			return nil, fmt.Errorf("bad type for field %s: %T", field.Name, field.Type.String())
		}

		if info.LocalMap[localMapAuthentication] == "" {
			info.Do = append(info.Do, doRequireAuthentication(requireAuthentication))
			info.LocalMap[localMapAuthentication] = "true"
		}

		permission := permissionset.Permission(apiTagValue)
		if !permission.Valid() {
			return nil, fmt.Errorf("invalid permission: %s", apiTagValue)
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()

			user, err := getUserFromRequest(req)
			if err != nil {
				return restfulwrapper.NewAPIResponseError(http.StatusForbidden, "Forbidden")
			}

			organizationIDInterface := req.Attribute(attributeOrganizationID)
			if organizationIDInterface == nil {
				return fmt.Errorf("could not find organization ID value %q in request attributes", attributeOrganizationID)
			}
			organizationID, ok := organizationIDInterface.(uint64)
			if !ok {
				return fmt.Errorf("organization ID value is not a uint64: %T", organizationIDInterface)
			}
			permissionSet := user.PermissionSetForOrganization(organizationID)
			slog.DebugContext(ctx, fmt.Sprintf("Permission set for organization.id=%d: %v", organizationID, permissionSet.Permissions()))

			matched := permissionSet.Match(permission)
			if !matched {
				return restfulwrapper.NewAPIResponseError(http.StatusForbidden, fmt.Sprintf("Forbidden: missing permission: %s", permission))
			}

			if v.CanSet() {
				switch v.Interface().(type) {
				case string:
					v.Set(reflect.ValueOf(string(permission)))
				case *string:
					newString := string(permission)
					v.Set(reflect.ValueOf(&newString))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			}
			return nil
		}, nil
	})
}
