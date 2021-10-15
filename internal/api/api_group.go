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

func (i *Instance) registerGroupEndpoints(ws *restful.WebService) {
	ws.Route(
		ws.POST("organization/{organization_id}/group").To(i.createGroup).
			Doc(`Create a new group`).
			Notes(`This creates a new group.`).
			Do(i.doRequireAuthentication).
			Param(restful.PathParameter("organization_id", "The organization ID.")).
			Reads(downballotapi.CreateGroupRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.CreateGroupResponse{}),
	)
	ws.Route(
		ws.GET("organization/{organization_id}/group").To(i.listGroups).
			Doc(`List the groups`).
			Notes(`This lists the groups.`).
			Do(i.doRequireAuthentication).
			Param(restful.PathParameter("organization_id", "The organization ID.")).
			Returns(http.StatusOK, "OK", downballotapi.ListGroupsResponse{}),
	)
}

func (i *Instance) createGroup(request *restful.Request, response *restful.Response) {
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

	var input downballotapi.CreateGroupRequest
	err = request.ReadEntity(&input)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	if input.Name == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing name")
		return
	}
	if input.ParentID == "" {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Missing name")
		return
	}

	groups, err := getGroupsForUser(i.App.DB, request.Attribute(AttributeUserID), organizationIDString)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}
	var parentGroup *schema.Group
	for _, g := range groups {
		if fmt.Sprintf("%v", g.ID) == input.ParentID {
			parentGroup = g
			break
		}
	}
	if parentGroup == nil {
		WriteHeaderAndText(ctx, response, http.StatusBadRequest, "Invalid parent_id")
		return
	}

	var owner schema.User
	err = i.App.DB.Session(&gorm.Session{NewDB: true}).
		Where("id = ?", request.Attribute(AttributeUserID)).
		First(&owner).
		Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	group := schema.Group{
		Name: input.Name,
	}

	output := downballotapi.CreateGroupResponse{
		// TODO
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&group).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		output.ID = fmt.Sprintf("%d", group.ID)
		output.ParentID = fmt.Sprintf("%d", group.ParentID)
		output.Name = group.Name

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

func (i *Instance) listGroups(request *restful.Request, response *restful.Response) {
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

	groups, err := getGroupsForUser(i.App.DB, request.Attribute(AttributeUserID), organizationIDString)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	output := downballotapi.ListGroupsResponse{
		Groups: []*downballotapi.Group{},
	}
	for _, group := range groups {
		o := &downballotapi.Group{
			ID:   fmt.Sprintf("%d", group.ID),
			Name: group.Name,
		}
		if group.ParentID != nil {
			o.ParentID = fmt.Sprintf("%d", group.ParentID)
		}
		output.Groups = append(output.Groups, o)
	}
	WriteEntity(ctx, response, output)
}
