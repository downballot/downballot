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

const attributeOrganizationID = "downballotwrapper.organizationID"

func init() {
	restfulwrapper.Register("downballot.organizationPermissionSet", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		requireAuthentication := false
		switch field.Type.String() {
		case "permissionset.PermissionSet":
			requireAuthentication = true
		case "*permissionset.PermissionSet":
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
				case permissionset.PermissionSet:
					return restfulwrapper.NewAPIResponseError(http.StatusForbidden, "Forbidden")
				case *permissionset.PermissionSet:
					v.Set(reflect.ValueOf((*User)(nil)))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			} else {
				organizationIDValue := req.Attribute(attributeOrganizationID)
				if organizationIDValue == nil {
					return fmt.Errorf("could not find organization ID value %q in request attributes", attributeOrganizationID)
				}

				organizationID, ok := organizationIDValue.(uint64)
				if !ok {
					return fmt.Errorf("organization ID value is not a uint64: %T", organizationIDValue)
				}
				permissionSet := user.PermissionSetForOrganization(organizationID)
				slog.DebugContext(ctx, fmt.Sprintf("Permission set for organization.id=%d: %v", organizationID, permissionSet.Permissions()))

				switch v.Interface().(type) {
				case permissionset.PermissionSet:
					v.Set(reflect.ValueOf(permissionSet))
				case *permissionset.PermissionSet:
					v.Set(reflect.ValueOf(&permissionSet))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			}
			return nil
		}, nil
	})
}
