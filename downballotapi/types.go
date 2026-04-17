package downballotapi

import "encoding/json"

// Envelope wraps all responses.
type Envelope[T any] struct {
	Message string `json:"message" description:"This is the error message."`
	Success bool   `json:"success" description:"This will always be false to indicate an error."`
	Data    T      `json:"data,omitempty" description:"This is the contents of the response."`
}

type RawEnvelope Envelope[json.RawMessage]
