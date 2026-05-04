package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
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

type GetOrganizationIDPersonIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	VoterID string               `api:"path:voter_id"`
	_       string               `api:"httppath:/organization/{organization_id}/person/{voter_id}"`
	_       string               `api:"doc" description:"Get the person."`
	_       string               `api:"notes" description:"This gets the person with the given voter ID."`
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

	output.Message = "OK"
	output.Success = true
	output.Data.Person = persons[0]
	return output, nil
}

type PatchOrganizationIDPersonIDMetadata struct {
	restfulwrapper.HTTPMethodPATCH
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	VoterID string                           `api:"path:voter_id"`
	_       string                           `api:"httppath:/organization/{organization_id}/person/{voter_id}"`
	_       string                           `api:"doc" description:"Update the person."`
	_       string                           `api:"notes" description:"This updates the person."`
	Body    downballotapi.PatchPersonRequest `api:"body"`
}

func (a *API) PatchOrganizationIDPersonID(ctx context.Context, meta PatchOrganizationIDPersonIDMetadata) (output downballotapi.Envelope[downballotapi.GetPersonResponse], err error) {
	filter := "voter_id = " + meta.VoterID
	limit := 1
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, &filter, nil /*no fields*/, limit)
	if err != nil {
		return output, err
	}
	if len(persons) == 0 {
		return output, restfulwrapper.NewAPIResponseError(http.StatusNotFound, "")
	}

	fieldDefinitionByIDMap := map[uint64]*schema.PersonFieldDefinition{}
	fieldDefinitionByNameMap := map[string]*schema.PersonFieldDefinition{}
	{
		var fieldDefinitions []*schema.PersonFieldDefinition
		err = meta.DB.Session(&gorm.Session{}).
			Where("organization_id = ?", meta.OrganizationID).
			Find(&fieldDefinitions).
			Error
		if err != nil {
			return output, fmt.Errorf("could not find field definitions: %w", err)
		}
		for _, fieldDefinition := range fieldDefinitions {
			fieldDefinitionByIDMap[fieldDefinition.ID] = fieldDefinition
			fieldDefinitionByNameMap[fieldDefinition.Name] = fieldDefinition
		}
	}

	{
		person := persons[0]

		personID, err := strconv.ParseUint(person.ID, 10, 64)
		if err != nil {
			return output, err
		}

		err = meta.DB.Transaction(func(tx *gorm.DB) error {
			for field, value := range meta.Body.Fields {
				fieldDefinition := fieldDefinitionByNameMap[field]
				if fieldDefinition == nil {
					return fmt.Errorf("unknown field: %s", field)
				}

				if value == nil {
					err := tx.Session(&gorm.Session{}).
						Where("person_id = ?", personID).
						Where("person_field_definition_id = ?", fieldDefinition.ID).
						Delete(&schema.PersonField{}).
						Error
					if err != nil {
						return err
					}
				} else {
					var fields []*schema.PersonField
					err := tx.Session(&gorm.Session{}).
						Where("person_id = ?", personID).
						Where("person_field_definition_id = ?", fieldDefinition.ID).
						Find(&fields).
						Error
					if err != nil {
						return err
					}

					if len(fields) == 0 {
						field := schema.PersonField{
							PersonID:                personID,
							PersonFieldDefinitionID: fieldDefinition.ID,
							Value:                   *value,
						}
						err := tx.Session(&gorm.Session{}).
							Create(&field).
							Error
						if err != nil {
							return err
						}
					} else {
						field := fields[0]
						err := tx.Session(&gorm.Session{}).
							Model(&schema.PersonField{}).
							Where("id = ?", field.ID).
							Update("value", *value).
							Error
						if err != nil {
							return err
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return output, err
		}
	}

	persons, err = filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, &filter, nil /*no fields*/, limit)
	if err != nil {
		return output, err
	}
	if len(persons) == 0 {
		return output, restfulwrapper.NewAPIResponseError(http.StatusNotFound, "")
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Person = persons[0]
	return output, nil
}
