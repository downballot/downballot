package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type PostOrganizationIDUserIDGroupMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	_              string                              `api:"httppath:/organization/{organization_id}/user/{user_id}/group"`
	_              string                              `api:"doc" description:"Add a user to a group."`
	_              string                              `api:"notes" description:"This adds a user to a group."`
	Body           downballotapi.AddUserToGroupRequest `api:"body"`
	OrganizationID string                              `api:"path:organization_id"`
	UserID         string                              `api:"path:user_id"`
}

func (a *API) PostOrganizationIDUserIDGroup(ctx context.Context, meta PostOrganizationIDUserIDGroupMetadata) (output downballotapi.Envelope[downballotapi.AddUserToGroupResponse], err error) {
	organization, err := getOrganizationForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID)
	if err != nil {
		return output, err
	}
	if organization == nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	var user *schema.User
	{
		users, err := getUsersForOrganization(a.App.DB(), organization.ID, map[string]any{"id": meta.UserID})
		if err != nil {
			return output, err
		}
		if len(users) > 0 {
			user = users[0]
		}
	}
	if user == nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	if meta.Body.GroupID == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing group_id"))
	}
	var group *schema.Group
	{
		groups, err := getGroupsForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID, nil)
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
		UserID:  user.ID,
		GroupID: group.ID,
	}

	output.Data.GroupID = fmt.Sprintf("%d", group.ID)
	err = a.App.DB().Transaction(func(tx *gorm.DB) error {
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
