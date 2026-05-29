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

type hasFilter struct {
	FilterID string        `api:"path:filter_id" description:"The filter ID"`
	Filter   schema.Filter `api:"database.query:where:id = ? AND organization_id IN (SELECT id FROM organization) AND (user_id IS NULL OR user_id = ?),FilterID,CurrentUser.ID"`
}

type DeleteOrganizationIDFilterIDMetadata struct {
	restfulwrapper.HTTPMethodDELETE
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasFilter
	_ string `api:"httppath:/organization/{organization_id}/filter/{filter_id}"`
	_ string `api:"doc" description:"Delete the filter."`
	_ string `api:"notes" description:"This deletes the filter."`
}

func (a *API) DeleteOrganizationIDFilterID(ctx context.Context, meta DeleteOrganizationIDFilterIDMetadata) error {
	err := meta.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Session(&gorm.Session{NewDB: true}).
			Where("id = ?", meta.Filter.ID).
			Delete(&schema.Filter{}).
			Error
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

type GetOrganizationIDFilterIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasFilter
	_ string `api:"httppath:/organization/{organization_id}/filter/{filter_id}"`
	_ string `api:"doc" description:"Get the filter."`
	_ string `api:"notes" description:"This gets the filter."`
}

func (a *API) GetOrganizationIDFilterID(ctx context.Context, meta GetOrganizationIDFilterIDMetadata) (output downballotapi.Envelope[downballotapi.GetFilterResponse], err error) {
	o := &downballotapi.Filter{
		ID:          fmt.Sprintf("%d", meta.Filter.ID),
		Name:        meta.Filter.Name,
		Description: meta.Filter.Description,
		Filter:      meta.Filter.Filter,
	}
	if meta.Filter.UserID != nil {
		o.UserID = new(string)
		*o.UserID = fmt.Sprintf("%d", *meta.Filter.UserID)
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Filter = o
	return output, nil
}

type PatchOrganizationIDFilterIDMetadata struct {
	restfulwrapper.HTTPMethodPATCH
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasFilter
	_    string                           `api:"httppath:/organization/{organization_id}/filter/{filter_id}"`
	_    string                           `api:"doc" description:"Patch the filter."`
	_    string                           `api:"notes" description:"This patches the filter."`
	Body downballotapi.PatchFilterRequest `api:"body"`
}

func (a *API) PatchOrganizationIDFilterID(ctx context.Context, meta PatchOrganizationIDFilterIDMetadata) (output downballotapi.Envelope[downballotapi.GetFilterResponse], err error) {
	updateMap := map[string]any{}
	if meta.Body.Name != nil {
		updateMap["name"] = *meta.Body.Name
	}
	if meta.Body.Description != nil {
		updateMap["description"] = *meta.Body.Description
	}
	if meta.Body.Filter != nil {
		updateMap["filter"] = *meta.Body.Filter
	}
	// TODO: Consider allowing the user to change the user_id.

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Model(&schema.Filter{}).
			Where("id = ?", meta.Filter.ID).
			Updates(updateMap).
			Error
		if err != nil {
			return err
		}

		var filter schema.Filter
		err = tx.Session(&gorm.Session{}).
			Where("id = ?", meta.Filter.ID).
			First(&filter).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.Filter = &downballotapi.Filter{
			ID:          fmt.Sprintf("%d", filter.ID),
			Name:        filter.Name,
			Description: filter.Description,
			Filter:      filter.Filter,
		}
		if filter.UserID != nil {
			output.Data.Filter.UserID = new(string)
			*output.Data.Filter.UserID = fmt.Sprintf("%d", *filter.UserID)
		}
		return nil
	})
	if err != nil {
		return output, err
	}
	return output, nil
}
