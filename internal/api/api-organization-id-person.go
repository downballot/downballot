package api

import (
	"context"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/threatmate/restfulwrapper"
)

type GetOrganizationIDPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_      string  `api:"httppath:/organization/{organization_id}/person"`
	_      string  `api:"doc" description:"List the persons."`
	_      string  `api:"notes" description:"This lists the persons."`
	Filter *string `api:"query:filter"`
	Fields *string `api:"query:fields"`
	Limit  int     `api:"query:limit" default:"1000"`
}

func (a *API) GetOrganizationIDPerson(ctx context.Context, meta GetOrganizationIDPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	var returnFields []string
	if meta.Fields != nil {
		returnFields = strings.Split(*meta.Fields, ",")
	}
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, meta.Filter, returnFields, meta.Limit)
	if err != nil {
		return output, err
	}
	output.Data.Persons = persons
	return output, nil
}
