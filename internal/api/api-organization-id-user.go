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

type GetOrganizationIDUserMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionOrganizationUserRead
	_ string `api:"httppath:/organization/{organization_id}/user"`
	_ string `api:"doc" description:"List the users."`
	_ string `api:"notes" description:"This lists the users."`
}

func (a *API) GetOrganizationIDUser(ctx context.Context, meta GetOrganizationIDUserMetadata) (output downballotapi.Envelope[downballotapi.ListUsersResponse], err error) {
	var users []*schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("id IN (SELECT user_id FROM user_organization_map WHERE organization_id = ?)", meta.Organization.ID).
		Find(&users).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find users: %w", err)
	}

	userIDToUserOrganizationMapMap := map[uint64]schema.UserOrganizationMap{}
	{
		var userOrganizationMaps []schema.UserOrganizationMap
		err = meta.DB.Session(&gorm.Session{}).
			Where("organization_id = ?", meta.Organization.ID).
			Find(&userOrganizationMaps).
			Error
		if err != nil {
			return output, fmt.Errorf("could not find user organization maps: %w", err)
		}
		for _, userOrganizationMap := range userOrganizationMaps {
			userIDToUserOrganizationMapMap[userOrganizationMap.UserID] = userOrganizationMap
		}
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Users = []*downballotapi.User{}
	for _, user := range users {
		u := &downballotapi.User{
			ID:       fmt.Sprintf("%d", user.ID),
			Name:     user.Name,
			Username: user.Username,
			Owner:    userIDToUserOrganizationMapMap[user.ID].Owner,
		}
		output.Data.Users = append(output.Data.Users, u)
	}
	return output, nil
}

type PostOrganizationIDUserMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_    string                                     `api:"httppath:/organization/{organization_id}/user"`
	_    string                                     `api:"doc" description:"Add a user to an organization."`
	_    string                                     `api:"notes" description:"This adds a user to an organization."`
	Body downballotapi.AddUserToOrganizationRequest `api:"body"`
}

func (a *API) PostOrganizationIDUser(ctx context.Context, meta PostOrganizationIDUserMetadata) (output downballotapi.Envelope[downballotapi.AddUserToOrganizationResponse], err error) {
	if meta.Body.Username == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing username"))
	}

	var users []*schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("username = ?", meta.Body.Username).
		Find(&users).
		Error
	if err != nil {
		return output, err
	}
	if len(users) == 0 {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid username"))
	}
	user := users[0]

	userOrganizationMapping := schema.UserOrganizationMap{
		UserID:         user.ID,
		OrganizationID: meta.Organization.ID,
		Owner:          meta.Body.Owner,
	}

	output.Message = "OK"
	output.Success = true
	output.Data.UserID = fmt.Sprintf("%d", user.ID)
	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userOrganizationMapping).
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
