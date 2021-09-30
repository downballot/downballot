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
