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

type GetOrganizationIDUserMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	_              string `api:"httppath:/organization/{organization_id}/user"`
	_              string `api:"doc" description:"List the users."`
	_              string `api:"notes" description:"This lists the users."`
	OrganizationID string `api:"path:organization_id"`
}

func (a *API) GetOrganizationIDUser(ctx context.Context, meta GetOrganizationIDUserMetadata) (output downballotapi.Envelope[downballotapi.ListUsersResponse], err error) {
	organization, err := getOrganizationForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID)
	if err != nil {
		return output, fmt.Errorf("could not get organization: %w", err)
	}
	if organization == nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	var users []*schema.User
	err = a.App.DB().Session(&gorm.Session{NewDB: true}).
		Where("id IN (SELECT user_id FROM user_organization_map WHERE organization_id = ?)", organization.ID).
		Find(&users).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find users: %w", err)
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Users = []*downballotapi.User{}
	for _, user := range users {
		u := &downballotapi.User{
			ID:       fmt.Sprintf("%d", user.ID),
			Name:     user.Name,
			Username: user.Username,
		}
		output.Data.Users = append(output.Data.Users, u)
	}
	return output, nil
}

type PostOrganizationIDUserMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	_              string                                     `api:"httppath:/organization/{organization_id}/user"`
	_              string                                     `api:"doc" description:"Add a user to an organization."`
	_              string                                     `api:"notes" description:"This adds a user to an organization."`
	Body           downballotapi.AddUserToOrganizationRequest `api:"body"`
	OrganizationID string                                     `api:"path:organization_id"`
}

func (a *API) PostOrganizationIDUser(ctx context.Context, meta PostOrganizationIDUserMetadata) (output downballotapi.Envelope[downballotapi.AddUserToOrganizationResponse], err error) {
	organization, err := getOrganizationForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID)
	if err != nil {
		return output, err
	}
	if organization == nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	if meta.Body.Username == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing username"))
	}

	var user schema.User
	err = a.App.DB().Session(&gorm.Session{NewDB: true}).
		Where("username = ?", meta.Body.Username).
		First(&user).
		Error
	if err != nil {
		return output, err
	}

	userOrganizationMapping := schema.UserOrganizationMap{
		UserID:         user.ID,
		OrganizationID: organization.ID,
	}

	output.Data.UserID = fmt.Sprintf("%d", user.ID)
	err = a.App.DB().Transaction(func(tx *gorm.DB) error {
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
