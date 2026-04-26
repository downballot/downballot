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

type PostOrganizationIDGroupMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_    string                           `api:"httppath:/organization/{organization_id}/group"`
	_    string                           `api:"doc" description:"Create a new group."`
	_    string                           `api:"notes" description:"This creates a new group."`
	Body downballotapi.CreateGroupRequest `api:"body"`
}

func (a *API) PostOrganizationIDGroup(ctx context.Context, meta PostOrganizationIDGroupMetadata) (output downballotapi.Envelope[downballotapi.CreateGroupResponse], err error) {
	if meta.Body.Name == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing name"))
	}
	if meta.Body.ParentID == "" {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("missing parent_id"))
	}

	groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID, nil)
	if err != nil {
		return output, err
	}
	var parentGroup *schema.Group
	for _, g := range groups {
		if fmt.Sprintf("%v", g.ID) == meta.Body.ParentID {
			parentGroup = g
			break
		}
	}
	if parentGroup == nil {
		return output, restfulwrapper.NewAPIBodyError(fmt.Errorf("invalid parent_id"))
	}

	var owner schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("id = ?", meta.CurrentUser.ID).
		First(&owner).
		Error
	if err != nil {
		return output, err
	}

	group := schema.Group{
		OrganizationID: meta.Organization.ID,
		Name:           meta.Body.Name,
		ParentID:       &parentGroup.ID,
		Filter:         meta.Body.Filter,
	}

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&group).
			Error
		if err != nil {
			return err
		}

		output.Data.ID = fmt.Sprintf("%d", group.ID)
		if group.ParentID != nil {
			output.Data.ParentID = fmt.Sprintf("%d", *group.ParentID)
		}
		output.Data.Name = group.Name
		output.Data.Filter = meta.Body.Filter

		userGroupMapping := schema.UserGroupMap{
			UserID:  owner.ID,
			GroupID: group.ID,
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&userGroupMapping).
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

type GetOrganizationIDGroupMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_        string  `api:"httppath:/organization/{organization_id}/group"`
	_        string  `api:"doc" description:"List the groups."`
	_        string  `api:"notes" description:"This lists the groups."`
	Name     *string `api:"query:name"`
	ParentID *string `api:"query:parent_id"`
}

func (a *API) GetOrganizationIDGroup(ctx context.Context, meta GetOrganizationIDGroupMetadata) (output downballotapi.Envelope[downballotapi.ListGroupsResponse], err error) {
	filters := map[string]any{}
	if meta.Name != nil {
		filters["name"] = *meta.Name
	}
	if meta.ParentID != nil {
		filters["parent_id"] = *meta.ParentID
	}
	groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID, filters)
	if err != nil {
		return output, err
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Groups = []*downballotapi.Group{}
	for _, group := range groups {
		o := &downballotapi.Group{
			ID:     fmt.Sprintf("%d", group.ID),
			Name:   group.Name,
			Filter: group.Filter,
		}
		if group.ParentID != nil {
			o.ParentID = fmt.Sprintf("%d", *group.ParentID)
		}
		output.Data.Groups = append(output.Data.Groups, o)
	}
	return output, nil
}
