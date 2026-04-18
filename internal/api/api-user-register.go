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

// PostUserRegister does not accept authentication, since this is what makes an account in the first place.
type PostUserMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.UseDatabase
	_    string                            `api:"httppath:/user"`
	_    string                            `api:"doc" description:"Register a new user."`
	_    string                            `api:"notes" description:"This registers a new user."`
	Body downballotapi.RegisterUserRequest `api:"body"`
}

func (a *API) PostUser(ctx context.Context, meta PostUserMetadata) (output downballotapi.Envelope[downballotapi.RegisterUserResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}
	if meta.Body.Username == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing username"))
	}
	if meta.Body.Password == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing password"))
	}

	{
		var testUsers []*schema.User
		err = meta.DB.Session(&gorm.Session{}).
			Where("username = ?", meta.Body.Username).
			Find(&testUsers).
			Error
		if err != nil {
			return output, fmt.Errorf("could not search for existing users: %w", err)
		}
		if len(testUsers) > 0 {
			return output, restfulwrapper.NewAPIResponseError(http.StatusConflict, "Username already taken")
		}
	}

	output.Message = "OK"
	output.Success = true
	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		user := schema.User{
			Name:     meta.Body.Name,
			Username: meta.Body.Username,
		}

		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&user).
			Error
		if err != nil {
			return fmt.Errorf("could not create user: %w", err)
		}

		output.Data.ID = fmt.Sprintf("%d", user.ID)
		output.Data.Name = user.Name
		output.Data.Username = user.Username
		return nil
	})
	if err != nil {
		return output, fmt.Errorf("could not execute transaction: %w", err)
	}

	return output, nil
}
