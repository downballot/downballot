package api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/tekkamanendless/restfulwrapper"
	"gorm.io/gorm"
)

type PostOrganizationIDFilterMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_    string                            `api:"httppath:/organization/{organization_id}/filter"`
	_    string                            `api:"doc" description:"Create a new filter."`
	_    string                            `api:"notes" description:"This creates a new filter."`
	Body downballotapi.CreateFilterRequest `api:"body"`
}

func (a *API) PostOrganizationIDFilter(ctx context.Context, meta PostOrganizationIDFilterMetadata) (output downballotapi.Envelope[downballotapi.CreateFilterResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}

	filter := schema.Filter{
		OrganizationID: meta.Organization.ID,
		Name:           meta.Body.Name,
		Description:    meta.Body.Description,
		Filter:         meta.Body.Filter,
	}

	if meta.Body.UserID != nil {
		if *meta.Body.UserID != fmt.Sprintf("%d", meta.CurrentUser.ID) {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("mismatched user_id")) // TODO: Consider allowing admins to create filters for other users.
		}
		v, err := strconv.ParseUint(*meta.Body.UserID, 10, 64)
		if err != nil {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid user_id"))
		}
		filter.UserID = &v
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&filter).
			Error
		if err != nil {
			return err
		}

		output.Data.ID = fmt.Sprintf("%d", filter.ID)
		output.Data.Name = filter.Name
		output.Data.Description = filter.Description
		output.Data.Filter = meta.Body.Filter
		if filter.UserID != nil {
			output.Data.UserID = new(string)
			*output.Data.UserID = fmt.Sprintf("%d", *filter.UserID)
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}

type GetOrganizationIDFilterMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_      string  `api:"httppath:/organization/{organization_id}/filter"`
	_      string  `api:"doc" description:"List the filters."`
	_      string  `api:"notes" description:"This lists the filters."`
	Name   *string `api:"query:name"`
	UserID *string `api:"query:user_id"`
}

func (a *API) GetOrganizationIDFilter(ctx context.Context, meta GetOrganizationIDFilterMetadata) (output downballotapi.Envelope[downballotapi.ListFiltersResponse], err error) {
	var filters []*schema.Filter
	query := meta.DB.Session(&gorm.Session{}).
		Where("organization_id = ?", meta.Organization.ID).
		Where("user_id IS NULL OR user_id = ?", meta.CurrentUser.ID)
	if meta.Name != nil {
		query = query.Where("name = ?", *meta.Name)
	}
	if meta.UserID != nil {
		if *meta.UserID == "self" {
			query = query.Where("user_id = ?", meta.CurrentUser.ID)
		} else {
			query = query.Where("user_id = ?", *meta.UserID)
		}
	}
	err = query.
		Find(&filters).
		Error
	if err != nil {
		return output, err
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Filters = []*downballotapi.Filter{}
	for _, filter := range filters {
		o := &downballotapi.Filter{
			ID:          fmt.Sprintf("%d", filter.ID),
			Name:        filter.Name,
			Description: filter.Description,
			Filter:      filter.Filter,
		}
		if filter.UserID != nil {
			o.UserID = new(string)
			*o.UserID = fmt.Sprintf("%d", *filter.UserID)
		}
		output.Data.Filters = append(output.Data.Filters, o)
	}
	return output, nil
}
