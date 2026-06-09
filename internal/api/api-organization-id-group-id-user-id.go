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

type hasGroupUser struct {
	UserID       string              `api:"path:user_id" description:"The user ID"`
	UserGroupMap schema.UserGroupMap `api:"database.query:where:user_id = ? AND group_id = ?,UserID,GroupID"`
	User         schema.User         `api:"database.query:where:id = ?,UserID"`
}

type GetOrganizationIDGroupIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionGroupRead
	downballotwrapper.RequirePermissionGroupUserRead
	hasGroup
	hasGroupUser
	_ string `api:"httppath:/organization/{organization_id}/group/{group_id}/user/{user_id}"`
	_ string `api:"doc" description:"Get the user in the group."`
	_ string `api:"notes" description:"This gets the user in the group."`
}

func (a *API) GetOrganizationIDGroupIDUserID(ctx context.Context, meta GetOrganizationIDGroupIDUserIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupUserResponse], err error) {
	output.Message = "OK"
	output.Success = true
	output.Data.GroupUser = downballotapi.GroupUser{
		User: downballotapi.User{
			ID:       fmt.Sprintf("%d", meta.User.ID),
			Name:     meta.User.Name,
			Username: meta.User.Username,
		},
		Owner: meta.UserGroupMap.Owner,
	}
	return output, nil
}

type PatchOrganizationIDGroupIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodPATCH
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionGroupRead
	downballotwrapper.RequirePermissionGroupUserUpdate
	hasGroup
	hasGroupUser
	_    string                              `api:"httppath:/organization/{organization_id}/group/{group_id}/user/{user_id}"`
	_    string                              `api:"doc" description:"Patch the user in the group."`
	_    string                              `api:"notes" description:"This patches the user in the group."`
	Body downballotapi.PatchGroupUserRequest `api:"body"`
}

func (a *API) PatchOrganizationIDGroupIDUserID(ctx context.Context, meta PatchOrganizationIDGroupIDUserIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupUserResponse], err error) {
	updateMap := map[string]any{}
	if meta.Body.Owner != nil {
		updateMap["owner"] = *meta.Body.Owner
	}

	// TODO: Ensure that the user cannot change the owner of the group.

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Session(&gorm.Session{NewDB: true}).
			Model(&schema.UserGroupMap{}).
			Where("user_id = ?", meta.User.ID).
			Where("group_id = ?", meta.Group.ID).
			Updates(updateMap).
			Error
		if err != nil {
			return err
		}

		var userGroupMap schema.UserGroupMap
		err = tx.Session(&gorm.Session{}).
			Where("user_id = ?", meta.User.ID).
			Where("group_id = ?", meta.Group.ID).
			First(&userGroupMap).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.GroupUser = downballotapi.GroupUser{
			User: downballotapi.User{
				ID:       fmt.Sprintf("%d", meta.User.ID),
				Name:     meta.User.Name,
				Username: meta.User.Username,
			},
			Owner: userGroupMap.Owner,
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}

type DeleteOrganizationIDGroupIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodDELETE
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionGroupRead
	downballotwrapper.RequirePermissionGroupUserDelete
	hasGroup
	hasGroupUser
	_ string `api:"httppath:/organization/{organization_id}/group/{group_id}/user/{user_id}"`
	_ string `api:"doc" description:"Delete the user from the group."`
	_ string `api:"notes" description:"This deletes the user from the group."`
}

func (a *API) DeleteOrganizationIDGroupIDUserID(ctx context.Context, meta DeleteOrganizationIDGroupIDUserIDMetadata) error {
	err := meta.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Session(&gorm.Session{NewDB: true}).
			Where("user_id = ?", meta.User.ID).
			Where("group_id = ?", meta.Group.ID).
			Delete(&schema.UserGroupMap{}).
			Error
		return err
	})
	if err != nil {
		return err
	}
	return nil
}
