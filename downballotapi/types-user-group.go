package downballotapi

// ListUserGroupsResponse is the response from listing the user groups.
type ListUserGroupsResponse struct {
	UserGroups []*UserGroup `json:"user_groups"`
}

// GetUserGroupResponse is the response from getting a user group.
type GetUserGroupResponse struct {
	UserGroup *UserGroup `json:"user_group"`
}

// UserGroup is a group that the user is in.
type UserGroup struct {
	Group
	Owner bool `json:"owner"` // Whether the user is an owner of the group.
}
