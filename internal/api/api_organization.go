package api

import (
	"fmt"
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/schema"
	restful "github.com/emicklei/go-restful/v3"
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
	}

	organization := schema.Organization{
		Name: input.Name,
	}

	output := downballotapi.RegisterOrganizationResponse{
		// TODO
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
		err = i.App.DB.Session(&gorm.Session{NewDB: true}).
			Create(&organization).Error
		if err != nil {
			return err
		}

		output.ID = fmt.Sprintf("%d", organization.ID)
		output.Name = organization.Name
		return nil
	})
	if err != nil {
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
