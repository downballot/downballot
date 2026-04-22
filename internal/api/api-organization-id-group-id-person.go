package api

import (
	"context"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
)

type GetOrganizationIDGroupIDPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_      string               `api:"httppath:/organization/{organization_id}/group/{group_id}/person"`
	_      string               `api:"doc" description:"Get the people in the group."`
	_      string               `api:"notes" description:"This gets the people in the group."`
	Filter *string              `api:"query:filter"`
	Fields *resttype.StringList `api:"query:fields"`
	Limit  int                  `api:"query:limit" default:"1000"`
}

func (a *API) GetOrganizationIDGroupIDPerson(ctx context.Context, meta GetOrganizationIDGroupIDPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, &meta.Group.ID, meta.Filter, (*[]string)(meta.Fields), meta.Limit)
	if err != nil {
		return output, err
	}

	output.Data.Persons = persons
	return output, nil
}

type GetOrganizationIDGroupRootPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	Group  schema.Group         `api:"database.query:where:parent_id IS NULL AND organization_id = ?, OrganizationID"`
	_      string               `api:"httppath:/organization/{organization_id}/group/root/person"`
	_      string               `api:"doc" description:"Get the persons in the group."`
	_      string               `api:"notes" description:"This gets the persons in the group."`
	Filter *string              `api:"query:filter"`
	Fields *resttype.StringList `api:"query:fields"`
	Limit  int                  `api:"query:limit" default:"1000"`
}

func (a *API) GetOrganizationIDGroupRootPerson(ctx context.Context, meta GetOrganizationIDGroupRootPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, &meta.Group.ID, meta.Filter, (*[]string)(meta.Fields), meta.Limit)
	if err != nil {
		return output, err
	}

	output.Data.Persons = persons
	return output, nil
}
