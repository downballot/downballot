package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type hasPersonField struct {
	PersonFieldID string                       `api:"path:person_field_id" description:"The person field ID"`
	PersonField   schema.PersonFieldDefinition `api:"database.query:where:id = ?,PersonFieldID"`
}

type PatchOrganizationIDPersonFieldIDMetadata struct {
	restfulwrapper.HTTPMethodPATCH
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasPersonField
	_    string                                `api:"httppath:/organization/{organization_id}/person-field/{person_field_id}"`
	_    string                                `api:"doc" description:"Add a person field."`
	_    string                                `api:"notes" description:"This adds a person field."`
	Body downballotapi.PatchPersonFieldRequest `api:"body"`
}

func (a *API) PostOrganizationIDPersonFieldID(ctx context.Context, meta PatchOrganizationIDPersonFieldIDMetadata) (output downballotapi.Envelope[downballotapi.PatchPersonFieldResponse], err error) {
	updateMap := map[string]any{}
	if meta.Body.Name != nil {
		updateMap["name"] = *meta.Body.Name
	}
	if meta.Body.Type != nil {
		updateMap["type"] = *meta.Body.Type
	}
	if meta.Body.AllowEmpty != nil {
		updateMap["allow_empty"] = *meta.Body.AllowEmpty
	}
	if meta.Body.AllowedValues != nil {
		updateMap["allowed_values"] = resttype.StringList(meta.Body.AllowedValues)
	}
	if meta.Body.AllowedRegex != nil {
		updateMap["allowed_regex"] = *meta.Body.AllowedRegex
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Model(&schema.PersonFieldDefinition{}).
			Where("id = ?", meta.PersonField.ID).
			Updates(updateMap).
			Error
		if err != nil {
			return err
		}

		var personField schema.PersonFieldDefinition
		err = tx.Session(&gorm.Session{}).
			Where("id = ?", meta.PersonField.ID).
			First(&personField).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.PersonField = downballotapi.PersonField{
			ID:            fmt.Sprintf("%d", personField.ID),
			Name:          personField.Name,
			Type:          personField.Type,
			AllowEmpty:    personField.AllowEmpty,
			AllowedValues: personField.AllowedValues,
			AllowedRegex:  personField.AllowedRegex,
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}
