package api

import (
	"context"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/threatmate/restfulwrapper"
)

type GetOrganizationIDPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_      string               `api:"httppath:/organization/{organization_id}/person"`
	_      string               `api:"produces:application/json,text/csv"`
	_      string               `api:"doc" description:"List the persons."`
	_      string               `api:"notes" description:"This lists the persons."`
	Filter *string              `api:"query:filter"`
	Fields *resttype.StringList `api:"query:fields"`
	Limit  int                  `api:"query:limit" default:"1000"`
}

func (a *API) GetOrganizationIDPerson(ctx context.Context, meta GetOrganizationIDPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, meta.Filter, (*[]string)(meta.Fields), meta.Limit)
	if err != nil {
		return output, err
	}
	output.Message = "OK"
	output.Success = true
	output.Data.Persons = persons
	return output, nil
}
