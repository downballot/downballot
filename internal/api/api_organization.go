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
			Create(&organization).Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			return err
		}

		output.ID = fmt.Sprintf("%d", organization.ID)
		output.Name = organization.Name

		mapping := schema.UserOrganizationMap{
			UserID:         owner.ID,
			OrganizationID: organization.ID,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&mapping).
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
	err := i.App.DB.Session(&gorm.Session{NewDB: true}).
		Find(&organizations).Error
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
