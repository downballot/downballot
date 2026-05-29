package api

import (
	"context"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/tekkamanendless/restfulwrapper"
)

type GetOrganizationIDPermissionMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_ string `api:"httppath:/organization/{organization_id}/permission"`
	_ string `api:"doc" description:"List the user's permissions."`
	_ string `api:"notes" description:"This lists the user's permissions."`
}

func (a *API) GetOrganizationIDPermission(ctx context.Context, meta GetOrganizationIDPermissionMetadata) (output downballotapi.Envelope[downballotapi.ListPermissionsResponse], err error) {
	permissionSet := meta.CurrentUser.PermissionSetForOrganization(meta.Organization.ID)

	output.Message = "OK"
	output.Success = true
	output.Data.Permissions = []string{}
	for _, permission := range permissionSet.Permissions() {
		output.Data.Permissions = append(output.Data.Permissions, string(permission))
	}
	return output, nil
}
