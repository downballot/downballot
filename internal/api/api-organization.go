package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type GetOrganizationMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	_ string `api:"httppath:/organization"`
	_ string `api:"doc" description:"List the organizations."`
	_ string `api:"notes" description:"This lists the organizations."`
}

func (a *API) GetOrganization(ctx context.Context, meta GetOrganizationMetadata) (output downballotapi.Envelope[downballotapi.ListOrganizationsResponse], err error) {
	var organizations []*schema.Organization
	query := a.App.DB().Session(&gorm.Session{NewDB: true})
	if meta.CurrentUserID != "0" { // "0" is the system token.
		query = query.
			Where("id IN (?)", a.App.DB().Session(&gorm.Session{NewDB: true}).
				Table(schema.UserOrganizationMap{}.TableName()).
				Select("id").
				Where("user_id = ?", meta.CurrentUserID),
			)
	}
	err = query.
		Find(&organizations).
		Error
	if err != nil {
		return output, err
	}

	output.Data.Organizations = []*downballotapi.Organization{}
	for _, organization := range organizations {
		o := &downballotapi.Organization{
			ID:   fmt.Sprintf("%d", organization.ID),
			Name: organization.Name,
		}
		output.Data.Organizations = append(output.Data.Organizations, o)
	}
	return output, nil
}
