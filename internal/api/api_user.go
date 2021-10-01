package api

import (
	"fmt"
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/schema"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func (i *Instance) registerUserEndpoints(ws *restful.WebService) {
	ws.Route(
		ws.POST("user/register").To(i.registerUser).
			Doc(`Register a new user`).
			Notes(`This registers a new user.`).
			Do(i.doRequireAuthentication).
			Reads(downballotapi.RegisterUserRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.RegisterUserResponse{}),
	)
	ws.Route(
		ws.GET("user").To(i.listUsers).
			Doc(`List the users`).
			Notes(`This lists the users.`).
			Do(i.doRequireAuthentication).
			Returns(http.StatusOK, "OK", downballotapi.ListUsersResponse{}),
	)
}

func (i *Instance) registerUser(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var input downballotapi.RegisterUserRequest
	err := request.ReadEntity(&input)
	if err != nil {
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	if input.Name == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing name")
		return
	}
	if input.Username == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing username")
		return
	}
	if input.Password == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing password")
		return
	}

	{
		var testUsers []*schema.User
		err = i.App.DB.Session(&gorm.Session{NewDB: true}).
			Where(map[string]interface{}{
				"username": input.Username,
			}).
			Find(&testUsers).Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
			return
		}
		if len(testUsers) > 0 {
			WriteHeaderAndText(ctx, response, http.StatusConflict, "Already taken username")
			return
		}
	}

	user := schema.User{
		Name:     input.Name,
		Username: input.Username,
	}

	output := downballotapi.RegisterUserResponse{
		// TODO
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = i.App.DB.Session(&gorm.Session{NewDB: true}).
			Create(&user).Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		output.ID = fmt.Sprintf("%d", user.ID)
		output.Name = user.Name
		return nil
	})
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	WriteEntity(ctx, response, output)
}

func (i *Instance) listUsers(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var users []*schema.User
	err := i.App.DB.Session(&gorm.Session{NewDB: true}).
		Find(&users).Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	output := downballotapi.ListUsersResponse{
		Users: []*downballotapi.User{},
	}
	for _, user := range users {
		u := &downballotapi.User{
			ID:       fmt.Sprintf("%d", user.ID),
			Name:     user.Name,
			Username: user.Username,
		}
		output.Users = append(output.Users, u)
	}
	WriteEntity(ctx, response, output)
}
