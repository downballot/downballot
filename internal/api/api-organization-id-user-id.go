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

type hasUser struct {
	UserID              string                     `api:"path:user_id" description:"The user ID"`
	UserOrganizationMap schema.UserOrganizationMap `api:"database.query:where:user_id = ? AND organization_id = ?,UserID,OrganizationID"`
	User                schema.User                `api:"database.query:where:id = ?,UserID"`
}

type GetOrganizationIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionOrganizationUserRead
	hasUser
	_ string `api:"httppath:/organization/{organization_id}/user/{user_id}"`
	_ string `api:"doc" description:"Get the user."`
	_ string `api:"notes" description:"This gets the user."`
}

func (a *API) GetOrganizationIDUserID(ctx context.Context, meta GetOrganizationIDUserIDMetadata) (output downballotapi.Envelope[downballotapi.GetUserResponse], err error) {
	output.Message = "OK"
	output.Success = true
	output.Data.User = &downballotapi.User{
		ID:       fmt.Sprintf("%d", meta.User.ID),
		Name:     meta.User.Name,
		Username: meta.User.Username,
		Owner:    meta.UserOrganizationMap.Owner,
	}
	return output, nil
}

type DeleteOrganizationIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodDELETE
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasUser
	_ string `api:"httppath:/organization/{organization_id}/user/{user_id}"`
	_ string `api:"doc" description:"Delete the user from the organization."`
	_ string `api:"notes" description:"This deletes the user from the organization."`
}

func (a *API) DeleteOrganizationIDUserID(ctx context.Context, meta DeleteOrganizationIDUserIDMetadata) error {
	// TODO: Ensure that the user cannot delete herself from the organization.

	err := meta.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Session(&gorm.Session{NewDB: true}).
			Where("user_id = ?", meta.User.ID).
			Where("organization_id = ?", meta.Organization.ID).
			Delete(&schema.UserOrganizationMap{}).
			Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
