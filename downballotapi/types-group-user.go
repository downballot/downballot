package downballotapi

// ListGroupUsersResponse is the response from listing the group users.
type ListGroupUsersResponse struct {
	GroupUsers []*GroupUser `json:"group_users"`
}

// GroupUser is a group user.
type GroupUser struct {
	User
	Owner bool `json:"owner"`
}

type GetGroupUserResponse struct {
	GroupUser GroupUser `json:"group_user"`
}

// PatchGroupUserRequest is the request for patching the group user.
type PatchGroupUserRequest struct {
	Owner *bool `json:"owner"`
}

// PatchGroupResponse is the response from patching the group.
type PatchGroupUserResponse GroupUser
