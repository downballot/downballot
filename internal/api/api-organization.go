package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type GetOrganizationMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	_    string  `api:"httppath:/organization"`
	_    string  `api:"doc" description:"List the organizations."`
	_    string  `api:"notes" description:"This lists the organizations."`
	Name *string `api:"query:name"`
}

func (a *API) GetOrganization(ctx context.Context, meta GetOrganizationMetadata) (output downballotapi.Envelope[downballotapi.ListOrganizationsResponse], err error) {
	var organizations []*schema.Organization
	query := meta.DB.Session(&gorm.Session{})
	if meta.Name != nil {
		query = query.Where("name = ?", *meta.Name)
	}
	err = query.
		Find(&organizations).
		Error
	if err != nil {
		return output, err
	}

	output.Data.Organizations = []*downballotapi.Organization{}
	for _, organization := range organizations {
		o := &downballotapi.Organization{
			ID:   fmt.Sprintf("%d", organization.ID),
			Name: organization.Name,
		}
		output.Data.Organizations = append(output.Data.Organizations, o)
	}
	return output, nil
}

type PostOrganizationMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	_    string                                    `api:"httppath:/organization"`
	_    string                                    `api:"doc" description:"Register a new organization."`
	_    string                                    `api:"notes" description:"This registers a new organization."`
	Body downballotapi.RegisterOrganizationRequest `api:"body"`
}

func (a *API) PostOrganization(ctx context.Context, meta PostOrganizationMetadata) (output downballotapi.Envelope[downballotapi.RegisterOrganizationResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}
	if meta.Body.OwnerID == "" {
		if meta.CurrentUser.ID == 0 {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing owner_id"))
		}
		meta.Body.OwnerID = fmt.Sprintf("%d", meta.CurrentUser.ID)
		slog.InfoContext(ctx, fmt.Sprintf("No owner ID given; using current user: %s", meta.Body.OwnerID))
	}
	if meta.Body.OwnerID != "" && meta.CurrentUser.ID != 0 {
		if fmt.Sprintf("%d", meta.CurrentUser.ID) != meta.Body.OwnerID {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("mismatched owner_id"))
		}
	}

	var owner schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("id = ?", meta.Body.OwnerID).
		First(&owner).
		Error
	if err != nil {
		return output, err
	}

	organization := schema.Organization{
		Name: meta.Body.Name,
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&organization).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.ID = fmt.Sprintf("%d", organization.ID)
		output.Data.Name = organization.Name

		userOrganizationMapping := schema.UserOrganizationMap{
			UserID:         owner.ID,
			OrganizationID: organization.ID,
			Owner:          true,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userOrganizationMapping).
			Error
		if err != nil {
			return err
		}

		group := schema.Group{
			OrganizationID: organization.ID,
			Name:           "Root",
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&group).
			Error
		if err != nil {
			return err
		}

		userGroupMapping := schema.UserGroupMap{
			UserID:  owner.ID,
			GroupID: group.ID,
			Owner:   true,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userGroupMapping).
			Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}
