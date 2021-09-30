package downballotapi

// AuthenticationStatusResponse is the authentication status response.
type AuthenticationStatusResponse struct {
	User *AuthenticationStatusUser `json:"user"`
}

// AuthenticationStatusUser is the current user information.
type AuthenticationStatusUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
}

// CreateAccountRequest is used to create an account.
type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// CreateAccountResponse is the response from creating an account.
type CreateAccountResponse struct {
	Email string `json:"email"`
}

// LoginRequest is used to sign in with an account.
type LoginRequest struct {
	Username string `json:"username" description:"(username/password) The username."`
	Password string `json:"password" description:"(username/password) The password."`
}

// LoginResponse is the response from signing in with an account.
type LoginResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// ResetPasswordRequest is used to reset a user's password.
type ResetPasswordRequest struct {
	Username string `json:"username"`
}

// ResetPasswordResponse is the response from resetting a user's password.
type ResetPasswordResponse struct {
	Email string `json:"email"`
}
