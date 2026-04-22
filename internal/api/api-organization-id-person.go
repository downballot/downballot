package api

import (
	"context"
	"net/http"

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
	output.Data.Persons = persons
	return output, nil
}

type GetOrganizationIDPersonIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	VoterID string               `api:"path:voter_id"`
	_       string               `api:"httppath:/organization/{organization_id}/person/{voter_id}"`
	_       string               `api:"doc" description:"List the persons."`
	_       string               `api:"notes" description:"This lists the persons."`
	Fields  *resttype.StringList `api:"query:fields"`
}

func (a *API) GetOrganizationIDPersonID(ctx context.Context, meta GetOrganizationIDPersonIDMetadata) (output downballotapi.Envelope[downballotapi.GetPersonResponse], err error) {
	filter := "voter_id = " + meta.VoterID
	limit := 1
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, &filter, (*[]string)(meta.Fields), limit)
	if err != nil {
		return output, err
	}
	if len(persons) == 0 {
		return output, restfulwrapper.NewAPIResponseError(http.StatusNotFound, "")
	}

	output.Data.Person = persons[0]
	return output, nil
}
