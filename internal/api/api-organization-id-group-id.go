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

type PatchOrganizationIDGroupIDMetadata struct {
	restfulwrapper.HTTPMethodPATCH
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_    string                          `api:"httppath:/organization/{organization_id}/group/{group_id}"`
	_    string                          `api:"doc" description:"Patch the group."`
	_    string                          `api:"notes" description:"This patches the group."`
	Body downballotapi.PatchGroupRequest `api:"body"`
}

func (a *API) PatchOrganizationIDGroupID(ctx context.Context, meta PatchOrganizationIDGroupIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	updateMap := map[string]any{}
	if meta.Body.Name != nil {
		updateMap["name"] = *meta.Body.Name
	}
	if meta.Body.Filter != nil {
		updateMap["filter"] = *meta.Body.Filter
	}
	if meta.Body.ParentID != nil {
		if *meta.Body.ParentID == fmt.Sprintf("%d", meta.Group.ID) {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("group cannot be its own parent"))
		}

		groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID, nil)
		if err != nil {
			return output, err
		}
		var parentGroup *schema.Group
		for _, g := range groups {
			if fmt.Sprintf("%v", g.ID) == *meta.Body.ParentID {
				parentGroup = g
				break
			}
		}
		if parentGroup == nil {
			return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid parent_id"))
		}

		groupMap := map[uint64]bool{}
		for _, g := range groups {
			if groupMap[g.ID] {
				return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("circular parent_id"))
			}
			groupMap[g.ID] = true
		}

		updateMap["parent_id"] = parentGroup.ID
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Model(&schema.Group{}).
			Where("id = ?", meta.Group.ID).
			Updates(updateMap).
			Error
		if err != nil {
			return err
		}

		var group schema.Group
		err = tx.Session(&gorm.Session{}).
			Where("id = ?", meta.Group.ID).
			First(&group).
			Error
		if err != nil {
			return err
		}

		output.Message = "OK"
		output.Success = true
		output.Data.Group = &downballotapi.Group{
			ID:     fmt.Sprintf("%d", group.ID),
			Name:   group.Name,
			Filter: group.Filter,
		}
		if group.ParentID != nil {
			output.Data.Group.ParentID = fmt.Sprintf("%d", *group.ParentID)
		}
		return nil
	})
	if err != nil {
		return output, err
	}
	return output, nil
}
