package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/threatmate/restfulwrapper"
)

type GetOrganizationIDGroupIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	_              string `api:"httppath:/organization/{organization_id}/group/{group_id}"`
	_              string `api:"doc" description:"Get the group."`
	_              string `api:"notes" description:"This gets the group."`
	OrganizationID string `api:"path:organization_id"`
	GroupID        string `api:"path:group_id"`
}

func (a *API) GetOrganizationIDGroupID(ctx context.Context, meta GetOrganizationIDGroupIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	organization, err := getOrganizationForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID)
	if err != nil {
		return output, err
	}
	if organization == nil {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	filters := map[string]interface{}{}
	if meta.GroupID == "root" {
		filters["parent_id"] = nil
	} else {
		filters["id"] = meta.GroupID
	}
	groups, err := getGroupsForUser(a.App.DB(), meta.CurrentUserID, meta.OrganizationID, filters)
	if err != nil {
		return output, err
	}
	if len(groups) == 0 {
		return output, restfulwrapper.NewAPIResponseError(http.StatusNotFound, "")
	}

	group := groups[0]
	o := &downballotapi.Group{
		ID:   fmt.Sprintf("%d", group.ID),
		Name: group.Name,
	}
	if group.ParentID != nil {
		o.ParentID = fmt.Sprintf("%d", *group.ParentID)
	}
	output.Data.Group = o
	return output, nil
}
