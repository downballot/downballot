package downballotapi

// RegisterOrganizationRequest is the request to register an organization.
type RegisterOrganizationRequest struct {
	Name    string `json:"name"`
	OwnerID string `json:"owner_id"`
}

// RegisterOrganizationResponse is the response from registering an organization
type RegisterOrganizationResponse Organization

// ListOrganizationsResponse is the response from listing the organizations.
type ListOrganizationsResponse struct {
	Organizations []*Organization `json:"organizations"`
}

// Organization is an organization.
type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AddUserToOrganizationRequest TODO
type AddUserToOrganizationRequest struct {
	Username string `json:"username"`
}

// AddUserToOrganizationResponse TODO
type AddUserToOrganizationResponse struct {
	UserID string `json:"user_id"`
}

// AddUserToGroupRequest TODO
type AddUserToGroupRequest struct {
	GroupID string `json:"group_id"`
}

// AddUserToGroupResponse TODO
type AddUserToGroupResponse struct {
	GroupID string `json:"group_id"`
}
