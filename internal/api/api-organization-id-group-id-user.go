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

type GetOrganizationIDGroupIDUserMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_ string `api:"httppath:/organization/{organization_id}/group/{group_id}/user"`
	_ string `api:"doc" description:"Get the users in the group."`
	_ string `api:"notes" description:"This gets the users in the group."`
}

func (a *API) GetOrganizationIDGroupIDUser(ctx context.Context, meta GetOrganizationIDGroupIDUserMetadata) (output downballotapi.Envelope[downballotapi.ListGroupUsersResponse], err error) {
	var user []*schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("id IN (SELECT user_id FROM user_group_map WHERE group_id = ?)", meta.Group.ID).
		Find(&user).
		Error
	if err != nil {
		return output, err
	}

	userIDToGroupUserMapMap := map[uint64]schema.UserGroupMap{}
	{
		var groupUserMaps []schema.UserGroupMap
		err = meta.DB.Session(&gorm.Session{}).
			Where("group_id = ?", meta.Group.ID).
			Find(&groupUserMaps).
			Error
		if err != nil {
			return output, err
		}
		for _, groupUserMap := range groupUserMaps {
			userIDToGroupUserMapMap[groupUserMap.UserID] = groupUserMap
		}
	}

	output.Message = "OK"
	output.Success = true
	output.Data.GroupUsers = make([]*downballotapi.GroupUser, 0, len(userIDToGroupUserMapMap))
	for _, groupUser := range user {
		output.Data.GroupUsers = append(output.Data.GroupUsers, &downballotapi.GroupUser{
			User: downballotapi.User{
				ID:       fmt.Sprintf("%d", groupUser.ID),
				Name:     groupUser.Name,
				Username: groupUser.Username,
			},
			Owner: userIDToGroupUserMapMap[groupUser.ID].Owner,
		})
	}
	return output, nil
}
