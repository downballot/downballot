package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
)

type hasOrganization struct {
	OrganizationID string              `api:"path:organization_id" description:"The organization ID"`
	Organization   schema.Organization `api:"database.query:where:id = ?,OrganizationID"`
}
type GetOrganizationIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_ string `api:"httppath:/organization/{organization_id}"`
	_ string `api:"doc" description:"Get the organization."`
	_ string `api:"notes" description:"This gets the organization."`
}

func (a *API) GetOrganizationID(ctx context.Context, meta GetOrganizationIDMetadata) (output downballotapi.Envelope[downballotapi.GetOrganizationResponse], err error) {
	o := downballotapi.Organization{
		ID:   fmt.Sprintf("%d", meta.Organization.ID),
		Name: meta.Organization.Name,
	}
	output.Message = "OK"
	output.Success = true
	output.Data.Organization = o
	return output, nil
}
