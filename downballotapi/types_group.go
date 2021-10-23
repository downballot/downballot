package downballotapi

// CreateGroupRequest is the request to create a group.
type CreateGroupRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
	Filter   string `json:"filter"`
}

// CreateGroupResponse is the response from creating a group
type CreateGroupResponse Group

// ListGroupsResponse is the response from listing the groups.
type ListGroupsResponse struct {
	Groups []*Group `json:"groups"`
}

// GetGroupResponse is the response from getting the group.
type GetGroupResponse struct {
	Group *Group `json:"group"`
}

// Group is a group.
type Group struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Filter   string `json:"filter"`
}
