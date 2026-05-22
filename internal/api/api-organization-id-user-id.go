package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type hasUser struct {
	UserID string      `api:"path:user_id" description:"The user ID"`
	User   schema.User `api:"database.query:where:id = ? AND id IN (SELECT DISTINCT user_id FROM user_organization_map WHERE organization_id IN (SELECT id FROM organization)),UserID"`
}

type GetOrganizationIDUserIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
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

type PostOrganizationIDUserIDGroupMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasUser
	_    string                              `api:"httppath:/organization/{organization_id}/user/{user_id}/group"`
	_    string                              `api:"doc" description:"Add a user to a group."`
	_    string                              `api:"notes" description:"This adds a user to a group."`
	Body downballotapi.AddUserToGroupRequest `api:"body"`
}

func (a *API) PostOrganizationIDUserIDGroup(ctx context.Context, meta PostOrganizationIDUserIDGroupMetadata) (output downballotapi.Envelope[downballotapi.AddUserToGroupResponse], err error) {
	if meta.Body.GroupID == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing group_id"))
	}
	var group *schema.Group
	{
		groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID)
		if err != nil {
			return output, err
		}
		for _, g := range groups {
			if fmt.Sprintf("%v", g.ID) == meta.Body.GroupID {
				group = g
				break
			}
		}
	}
	if group == nil {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid group_id"))
	}

	userGroupMapping := schema.UserGroupMap{
		UserID:  meta.User.ID,
		GroupID: group.ID,
	}

	output.Message = "OK"
	output.Success = true
	output.Data.GroupID = fmt.Sprintf("%d", group.ID)
	err = meta.DB.Transaction(func(tx *gorm.DB) error {
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
