package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/tekkamanendless/restfulwrapper"
	"gorm.io/gorm"
)

type GetOrganizationIDPersonFieldMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionPersonFieldDefinitionRead
	_ string `api:"httppath:/organization/{organization_id}/person-field"`
	_ string `api:"doc" description:"List the person fields."`
	_ string `api:"notes" description:"This lists the person fields."`
}

func (a *API) GetOrganizationIDPersonField(ctx context.Context, meta GetOrganizationIDPersonFieldMetadata) (output downballotapi.Envelope[downballotapi.ListPersonFieldsResponse], err error) {
	var personFields []*schema.PersonFieldDefinition
	err = meta.DB.Session(&gorm.Session{}).
		Where("organization_id = ?", meta.Organization.ID).
		Find(&personFields).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find person fields: %w", err)
	}

	output.Message = "OK"
	output.Success = true
	output.Data.PersonFields = []*downballotapi.PersonField{}
	for _, personField := range personFields {
		u := &downballotapi.PersonField{
			ID:            fmt.Sprintf("%d", personField.ID),
			Name:          personField.Name,
			Type:          downballotapi.PersonFieldDefinitionType(personField.Type),
			AllowEmpty:    personField.AllowEmpty,
			AllowedValues: personField.AllowedValues,
			AllowedRegex:  personField.AllowedRegex,
		}
		output.Data.PersonFields = append(output.Data.PersonFields, u)
	}
	return output, nil
}

type PostOrganizationIDPersonFieldMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionPersonFieldDefinitionCreate
	_    string                                 `api:"httppath:/organization/{organization_id}/person-field"`
	_    string                                 `api:"doc" description:"Add a person field."`
	_    string                                 `api:"notes" description:"This adds a person field."`
	Body downballotapi.CreatePersonFieldRequest `api:"body"`
}

func (a *API) PostOrganizationIDPersonField(ctx context.Context, meta PostOrganizationIDPersonFieldMetadata) (output downballotapi.Envelope[downballotapi.CreatePersonFieldResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}

	switch schema.PersonFieldDefinitionType(meta.Body.Type) {
	case schema.PersonFieldDefinitionTypeBoolean:
	case schema.PersonFieldDefinitionTypeCoordinates:
	case schema.PersonFieldDefinitionTypeDate:
	case schema.PersonFieldDefinitionTypeEnum:
	case schema.PersonFieldDefinitionTypeInteger:
	case schema.PersonFieldDefinitionTypeSet:
	case schema.PersonFieldDefinitionTypeString:
	default:
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("unknown type: %q", meta.Body.Type))
	}

	personField := schema.PersonFieldDefinition{
		OrganizationID: meta.Organization.ID,
		Name:           meta.Body.Name,
		Type:           schema.PersonFieldDefinitionType(meta.Body.Type),
		AllowEmpty:     meta.Body.AllowEmpty,
		AllowedValues:  meta.Body.AllowedValues,
		AllowedRegex:   meta.Body.AllowedRegex,
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&personField).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.PersonField = downballotapi.PersonField{
			ID:            fmt.Sprintf("%d", personField.ID),
			Name:          personField.Name,
			Type:          downballotapi.PersonFieldDefinitionType(personField.Type),
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
