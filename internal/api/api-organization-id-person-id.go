package api

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

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

	person := persons[0]

	personID, err := strconv.ParseUint(person.ID, 10, 64)
	if err != nil {
		return output, err
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

type GetOrganizationIDPersonIDAuditMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	VoterID string               `api:"path:voter_id"`
	_       string               `api:"httppath:/organization/{organization_id}/person/{voter_id}/audit"`
	_       string               `api:"doc" description:"Get the person audit."`
	_       string               `api:"notes" description:"This gets the person audit with the given voter ID."`
	Fields  *resttype.StringList `api:"query:fields"`
}

func (a *API) GetOrganizationIDPersonIDAudit(ctx context.Context, meta GetOrganizationIDPersonIDAuditMetadata) (output downballotapi.Envelope[downballotapi.ListPersonAuditsResponse], err error) {
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

	var audits []*schema.PersonAudit
	query := meta.DB.Session(&gorm.Session{}).
		Where("person_id = ?", persons[0].ID)
	if meta.Fields != nil {
		fieldDefintionIDs := []uint64{}
		for _, field := range *meta.Fields {
			fieldDefinition := fieldDefinitionByNameMap[field]
			if fieldDefinition == nil {
				return output, fmt.Errorf("unknown field: %s", field)
			}
			fieldDefintionIDs = append(fieldDefintionIDs, fieldDefinition.ID)
		}
		query = query.Where("person_field_definition_id IN (?)", fieldDefintionIDs)
	}
	err = query.
		Find(&audits).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find audits: %w", err)
	}

	userIDToUsernameMap := map[uint64]string{}
	{
		userIDMap := map[uint64]bool{}
		for _, audit := range audits {
			userIDMap[audit.UserID] = true
		}
		var users []*schema.User
		err = meta.DB.Session(&gorm.Session{}).
			Where("id IN (?)", slices.Collect(maps.Keys(userIDMap))).
			Find(&users).
			Error
		if err != nil {
			return output, fmt.Errorf("could not find users: %w", err)
		}
		for userID := range userIDMap {
			userIDToUsernameMap[userID] = "user #" + fmt.Sprintf("%d", userID)
		}
		for _, user := range users {
			userIDToUsernameMap[user.ID] = user.Username
		}
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Audits = []*downballotapi.PersonAudit{}
	for _, audit := range audits {
		fieldDefinition := fieldDefinitionByIDMap[audit.PersonFieldDefinitionID]
		if fieldDefinition == nil {
			return output, fmt.Errorf("unknown field definition: %d", audit.PersonFieldDefinitionID)
		}

		output.Data.Audits = append(output.Data.Audits, &downballotapi.PersonAudit{
			ID:        fmt.Sprintf("%d", audit.ID),
			Username:  userIDToUsernameMap[audit.UserID],
			VoterID:   meta.VoterID,
			Timestamp: resttype.DateTime(audit.Timestamp),
			Field:     fieldDefinition.Name,
			OldValue:  audit.OldValue,
			NewValue:  audit.NewValue,
		})
	}
	return output, nil
}
