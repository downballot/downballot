package downballotapi

// RegisterUserRequest is the request to register a user.
type RegisterUserRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterUserResponse is the response from registering a user
type RegisterUserResponse User

// ListUsersResponse is the response from listing the users.
type ListUsersResponse struct {
	Users []*User `json:"users"`
}

// User is an user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}
