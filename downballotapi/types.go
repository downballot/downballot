package downballotapi

import "encoding/json"

// Envelope wraps all responses.
type Envelope struct {
	Message string          `json:"message" description:"This is the error message."`
	Success bool            `json:"success" description:"This will always be false to indicate an error."`
	Data    json.RawMessage `json:"data,omitempty" description:"This is the contents of the response."`
}
