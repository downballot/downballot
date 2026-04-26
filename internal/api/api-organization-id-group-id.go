package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
)

type hasGroup struct {
	GroupID string       `api:"path:group_id" description:"The group ID"`
	Group   schema.Group `api:"database.query:where:id = ? AND organization_id IN (SELECT id FROM organization),GroupID"`
}

type GetOrganizationIDGroupIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_ string `api:"httppath:/organization/{organization_id}/group/{group_id}"`
	_ string `api:"doc" description:"Get the group."`
	_ string `api:"notes" description:"This gets the group."`
}

func (a *API) GetOrganizationIDGroupID(ctx context.Context, meta GetOrganizationIDGroupIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	o := &downballotapi.Group{
		ID:     fmt.Sprintf("%d", meta.Group.ID),
		Name:   meta.Group.Name,
		Filter: meta.Group.Filter,
	}
	if meta.Group.ParentID != nil {
		o.ParentID = fmt.Sprintf("%d", *meta.Group.ParentID)
	}
	output.Message = "OK"
	output.Success = true
	output.Data.Group = o
	return output, nil
}

type GetOrganizationIDGroupRootMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	Group schema.Group `api:"database.query:where:parent_id IS NULL AND organization_id = ?, OrganizationID"`
	_     string       `api:"httppath:/organization/{organization_id}/group/root"`
	_     string       `api:"doc" description:"Get the group."`
	_     string       `api:"notes" description:"This gets the group."`
}

func (a *API) GetOrganizationIDGroupRoot(ctx context.Context, meta GetOrganizationIDGroupRootMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	o := &downballotapi.Group{
		ID:     fmt.Sprintf("%d", meta.Group.ID),
		Name:   meta.Group.Name,
		Filter: meta.Group.Filter,
	}
	if meta.Group.ParentID != nil {
		o.ParentID = fmt.Sprintf("%d", *meta.Group.ParentID)
	}
	output.Message = "OK"
	output.Success = true
	output.Data.Group = o
	return output, nil
}
