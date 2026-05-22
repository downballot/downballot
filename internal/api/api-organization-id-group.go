package api

import (
	"context"
	"fmt"
	"strings"

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

	groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID)
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
	var groups []*schema.Group
	{
		groupHierarchies, err := getGroupHierarchiesForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID)
		if err != nil {
			return output, err
		}

		/*
			slog.Info("groupHierarchies", "groupHierarchies", len(groupHierarchies))
			for _, groupHierarchy := range groupHierarchies {
				slog.Info("group", "group", groupHierarchy[len(groupHierarchy)-1].Name, "id", groupHierarchy[len(groupHierarchy)-1].ID)
			}
			//*/

		userRootGroupHierarchies := condenseHierarchies(groupHierarchies)

		/*
			slog.Info("userRootGroupHierarchies", "userRootGroupHierarchies", len(userRootGroupHierarchies))
			for _, userRootGroupHierarchy := range userRootGroupHierarchies {
				slog.Info("group", "group", userRootGroupHierarchy[len(userRootGroupHierarchy)-1].Name, "id", userRootGroupHierarchy[len(userRootGroupHierarchy)-1].ID)
			}
			//*/

		var groupsToConsider []*schema.Group
		if meta.ParentID != nil {
			if *meta.ParentID == "null" || *meta.ParentID == "0" {
				for _, hierarchy := range userRootGroupHierarchies {
					groupsToConsider = append(groupsToConsider, hierarchy[len(hierarchy)-1])
				}
			} else {
				for _, hierarchy := range groupHierarchies {
					group := hierarchy[len(hierarchy)-1]
					if group.ParentID != nil && fmt.Sprintf("%d", *group.ParentID) == *meta.ParentID {
						groupsToConsider = append(groupsToConsider, group)
					}
				}
			}
		} else {
			for _, hierarchy := range groupHierarchies {
				group := hierarchy[len(hierarchy)-1]
				groupsToConsider = append(groupsToConsider, group)
			}
		}

		for _, group := range groupsToConsider {
			if meta.Name != nil && !strings.EqualFold(group.Name, *meta.Name) {
				continue
			}
			groups = append(groups, group)
		}
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
