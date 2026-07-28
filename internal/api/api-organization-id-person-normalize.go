package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/normalize"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"github.com/tekkamanendless/restfulwrapper"
	"googlemaps.github.io/maps"
	"gorm.io/gorm"
)

type PostOrganizationIDPersonNormalizeMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionPersonUpdate
	_    string                                   `api:"httppath:/organization/{organization_id}/person/normalize"`
	_    string                                   `api:"doc" description:"Normalize the persons."`
	_    string                                   `api:"notes" description:"This normalizes the persons."`
	Body downballotapi.PostPersonNormalizeRequest `api:"body"`
}

func (a *API) PostOrganizationIDPersonNormalize(ctx context.Context, meta PostOrganizationIDPersonNormalizeMetadata) (output downballotapi.Envelope[downballotapi.PostPersonNormalizeResponse], err error) {
	output.Data.Pretend = meta.Body.Pretend
	output.Data.Persons = []*downballotapi.NormalizePerson{}

	// A group ID is required.
	if meta.Body.GroupID == "" {
		return output, restfulwrapper.NewAPIResponseError(http.StatusBadRequest, "group ID is required")
	}
	groupID, err := strconv.ParseUint(meta.Body.GroupID, 10, 64)
	if err != nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusBadRequest, fmt.Sprintf("invalid group ID: %s", err))
	}

	var filterString *string
	if meta.Body.Filter != "" {
		filterString = &meta.Body.Filter
	}

	fields := []string{
		"residential_address",
		"residential_address_development",
		"coordinates",
	}

	limit := 100000000 // No practical limit.

	persons, err := filterPersons(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, &groupID, filterString, &fields, limit)
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

	mapsClient, err := maps.NewClient(maps.WithAPIKey(a.googleMapsServerAPIKey))
	if err != nil {
		return output, fmt.Errorf("could not create Google Maps client: %w", err)
	}

	nccClient := &normalize.NewCastleCountyGIS{}

	for _, person := range persons {
		address := person.Fields["residential_address"]
		if address == "" {
			continue
		}

		normalizePerson := downballotapi.NormalizePerson{
			ID:        person.ID,
			VoterID:   person.VoterID,
			OldFields: map[string]*string{},
			NewFields: map[string]*string{},
		}
		for name, value := range person.Fields {
			normalizePerson.OldFields[name] = &value
			normalizePerson.NewFields[name] = &value
		}

		err := normalize.Location(ctx, mapsClient, nccClient, &normalizePerson)
		if err != nil {
			return output, fmt.Errorf("could not normalize location: %w", err)
		}

		for name, value := range normalizePerson.NewFields {
			if (normalizePerson.OldFields[name] == nil && value == nil) || (normalizePerson.OldFields[name] != nil && value != nil && *normalizePerson.OldFields[name] == *value) {
				delete(normalizePerson.OldFields, name)
				delete(normalizePerson.NewFields, name)
				continue
			}

			fieldDefinition := fieldDefinitionByNameMap[name]
			if fieldDefinition == nil {
				return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("unknown field: %s", name))
			}

			{
				err = fieldDefinition.Validate(*value)
				if err != nil {
					return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid value for field %s: %w", name, err))
				}
			}
		}

		output.Data.Persons = append(output.Data.Persons, &normalizePerson)
	}

	if !meta.Body.Pretend {
		err = meta.DB.Transaction(func(tx *gorm.DB) error {
			for _, person := range output.Data.Persons {
				personID, err := strconv.ParseUint(person.ID, 10, 64)
				if err != nil {
					return err
				}

				for field, value := range person.NewFields {
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
					if oldValue, ok := person.OldFields[field]; ok && oldValue != nil {
						audit.OldValue = new(string)
						*audit.OldValue = *oldValue
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

	output.Message = "OK"
	output.Success = true
	return output, nil
}
