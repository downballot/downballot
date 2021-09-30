package api

// Error is an API error.
// These are serializable to JSON for compatibility so that even error responses
// come back as proper JSON.
type Error struct {
	Code    int    `json:"code,omitempty" description:"This is the error code, if any."`
	Message string `json:"message" description:"This is the error message."`
}

// NewError returns a new Error instance from a standard Go error.
func NewError(err error) *Error {
	return &Error{
		Message: err.Error(),
	}
}
