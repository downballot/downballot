package api

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/resttype"
	"github.com/downballot/downballot/internal/schema"
	"github.com/tekkamanendless/restfulwrapper"
)

type GetOrganizationIDGroupPersonCountMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	downballotwrapper.RequirePermissionGroupRead
	_        string              `api:"httppath:/organization/{organization_id}/group-person-count"`
	_        string              `api:"doc" description:"Get the person count for each group."`
	_        string              `api:"notes" description:"This gets the person count for each group."`
	Filter   *string             `api:"query:filter"`
	GroupIDs resttype.StringList `api:"query:group_ids"`
}

func (a *API) GetOrganizationIDGroupPersonCount(ctx context.Context, meta GetOrganizationIDGroupPersonCountMetadata) (output downballotapi.Envelope[downballotapi.GetGroupPersonCountResponse], err error) {
	groups, err := getGroupsForUser(meta.DB, meta.CurrentUser.ID, meta.OrganizationID)
	if err != nil {
		return output, err
	}

	var groupIDs []uint64
	for _, groupID := range meta.GroupIDs {
		index := slices.IndexFunc(groups, func(g *schema.Group) bool {
			return fmt.Sprintf("%v", g.ID) == groupID
		})
		if index < 0 {
			return output, restfulwrapper.NewAPIQueryParameterError("group_ids", fmt.Errorf("invalid group_id: %s", groupID))
		}
		groupIDs = append(groupIDs, groups[index].ID)
	}
	slog.InfoContext(ctx, fmt.Sprintf("Initial group IDs: (%d)", len(meta.GroupIDs)))
	slog.InfoContext(ctx, fmt.Sprintf("Initial group IDs: %v", meta.GroupIDs))
	slog.InfoContext(ctx, fmt.Sprintf("Final group IDs: (%d)", len(groupIDs)))

	groupIDToCountMap, err := filterPersonsCount(ctx, meta.DB, meta.CurrentUser.ID, meta.Organization.ID, groupIDs, meta.Filter)
	if err != nil {
		return output, err
	}

	groupIDToGroupMap := map[uint64]*schema.Group{}
	for _, group := range groups {
		groupIDToGroupMap[group.ID] = group
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Groups = make([]*downballotapi.GroupPersonCount, 0, len(groupIDToCountMap))
	for groupID, count := range groupIDToCountMap {
		group := groupIDToGroupMap[groupID]
		output.Data.Groups = append(output.Data.Groups, &downballotapi.GroupPersonCount{
			ID:       fmt.Sprintf("%d", groupID),
			ParentID: fmt.Sprintf("%d", group.ParentID),
			Name:     group.Name,
			Filter:   group.Filter,
			Count:    count,
		})
	}

	return output, nil
}
