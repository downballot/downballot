package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/filter"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"github.com/tekkamanendless/restfulwrapper"
	"gorm.io/gorm"
)

type PostOrganizationIDPersonUpdateMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionPersonUpdate
	_    string                                `api:"httppath:/organization/{organization_id}/person/update"`
	_    string                                `api:"doc" description:"Update the persons."`
	_    string                                `api:"notes" description:"This updates the persons."`
	Body downballotapi.PostPersonUpdateRequest `api:"body"`
}

func (a *API) PostOrganizationIDPersonUpdate(ctx context.Context, meta PostOrganizationIDPersonUpdateMetadata) (output downballotapi.Envelope[downballotapi.PostPersonUpdateResponse], err error) {
	output.Data.Persons = []*downballotapi.Person{}

	// If we weren't given any voter IDs, then don't do anything.
	if len(meta.Body.VoterIDs) == 0 {
		return output, nil
	}

	var filterString string
	{
		var voterIDs []string
		for _, voterID := range meta.Body.VoterIDs {
			voterIDs = append(voterIDs, filter.QuoteIfNecessary(voterID))
		}
		filterString = "voter_id = (" + strings.Join(voterIDs, ", ") + ")"
	}
	limit := len(meta.Body.VoterIDs)
	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, &filterString, nil /*no fields*/, limit)
	if err != nil {
		return output, err
	}

	// This is a map of every unique voter ID that was passed in.
	desiredVoterIDMap := map[string]bool{}
	for _, voterID := range meta.Body.VoterIDs {
		desiredVoterIDMap[voterID] = true
	}
	// This is a map of every unique voter ID that was found in the database.
	foundVoterIDMap := map[string]bool{}
	for _, person := range persons {
		foundVoterIDMap[person.VoterID] = true
	}

	// Make sure that every voter ID that was passed in was found in the database.
	for desiredVoterID := range desiredVoterIDMap {
		if !foundVoterIDMap[desiredVoterID] {
			return output, restfulwrapper.NewAPIResponseError(http.StatusBadRequest, fmt.Sprintf("voter ID not found: %s", desiredVoterID))
		}
	}
	// Make sure that every voter ID that was found in the database was passed in.
	for foundVoterID := range foundVoterIDMap {
		if !desiredVoterIDMap[foundVoterID] {
			return output, restfulwrapper.NewAPIResponseError(http.StatusBadRequest, fmt.Sprintf("voter ID not requested: %s", foundVoterID))
		}
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

	for field, value := range meta.Body.Fields {
		fieldDefinition := fieldDefinitionByNameMap[field]
		if fieldDefinition == nil {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("unknown field: %s", field))
		}

		if value != nil {
			err = fieldDefinition.Validate(*value)
			if err != nil {
				return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid value for field %s: %w", field, err))
			}
		}
	}

	{
		err = meta.DB.Transaction(func(tx *gorm.DB) error {
			for _, person := range persons {
				personID, err := strconv.ParseUint(person.ID, 10, 64)
				if err != nil {
					return err
				}

				for field, value := range meta.Body.Fields {
					fieldDefinition := fieldDefinitionByNameMap[field]
					if fieldDefinition == nil {
						// We should have already defended against this, but play it safe.
						return fmt.Errorf("unknown field: %s", field)
					}

					audit := schema.PersonAudit{
						PersonID:                personID,
						UserID:                  meta.CurrentUser.ID,
						PersonFieldDefinitionID: fieldDefinition.ID,
						Timestamp:               sqltype.DateTime(time.Now()),
					}

					// If the field had a value, then record its old value.
					if oldValue, ok := person.Fields[field]; ok {
						audit.OldValue = new(string)
						*audit.OldValue = oldValue
					}
					// If the field has a new value, then record its new value.
					if value != nil {
						audit.NewValue = value
					}

					if audit.OldValue == nil && audit.NewValue == nil {
						// If the field was added and deleted, then don't do anything.
						continue
					}
					if audit.OldValue != nil && audit.NewValue != nil && *audit.OldValue == *audit.NewValue {
						// If the field was not changed, then don't do anything.
						continue
					}

					if value == nil {
						err := tx.Session(&gorm.Session{}).
							Where("person_id = ?", personID).
							Where("person_field_definition_id = ?", fieldDefinition.ID).
							Delete(&schema.PersonField{}).
							Error
						if err != nil {
							return fmt.Errorf("could not delete field: %w", err)
						}
					} else {
						var fields []*schema.PersonField
						err := tx.Session(&gorm.Session{}).
							Where("person_id = ?", personID).
							Where("person_field_definition_id = ?", fieldDefinition.ID).
							Find(&fields).
							Error
						if err != nil {
							return fmt.Errorf("could not find fields: %w", err)
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
								return fmt.Errorf("could not create field: %w", err)
							}
						} else {
							field := fields[0]
							err := tx.Session(&gorm.Session{}).
								Model(&schema.PersonField{}).
								Where("id = ?", field.ID).
								Update("value", *value).
								Error
							if err != nil {
								return fmt.Errorf("could not update field: %w", err)
							}
						}
					}

					err := tx.Session(&gorm.Session{}).
						Create(&audit).
						Error
					if err != nil {
						return fmt.Errorf("could not create audit: %w", err)
					}
				}
			}
			return nil
		})
		if err != nil {
			return output, err
		}
	}

	persons, err = filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, nil /*no group ID*/, &filterString, nil /*no fields*/, limit)
	if err != nil {
		return output, err
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Persons = persons
	return output, nil
}
