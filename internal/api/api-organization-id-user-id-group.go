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

type GetOrganizationIDUserIDGroupMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasUser
	_ string `api:"httppath:/organization/{organization_id}/user/{user_id}/group"`
	_ string `api:"doc" description:"Get the groups the user is in."`
	_ string `api:"notes" description:"This gets the groups the user is in."`
}

func (a *API) GetOrganizationIDUserIDGroup(ctx context.Context, meta GetOrganizationIDUserIDGroupMetadata) (output downballotapi.Envelope[downballotapi.ListUserGroupsResponse], err error) {
	var groups []*schema.Group
	err = meta.DB.Session(&gorm.Session{}).
		Where("organization_id = ?", meta.Organization.ID).
		Where("id IN (SELECT group_id FROM user_group_map WHERE user_id = ?)", meta.User.ID).
		Find(&groups).
		Error
	if err != nil {
		return output, err
	}

	userIDToGroupMapMap := map[uint64]schema.UserGroupMap{}
	{
		var userGroupMaps []schema.UserGroupMap
		err = meta.DB.Session(&gorm.Session{}).
			Where("user_id = ?", meta.User.ID).
			Find(&userGroupMaps).
			Error
		if err != nil {
			return output, err
		}
		for _, userGroupMap := range userGroupMaps {
			userIDToGroupMapMap[userGroupMap.GroupID] = userGroupMap
		}
	}

	output.Message = "OK"
	output.Success = true
	output.Data.UserGroups = make([]*downballotapi.UserGroup, 0, len(groups))
	for _, group := range groups {
		output.Data.UserGroups = append(output.Data.UserGroups, &downballotapi.UserGroup{
			Group: downballotapi.Group{
				ID:     fmt.Sprintf("%d", group.ID),
				Name:   group.Name,
				Filter: group.Filter,
			},
			Owner: userIDToGroupMapMap[group.ID].Owner,
		})
	}
	return output, nil
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

	if meta.Body.Owner {
		// TODO: Make sure that this user can add the user as an owner.
	}

	userGroupMapping := schema.UserGroupMap{
		UserID:  meta.User.ID,
		GroupID: group.ID,
		Owner:   meta.Body.Owner,
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
