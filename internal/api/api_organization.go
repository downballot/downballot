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

func (i *Instance) registerOrganizationEndpoints(ws *restful.WebService) {
	ws.Route(
		ws.POST("organization/register").To(i.registerOrganization).
			Doc(`Register a new organization`).
			Notes(`This registers a new organization.`).
			Do(i.doRequireAuthentication).
			Reads(downballotapi.RegisterOrganizationRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.RegisterOrganizationResponse{}),
	)
	ws.Route(
		ws.GET("organization").To(i.listOrganizations).
			Doc(`List the organizations`).
			Notes(`This lists the organizations.`).
			Do(i.doRequireAuthentication).
			Returns(http.StatusOK, "OK", downballotapi.ListOrganizationsResponse{}),
	)
	ws.Route(
		ws.POST("organization/{organization_id}/user").To(i.addUserToOrganization).
			Doc(`Add a user to an organization`).
			Notes(`This adds a user to an organization.`).
			Do(i.doRequireAuthentication).
			Param(restful.PathParameter("organization_id", "The organization ID.")).
			Reads(downballotapi.AddUserToOrganizationRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.AddUserToOrganizationResponse{}),
	)
	ws.Route(
		ws.POST("organization/{organization_id}/user/{user_id}/group").To(i.addUserToGroup).
			Doc(`Add a user to a group`).
			Notes(`This adds a user to a group.`).
			Do(i.doRequireAuthentication).
			Param(restful.PathParameter("organization_id", "The organization ID.")).
			Param(restful.PathParameter("user_id", "The user ID.")).
			Reads(downballotapi.AddUserToGroupRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.AddUserToGroupResponse{}),
	)
}

func (i *Instance) registerOrganization(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var input downballotapi.RegisterOrganizationRequest
	err := request.ReadEntity(&input)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	if input.Name == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing name")
		return
	}
	if input.OwnerID == "" {
		if request.Attribute(AttributeUserID) == nil {
			WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing owner_id")
			return
		}
		input.OwnerID = fmt.Sprintf("%v", request.Attribute(AttributeUserID))
		logrus.WithContext(ctx).Infof("No owner ID given; using current user: %s", input.OwnerID)
	}
	if input.OwnerID != "" && request.Attribute(AttributeUserID) != nil {
		if fmt.Sprintf("%v", request.Attribute(AttributeUserID)) != input.OwnerID {
			WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Mismatched owner_id")
			return
		}
	}

	var owner schema.User
	err = i.App.DB.Session(&gorm.Session{NewDB: true}).
		Where("id = ?", input.OwnerID).
		First(&owner).
		Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	organization := schema.Organization{
		Name: input.Name,
	}

	output := downballotapi.RegisterOrganizationResponse{
		// TODO
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&organization).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		output.ID = fmt.Sprintf("%d", organization.ID)
		output.Name = organization.Name

		userOrganizationMapping := schema.UserOrganizationMap{
			UserID:         owner.ID,
			OrganizationID: organization.ID,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userOrganizationMapping).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		group := schema.Group{
			OrganizationID: organization.ID,
			Name:           "Root",
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&group).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		userGroupMapping := schema.UserGroupMap{
			UserID:  owner.ID,
			GroupID: group.ID,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userGroupMapping).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}
		return nil
	})
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	WriteEntity(ctx, response, output)
}

func (i *Instance) listOrganizations(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var organizations []*schema.Organization
	query := i.App.DB.Session(&gorm.Session{NewDB: true})
	if request.Attribute(AttributeUserID) != nil {
		query = query.Where("id IN (?)", i.App.DB.Session(&gorm.Session{NewDB: true}).Table(schema.UserOrganizationMap{}.TableName()).Select("id").Where("user_id = ?", request.Attribute(AttributeUserID)))
	}
	err := query.
		Find(&organizations).
		Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	output := downballotapi.ListOrganizationsResponse{
		Organizations: []*downballotapi.Organization{},
	}
	for _, organization := range organizations {
		o := &downballotapi.Organization{
			ID:   fmt.Sprintf("%d", organization.ID),
			Name: organization.Name,
		}
		output.Organizations = append(output.Organizations, o)
	}
	WriteEntity(ctx, response, output)
}

func (i *Instance) addUserToOrganization(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	organizationIDString := request.PathParameter("organization_id")
	organization, err := getOrganizationForUser(i.App.DB, request.Attribute(AttributeUserID), organizationIDString)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}
	if organization == nil {
		WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input downballotapi.AddUserToOrganizationRequest
	err = request.ReadEntity(&input)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	if input.Username == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing username")
		return
	}

	var user schema.User
	err = i.App.DB.Session(&gorm.Session{NewDB: true}).
		Where("username = ?", input.Username).
		First(&user).
		Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	userOrganizationMapping := schema.UserOrganizationMap{
		UserID:         user.ID,
		OrganizationID: organization.ID,
	}

	output := downballotapi.AddUserToOrganizationResponse{
		UserID: fmt.Sprintf("%d", user.ID),
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userOrganizationMapping).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		return nil
	})
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	WriteEntity(ctx, response, output)
}

func (i *Instance) addUserToGroup(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	organizationIDString := request.PathParameter("organization_id")
	organization, err := getOrganizationForUser(i.App.DB, request.Attribute(AttributeUserID), organizationIDString)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}
	if organization == nil {
		WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userIDString := request.PathParameter("user_id")
	var user *schema.User
	{
		users, err := getUsersForOrganization(i.App.DB, organization.ID, map[string]interface{}{"id": userIDString})
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
			return
		}
		if len(users) > 0 {
			user = users[0]
		}
	}
	if user == nil {
		WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var input downballotapi.AddUserToGroupRequest
	err = request.ReadEntity(&input)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	if input.GroupID == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing group_id")
		return
	}
	var group *schema.Group
	{
		groups, err := getGroupsForUser(i.App.DB, request.Attribute(AttributeUserID), organizationIDString, nil)
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
			return
		}
		for _, g := range groups {
			if fmt.Sprintf("%v", g.ID) == input.GroupID {
				group = g
				break
			}
		}
	}
	if group == nil {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Invalid group_id")
		return
	}

	userGroupMapping := schema.UserGroupMap{
		UserID:  user.ID,
		GroupID: group.ID,
	}

	output := downballotapi.AddUserToGroupResponse{
		GroupID: fmt.Sprintf("%d", group.ID),
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userGroupMapping).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		return nil
	})
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	WriteEntity(ctx, response, output)
}
