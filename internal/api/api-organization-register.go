package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/sirupsen/logrus"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type PostOrganizationRegisterMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	_    string                                    `api:"httppath:/organization/register"`
	_    string                                    `api:"doc" description:"Register a new organization."`
	_    string                                    `api:"notes" description:"This registers a new organization."`
	Body downballotapi.RegisterOrganizationRequest `api:"body"`
}

func (a *API) PostOrganizationRegister(ctx context.Context, meta PostOrganizationRegisterMetadata) (output downballotapi.Envelope[downballotapi.RegisterOrganizationResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}
	if meta.Body.OwnerID == "" {
		if meta.CurrentUser.ID == "0" {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing owner_id"))
		}
		meta.Body.OwnerID = meta.CurrentUser.ID
		logrus.WithContext(ctx).Infof("No owner ID given; using current user: %s", meta.Body.OwnerID)
	}
	if meta.Body.OwnerID != "" && meta.CurrentUser.ID != "0" {
		if meta.CurrentUser.ID != meta.Body.OwnerID {
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

		output.Data.ID = fmt.Sprintf("%d", organization.ID)
		output.Data.Name = organization.Name

		userOrganizationMapping := schema.UserOrganizationMap{
			UserID:         owner.ID,
			OrganizationID: organization.ID,
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
