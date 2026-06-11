package downballotapi

// RegisterUserRequest is the request to register a user.
type RegisterUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
}

// RegisterUserResponse is the response from registering a user
type RegisterUserResponse User

// ListUsersResponse is the response from listing the users.
type ListUsersResponse struct {
	Users []*User `json:"users"`
}

// GetUserResponse is the response from getting a user.
type GetUserResponse struct {
	User *User `json:"user"`
}

// PatchOrganizationUserRequest is the request for patching an organization user.
type PatchOrganizationUserRequest struct {
	Owner *bool `json:"owner"`
}

// PatchOrganizationUserResponse is the response from patching an organization user.
type PatchOrganizationUserResponse struct {
	User User `json:"user"`
}

// User is an user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Owner    bool   `json:"owner"` // Whether the user is an owner of the organization.
}
